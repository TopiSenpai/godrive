package godrive

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"golang.org/x/exp/slices"
	"io"
	"log"
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

	var (
		start *int64
		end   *int64
	)
	if rangeHeader := r.Header.Get("Range"); rangeHeader != "" {
		fmt.Println("rangeHeader:", rangeHeader)
		if _, err = fmt.Sscanf(rangeHeader, "bytes=%d-%d", start, end); err != nil {
			s.error(w, r, errors.New("invalid range header"), http.StatusRequestedRangeNotSatisfiable)
			return
		}
	}

	if len(files) == 1 && (rPath != "/" || path.Join(files[0].Dir, files[0].Name) == rPath) {
		file := files[0]
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
		if err = s.writeFile(r.Context(), w, path.Join(file.Dir, file.Name), start, end); err != nil {
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
			if file.Private {
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
		if file.Private {
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
				IsDir:       true,
				Name:        relativePath,
				Dir:         rPath,
				Size:        file.Size,
				Description: "",
				Date:        updatedAt,
			})
			continue
		}

		templateFiles = append(templateFiles, TemplateFile{
			IsDir:       false,
			Name:        file.Name,
			Dir:         file.Dir,
			Size:        file.Size,
			Description: file.Description,
			Date:        updatedAt,
		})
	}

	vars := TemplateVariables{
		Path:      rPath,
		PathParts: strings.FieldsFunc(rPath, func(r rune) bool { return r == '/' }),
		Files:     templateFiles,
		Theme:     "dark",
	}
	if err = s.tmpl(w, "index.gohtml", vars); err != nil {
		log.Println("failed to execute template:", err)
	}
}

func (s *Server) writeFile(ctx context.Context, w io.Writer, fullName string, start *int64, end *int64) error {
	obj, err := s.storage.GetObject(ctx, fullName, start, end)
	if err != nil {
		return err
	}
	if _, err = io.Copy(w, obj); err != nil {
		return err
	}
	return nil
}

func (s *Server) PostFiles(w http.ResponseWriter, r *http.Request) {
	if err := s.parseMultipart(r, func(file ParsedFile, reader io.Reader) error {
		if err := s.storage.PutObject(r.Context(), path.Join(file.Dir, file.Name), file.Size, reader, file.ContentType); err != nil {
			return err
		}

		_, err := s.db.CreateFile(r.Context(), file.Dir, file.Name, file.Size, file.ContentType, file.Description, file.Private)
		return err
	}); err != nil {
		s.error(w, r, err, http.StatusInternalServerError)
		return
	}

	s.ok(w, r, nil)
}

func (s *Server) PatchFiles(w http.ResponseWriter, r *http.Request) {
	if err := s.parseMultipart(r, func(file ParsedFile, reader io.Reader) error {
		if err := s.db.UpdateFile(r.Context(), file.Dir, file.Name, file.Size, file.ContentType, file.Description, file.Private); err != nil {
			return err
		}

		return s.storage.PutObject(r.Context(), path.Join(file.Dir, file.Name), file.Size, reader, file.ContentType)
	}); err != nil {
		s.error(w, r, err, http.StatusInternalServerError)
		return
	}

	s.ok(w, r, nil)
}

func (s *Server) DeleteFiles(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var files []string
	if err := json.NewDecoder(r.Body).Decode(&files); err != nil {
		s.error(w, r, err, http.StatusBadRequest)
		return
	}

	var finalErr error
	for _, file := range files {
		if err := s.db.DeleteFile(r.Context(), r.URL.Path, file); err != nil {
			finalErr = errors.Join(finalErr, err)
			continue
		}
		if err := s.storage.DeleteObject(r.Context(), path.Join(r.URL.Path, file)); err != nil {
			finalErr = errors.Join(finalErr, err)
			continue
		}
	}

	if finalErr != nil {
		s.error(w, r, finalErr, http.StatusInternalServerError)
		return
	}

	s.json(w, r, nil, http.StatusNoContent)
}

type ParsedFile struct {
	Dir         string
	Name        string
	Size        uint64
	ContentType string
	Description string
	Private     bool
}

func (s *Server) parseMultipart(r *http.Request, fileFunc func(file ParsedFile, reader io.Reader) error) error {
	defer r.Body.Close()

	mr, err := r.MultipartReader()
	if err != nil {
		return err
	}

	part, err := mr.NextPart()
	if err != nil {
		return err
	}

	if part.FormName() != "json" {
		return errors.New("json field not found")
	}

	var files []FileRequest
	if err = json.NewDecoder(part).Decode(&files); err != nil {
		return err
	}

	for _, file := range files {
		part, err = mr.NextPart()
		if err == io.EOF {
			return errors.New("not enough files")
		}
		if err != nil {
			return err
		}

		contentType := part.Header.Get("Content-Type")
		if contentType == "" {
			contentType = "application/octet-stream"
		}

		parsedFile := ParsedFile{
			Dir:         r.URL.Path,
			Name:        part.FileName(),
			Size:        file.Size,
			ContentType: contentType,
			Description: file.Description,
			Private:     file.Private,
		}

		if err = fileFunc(parsedFile, part); err != nil {
			return err
		}

	}
	return nil
}
