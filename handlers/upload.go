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
	fmt.Println("ClamAV scan output:", string(output))

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 1 {
				return fmt.Errorf("virus detected")
			}
		}

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

		dst, err := os.Create(finalPath)
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

		src, _ := header.Open()
		io.Copy(dst, src)
		src.Close()

		BASE_URL := os.Getenv("BASE_URL")
		responses = append(responses, UploadResponse{
			ID:           finalID,
			OriginalName: header.Filename,
			URL:          fmt.Sprintf("%s/api/v1/files/%s", BASE_URL, finalID),
		})

		go func(path string) {
			if err := scanWithClamAV(path); err != nil {
				fmt.Printf("⚠️ Virus found in %s. Deleting...\n", path)
				if removeErr := os.Remove(path); removeErr != nil {
					fmt.Printf("❌ Failed to delete %s: %v\n", path, removeErr)
				} else {
					fmt.Printf("✅ Successfully deleted %s\n", path)
				}
			}
		}(finalPath)
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
