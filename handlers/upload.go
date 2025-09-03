package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
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
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "error",
			"message": "Unable to parse form",
			"data":    nil,
		})
		return
	}

	var responses []UploadResponse

	files := r.MultipartForm.File["file"]
	for _, header := range files {
		ext := strings.ToLower(filepath.Ext(header.Filename))
		if !allowedExtensions[ext] {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  "error",
				"message": "Unsupported file type: " + header.Filename,
				"data":    nil,
			})
			return
		}

		dateFolder := time.Now().Format("2006-01-02")
		storagePath := filepath.Join("uploads", dateFolder)
		if err := os.MkdirAll(storagePath, os.ModePerm); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  "error",
				"message": "Failed to create folder",
				"data":    nil,
			})
			return
		}

		file, err := header.Open()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  "error",
				"message": "Failed to open file",
				"data":    nil,
			})
			return
		}
		defer file.Close()

		fileID := uuid.New().String()
		newFileName := fileID + ext
		fullPath := filepath.Join(storagePath, newFileName)

		dst, err := os.Create(fullPath)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  "error",
				"message": "Failed to save file",
				"data":    nil,
			})
			return
		}
		defer dst.Close()
		io.Copy(dst, file)

		BASE_URL := os.Getenv("BASE_URL")
		responses = append(responses, UploadResponse{
			ID:           fileID,
			OriginalName: header.Filename,
			URL:          fmt.Sprintf("%s/api/v1/files/%s", BASE_URL, fileID),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "File(s) uploaded successfully",
		"data":    responses,
	})
}

func DeleteFile(w http.ResponseWriter, r *http.Request) {
	fileID := chi.URLParam(r, "id")
	if fileID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "error",
			"message": "File ID is required",
			"data":    nil,
		})
		return
	}

	uploadsRoot := "uploads"
	var filePath string
	err := filepath.Walk(uploadsRoot, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && strings.HasPrefix(info.Name(), fileID) {
			filePath = path
			return filepath.SkipDir
		}
		return nil
	})

	if err != nil || filePath == "" {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "error",
			"message": "File not found",
			"data":    nil,
		})
		return
	}

	if removeErr := os.Remove(filePath); removeErr != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "error",
			"message": "Failed to delete file",
			"data":    nil,
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "File deleted successfully",
		"data": map[string]string{
			"id": fileID,
		},
	})
}
