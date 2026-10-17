package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"

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
	databaseURL  string
	ServerPort   string
	R2AccountID  string
	R2AccessKey  string
	R2SecretKey  string
	R2Bucket     string
	R2PublicURL  string
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
		DBSSLMode:    getEnv("DB_SSLMODE", "require"),
		ServerPort:   getEnv("PORT", getEnv("SERVER_PORT", "8080")),
		R2AccountID:  getEnv("R2_ACCOUNT_ID", ""),
		R2AccessKey:  getEnv("R2_ACCESS_KEY_ID", ""),
		R2SecretKey:  getEnv("R2_SECRET_ACCESS_KEY", ""),
		R2Bucket:     getEnv("R2_BUCKET_NAME", "movies"),
		R2PublicURL:  getEnv("R2_PUBLIC_URL", ""),
	}

	rawDatabaseURL := getEnv("DATABASE_URL", "")
	if rawDatabaseURL != "" {
		var err error
		cfg.databaseURL, err = normalizeDatabaseURL(rawDatabaseURL)
		if err != nil {
			return nil, err
		}
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName, err = parseDatabaseURL(cfg.databaseURL)
		if err != nil {
			return nil, err
		}
	} else {
		cfg.DBHost = getEnv("DB_HOST", "localhost")
		cfg.DBPort = getEnv("DB_PORT", "5432")
		cfg.DBUser = getEnv("DB_USER", "postgres")
		cfg.DBPassword = getEnv("DB_PASSWORD", "postgres")
		cfg.DBName = getEnv("DB_NAME", "movienova")
		cfg.databaseURL = legacyDatabaseURL(cfg)
	}

	if cfg.TMDBAPIKey == "" || cfg.TMDBAPIKey == "YOUR_TMDB_API_KEY_HERE" {
		logger.Error("TMDB_API_KEY is not set")
		logger.Info("Get your free API key at: https://www.themoviedb.org/settings/api")
		return nil, fmt.Errorf("TMDB_API_KEY is required")
	}

	return cfg, nil
}

func (c *Config) DatabaseURL() string {
	return c.databaseURL
}

func normalizeDatabaseURL(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") || u.Hostname() == "" {
		return "", fmt.Errorf("invalid DATABASE_URL: provide a complete postgres:// or postgresql:// URL")
	}
	if strings.Trim(u.Path, "/") == "" {
		return "", fmt.Errorf("invalid DATABASE_URL: include the target database name in the URL path")
	}

	query, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return "", fmt.Errorf("invalid DATABASE_URL query parameters")
	}

	mode := strings.ToLower(query.Get("sslmode"))
	if mode != "require" && mode != "verify-ca" && mode != "verify-full" {
		query.Set("sslmode", "require")
	}
	u.RawQuery = query.Encode()
	return u.String(), nil
}

func parseDatabaseURL(raw string) (host, port, user, pass, name string, err error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", "", "", "", "", fmt.Errorf("invalid DATABASE_URL")
	}
	host = u.Hostname()
	port = u.Port()
	if port == "" {
		port = "5432"
	}
	if u.User != nil {
		user = u.User.Username()
		pass, _ = u.User.Password()
	}
	name = strings.TrimPrefix(u.Path, "/")
	return host, port, user, pass, name, nil
}

func legacyDatabaseURL(cfg *Config) string {
	u := &url.URL{
		Scheme: "postgres",
		Host:   net.JoinHostPort(cfg.DBHost, cfg.DBPort),
		User:   url.UserPassword(cfg.DBUser, cfg.DBPassword),
		Path:   "/" + cfg.DBName,
	}
	query := url.Values{}
	query.Set("sslmode", cfg.DBSSLMode)
	u.RawQuery = query.Encode()
	result, err := normalizeDatabaseURL(u.String())
	if err != nil {
		return u.String()
	}
	return result
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
