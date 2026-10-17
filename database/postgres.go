package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"backend_movies/config"

	"github.com/jackc/pgx/v5/pgconn"
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

	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL())
	if err != nil {
		return fmt.Errorf("invalid PostgreSQL connection configuration; check DATABASE_URL or DB_* settings")
	}

	poolConfig.ConnConfig.ConnectTimeout = 10 * time.Second
	poolConfig.MaxConns = 10
	poolConfig.MinConns = 0
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return fmt.Errorf("PostgreSQL connection pool initialization failed: %w", err)
	}

	if err = pool.Ping(ctx); err != nil {
		pool.Close()
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "3D000" {
			return fmt.Errorf("PostgreSQL database %q does not exist; set DATABASE_URL to Neon's existing neondb database", cfg.DBName)
		}
		return fmt.Errorf("PostgreSQL ping failed: %w", err)
	}

	DB = pool
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
		`CREATE TABLE IF NOT EXISTS media_files (
			id SERIAL PRIMARY KEY,
			tmdb_id INTEGER UNIQUE NOT NULL,
			movie_name VARCHAR(255) NOT NULL,
			qualities TEXT[] DEFAULT '{}',
			created_at TIMESTAMP DEFAULT NOW()
		)`,
		`INSERT INTO media_files (tmdb_id, movie_name, qualities) VALUES (969681, 'Spider-Man-Brand-New-Day', ARRAY['1080p','720p','480p']) ON CONFLICT (tmdb_id) DO NOTHING`,
	}

	for i, migration := range migrations {
		if _, err := DB.Exec(ctx, migration); err != nil {
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
