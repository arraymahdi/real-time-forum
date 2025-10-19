package post

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

var uploadDir = "uploads"

// File uploader
func SaveFile(r *http.Request, fieldName string) (string, error) {
	file, header, err := r.FormFile(fieldName)
	if err != nil {
		if err == http.ErrMissingFile {
			return "", nil
		}
		return "", err
	}
	defer file.Close()

	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		os.MkdirAll(uploadDir, os.ModePerm)
	}

	fileName := fmt.Sprintf("%d_%s", time.Now().UnixNano(), header.Filename)
	filePath := filepath.Join(uploadDir, fileName)

	dst, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		return "", err
	}

	return filePath, nil
}
