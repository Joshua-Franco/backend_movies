package config

import (
	"fmt"
	"os"
	"path/filepath"

	"backend_movies/logger"

	"github.com/joho/godotenv"
)

type Config struct {
	TMDBAPIKey   string
	TMDBBaseURL  string
	TMDBImageURL string
	DBHost       string
	DBPort       string
	DBUser       string
	DBPassword   string
	DBName       string
	DBSSLMode    string
	ServerPort   string
}

func Load() (*Config, error) {
	exePath, _ := os.Executable()
	exeDir := filepath.Dir(exePath)
	envPath := filepath.Join(exeDir, ".env")
	godotenv.Load(envPath)
	godotenv.Load()

	cfg := &Config{
		TMDBAPIKey:   getEnv("TMDB_API_KEY", ""),
		TMDBBaseURL:  getEnv("TMDB_BASE_URL", "https://api.themoviedb.org/3"),
		TMDBImageURL: getEnv("TMDB_IMAGE_URL", "https://image.tmdb.org/t/p"),
		DBHost:       getEnv("DB_HOST", "localhost"),
		DBPort:       getEnv("DB_PORT", "5432"),
		DBUser:       getEnv("DB_USER", "postgres"),
		DBPassword:   getEnv("DB_PASSWORD", "postgres"),
		DBName:       getEnv("DB_NAME", "movienova"),
		DBSSLMode:    getEnv("DB_SSLMODE", "disable"),
		ServerPort:   getEnv("SERVER_PORT", "8080"),
	}

	if cfg.TMDBAPIKey == "" || cfg.TMDBAPIKey == "YOUR_TMDB_API_KEY_HERE" {
		logger.Error("TMDB_API_KEY is not set in .env")
		logger.Info("Get your free API key at: https://www.themoviedb.org/settings/api")
		return nil, fmt.Errorf("TMDB_API_KEY is required in .env file")
	}

	return cfg, nil
}

func (c *Config) DatabaseURL() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName, c.DBSSLMode,
	)
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
