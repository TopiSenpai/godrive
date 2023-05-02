package godrive

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"

	"golang.org/x/exp/slices"
)

func (s *Server) GetFiles(w http.ResponseWriter, r *http.Request) {
	var (
		download    bool
		filesFilter []string
	)
	if dl := r.URL.Query().Get("dl"); dl != "" && dl != "0" && strings.ToLower(dl) != "false" {
		download = true
		if dl != "1" && strings.ToLower(dl) != "true" {
			filesFilter = strings.Split(dl, ",")
		}
	}

	files, err := s.db.FindFiles(r.Context(), r.URL.Path)
	if err != nil {
		s.error(w, r, err, http.StatusInternalServerError)
		return
	}

	if download && len(files) == 0 {
		s.notFound(w, r)
		return
	}
	if r.URL.Path != "/" && len(files) == 0 {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	userInfo := GetUserInfo(r)
	if len(files) == 1 && files[0].Path == r.URL.Path {
		start, end, err := parseRange(r.Header.Get("Range"))
		if err != nil {
			s.error(w, r, err, http.StatusRequestedRangeNotSatisfiable)
			return
		}
		file := files[0]
		if download {
			w.Header().Set("Content-Disposition", "attachment; filename="+path.Base(file.Path))
		}
		w.Header().Set("Content-Type", file.ContentType)
		w.Header().Set("Content-Length", strconv.FormatUint(file.Size, 10))
		w.Header().Set("Accept-Ranges", "bytes")
		if start != nil || end != nil {
			w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, file.Size))
			w.WriteHeader(http.StatusPartialContent)
		}
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusOK)
			return
		}
		if err = s.writeFile(r.Context(), w, file.Path, start, end); err != nil {
			s.log(r, "error writing file", err)
		}
		return
	}

	if download {
		zipName := path.Dir(r.URL.Path)
		if zipName == "/" || zipName == "." {
			zipName = "godrive"
		}
		w.Header().Set("Content-Disposition", "attachment; filename="+zipName+".zip")

		zw := zip.NewWriter(w)
		defer zw.Close()

		rPath := r.URL.Path
		if !strings.HasSuffix(rPath, "/") {
			rPath += "/"
		}

		var addedFiles int
		for _, file := range files {
			if len(filesFilter) > 0 && !slices.Contains(filesFilter, strings.SplitN(strings.TrimPrefix(file.Path, rPath), "/", 2)[0]) {
				continue
			}
			fw, err := zw.CreateHeader(&zip.FileHeader{
				Name:               strings.TrimPrefix(file.Path, "/"),
				UncompressedSize64: file.Size,
				Modified:           file.UpdatedAt,
				Comment:            file.Description,
				Method:             zip.Deflate,
			})
			addedFiles++
			if err != nil {
				s.error(w, r, err, http.StatusInternalServerError)
				return
			}
			if err = s.writeFile(r.Context(), fw, file.Path, nil, nil); err != nil {
				s.error(w, r, err, http.StatusInternalServerError)
				return
			}
		}
		if addedFiles == 0 {
			s.notFound(w, r)
			return
		}
		if err = zw.SetComment("Generated by godrive"); err != nil {
			s.error(w, r, err, http.StatusInternalServerError)
			return
		}
		return
	}

	var templateFiles []TemplateFile
	for _, file := range files {
		owner := "Unknown"
		if file.Username != nil {
			owner = *file.Username
		}
		isOwner := file.UserID == userInfo.Subject || s.isAdmin(userInfo)
		date := file.CreatedAt
		if file.UpdatedAt.After(date) {
			date = file.UpdatedAt
		}

		if dir := strings.TrimPrefix(path.Dir(file.Path), r.URL.Path); dir != "" {
			baseDir := strings.TrimPrefix(dir, "/")
			if strings.Count(baseDir, "/") > 0 {
				baseDir = strings.SplitN(baseDir, "/", 2)[0]
			}
			index := slices.IndexFunc(templateFiles, func(file TemplateFile) bool {
				return file.Name == baseDir
			})
			if index == -1 {
				templateFiles = append(templateFiles, TemplateFile{
					IsDir:   true,
					Path:    path.Join(r.URL.Path, baseDir),
					Dir:     r.URL.Path,
					Name:    baseDir,
					Size:    file.Size,
					Date:    date,
					Owner:   owner,
					IsOwner: isOwner,
				})
				continue
			}
			templateFiles[index].Size += file.Size
			if templateFiles[index].Date.Before(date) {
				templateFiles[index].Date = date
			}
			if !strings.Contains(templateFiles[index].Owner, owner) {
				templateFiles[index].Owner += ", " + owner
			}
			if !templateFiles[index].IsOwner && isOwner {
				templateFiles[index].IsOwner = true
			}
			continue
		}

		templateFiles = append(templateFiles, TemplateFile{
			IsDir:       false,
			Path:        file.Path,
			Name:        path.Base(file.Path),
			Dir:         path.Dir(file.Path),
			Size:        file.Size,
			Description: file.Description,
			Date:        date,
			Owner:       owner,
			IsOwner:     file.UserID == userInfo.Subject || s.isAdmin(userInfo),
		})
	}

	vars := IndexVariables{
		BaseVariables: BaseVariables{
			Theme: "dark",
			Auth:  s.cfg.Auth != nil,
			User:  s.ToTemplateUser(userInfo),
		},
		Path:      r.URL.Path,
		PathParts: strings.FieldsFunc(r.URL.Path, func(r rune) bool { return r == '/' }),
		Files:     templateFiles,
	}
	if err = s.tmpl(w, "index.gohtml", vars); err != nil {
		s.log(r, "template", err)
	}
}

