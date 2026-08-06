package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"backend_movies/database"
	"backend_movies/logger"
	"backend_movies/models"

	"github.com/go-chi/chi/v5"
)

type UserHandler struct{}

func NewUserHandler() *UserHandler {
	return &UserHandler{}
}

func (h *UserHandler) GetMyList(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := database.DB.Query(ctx,
		"SELECT id, tmdb_id, media_type, title, poster_path, added_at FROM user_lists ORDER BY added_at DESC",
	)
	if err != nil {
		logger.Error("Failed to fetch my list: %v", err)
		http.Error(w, "Failed to fetch list", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	items := []models.UserListItem{}
	for rows.Next() {
		var item models.UserListItem
		if err := rows.Scan(&item.ID, &item.TMDBID, &item.MediaType, &item.Title, &item.PosterPath, &item.AddedAt); err != nil {
			logger.Warn("Failed to scan list item: %v", err)
			continue
		}
		items = append(items, item)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

func (h *UserHandler) AddToList(w http.ResponseWriter, r *http.Request) {
	var req models.AddToListRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error("Invalid request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := database.DB.Exec(ctx,
		`INSERT INTO user_lists (tmdb_id, media_type, title, poster_path)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (tmdb_id, media_type) DO NOTHING`,
		req.TMDBID, req.MediaType, req.Title, req.PosterPath,
	)
	if err != nil {
		logger.Error("Failed to add to list: %v", err)
		http.Error(w, "Failed to add to list", http.StatusInternalServerError)
		return
	}

	logger.Success("Added %s #%d to my list", req.MediaType, req.TMDBID)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "added"})
}

func (h *UserHandler) RemoveFromList(w http.ResponseWriter, r *http.Request) {
	tmdbIdStr := chi.URLParam(r, "tmdbId")
	if tmdbIdStr == "" {
		logger.Error("Missing tmdbId parameter")
		http.Error(w, "tmdbId is required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := database.DB.Exec(ctx,
		"DELETE FROM user_lists WHERE tmdb_id = $1",
		tmdbIdStr,
	)
	if err != nil {
		logger.Error("Failed to remove from list: %v", err)
		http.Error(w, "Failed to remove from list", http.StatusInternalServerError)
		return
	}

	logger.Success("Removed #%s from my list", tmdbIdStr)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "removed"})
}

func (h *UserHandler) GetContinueWatching(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := database.DB.Query(ctx,
		`SELECT id, tmdb_id, media_type, title, poster_path, progress, season, episode, time_left, updated_at
		 FROM continue_watching ORDER BY updated_at DESC`,
	)
	if err != nil {
		logger.Error("Failed to fetch continue watching: %v", err)
		http.Error(w, "Failed to fetch continue watching", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	items := []models.ContinueWatchingItem{}
	for rows.Next() {
		var item models.ContinueWatchingItem
		if err := rows.Scan(&item.ID, &item.TMDBID, &item.MediaType, &item.Title, &item.PosterPath, &item.Progress, &item.Season, &item.Episode, &item.TimeLeft, &item.UpdatedAt); err != nil {
			logger.Warn("Failed to scan continue watching item: %v", err)
			continue
		}
		items = append(items, item)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

func (h *UserHandler) UpdateProgress(w http.ResponseWriter, r *http.Request) {
	var req models.UpdateProgressRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error("Invalid request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := database.DB.Exec(ctx,
		`INSERT INTO continue_watching (tmdb_id, media_type, title, poster_path, progress, season, episode, time_left)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 ON CONFLICT (tmdb_id, media_type) DO UPDATE SET
		 progress = $5, season = $6, episode = $7, time_left = $8, title = $3, poster_path = $4, updated_at = NOW()`,
		req.TMDBID, req.MediaType, req.Title, req.PosterPath, req.Progress, req.Season, req.Episode, req.TimeLeft,
	)
	if err != nil {
		logger.Error("Failed to update progress: %v", err)
		http.Error(w, "Failed to update progress", http.StatusInternalServerError)
		return
	}

	logger.Success("Updated progress for %s #%d (%d%%)", req.MediaType, req.TMDBID, req.Progress)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

func (h *UserHandler) RemoveFromContinueWatching(w http.ResponseWriter, r *http.Request) {
	tmdbIdStr := chi.URLParam(r, "tmdbId")
	if tmdbIdStr == "" {
		logger.Error("Missing tmdbId parameter")
		http.Error(w, "tmdbId is required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := database.DB.Exec(ctx,
		"DELETE FROM continue_watching WHERE tmdb_id = $1",
		tmdbIdStr,
	)
	if err != nil {
		logger.Error("Failed to remove from continue watching: %v", err)
		http.Error(w, "Failed to remove from continue watching", http.StatusInternalServerError)
		return
	}

	logger.Success("Removed #%s from continue watching", tmdbIdStr)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "removed"})
}
