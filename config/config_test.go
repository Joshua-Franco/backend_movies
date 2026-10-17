package config

import (
	"net/url"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestNormalizeDatabaseURLRequiresTLSAndPreservesNeonParameters(t *testing.T) {
	raw := "postgresql://movie_user:p%40ss@ep-example-pooler.neon.tech:5432/neondb?sslmode=disable&channel_binding=require"

	got, err := normalizeDatabaseURL(raw)
	if err != nil {
		t.Fatalf("normalizeDatabaseURL() error = %v", err)
	}

	parsed, err := url.Parse(got)
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}
	query := parsed.Query()
	if got := query.Get("sslmode"); got != "require" {
		t.Errorf("sslmode = %q, want %q", got, "require")
	}
	if got := query.Get("channel_binding"); got != "require" {
		t.Errorf("channel_binding = %q, want %q", got, "require")
	}
	if got := parsed.User.Username(); got != "movie_user" {
		t.Errorf("username = %q, want %q", got, "movie_user")
	}
	if got, _ := parsed.User.Password(); got != "p@ss" {
		t.Errorf("password was not preserved correctly")
	}
	if _, err := pgxpool.ParseConfig(got); err != nil {
		t.Errorf("pgxpool rejected Neon connection parameters: %v", err)
	}
}

func TestNormalizeDatabaseURLPreservesVerifiedTLSMode(t *testing.T) {
	got, err := normalizeDatabaseURL("postgres://user@db.example:5432/neondb?sslmode=verify-full")
	if err != nil {
		t.Fatalf("normalizeDatabaseURL() error = %v", err)
	}

	parsed, err := url.Parse(got)
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}
	if got := parsed.Query().Get("sslmode"); got != "verify-full" {
		t.Errorf("sslmode = %q, want %q", got, "verify-full")
	}
}

func TestLegacyDatabaseURLUsesTLSAndEscapesCredentials(t *testing.T) {
	cfg := &Config{
		DBHost:     "db.example",
		DBPort:     "5432",
		DBUser:     "movie_user",
		DBPassword: "p@ss",
		DBName:     "movie database",
		DBSSLMode:  "disable",
	}

	got := legacyDatabaseURL(cfg)
	parsed, err := url.Parse(got)
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}
	if parsed.Query().Get("sslmode") != "require" {
		t.Errorf("sslmode = %q, want %q", parsed.Query().Get("sslmode"), "require")
	}
	if password, _ := parsed.User.Password(); password != "p@ss" {
		t.Errorf("password was not preserved correctly")
	}
	if parsed.Path != "/movie database" {
		t.Errorf("database path = %q, want %q", parsed.Path, "/movie database")
	}
}
