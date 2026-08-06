package handlers

import (
	"net/http"
	"strconv"
	"time"

	"backend_movies/logger"
	"backend_movies/services"

	"github.com/go-chi/chi/v5"
)

type DetailHandler struct {
	tmdb  *services.TMDBClient
	cache *services.CacheService
}

func NewDetailHandler(tmdb *services.TMDBClient, cache *services.CacheService) *DetailHandler {
	return &DetailHandler{tmdb: tmdb, cache: cache}
}

func (h *DetailHandler) GetDetail(w http.ResponseWriter, r *http.Request) {
	mediaType := chi.URLParam(r, "mediaType")
	if mediaType != "movie" && mediaType != "tv" {
		logger.Error("Invalid mediaType: %s", mediaType)
		http.Error(w, "mediaType must be 'movie' or 'tv'", http.StatusBadRequest)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		logger.Error("Invalid ID: %s", idStr)
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	key := services.CacheKey("detail", mediaType, id)

	if data, found, err := h.cache.Get(key); found {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "HIT")
		w.Write(data)
		return
	} else if err != nil {
		logger.Warn("Cache read error: %v", err)
	}

	logger.Info("Fetching %s detail #%d from TMDB...", mediaType, id)
	data, err := h.tmdb.GetDetail(mediaType, id)
	if err != nil {
		logger.Error("TMDB request failed: %v", err)
		http.Error(w, "Failed to fetch detail", http.StatusInternalServerError)
		return
	}

	h.cache.Set(key, data, 1*time.Hour)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cache", "MISS")
	w.Write(data)
}
