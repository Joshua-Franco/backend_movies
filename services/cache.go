package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"backend_movies/database"
	"backend_movies/logger"
)

type CacheService struct{}

func NewCacheService() *CacheService {
	return &CacheService{}
}

func (c *CacheService) Get(key string) ([]byte, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var response []byte
	var expiresAt time.Time

	err := database.DB.QueryRow(ctx,
		"SELECT response, expires_at FROM tmdb_cache WHERE cache_key = $1",
		key,
	).Scan(&response, &expiresAt)

	if err != nil {
		return nil, false, nil
	}

	if time.Now().After(expiresAt) {
		go c.Delete(key)
		return nil, false, nil
	}

	logger.CacheHit(key)
	return response, true, nil
}

func (c *CacheService) Set(key string, data []byte, ttl time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	expiresAt := time.Now().Add(ttl)

	_, err := database.DB.Exec(ctx,
		`INSERT INTO tmdb_cache (cache_key, response, expires_at)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (cache_key) DO UPDATE SET response = $2, expires_at = $3`,
		key, data, expiresAt,
	)

	if err != nil {
		logger.Warn("Cache set failed for %s: %v", key, err)
		return err
	}

	logger.CacheMiss(key)
	return nil
}

func (c *CacheService) Delete(key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := database.DB.Exec(ctx, "DELETE FROM tmdb_cache WHERE cache_key = $1", key)
	return err
}

func (c *CacheService) CleanExpired() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := database.DB.Exec(ctx, "DELETE FROM tmdb_cache WHERE expires_at < NOW()")
	if err != nil {
		logger.Warn("Cache cleanup failed: %v", err)
		return err
	}

	return nil
}

func CacheKey(parts ...interface{}) string {
	key := ""
	for i, p := range parts {
		if i > 0 {
			key += ":"
		}
		key += fmt.Sprintf("%v", p)
	}
	return key
}

func MarshalAndCache(key string, data interface{}, ttl time.Duration) ([]byte, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	cache := NewCacheService()
	cache.Set(key, jsonData, ttl)

	return jsonData, nil
}
