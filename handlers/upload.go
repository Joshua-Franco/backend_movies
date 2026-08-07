package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"backend_movies/logger"
	"backend_movies/services"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type UploadHandler struct {
	converter *services.ConverterService
}

func NewUploadHandler(converter *services.ConverterService) *UploadHandler {
	return &UploadHandler{
		converter: converter,
	}
}

func (h *UploadHandler) Upload(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(100 << 20)
	if err != nil {
		http.Error(w, "Error parsing multipart form: "+err.Error(), http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "No file provided: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	tmdbIDStr := r.FormValue("tmdb_id")
	if tmdbIDStr == "" {
		http.Error(w, "tmdb_id is required", http.StatusBadRequest)
		return
	}

	var tmdbID int
	if _, err := fmt.Sscanf(tmdbIDStr, "%d", &tmdbID); err != nil {
		http.Error(w, "Invalid tmdb_id", http.StatusBadRequest)
		return
	}

	movieName := r.FormValue("movie_name")
	if movieName == "" {
		movieName = header.Filename
		movieName = movieName[:len(movieName)-len(filepath.Ext(movieName))]
	}

	tmpDir := os.TempDir()
	tmpFile := filepath.Join(tmpDir, fmt.Sprintf("upload-%s%s", uuid.New().String(), filepath.Ext(header.Filename)))

	dst, err := os.Create(tmpFile)
	if err != nil {
		http.Error(w, "Error creating temp file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if _, err := io.Copy(dst, file); err != nil {
		dst.Close()
		os.Remove(tmpFile)
		http.Error(w, "Error saving file: "+err.Error(), http.StatusInternalServerError)
		return
	}
	dst.Close()

	uploadID := uuid.New().String()
	go func() {
		ctx := context.Background()
		if err := h.converter.ConvertAndUpload(ctx, tmpFile, tmdbID, movieName); err != nil {
			logger.Error("Conversión falló para upload %s: %v", uploadID, err)
		}
		os.Remove(tmpFile)
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"upload_id": uploadID,
		"status":    "started",
		"message":   "Conversión iniciada en segundo plano",
	})
}

func (h *UploadHandler) Status(w http.ResponseWriter, r *http.Request) {
	uploadID := chi.URLParam(r, "id")

	status := h.converter.GetUploadStatus(uploadID)
	if status == nil {
		http.Error(w, "Upload not found", http.StatusNotFound)
		return
	}

	response := map[string]interface{}{
		"upload_id":        status.UploadID,
		"tmdb_id":          status.TMDBID,
		"movie_name":       status.MovieName,
		"status":           status.Status,
		"current_quality":  status.CurrentQuality,
		"progress":         status.Progress,
		"started_at":       status.StartedAt.Format(time.RFC3339),
	}

	if status.CompletedAt != nil {
		response["completed_at"] = status.CompletedAt.Format(time.RFC3339)
	}
	if status.Error != "" {
		response["error"] = status.Error
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}