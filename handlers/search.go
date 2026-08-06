package handlers

import (
	"net/http"
	"time"

	"backend_movies/logger"
	"backend_movies/services"
)

type SearchHandler struct {
	tmdb  *services.TMDBClient
	cache *services.CacheService
}

func NewSearchHandler(tmdb *services.TMDBClient, cache *services.CacheService) *SearchHandler {
	return &SearchHandler{tmdb: tmdb, cache: cache}
}

func (h *SearchHandler) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		logger.Error("Missing required parameter: q")
		http.Error(w, "query parameter 'q' is required", http.StatusBadRequest)
		return
	}

	page := getPageParam(r)
	key := services.CacheKey("search", query, page)

	if data, found, err := h.cache.Get(key); found {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "HIT")
		w.Write(data)
		return
	} else if err != nil {
		logger.Warn("Cache read error: %v", err)
	}

	logger.Info("Searching TMDB: '%s' (page %d)...", query, page)
	data, err := h.tmdb.Search(query, page)
	if err != nil {
		logger.Error("TMDB search failed: %v", err)
		http.Error(w, "Failed to search", http.StatusInternalServerError)
		return
	}

	h.cache.Set(key, data, 5*time.Minute)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cache", "MISS")
	w.Write(data)
}
