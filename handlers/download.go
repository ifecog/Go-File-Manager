package handlers

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

func GetFile(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	// fmt.Println("Requested ID:", chi.URLParam(r, "id"))

	foundPath := ""
	err := filepath.Walk("uploads", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasPrefix(info.Name(), id) {
			foundPath = path
			return filepath.SkipDir
		}
		return nil
	})

	if err != nil || foundPath == "" {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	f, err := os.Open(foundPath)
	if err != nil {
		http.Error(w, "Unable to open file", http.StatusInternalServerError)
		return
	}
	defer f.Close()

	buff := make([]byte, 512)
	_, _ = f.Read(buff)
	contentType := http.DetectContentType(buff)

	f.Seek(0, 0)

	w.Header().Set("Content-Type", contentType)

	if r.URL.Query().Get("download") == "true" {
		w.Header().Set("Content-Disposition", "attachment; filename="+filepath.Base(foundPath))
	} else {
		w.Header().Set("Content-Disposition", "inline; filename="+filepath.Base(foundPath))
	}

	http.ServeContent(w, r, filepath.Base(foundPath), time.Now(), f)
}
