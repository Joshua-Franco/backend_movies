package handlers

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"backend_movies/logger"
	"backend_movies/services"

	"github.com/go-chi/chi/v5"
)

type PopularHandler struct {
	tmdb  *services.TMDBClient
	cache *services.CacheService
}

func NewPopularHandler(tmdb *services.TMDBClient, cache *services.CacheService) *PopularHandler {
	return &PopularHandler{tmdb: tmdb, cache: cache}
}

func (h *PopularHandler) GetPopular(w http.ResponseWriter, r *http.Request) {
	mediaType := chi.URLParam(r, "mediaType")
	if mediaType != "movie" && mediaType != "tv" {
		logger.Error("Invalid mediaType: %s", mediaType)
		http.Error(w, "mediaType must be 'movie' or 'tv'", http.StatusBadRequest)
		return
	}

	page := getPageParam(r)
	key := services.CacheKey("popular", mediaType, page)

	if data, found, err := h.cache.Get(key); found {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "HIT")
		w.Write(data)
		return
	} else if err != nil {
		logger.Warn("Cache read error: %v", err)
	}

	logger.Info("Fetching popular %s (page %d) from TMDB...", mediaType, page)
	data, err := h.tmdb.GetPopular(mediaType, page)
	if err != nil {
		logger.Error("TMDB request failed: %v", err)
		http.Error(w, "Failed to fetch popular", http.StatusInternalServerError)
		return
	}

	h.cache.Set(key, data, 30*time.Minute)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cache", "MISS")
	w.Write(data)
}

func (h *PopularHandler) GetTopRated(w http.ResponseWriter, r *http.Request) {
	mediaType := chi.URLParam(r, "mediaType")
	if mediaType != "movie" && mediaType != "tv" {
		logger.Error("Invalid mediaType: %s", mediaType)
		http.Error(w, "mediaType must be 'movie' or 'tv'", http.StatusBadRequest)
		return
	}

	page := getPageParam(r)
	key := services.CacheKey("top_rated", mediaType, page)

	if data, found, err := h.cache.Get(key); found {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "HIT")
		w.Write(data)
		return
	} else if err != nil {
		logger.Warn("Cache read error: %v", err)
	}

	logger.Info("Fetching top rated %s (page %d) from TMDB...", mediaType, page)
	data, err := h.tmdb.GetTopRated(mediaType, page)
	if err != nil {
		logger.Error("TMDB request failed: %v", err)
		http.Error(w, "Failed to fetch top rated", http.StatusInternalServerError)
		return
	}

	h.cache.Set(key, data, 30*time.Minute)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cache", "MISS")
	w.Write(data)
}

func (h *PopularHandler) GetNowPlaying(w http.ResponseWriter, r *http.Request) {
	mediaType := chi.URLParam(r, "mediaType")
	if mediaType != "movie" && mediaType != "tv" {
		logger.Error("Invalid mediaType: %s", mediaType)
		http.Error(w, "mediaType must be 'movie' or 'tv'", http.StatusBadRequest)
		return
	}

	page := getPageParam(r)
	key := services.CacheKey("now_playing", mediaType, page)

	if data, found, err := h.cache.Get(key); found {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "HIT")
		w.Write(data)
		return
	} else if err != nil {
		logger.Warn("Cache read error: %v", err)
	}

	logger.Info("Fetching now playing %s (page %d) from TMDB...", mediaType, page)
	data, err := h.tmdb.GetNowPlaying(mediaType, page)
	if err != nil {
		logger.Error("TMDB request failed: %v", err)
		http.Error(w, "Failed to fetch now playing", http.StatusInternalServerError)
		return
	}

	h.cache.Set(key, data, 30*time.Minute)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cache", "MISS")
	w.Write(data)
}

func getPageParam(r *http.Request) int {
	pageStr := r.URL.Query().Get("page")
	if pageStr == "" {
		return 1
	}
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		return 1
	}
	return page
}

func init() {
	// suppress unused import warning
	_ = log.Fatal
}
