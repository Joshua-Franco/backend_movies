package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"backend_movies/database"
	"backend_movies/services"

	"github.com/go-chi/chi/v5"
)

type StreamHandler struct {
	R2 *services.R2Client
}

func NewStreamHandler(r2 *services.R2Client) *StreamHandler {
	return &StreamHandler{R2: r2}
}

func (h *StreamHandler) GetStreamInfo(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "tmdbId")
	tmdbID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid tmdb id", http.StatusBadRequest)
		return
	}

	var movieName string
	var qualities []string

	err = database.DB.QueryRow(r.Context(),
		`SELECT movie_name, qualities FROM media_files WHERE tmdb_id = $1`, tmdbID,
	).Scan(&movieName, &qualities)

	if err == sql.ErrNoRows {
		writeJSON(w, http.StatusOK, services.StreamInfo{
			TMDBID:    strconv.Itoa(tmdbID),
			Available: false,
		})
		return
	}
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}

	if len(qualities) == 0 {
		qualities = []string{"1080p", "720p", "480p"}
	}

	writeJSON(w, http.StatusOK, services.StreamInfo{
		TMDBID:    strconv.Itoa(tmdbID),
		Available: true,
		MovieName: movieName,
		Qualities: qualities,
	})
}

func (h *StreamHandler) ServeMasterPlaylist(w http.ResponseWriter, r *http.Request) {
	movieName := chi.URLParam(r, "movieName")
	http.Redirect(w, r, h.R2.GetMasterURL(movieName), http.StatusFound)
}

func (h *StreamHandler) ServeQualityPlaylist(w http.ResponseWriter, r *http.Request) {
	movieName := chi.URLParam(r, "movieName")
	quality := chi.URLParam(r, "quality")
	http.Redirect(w, r, h.R2.GetHLSURL(movieName, quality, "playlist.m3u8"), http.StatusFound)
}

func (h *StreamHandler) ServeSegment(w http.ResponseWriter, r *http.Request) {
	movieName := chi.URLParam(r, "movieName")
	quality := chi.URLParam(r, "quality")
	segment := chi.URLParam(r, "segment")
	http.Redirect(w, r, h.R2.GetHLSURL(movieName, quality, segment), http.StatusFound)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	jsonBytes, _ := json.Marshal(data)
	w.Write(jsonBytes)
}
