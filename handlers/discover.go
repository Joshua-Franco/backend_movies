package handlers

import (
	"net/http"
	"time"

	"backend_movies/logger"
	"backend_movies/services"
)

type DiscoverHandler struct {
	tmdb  *services.TMDBClient
	cache *services.CacheService
}

func NewDiscoverHandler(tmdb *services.TMDBClient, cache *services.CacheService) *DiscoverHandler {
	return &DiscoverHandler{tmdb: tmdb, cache: cache}
}

func (h *DiscoverHandler) DiscoverByCompanies(w http.ResponseWriter, r *http.Request) {
	page := getPageParam(r)
	key := services.CacheKey("discover", "companies", page)

	if data, found, err := h.cache.Get(key); found {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "HIT")
		w.Write(data)
		return
	} else if err != nil {
		logger.Warn("Cache read error: %v", err)
	}

	logger.Info("Discovering movies by companies (page %d)...", page)
	data, err := h.tmdb.DiscoverByCompanies(page)
	if err != nil {
		logger.Error("TMDB request failed: %v", err)
		http.Error(w, "Failed to discover movies", http.StatusInternalServerError)
		return
	}

	h.cache.Set(key, data, 30*time.Minute)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cache", "MISS")
	w.Write(data)
}

func (h *DiscoverHandler) DiscoverTVByCompanies(w http.ResponseWriter, r *http.Request) {
	page := getPageParam(r)
	key := services.CacheKey("discover", "tv_companies", page)

	if data, found, err := h.cache.Get(key); found {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "HIT")
		w.Write(data)
		return
	} else if err != nil {
		logger.Warn("Cache read error: %v", err)
	}

	logger.Info("Discovering TV shows by companies (page %d)...", page)
	data, err := h.tmdb.DiscoverTVByCompanies(page)
	if err != nil {
		logger.Error("TMDB request failed: %v", err)
		http.Error(w, "Failed to discover TV shows", http.StatusInternalServerError)
		return
	}

	h.cache.Set(key, data, 30*time.Minute)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cache", "MISS")
	w.Write(data)
}
