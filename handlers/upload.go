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

func writeError(w http.ResponseWriter, msg string, code int) {
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "error",
		"message": msg,
		"data":    nil,
	})
}

func writeSuccess(w http.ResponseWriter, msg string, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": msg,
		"data":    data,
	})
}

func UploadFile(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(100 << 20)
	if err != nil {
		writeError(w, "Unable to parse form", http.StatusBadRequest)
		return
	}

	var responses []UploadResponse

	files := r.MultipartForm.File["file"]
	for _, header := range files {
		ext := strings.ToLower(filepath.Ext(header.Filename))
		if !allowedExtensions[ext] {
			writeError(w, "Unsupported file type: "+header.Filename, http.StatusBadRequest)
			return
		}

		dateFolder := time.Now().Format("2006-01-02")
		storagePath := filepath.Join("uploads", dateFolder)
		if err := os.MkdirAll(storagePath, os.ModePerm); err != nil {
			writeError(w, "Failed to create folder", http.StatusInternalServerError)
			return
		}

		finalID := uuid.New().String()
		finalPath := filepath.Join(storagePath, finalID+ext)

		dst, err := os.Create(finalPath)
		if err != nil {
			writeError(w, "Failed to save file", http.StatusInternalServerError)
			return
		}

		src, err := header.Open()
		if err != nil {
			dst.Close()
			writeError(w, "Failed to open uploaded file", http.StatusInternalServerError)
			return
		}

		_, err = io.Copy(dst, src)
		dst.Close()
		src.Close()

		if err != nil {
			os.Remove(finalPath)
			writeError(w, "Failed to save file", http.StatusInternalServerError)
			return
		}

		// Scan the file synchronously before adding to response
		if err := ScanWithClamAVDaemon(finalPath); err != nil {
			fmt.Printf("Virus found in %s: %v. Deleting...\n", finalPath, err)
			if removeErr := os.Remove(finalPath); removeErr != nil {
				fmt.Printf("Failed to delete %s: %v\n", finalPath, removeErr)
			}
			writeError(w, "File failed security scan: "+header.Filename, http.StatusBadRequest)
			return
		}

		BASE_URL := os.Getenv("BASE_URL")
		responses = append(responses, UploadResponse{
			ID:           finalID,
			OriginalName: header.Filename,
			URL:          fmt.Sprintf("%s/api/v1/files/%s", BASE_URL, finalID),
		})
	}

	writeSuccess(w, "File(s) uploaded successfully", responses)
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
