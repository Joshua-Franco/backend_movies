package models

import "time"

type UserListItem struct {
	ID          int       `json:"id"`
	TMDBID      int       `json:"tmdb_id"`
	MediaType   string    `json:"media_type"`
	Title       string    `json:"title"`
	PosterPath  *string   `json:"poster_path"`
	AddedAt     time.Time `json:"added_at"`
}

type ContinueWatchingItem struct {
	ID          int       `json:"id"`
	TMDBID      int       `json:"tmdb_id"`
	MediaType   string    `json:"media_type"`
	Title       string    `json:"title"`
	PosterPath  *string   `json:"poster_path"`
	Progress    int       `json:"progress"`
	Season      *int      `json:"season"`
	Episode     *int      `json:"episode"`
	TimeLeft    string    `json:"time_left"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type AddToListRequest struct {
	TMDBID     int    `json:"tmdb_id"`
	MediaType  string `json:"media_type"`
	Title      string `json:"title"`
	PosterPath string `json:"poster_path"`
}

type UpdateProgressRequest struct {
	TMDBID     int    `json:"tmdb_id"`
	MediaType  string `json:"media_type"`
	Title      string `json:"title"`
	PosterPath string `json:"poster_path"`
	Progress   int    `json:"progress"`
	Season     *int   `json:"season"`
	Episode    *int   `json:"episode"`
	TimeLeft   string `json:"time_left"`
}
