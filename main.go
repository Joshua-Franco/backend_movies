package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"backend_movies/config"
	"backend_movies/database"
	"backend_movies/logger"
	"backend_movies/routes"
	"backend_movies/services"
)

func main() {
	logger.Banner()

	logger.Info("Loading configuration...")
	cfg, err := config.Load()
	if err != nil {
		logger.Error("Failed to load config: %v", err)
		waitForInput()
		return
	}
	logger.Success("Configuration loaded")

	logger.Info("Connecting to PostgreSQL (%s:%s/%s)...", cfg.DBHost, cfg.DBPort, cfg.DBName)
	if err := database.Connect(cfg); err != nil {
		logger.Error("Could not connect to PostgreSQL")
		fmt.Println()
		logger.Error("Reason: %v", err)
		fmt.Println()
		logger.Warn("Make sure PostgreSQL is running on %s:%s", cfg.DBHost, cfg.DBPort)
		logger.Warn("Windows:  net start postgresql-x64-18")
		fmt.Println()
		waitForInput()
		return
	}
	defer database.Close()
	logger.Success("Connected to PostgreSQL")

	logger.Info("Running migrations...")
	if err := database.RunMigrations(); err != nil {
		logger.Error("Failed to run migrations: %v", err)
		waitForInput()
		return
	}
	logger.Success("Migrations completed")

	logger.Info("Scanning local media files...")
	database.ScanMediaFiles()

	logger.Info("Initializing TMDB client...")
	tmdb := services.NewTMDBClient(cfg)
	cache := services.NewCacheService()
	logger.Success("TMDB client ready")

	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		for range ticker.C {
			cache.CleanExpired()
		}
	}()

	router := routes.SetupRoutes(tmdb, cache)

	port := cfg.ServerPort
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	logger.Separator()
	logger.Success("Server running on %s", fmt.Sprintf("http://localhost:%s", port))
	logger.Info("TMDB API key: %s...", cfg.TMDBAPIKey[:8])
	logger.Info("Press Ctrl+C to stop")
	logger.Separator()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server failed: %v", err)
		}
	}()

	<-quit
	logger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown: %v", err)
	}

	logger.Success("Server stopped")
}

func waitForInput() {
	fmt.Println()
	logger.Info("Press Enter to close...")
	bufio := make([]byte, 1)
	os.Stdin.Read(bufio)
}