func (s *Server) writeFile(ctx context.Context, w io.Writer, fullPath string, start *int64, end *int64) error {
	obj, err := s.storage.GetObject(ctx, fullPath, start, end)
	if err != nil {
		return err
	}
	if _, err = io.Copy(w, obj); err != nil {
		return err
	}
	return nil
}

func (s *Server) PostFiles(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	userInfo := GetUserInfo(r)
	if userInfo == nil {
		s.error(w, r, errors.New("unauthorized"), http.StatusUnauthorized)
		return
	}
	file, reader, err := s.parseMultipart(r)
	if err != nil {
		s.error(w, r, err, http.StatusInternalServerError)
		return
	}

	defer reader.Close()
	if err = s.storage.PutObject(r.Context(), file.Path, file.Size, reader, file.ContentType); err != nil {
		s.error(w, r, err, http.StatusInternalServerError)
		return
	}

	if _, err = s.db.CreateFile(r.Context(), file.Path, file.Size, file.ContentType, file.Description, userInfo.Subject); err != nil {
		s.error(w, r, err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) PatchFile(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	file, reader, err := s.parseMultipart(r)
	if err != nil {
		s.error(w, r, err, http.StatusInternalServerError)
		return
	}

	dbFile, err := s.db.GetFile(r.Context(), r.URL.Path)
	if err != nil {
		s.error(w, r, err, http.StatusInternalServerError)
		return
	}

	userInfo := GetUserInfo(r)
	if !s.hasFileAccess(userInfo, *dbFile) {
		s.error(w, r, errors.New("unauthorized"), http.StatusUnauthorized)
		return
	}

	defer reader.Close()
	if err = s.db.UpdateFile(r.Context(), r.URL.Path, file.Path, file.Size, file.ContentType, file.Description); err != nil {
		s.error(w, r, err, http.StatusInternalServerError)
		return
	}
	if file.Size > 0 {
		if err = s.storage.PutObject(r.Context(), file.Path, file.Size, reader, file.ContentType); err != nil {
			s.error(w, r, err, http.StatusInternalServerError)
			return
		}
		if r.URL.Path != file.Path {
			if err = s.storage.DeleteObject(r.Context(), r.URL.Path); err != nil {
				s.error(w, r, err, http.StatusInternalServerError)
				return
			}
		}
	} else if r.URL.Path != file.Path {
		if err = s.storage.MoveObject(r.Context(), r.URL.Path, file.Path); err != nil {
			s.error(w, r, err, http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) MoveFiles(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	destination := r.Header.Get("Destination")
	if destination == "" {
		s.error(w, r, errors.New("missing destination header"), http.StatusBadRequest)
		return
	}
	if destination == r.URL.Path {
		s.error(w, r, errors.New("destination cannot be the same as source"), http.StatusBadRequest)
		return
	}

	var fileNames []string
	if err := json.NewDecoder(r.Body).Decode(&fileNames); err != nil && err != io.EOF {
		s.error(w, r, err, http.StatusBadRequest)
		return
	}

	rPath := r.URL.Path
	if !strings.HasSuffix(rPath, "/") {
		rPath += "/"
	}

	files, err := s.db.FindFiles(r.Context(), r.URL.Path)
	if err != nil {
		return
	}

	userInfo := GetUserInfo(r)

	var (
		errs  error
		warns []string
	)
	for _, file := range files {
		rFilePath := strings.TrimPrefix(file.Path, rPath)
		if len(fileNames) > 0 && !slices.Contains(fileNames, strings.SplitN(rFilePath, "/", 2)[0]) {
			warns = append(warns, fmt.Sprintf("file %s not found", rFilePath))
			continue
		}
		if !s.hasFileAccess(userInfo, file) {
			warns = append(warns, fmt.Sprintf("unauthorized to move file %s", rFilePath))
			continue
		}
		newPath := path.Join(destination, rFilePath)
		if err = s.db.UpdateFile(r.Context(), file.Path, newPath, 0, "", file.Description); err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		if err = s.storage.MoveObject(r.Context(), file.Path, newPath); err != nil {
			errs = errors.Join(errs, err)
			continue
		}
	}

	if errs != nil {
		s.error(w, r, errs, http.StatusInternalServerError)
		return
	}
	if warns != nil {
		s.warn(w, r, strings.Join(warns, ", "), http.StatusPartialContent)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) DeleteFiles(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var fileNames []string
	if err := json.NewDecoder(r.Body).Decode(&fileNames); err != nil && err != io.EOF {
		s.error(w, r, err, http.StatusBadRequest)
		return
	}

	files, err := s.db.FindFiles(r.Context(), r.URL.Path)
	if err != nil {
		return
	}

	rPath := r.URL.Path
	if !strings.HasSuffix(rPath, "/") {
		rPath += "/"
	}

	userInfo := GetUserInfo(r)

	var (
		errs  error
		warns []string
	)
	for _, file := range files {
		if len(fileNames) > 0 && !slices.Contains(fileNames, strings.SplitN(strings.TrimPrefix(file.Path, rPath), "/", 2)[0]) {
			warns = append(warns, fmt.Sprintf("file %s not found", file.Path))
			continue
		}
		if !s.hasFileAccess(userInfo, file) {
			warns = append(warns, fmt.Sprintf("unauthorized to delete file %s", file.Path))
			continue
		}
		if err = s.db.DeleteFile(r.Context(), file.Path); err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		if err = s.storage.DeleteObject(r.Context(), file.Path); err != nil {
			errs = errors.Join(errs, err)
			continue
		}
	}

	if errs != nil {
		s.error(w, r, errs, http.StatusInternalServerError)
		return
	}
	if warns != nil {
		s.warn(w, r, strings.Join(warns, ", "), http.StatusPartialContent)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type ParsedFile struct {
	Path        string
	Size        uint64
	ContentType string
	Description string
}

func parseRange(rangeHeader string) (*int64, *int64, error) {
	if rangeHeader == "" {
		return nil, nil, nil
	}

	if !strings.HasPrefix(rangeHeader, "bytes=") {
		return nil, nil, errors.New("invalid range header, must start with 'bytes='")
	}

	var (
		start int64
		end   int64
	)
	if _, err := fmt.Sscanf(rangeHeader, "bytes=%d-%d", &start, &end); err == nil {
		return &start, &end, nil
	}
	if _, err := fmt.Sscanf(rangeHeader, "bytes=-%d", &end); err == nil {
		return nil, &end, nil
	}
	if _, err := fmt.Sscanf(rangeHeader, "bytes=%d-", &start); err == nil {
		return &start, nil, nil
	}

	return nil, nil, fmt.Errorf("invalid range header: %s", rangeHeader)
}

func (s *Server) parseMultipart(r *http.Request) (*ParsedFile, io.ReadCloser, error) {
	mr, err := r.MultipartReader()
	if err != nil {
		return nil, nil, err
	}

	part, err := mr.NextPart()
	if err != nil {
		return nil, nil, err
	}

	if part.FormName() != "json" {
		return nil, nil, errors.New("json field not found")
	}

	var file FileRequest
	if err = json.NewDecoder(part).Decode(&file); err != nil {
		return nil, nil, err
	}

	part, err = mr.NextPart()
	if err == io.EOF {
		return nil, nil, errors.New("not enough files")
	}
	if err != nil {
		return nil, nil, err
	}

	contentType := part.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	dir := r.URL.Path
	if r.Method == http.MethodPatch {
		dir = file.Dir
	}
	parsedFile := ParsedFile{
		Path:        path.Join(dir, part.FileName()),
		Size:        file.Size,
		ContentType: contentType,
		Description: file.Description,
	}

	return &parsedFile, part, nil
}
