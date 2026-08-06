package handlers

import (
	"net/http"
	"time"

	"backend_movies/logger"
	"backend_movies/services"

	"github.com/go-chi/chi/v5"
)

type TrendingHandler struct {
	tmdb  *services.TMDBClient
	cache *services.CacheService
}

func NewTrendingHandler(tmdb *services.TMDBClient, cache *services.CacheService) *TrendingHandler {
	return &TrendingHandler{tmdb: tmdb, cache: cache}
}

func (h *TrendingHandler) GetTrending(w http.ResponseWriter, r *http.Request) {
	mediaType := chi.URLParam(r, "mediaType")
	if mediaType != "movie" && mediaType != "tv" {
		logger.Error("Invalid mediaType: %s (must be 'movie' or 'tv')", mediaType)
		http.Error(w, "mediaType must be 'movie' or 'tv'", http.StatusBadRequest)
		return
	}

	key := services.CacheKey("trending", mediaType, "week")

	if data, found, err := h.cache.Get(key); found {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "HIT")
		w.Write(data)
		return
	} else if err != nil {
		logger.Warn("Cache read error: %v", err)
	}

	logger.Info("Fetching trending %s from TMDB...", mediaType)
	data, err := h.tmdb.GetTrending(mediaType)
	if err != nil {
		logger.Error("TMDB request failed: %v", err)
		http.Error(w, "Failed to fetch trending", http.StatusInternalServerError)
		return
	}

	h.cache.Set(key, data, 30*time.Minute)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cache", "MISS")
	w.Write(data)
}
