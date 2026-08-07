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

type QualityInfo struct {
	Quality          string `json:"quality"`
	HLSPath          string `json:"hls_path"`
	CloudinaryFolder string `json:"cloudinary_folder"`
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
		`CREATE TABLE IF NOT EXISTS media_files (
			id SERIAL PRIMARY KEY,
			tmdb_id INTEGER NOT NULL,
			media_type VARCHAR(10) NOT NULL DEFAULT 'movie',
			movie_name VARCHAR(255) NOT NULL,
			quality VARCHAR(10) NOT NULL DEFAULT '1080p',
			cloudinary_folder VARCHAR(255) NOT NULL,
			created_at TIMESTAMP DEFAULT NOW(),
			UNIQUE(tmdb_id, quality)
		)`,
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

func ScanMediaFiles() {
	logger.Info("Escaneando media_files desde base de datos...")
	
	ctx := context.Background()
	rows, err := DB.Query(ctx, `
		SELECT tmdb_id, media_type, movie_name, quality, cloudinary_folder
		FROM media_files
		ORDER BY tmdb_id, quality
	`)
	if err != nil {
		logger.Warn("Error escaneando media_files: %v", err)
		return
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var tmdbID int
		var mediaType, movieName, quality, cloudinaryFolder string
		if err := rows.Scan(&tmdbID, &mediaType, &movieName, &quality, &cloudinaryFolder); err != nil {
			logger.Warn("Error leyendo fila: %v", err)
			continue
		}
		count++
		logger.Info("Encontrado: tmdb_id=%d, quality=%s, folder=%s", tmdbID, quality, cloudinaryFolder)
	}

	if count > 0 {
		logger.Success("Found %d media file(s) in database", count)
	} else {
		logger.Info("No media files found in database")
	}
}

func SaveMediaFile(tmdbID int, mediaType, movieName, quality, cloudinaryFolder string) error {
	ctx := context.Background()
	_, err := DB.Exec(ctx,
		`INSERT INTO media_files (tmdb_id, media_type, movie_name, quality, cloudinary_folder)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (tmdb_id, quality) DO UPDATE SET 
		 	movie_name = $3, 
		 	cloudinary_folder = $5`,
		tmdbID, mediaType, movieName, quality, cloudinaryFolder,
	)
	return err
}

func GetStreamInfo(tmdbID string) (string, []QualityInfo, bool) {
	ctx := context.Background()
	rows, err := DB.Query(ctx,
		"SELECT quality, cloudinary_folder FROM media_files WHERE tmdb_id = $1 ORDER BY quality DESC",
		tmdbID,
	)
	if err != nil {
		return "", nil, false
	}
	defer rows.Close()

	var qualities []QualityInfo
	var movieName string

	for rows.Next() {
		var q QualityInfo
		if err := rows.Scan(&q.Quality, &q.CloudinaryFolder); err != nil {
			continue
		}
		qualities = append(qualities, q)

		if movieName == "" {
			DB.QueryRow(ctx,
				"SELECT movie_name FROM media_files WHERE tmdb_id = $1 LIMIT 1",
				tmdbID,
			).Scan(&movieName)
		}
	}

	if len(qualities) == 0 {
		return "", nil, false
	}

	return movieName, qualities, true
}

func GetCloudinaryFolder(movieName, quality string) (string, error) {
	ctx := context.Background()
	var folder string
	err := DB.QueryRow(ctx,
		"SELECT cloudinary_folder FROM media_files WHERE movie_name = $1 AND quality = $2",
		movieName, quality,
	).Scan(&folder)
	return folder, err
}
