package database

import (
	"database/sql"
	"fmt"
	"os"
	"time"  // Adicionar este import

	_ "github.com/lib/pq"
)

// Config holds database configuration
type Config struct {
	URL string
}

// NewConnection creates a new database connection
func NewConnection(cfg Config) (*sql.DB, error) {
	url := cfg.URL
	if url == "" {
		url = os.Getenv("DATABASE_URL")
	}

	if url == "" {
		return nil, fmt.Errorf("DATABASE_URL not configured")
	}

	db, err := sql.Open("postgres", url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %v", err)
	}

	// Connection pool configuration
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	return db, nil
}

// DomainExists checks if a domain is active in proxy_routes
func DomainExists(db *sql.DB, domain string) (bool, error) {
	var exists bool
	err := db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM proxy_routes 
			WHERE domain = $1 AND active = true
		)
	`, domain).Scan(&exists)

	return exists, err
}

// GetUpstream returns the upstream for a domain
func GetUpstream(db *sql.DB, domain string) (string, error) {
	var upstream string
	err := db.QueryRow(`
		SELECT upstream FROM proxy_routes 
		WHERE domain = $1 AND active = true
	`, domain).Scan(&upstream)

	return upstream, err
}

// IsSSLEnabled checks if SSL is enabled for a domain
func IsSSLEnabled(db *sql.DB, domain string) (bool, error) {
	var enabled bool
	err := db.QueryRow(`
		SELECT ssl_enabled FROM proxy_routes 
		WHERE domain = $1 AND active = true
	`, domain).Scan(&enabled)

	return enabled, err
}