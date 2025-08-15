package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
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

func scanWithClamAV(filePath string) error {
	cmd := exec.Command("clamscan", "--no-summary", filePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("clamav scan error: %v", err)
	}
	if strings.Contains(string(output), "FOUND") {
		return fmt.Errorf("virus detected")
	}
	return nil
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

		tmpDir := "temp_uploads"
		if err := os.MkdirAll(tmpDir, os.ModePerm); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  "error",
				"message": "Failed to create temp folder",
				"data":    nil,
			})
			return
		}

		tempFilePath := filepath.Join(tmpDir, uuid.New().String()+ext)
		tempFile, err := os.Create(tempFilePath)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  "error",
				"message": "Failed to create temp file",
				"data":    nil,
			})
			return
		}

		src, _ := header.Open()
		io.Copy(tempFile, src)
		tempFile.Close()
		src.Close()

		// Scan file for viruses
		if err := scanWithClamAV(tempFilePath); err != nil {
			os.Remove(tempFilePath) // delete infected file
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  "error",
				"message": "File rejected: contains a virus",
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

		finalID := uuid.New().String()
		finalPath := filepath.Join(storagePath, finalID+ext)
		if err := os.Rename(tempFilePath, finalPath); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  "error",
				"message": "Failed to move file to final location",
				"data":    nil,
			})
			return
		}

		responses = append(responses, UploadResponse{
			ID:           finalID,
			OriginalName: header.Filename,
			URL:          fmt.Sprintf("/api/v1/files/%s", finalID),
		})
	}

	// 	dateFolder := time.Now().Format("2006-01-02")
	// 	storagePath := filepath.Join("uploads", dateFolder)
	// 	if err := os.MkdirAll(storagePath, os.ModePerm); err != nil {
	// 		w.WriteHeader(http.StatusInternalServerError)
	// 		json.NewEncoder(w).Encode(map[string]interface{}{
	// 			"status":  "error",
	// 			"message": "Failed to create folder",
	// 			"data":    nil,
	// 		})
	// 		return
	// 	}

	// 	file, err := header.Open()
	// 	if err != nil {
	// 		w.WriteHeader(http.StatusInternalServerError)
	// 		json.NewEncoder(w).Encode(map[string]interface{}{
	// 			"status":  "error",
	// 			"message": "Failed to open file",
	// 			"data":    nil,
	// 		})
	// 		return
	// 	}
	// 	defer file.Close()

	// 	fileID := uuid.New().String()
	// 	newFileName := fileID + ext
	// 	fullPath := filepath.Join(storagePath, newFileName)

	// 	dst, err := os.Create(fullPath)
	// 	if err != nil {
	// 		w.WriteHeader(http.StatusInternalServerError)
	// 		json.NewEncoder(w).Encode(map[string]interface{}{
	// 			"status":  "error",
	// 			"message": "Failed to save file",
	// 			"data":    nil,
	// 		})
	// 		return
	// 	}
	// 	defer dst.Close()
	// 	io.Copy(dst, file)

	// 	responses = append(responses, UploadResponse{
	// 		ID:           fileID,
	// 		OriginalName: header.Filename,
	// 		URL:          fmt.Sprintf("/api/v1/files/%s", fileID),
	// 	})
	// }

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "File(s) uploaded successfully",
		"data":    responses,
	})
}
