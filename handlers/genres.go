package handlers

import (
	"net/http"
	"strconv"
	"time"

	"backend_movies/logger"
	"backend_movies/services"

	"github.com/go-chi/chi/v5"
)

type GenresHandler struct {
	tmdb  *services.TMDBClient
	cache *services.CacheService
}

func NewGenresHandler(tmdb *services.TMDBClient, cache *services.CacheService) *GenresHandler {
	return &GenresHandler{tmdb: tmdb, cache: cache}
}

func (h *GenresHandler) GetGenres(w http.ResponseWriter, r *http.Request) {
	mediaType := chi.URLParam(r, "mediaType")
	if mediaType != "movie" && mediaType != "tv" {
		logger.Error("Invalid mediaType: %s", mediaType)
		http.Error(w, "mediaType must be 'movie' or 'tv'", http.StatusBadRequest)
		return
	}

	key := services.CacheKey("genres", mediaType)

	if data, found, err := h.cache.Get(key); found {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "HIT")
		w.Write(data)
		return
	} else if err != nil {
		logger.Warn("Cache read error: %v", err)
	}

	logger.Info("Fetching %s genres from TMDB...", mediaType)
	data, err := h.tmdb.GetGenres(mediaType)
	if err != nil {
		logger.Error("TMDB request failed: %v", err)
		http.Error(w, "Failed to fetch genres", http.StatusInternalServerError)
		return
	}

	h.cache.Set(key, data, 24*time.Hour)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cache", "MISS")
	w.Write(data)
}

func (h *GenresHandler) DiscoverByGenre(w http.ResponseWriter, r *http.Request) {
	genreIdStr := r.URL.Query().Get("genreId")
	if genreIdStr == "" {
		logger.Error("Missing required parameter: genreId")
		http.Error(w, "genreId query parameter is required", http.StatusBadRequest)
		return
	}

	genreId, err := strconv.Atoi(genreIdStr)
	if err != nil {
		logger.Error("Invalid genreId: %s", genreIdStr)
		http.Error(w, "Invalid genreId", http.StatusBadRequest)
		return
	}

	page := getPageParam(r)
	key := services.CacheKey("discover", "movie", genreId, page)

	if data, found, err := h.cache.Get(key); found {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "HIT")
		w.Write(data)
		return
	} else if err != nil {
		logger.Warn("Cache read error: %v", err)
	}

	logger.Info("Discovering movies by genre %d (page %d)...", genreId, page)
	data, err := h.tmdb.DiscoverMoviesByGenre(genreId, page)
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
