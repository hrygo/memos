package db

import (
	"github.com/pkg/errors"

	"github.com/usememos/memos/internal/profile"
	"github.com/usememos/memos/store"
	"github.com/usememos/memos/store/db/postgres"
	"github.com/usememos/memos/store/db/sqlite"
)

// ============================================================================
// DATABASE SUPPORT POLICY
// ============================================================================
// This project supports only PostgreSQL and SQLite databases.
//
// PostgreSQL: Full support for production use with all AI features.
// SQLite: Limited support for development/testing only (no advanced AI).
// MySQL: NOT SUPPORTED - all MySQL code has been removed.
//
// When adding new features:
// - Implement fully for PostgreSQL
// - Implement for SQLite ONLY if high ROI and low maintenance cost
// - Do NOT add MySQL support under any circumstances
// ============================================================================

// NewDBDriver creates new db driver based on profile.
func NewDBDriver(profile *profile.Profile) (store.Driver, error) {
	var driver store.Driver
	var err error

	switch profile.Driver {
	case "sqlite":
		driver, err = sqlite.NewDB(profile)
	case "postgres":
		driver, err = postgres.NewDB(profile)
	default:
		return nil, errors.New("unknown db driver: only 'postgres' and 'sqlite' are supported (MySQL is not supported)")
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed to create db driver")
	}
	return driver, nil
}
