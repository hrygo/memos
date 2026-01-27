package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/usememos/memos/store"
)

func (d *DB) UpsertUserPreferences(ctx context.Context, upsert *store.UpsertUserPreferences) (*store.UserPreferences, error) {
	now := time.Now().Unix()

	stmt := `INSERT INTO user_preferences (user_id, preferences, created_ts, updated_ts)
		VALUES (` + placeholder(1) + `, ` + placeholder(2) + `, ` + placeholder(3) + `, ` + placeholder(4) + `)
		ON CONFLICT (user_id) DO UPDATE SET
			preferences = EXCLUDED.preferences,
			updated_ts = EXCLUDED.updated_ts
		RETURNING user_id, preferences, created_ts, updated_ts`

	result := &store.UserPreferences{}
	err := d.db.QueryRowContext(ctx, stmt, upsert.UserID, upsert.Preferences, now, now).Scan(
		&result.UserID,
		&result.Preferences,
		&result.CreatedTs,
		&result.UpdatedTs,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert user_preferences: %w", err)
	}

	return result, nil
}

func (d *DB) GetUserPreferences(ctx context.Context, find *store.FindUserPreferences) (*store.UserPreferences, error) {
	if find.UserID == nil {
		return nil, fmt.Errorf("user_id is required")
	}

	query := `SELECT user_id, preferences, created_ts, updated_ts FROM user_preferences WHERE user_id = ` + placeholder(1)

	result := &store.UserPreferences{}
	err := d.db.QueryRowContext(ctx, query, *find.UserID).Scan(
		&result.UserID,
		&result.Preferences,
		&result.CreatedTs,
		&result.UpdatedTs,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Not found, return nil without error
		}
		return nil, fmt.Errorf("failed to get user_preferences: %w", err)
	}

	return result, nil
}
