package handlers

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupTestFile(t *testing.T, filename, content string) string {
	t.Helper()
	os.MkdirAll("uploads", 0755)
	path := filepath.Join("uploads", filename)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	return path
}

func cleanupUploads() {
	os.RemoveAll("uploads")
}

func TestGetFile_PreviewMode(t *testing.T) {
	defer cleanupUploads()
	id := "test-id"
	setupTestFile(t, id+"_file.txt", "hello world")

	req := httptest.NewRequest("GET", "/files/"+id, nil)
	w := httptest.NewRecorder()

	GetFile(w, req)
	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", res.StatusCode)
	}

	if disp := res.Header.Get("Content-Disposition"); !strings.Contains(disp, "inline") {
		t.Errorf("expected inline disposition, got %s", disp)
	}

	body, _ := io.ReadAll(res.Body)
	if string(body) != "hello world" {
		t.Errorf("expected body %q, got %q", "hello world", string(body))
	}
}

func TestGetFile_DownloadMode(t *testing.T) {
	defer cleanupUploads()
	id := "test-id"
	setupTestFile(t, id+"_file.txt", "download me")

	req := httptest.NewRequest("GET", "/files/"+id+"?download=true", nil)
	w := httptest.NewRecorder()

	GetFile(w, req)
	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", res.StatusCode)
	}

	if disp := res.Header.Get("Content-Disposition"); !strings.Contains(disp, "attachment") {
		t.Errorf("expected attachment disposition, got %s", disp)
	}
}

func TestGetFile_NotFound(t *testing.T) {
	defer cleanupUploads()

	req := httptest.NewRequest("GET", "/files/nonexistent", nil)
	w := httptest.NewRecorder()

	GetFile(w, req)
	res := w.Result()

	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", res.StatusCode)
	}
}
