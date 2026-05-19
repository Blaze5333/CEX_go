package db

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // registers "pgx" driver with database/sql
)

const postgresTag = "[db/postgres]"

// PostgresConfig holds the connection-pool settings.
type PostgresConfig struct {
	DSN             string        // e.g. "postgres://user:pass@host:5432/dbname?sslmode=disable"
	MaxOpenConns    int           // max open connections (default 25)
	MaxIdleConns    int           // max idle connections (default 10)
	ConnMaxLifetime time.Duration // max connection lifetime (default 30 min)
	ConnMaxIdleTime time.Duration // max idle time before closing (default 10 min)
}

// NewPostgres opens a *sql.DB backed by pgx and verifies connectivity with
// a ping.  Caller is responsible for calling db.Close() on shutdown.
func NewPostgres(cfg PostgresConfig) (*sql.DB, error) {
	log.Printf("%s NewPostgres: initializing postgres connection", postgresTag)
	if cfg.DSN == "" {
		log.Printf("%s NewPostgres: DSN is empty", postgresTag)
		return nil, fmt.Errorf("db: postgres DSN must not be empty")
	}

	// Apply sensible defaults.
	if cfg.MaxOpenConns == 0 {
		cfg.MaxOpenConns = 25
	}
	if cfg.MaxIdleConns == 0 {
		cfg.MaxIdleConns = 10
	}
	if cfg.ConnMaxLifetime == 0 {
		cfg.ConnMaxLifetime = 30 * time.Minute
	}
	if cfg.ConnMaxIdleTime == 0 {
		cfg.ConnMaxIdleTime = 10 * time.Minute
	}

	log.Printf("%s NewPostgres: opening connection maxOpenConns=%d maxIdleConns=%d", postgresTag, cfg.MaxOpenConns, cfg.MaxIdleConns)
	db, err := sql.Open("pgx", cfg.DSN)
	if err != nil {
		log.Printf("%s NewPostgres: failed to open postgres connection: %v", postgresTag, err)
		return nil, fmt.Errorf("db: open postgres: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	if err := db.Ping(); err != nil {
		log.Printf("%s NewPostgres: failed to ping postgres: %v", postgresTag, err)
		db.Close()
		return nil, fmt.Errorf("db: ping postgres: %w", err)
	}

	log.Printf("%s NewPostgres: postgres connection established successfully", postgresTag)
	return db, nil
}
