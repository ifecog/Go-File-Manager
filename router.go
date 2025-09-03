package main

import (
	"go-file-api/handlers"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func SetupRouter() *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"service":"CLRD File API","status":"running","version":"1.0.0"}`))
	})

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	r.Route("/api/v1", func(r chi.Router) {
		// r.Use(APIKeyMiddleware)

		r.Post("/upload", handlers.UploadFile)
		r.Get("/files/{id}", handlers.GetFile)
		r.Delete("/files/{id}", handlers.DeleteFile)
	})

	return r
}
