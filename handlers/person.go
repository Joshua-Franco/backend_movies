package handlers

import (
	"net/http"
	"strconv"
	"time"

	"backend_movies/logger"
	"backend_movies/services"

	"github.com/go-chi/chi/v5"
)

type PersonHandler struct {
	tmdb  *services.TMDBClient
	cache *services.CacheService
}

func NewPersonHandler(tmdb *services.TMDBClient, cache *services.CacheService) *PersonHandler {
	return &PersonHandler{tmdb: tmdb, cache: cache}
}

func (h *PersonHandler) GetPerson(w http.ResponseWriter, r *http.Request) {
	personIdStr := chi.URLParam(r, "personId")
	personId, err := strconv.Atoi(personIdStr)
	if err != nil {
		logger.Error("Invalid person ID: %s", personIdStr)
		http.Error(w, "Invalid person ID", http.StatusBadRequest)
		return
	}

	key := services.CacheKey("person", personId)

	if data, found, err := h.cache.Get(key); found {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "HIT")
		w.Write(data)
		return
	} else if err != nil {
		logger.Warn("Cache read error: %v", err)
	}

	logger.Info("Fetching person detail #%d from TMDB...", personId)
	data, err := h.tmdb.GetPerson(personId)
	if err != nil {
		logger.Error("TMDB request failed: %v", err)
		http.Error(w, "Failed to fetch person", http.StatusInternalServerError)
		return
	}

	h.cache.Set(key, data, 1*time.Hour)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cache", "MISS")
	w.Write(data)
}
