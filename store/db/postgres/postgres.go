package postgres

import (
	"context"
	"database/sql"
	"log"
	"time"

	// Import the PostgreSQL driver.
	_ "github.com/lib/pq"
	"github.com/pkg/errors"

	"github.com/usememos/memos/internal/profile"
	"github.com/usememos/memos/store"
)

type DB struct {
	db      *sql.DB
	profile *profile.Profile
}

func NewDB(profile *profile.Profile) (store.Driver, error) {
	if profile == nil {
		return nil, errors.New("profile is nil")
	}

	// Open the PostgreSQL connection
	db, err := sql.Open("postgres", profile.DSN)
	if err != nil {
		log.Printf("Failed to open database: %s", err)
		return nil, errors.Wrapf(err, "failed to open database: %s", profile.DSN)
	}

	// Configure connection pool for 2C2G environment
	// These settings are optimized for low-resource environments
	db.SetMaxOpenConns(10)  // Limit total connections (default is unlimited)
	db.SetMaxIdleConns(5)   // Keep idle connections ready (default is 2)
	db.SetConnMaxLifetime(1 * time.Hour)       // Recycle connections after 1 hour
	db.SetConnMaxIdleTime(10 * time.Minute)     // Don't keep idle connections too long

	var driver store.Driver = &DB{
		db:      db,
		profile: profile,
	}

	// Return the DB struct
	return driver, nil
}

func (d *DB) GetDB() *sql.DB {
	return d.db
}

func (d *DB) Close() error {
	return d.db.Close()
}

func (d *DB) IsInitialized(ctx context.Context) (bool, error) {
	var exists bool
	err := d.db.QueryRowContext(ctx, "SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_catalog = current_database() AND table_name = 'memo' AND table_type = 'BASE TABLE')").Scan(&exists)
	if err != nil {
		return false, errors.Wrap(err, "failed to check if database is initialized")
	}
	return exists, nil
}
