package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"backend_movies/config"
)

type TMDBClient struct {
	apiKey   string
	baseURL  string
	client   *http.Client
}

func NewTMDBClient(cfg *config.Config) *TMDBClient {
	return &TMDBClient{
		apiKey:  cfg.TMDBAPIKey,
		baseURL: cfg.TMDBBaseURL,
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (t *TMDBClient) get(path string) ([]byte, error) {
	separator := "?"
	if strings.Contains(path, "?") {
		separator = "&"
	}
	url := fmt.Sprintf("%s%s%sapi_key=%s&language=en-US", t.baseURL, path, separator, t.apiKey)
	
	resp, err := t.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("TMDB request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("TMDB returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read TMDB response: %w", err)
	}

	return body, nil
}

func (t *TMDBClient) GetTrending(mediaType string) ([]byte, error) {
	return t.get(fmt.Sprintf("/trending/%s/week", mediaType))
}

func (t *TMDBClient) GetPopular(mediaType string, page int) ([]byte, error) {
	return t.get(fmt.Sprintf("/%s/popular?page=%d", mediaType, page))
}

func (t *TMDBClient) GetTopRated(mediaType string, page int) ([]byte, error) {
	return t.get(fmt.Sprintf("/%s/top_rated?page=%d", mediaType, page))
}

func (t *TMDBClient) GetNowPlaying(mediaType string, page int) ([]byte, error) {
	return t.get(fmt.Sprintf("/%s/now_playing?page=%d", mediaType, page))
}

func (t *TMDBClient) GetDetail(mediaType string, id int) ([]byte, error) {
	return t.get(fmt.Sprintf("/%s/%d?append_to_response=videos,credits,similar", mediaType, id))
}

func (t *TMDBClient) GetPerson(id int) ([]byte, error) {
	return t.get(fmt.Sprintf("/person/%d?append_to_response=combined_credits", id))
}

func (t *TMDBClient) Search(query string, page int) ([]byte, error) {
	return t.get(fmt.Sprintf("/search/multi?query=%s&page=%d", query, page))
}

func (t *TMDBClient) GetGenres(mediaType string) ([]byte, error) {
	return t.get(fmt.Sprintf("/genre/%s/list", mediaType))
}

func (t *TMDBClient) DiscoverMoviesByGenre(genreId int, page int) ([]byte, error) {
	return t.get(fmt.Sprintf("/discover/movie?with_genres=%d&page=%d", genreId, page))
}

func (t *TMDBClient) DiscoverByCompanies(page int) ([]byte, error) {
	companies := "2|3|420|1|9993|3255|27|11565|174|36323|213"
	return t.get(fmt.Sprintf("/discover/movie?with_companies=%s&page=%d&sort_by=popularity.desc", companies, page))
}

func (t *TMDBClient) DiscoverTVByCompanies(page int) ([]byte, error) {
	companies := "2|3|420|1|9993|3255|27|11565|174|36323|213"
	return t.get(fmt.Sprintf("/discover/tv?with_companies=%s&page=%d&sort_by=popularity.desc", companies, page))
}

func ParseResponse(data []byte) (map[string]interface{}, error) {
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}
