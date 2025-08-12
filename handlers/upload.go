package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

type UploadResponse struct {
	ID           string `json:"id"`
	OriginalName string `json:"original_filename"`
	URL          string `json:"url"`
}

var allowedExtensions = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true,
	".pdf": true, ".doc": true, ".docx": true,
	".xls": true, ".xlsx": true,
	".mp4": true, ".avi": true, ".mov": true, ".mkv": true,
}

func UploadFile(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(100 << 20)
	if err != nil {
		http.Error(w, "Unablle to parse form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "File is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !allowedExtensions[ext] {
		http.Error(w, "Unsupported file type", http.StatusBadRequest)
		return
	}

	dateFolder := time.Now().Format("2006-01-02")
	storagePath := filepath.Join("uploads", dateFolder)
	if err := os.MkdirAll(storagePath, os.ModePerm); err != nil {
		http.Error(w, "Failed to create folder", http.StatusInternalServerError)
		return
	}

	fileID := uuid.New().String()
	newFileName := fileID + ext
	fullPath := filepath.Join(storagePath, newFileName)

	dst, err := os.Create(fullPath)
	if err != nil {
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()
	io.Copy(dst, file)

	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	resp := UploadResponse{
		ID:           fileID,
		OriginalName: header.Filename,
		URL:          fmt.Sprintf("/api/v1/files/%s", fileID),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
