package database

import (
	"context"
	"fmt"
	"time"

	"backend_movies/config"
	"backend_movies/logger"

	"github.com/jackc/pgx/v5/pgxpool"
)

var DB *pgxpool.Pool

type MovieInfo struct {
	TMDBID int    `json:"tmdb_id"`
	Title  string `json:"title"`
}

func Connect(cfg *config.Config) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var err error
	DB, err = pgxpool.New(ctx, cfg.DatabaseURL())
	if err != nil {
		return fmt.Errorf("connection pool creation failed: %w", err)
	}

	if err = DB.Ping(ctx); err != nil {
		DB.Close()
		logger.Warn("Database '%s' does not exist, creating it...", cfg.DBName)
		createErr := createDatabase(cfg)
		if createErr != nil {
			return fmt.Errorf("could not create database '%s': %w", cfg.DBName, createErr)
		}
		logger.Success("Database '%s' created", cfg.DBName)

		DB, err = pgxpool.New(ctx, cfg.DatabaseURL())
		if err != nil {
			return fmt.Errorf("connection pool creation failed after db creation: %w", err)
		}

		if err = DB.Ping(ctx); err != nil {
			return fmt.Errorf("ping failed after db creation: %w", err)
		}
	}

	return nil
}

func createDatabase(cfg *config.Config) error {
	connStr := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/postgres?sslmode=%s",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBSSLMode,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		return fmt.Errorf("cannot connect to postgres server: %w", err)
	}
	defer pool.Close()

	_, err = pool.Exec(ctx, fmt.Sprintf(`CREATE DATABASE "%s"`, cfg.DBName))
	if err != nil {
		return fmt.Errorf("CREATE DATABASE failed: %w", err)
	}

	return nil
}

func RunMigrations() error {
	ctx := context.Background()

	migrations := []string{
		`CREATE TABLE IF NOT EXISTS tmdb_cache (
			id SERIAL PRIMARY KEY,
			cache_key VARCHAR(255) UNIQUE NOT NULL,
			response JSONB NOT NULL,
			expires_at TIMESTAMP NOT NULL,
			created_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS user_lists (
			id SERIAL PRIMARY KEY,
			tmdb_id INTEGER NOT NULL,
			media_type VARCHAR(10) NOT NULL,
			title VARCHAR(255) NOT NULL DEFAULT '',
			poster_path VARCHAR(255),
			added_at TIMESTAMP DEFAULT NOW(),
			UNIQUE(tmdb_id, media_type)
		)`,
		`CREATE TABLE IF NOT EXISTS continue_watching (
			id SERIAL PRIMARY KEY,
			tmdb_id INTEGER NOT NULL,
			media_type VARCHAR(10) NOT NULL,
			title VARCHAR(255) NOT NULL,
			poster_path VARCHAR(255),
			progress INTEGER DEFAULT 0,
			season INTEGER,
			episode INTEGER,
			time_left VARCHAR(50),
			updated_at TIMESTAMP DEFAULT NOW(),
			UNIQUE(tmdb_id, media_type)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_tmdb_cache_key ON tmdb_cache(cache_key)`,
		`CREATE INDEX IF NOT EXISTS idx_tmdb_cache_expires ON tmdb_cache(expires_at)`,
		`DROP TABLE IF EXISTS media_files`,
	}

	for i, m := range migrations {
		if _, err := DB.Exec(ctx, m); err != nil {
			return fmt.Errorf("migration #%d failed: %w", i+1, err)
		}
	}

	return nil
}

func Close() {
	if DB != nil {
		DB.Close()
	}
}
