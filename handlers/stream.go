package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"backend_movies/database"
	"backend_movies/logger"
	"backend_movies/services"

	"github.com/go-chi/chi/v5"
)

type StreamHandler struct {
	cloudinary *services.CloudinaryService
}

func NewStreamHandler(cld *services.CloudinaryService) *StreamHandler {
	return &StreamHandler{
		cloudinary: cld,
	}
}

func (h *StreamHandler) GetStreamInfo(w http.ResponseWriter, r *http.Request) {
	tmdbId := chi.URLParam(r, "tmdbId")

	movieName, qualities, found := database.GetStreamInfo(tmdbId)

	w.Header().Set("Content-Type", "application/json")

	if !found {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"tmdb_id":   tmdbId,
			"available": false,
		})
		return
	}

	qualityNames := make([]string, len(qualities))
	for i, q := range qualities {
		qualityNames[i] = q.Quality
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"tmdb_id":    tmdbId,
		"available":  true,
		"movie_name": movieName,
		"qualities":  qualityNames,
	})
}

func (h *StreamHandler) ServePlaylist(w http.ResponseWriter, r *http.Request) {
	movieName := chi.URLParam(r, "movieName")
	quality := chi.URLParam(r, "quality")

	folder, err := database.GetCloudinaryFolder(movieName, quality)
	if err != nil {
		logger.Warn("Cloudinary folder not found for %s/%s: %v", movieName, quality, err)
		http.Error(w, "Playlist not found", http.StatusNotFound)
		return
	}

	url := h.cloudinary.GetHLSURL(folder, quality)
	logger.Info("Redirecting playlist to Cloudinary: %s", url)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (h *StreamHandler) ServeSegment(w http.ResponseWriter, r *http.Request) {
	movieName := chi.URLParam(r, "movieName")
	quality := chi.URLParam(r, "quality")
	segment := chi.URLParam(r, "segment")

	if !strings.HasSuffix(segment, ".ts") {
		segment = segment + ".ts"
	}

	folder, err := database.GetCloudinaryFolder(movieName, quality)
	if err != nil {
		logger.Warn("Cloudinary folder not found for %s/%s: %v", movieName, quality, err)
		http.Error(w, "Segment not found", http.StatusNotFound)
		return
	}

	url := h.cloudinary.GetSegmentURL(folder, quality, segment)
	logger.Info("Redirecting segment to Cloudinary: %s", url)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (h *StreamHandler) ScanMedia(w http.ResponseWriter, r *http.Request) {
	database.ScanMediaFiles()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "scanned"})
}

func (h *StreamHandler) GetDownload(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Download not available", http.StatusNotImplemented)
}
