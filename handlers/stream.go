package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"backend_movies/database"
	"backend_movies/logger"

	"github.com/go-chi/chi/v5"
)

type StreamHandler struct{}

func NewStreamHandler() *StreamHandler {
	return &StreamHandler{}
}

func (h *StreamHandler) GetStreamInfo(w http.ResponseWriter, r *http.Request) {
	tmdbId := chi.URLParam(r, "tmdbId")

	movieName, qualities, found := database.GetStreamInfo(tmdbId)

	w.Header().Set("Content-Type", "application/json")

	if !found {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"tmdb_id":   tmdbId,
			"available": false,
		})
		return
	}

	qualityNames := make([]string, len(qualities))
	for i, q := range qualities {
		qualityNames[i] = q.Quality
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"tmdb_id":    tmdbId,
		"available":  true,
		"movie_name": movieName,
		"qualities":  qualityNames,
	})
}

func (h *StreamHandler) ServePlaylist(w http.ResponseWriter, r *http.Request) {
	movieName := chi.URLParam(r, "movieName")
	quality := chi.URLParam(r, "quality")

	filePath := filepath.Join("storage", "peliculas", movieName, quality, "playlist.m3u8")

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		logger.Warn("Playlist not found: %s", filePath)
		http.Error(w, "Playlist not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	w.Header().Set("Cache-Control", "no-cache")
	http.ServeFile(w, r, filePath)
}

func (h *StreamHandler) ServeSegment(w http.ResponseWriter, r *http.Request) {
	movieName := chi.URLParam(r, "movieName")
	quality := chi.URLParam(r, "quality")
	segment := chi.URLParam(r, "segment")

	if !strings.HasSuffix(segment, ".ts") {
		segment = segment + ".ts"
	}

	filePath := filepath.Join("storage", "peliculas", movieName, quality, segment)

	fileInfo, err := os.Stat(filePath)
	if err != nil {
		http.Error(w, "Segment not found", http.StatusNotFound)
		return
	}

	fileSize := fileInfo.Size()

	w.Header().Set("Content-Type", "video/mp2t")
	w.Header().Set("Cache-Control", "max-age=3600")
	w.Header().Set("Accept-Ranges", "bytes")

	rangeHeader := r.Header.Get("Range")
	if rangeHeader == "" {
		http.ServeFile(w, r, filePath)
		return
	}

	rangeStr := strings.TrimPrefix(rangeHeader, "bytes=")
	parts := strings.SplitN(rangeStr, "-", 2)
	if len(parts) != 2 {
		http.ServeFile(w, r, filePath)
		return
	}

	var start, end int64
	if parts[0] != "" {
		start, _ = strconv.ParseInt(parts[0], 10, 64)
	}
	if parts[1] != "" {
		end, _ = strconv.ParseInt(parts[1], 10, 64)
	} else {
		end = fileSize - 1
	}

	if start < 0 || start >= fileSize || start > end {
		w.Header().Set("Content-Range", fmt.Sprintf("bytes */%d", fileSize))
		w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
		return
	}

	if end >= fileSize {
		end = fileSize - 1
	}

	contentLength := end - start + 1
	w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, fileSize))
	w.Header().Set("Content-Length", strconv.FormatInt(contentLength, 10))
	w.WriteHeader(http.StatusPartialContent)

	f, err := os.Open(filePath)
	if err != nil {
		http.Error(w, "Failed to open segment", http.StatusInternalServerError)
		return
	}
	defer f.Close()

	f.Seek(start, 0)
	io.CopyN(w, f, contentLength)
}

func (h *StreamHandler) ScanMedia(w http.ResponseWriter, r *http.Request) {
	database.ScanMediaFiles()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "scanned"})
}

func (h *StreamHandler) GetDownload(w http.ResponseWriter, r *http.Request) {
	tmdbId := chi.URLParam(r, "tmdbId")

	movieName, _, found := database.GetStreamInfo(tmdbId)
	if !found {
		http.Error(w, "Movie not found", http.StatusNotFound)
		return
	}

	origDir := filepath.Join("storage", "peliculas", movieName, "archivo_original")

	entries, err := os.ReadDir(origDir)
	if err != nil || len(entries) == 0 {
		http.Error(w, "Original file not available", http.StatusNotFound)
		return
	}

	originalFile := entries[0]
	filePath := filepath.Join(origDir, originalFile.Name())

	fileInfo, err := os.Stat(filePath)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	ext := strings.ToLower(filepath.Ext(originalFile.Name()))
	contentType := "application/octet-stream"
	switch ext {
	case ".mp4":
		contentType = "video/mp4"
	case ".mkv":
		contentType = "video/x-matroska"
	case ".avi":
		contentType = "video/x-msvideo"
	case ".mov":
		contentType = "video/quicktime"
	case ".webm":
		contentType = "video/webm"
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.FormatInt(fileInfo.Size(), 10))
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, originalFile.Name()))
	w.Header().Set("Accept-Ranges", "bytes")
	http.ServeFile(w, r, filePath)
}
