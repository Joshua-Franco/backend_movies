package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"backend_movies/logger"
	"backend_movies/services"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type UploadHandler struct {
	converter *services.ConverterService
	cloudinary *services.CloudinaryService
}

func NewUploadHandler(converter *services.ConverterService, cld *services.CloudinaryService) *UploadHandler {
	return &UploadHandler{
		converter:  converter,
		cloudinary: cld,
	}
}

func (h *UploadHandler) SignUpload(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TMDBID    int    `json:"tmdb_id"`
		MovieName string `json:"movie_name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.TMDBID == 0 {
		http.Error(w, "tmdb_id is required", http.StatusBadRequest)
		return
	}

	if req.MovieName == "" {
		req.MovieName = fmt.Sprintf("movie-%d", req.TMDBID)
	}

	publicID := fmt.Sprintf("%d", req.TMDBID)
	params := h.cloudinary.GenerateSignedUploadURL(publicID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(params)
}

func (h *UploadHandler) CompleteUpload(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TMDBID     int    `json:"tmdb_id"`
		MovieName  string `json:"movie_name"`
		PublicID   string `json:"public_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.TMDBID == 0 {
		http.Error(w, "tmdb_id is required", http.StatusBadRequest)
		return
	}
	if req.PublicID == "" {
		http.Error(w, "public_id is required", http.StatusBadRequest)
		return
	}
	if req.MovieName == "" {
		req.MovieName = fmt.Sprintf("movie-%d", req.TMDBID)
	}

	uploadID := uuid.New().String()

	go func() {
		ctx := context.Background()
		
		// Descargar original de Cloudinary
		tmpDir := fmt.Sprintf("%s/movienova-%s", os.TempDir(), uploadID)
		inputPath, err := h.cloudinary.DownloadOriginal(req.PublicID, tmpDir)
		if err != nil {
			logger.Error("Error descargando original: %v", err)
			return
		}

		// Convertir y subir HLS
		if err := h.converter.ConvertAndUpload(ctx, inputPath, req.TMDBID, req.MovieName); err != nil {
			logger.Error("Conversión falló para upload %s: %v", uploadID, err)
		}

		// Limpiar archivo temporal
		os.Remove(inputPath)
		os.RemoveAll(tmpDir)
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