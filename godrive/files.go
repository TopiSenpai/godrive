package godrive

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"golang.org/x/exp/slices"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"
)

func (s *Server) GetFiles(w http.ResponseWriter, r *http.Request) {
	rPath := r.URL.Path

	var download bool
	if dl := r.URL.Query().Get("dl"); dl != "" && dl != "0" {
		download = true
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

	userInfo := GetUserInfo(r)
	if len(files) == 1 && path.Join(files[0].Dir, files[0].Name) == rPath {
		start, end, err := parseRange(r.Header.Get("Range"))
		if err != nil {
			s.error(w, r, err, http.StatusRequestedRangeNotSatisfiable)
			return
		}
		file := files[0]
		if !s.hasFileAccess(userInfo, file) {
			s.prettyError(w, r, errors.New("file is private"), http.StatusForbidden)
			return
		}
		if download {
			w.Header().Set("Content-Disposition", "attachment; filename="+file.Name)
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
		if err = s.writeFile(r.Context(), w, file.ObjectID, start, end); err != nil {
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
		for _, file := range files {
			if !s.hasFileAccess(userInfo, file) {
				continue
			}
			fullName := path.Join(file.Dir, file.Name)
			var fw io.Writer
			fw, err = zw.CreateHeader(&zip.FileHeader{
				Name:               strings.TrimPrefix(fullName, "/"),
				UncompressedSize64: file.Size,
				Modified:           file.UpdatedAt,
				Comment:            file.Description,
				Method:             zip.Deflate,
			})
			if err != nil {
				s.error(w, r, err, http.StatusInternalServerError)
				return
			}
			if err = s.writeFile(r.Context(), fw, fullName, nil, nil); err != nil {
				s.error(w, r, err, http.StatusInternalServerError)
				return
			}
		}
		if err = zw.SetComment("Generated by godrive"); err != nil {
			s.error(w, r, err, http.StatusInternalServerError)
			return
		}
		return
	}

	var templateFiles []TemplateFile
	for _, file := range files {
		if !s.hasFileAccess(userInfo, file) {
			continue
		}
		updatedAt := file.UpdatedAt
		if updatedAt.IsZero() {
			updatedAt = file.CreatedAt
		}

		relativePath := strings.TrimPrefix(file.Dir, rPath)
		if relativePath != "" {
			if strings.Count(relativePath, "") > 0 {
				parts := strings.Split(relativePath, "/")
				if len(parts) > 1 {
					relativePath = parts[1]
				} else {
					relativePath = parts[0]
				}
			}

			index := slices.IndexFunc(templateFiles, func(f TemplateFile) bool {
				return f.Name == relativePath
			})
			if index > -1 {
				templateFiles[index].Size += file.Size
				if templateFiles[index].Date.Before(updatedAt) {
					templateFiles[index].Date = updatedAt
				}
				continue
			}

			templateFiles = append(templateFiles, TemplateFile{
				IsDir: true,
				Name:  relativePath,
				Dir:   rPath,
				Size:  file.Size,
				Date:  updatedAt,
			})
			continue
		}

		owner := "Unknown"
		if file.Username != nil {
			owner = *file.Username
		}

		templateFiles = append(templateFiles, TemplateFile{
			IsDir:       false,
			Name:        file.Name,
			Dir:         file.Dir,
			Size:        file.Size,
			Description: file.Description,
			Private:     file.Private,
			Date:        updatedAt,
			Owner:       owner,
			IsOwner:     userInfo.Subject == file.UserID,
		})
	}

	vars := IndexVariables{
		BaseVariables: BaseVariables{
			Theme: "dark",
			Auth:  s.cfg.Auth != nil,
			User:  s.ToTemplateUser(userInfo),
		},
		Path:      rPath,
		PathParts: strings.FieldsFunc(rPath, func(r rune) bool { return r == '/' }),
		Files:     templateFiles,
	}
	if err = s.tmpl(w, "index.gohtml", vars); err != nil {
		s.log(r, "template", err)
	}
}

func (s *Server) writeFile(ctx context.Context, w io.Writer, id string, start *int64, end *int64) error {
	obj, err := s.storage.GetObject(ctx, id, start, end)
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
	id := s.newID()
	if err = s.storage.PutObject(r.Context(), id, file.Size, reader, file.ContentType); err != nil {
		s.error(w, r, err, http.StatusInternalServerError)
		return
	}

	if _, err = s.db.CreateFile(r.Context(), file.Dir, file.Name, id, file.Size, file.ContentType, file.Description, file.Private, userInfo.Subject); err != nil {
		s.error(w, r, err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) PatchFiles(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	file, reader, err := s.parseMultipart(r)
	if err != nil {
		s.error(w, r, err, http.StatusInternalServerError)
		return
	}

	defer reader.Close()
	id, err := s.db.UpdateFile(r.Context(), file.Dir, path.Base(r.URL.Path), file.Dir, file.Name, file.Size, file.ContentType, file.Description, file.Private)
	if err != nil {
		s.error(w, r, err, http.StatusInternalServerError)
		return
	}
	if file.Size > 0 {
		if err = s.storage.PutObject(r.Context(), id, file.Size, reader, file.ContentType); err != nil {
			s.error(w, r, err, http.StatusInternalServerError)
			return
		}
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

	var errs error
	for _, file := range files {
		if len(fileNames) > 0 && !slices.Contains(fileNames, file.Name) {
			continue
		}
		id, err := s.db.DeleteFile(r.Context(), file.Dir, file.Name)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		if err = s.storage.DeleteObject(r.Context(), id); err != nil {
			errs = errors.Join(errs, err)
			continue
		}
	}

	if errs != nil {
		s.error(w, r, errs, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type ParsedFile struct {
	Name        string
	Dir         string
	Size        uint64
	ContentType string
	Description string
	Private     bool
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
		dir = path.Dir(dir)
	}

	parsedFile := ParsedFile{
		Dir:         dir,
		Name:        part.FileName(),
		Size:        file.Size,
		ContentType: contentType,
		Description: file.Description,
		Private:     file.Private,
	}

	return &parsedFile, part, nil
}
