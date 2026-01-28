package sqlite

import (
	"context"
	"errors"
	"fmt"

	"github.com/hrygo/divinesense/store"
)

// AI conversation features are NOT supported on SQLite.
// SQLite is intended for development/testing only and does not support
// the advanced AI features including conversation persistence, vector search,
// and reranking. Use PostgreSQL for production AI features.
//
// Ref: https://github.com/hrygo/divinesense/docs/dev-guides/BACKEND_DB.md

var (
	// ErrSQLiteAINotSupported is returned when AI features are requested on SQLite.
	ErrSQLiteAINotSupported = errors.New("AI conversation features are not supported on SQLite. Use PostgreSQL for production AI features")
)

func (d *DB) CreateAIConversation(ctx context.Context, create *store.AIConversation) (*store.AIConversation, error) {
	return nil, ErrSQLiteAINotSupported
}

func (d *DB) ListAIConversations(ctx context.Context, find *store.FindAIConversation) ([]*store.AIConversation, error) {
	return nil, ErrSQLiteAINotSupported
}

func (d *DB) UpdateAIConversation(ctx context.Context, update *store.UpdateAIConversation) (*store.AIConversation, error) {
	return nil, ErrSQLiteAINotSupported
}

func (d *DB) DeleteAIConversation(ctx context.Context, delete *store.DeleteAIConversation) error {
	return ErrSQLiteAINotSupported
}

func (d *DB) CreateAIMessage(ctx context.Context, create *store.AIMessage) (*store.AIMessage, error) {
	return nil, ErrSQLiteAINotSupported
}

func (d *DB) ListAIMessages(ctx context.Context, find *store.FindAIMessage) ([]*store.AIMessage, error) {
	return nil, ErrSQLiteAINotSupported
}

func (d *DB) DeleteAIMessage(ctx context.Context, delete *store.DeleteAIMessage) error {
	return ErrSQLiteAINotSupported
}

// ListAIConversationsBasic is a minimal implementation for UI rendering only.
// Returns empty list since SQLite doesn't support AI conversations.
// This prevents the UI from breaking but doesn't provide actual functionality.
func (d *DB) ListAIConversationsBasic(ctx context.Context, find *store.FindAIConversation) ([]*store.AIConversation, error) {
	// Return empty list - UI will show "no conversations" message
	return []*store.AIConversation{}, nil
}

// Deprecated: Use CreateAIConversation instead (which will return an error on SQLite).
// This function is kept for backward compatibility during migration.
func (d *DB) createAIConversationLegacy(ctx context.Context, create *store.AIConversation) (*store.AIConversation, error) {
	var fields []string
	var placeholder []string
	var args []any

	// If ID is specified, use it (for fixed conversations)
	// Otherwise, let the database generate it
	if create.ID != 0 {
		fields = []string{"`id`", "`uid`", "`creator_id`", "`title`", "`parrot_id`", "`pinned`", "`created_ts`", "`updated_ts`"}
		placeholder = []string{"?", "?", "?", "?", "?", "?", "?", "?"}
		args = []any{create.ID, create.UID, create.CreatorID, create.Title, create.ParrotID, create.Pinned, create.CreatedTs, create.UpdatedTs}
		stmt := "INSERT INTO `ai_conversation` (" + fmt.Sprint(fields) + ") VALUES (" + fmt.Sprint(placeholder) + ")"
		if _, err := d.db.ExecContext(ctx, stmt, args...); err != nil {
			return nil, fmt.Errorf("failed to create ai_conversation with fixed id: %w", err)
		}
	} else {
		fields = []string{"`uid`", "`creator_id`", "`title`", "`parrot_id`", "`pinned`", "`created_ts`", "`updated_ts`"}
		placeholder = []string{"?", "?", "?", "?", "?", "?", "?"}
		args = []any{create.UID, create.CreatorID, create.Title, create.ParrotID, create.Pinned, create.CreatedTs, create.UpdatedTs}
		stmt := "INSERT INTO `ai_conversation` (" + fmt.Sprint(fields) + ") VALUES (" + fmt.Sprint(placeholder) + ")"
		res, err := d.db.ExecContext(ctx, stmt, args...)
		if err != nil {
			return nil, fmt.Errorf("failed to create ai_conversation: %w", err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			return nil, fmt.Errorf("failed to get last insert id: %w", err)
		}
		create.ID = int32(id)
	}

	return create, nil
}
