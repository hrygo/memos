// Package mysql provides MySQL database driver for Memos.
//
// ⚠️ DEPRECATION NOTICE FOR SCHEDULE FEATURE
//
// The schedule assistant feature and all subsequent new features are NOT
// supported on MySQL. Please migrate to PostgreSQL or SQLite.
//
// Reasons:
//   - MySQL lacks advanced JSON field constraints
//   - Limited trigger capabilities compared to PostgreSQL/SQLite
//   - No vector search support (pgvector is PostgreSQL-only)
//   - High maintenance cost for three-database compatibility
//
// The Schedule implementation in this file is provided AS-IS without
// guarantees. Migration files contain known syntax errors that will not
// be fixed. Use PostgreSQL or SQLite instead.
package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/usememos/memos/store"
)

func (d *DB) CreateSchedule(ctx context.Context, create *store.Schedule) (*store.Schedule, error) {
	fields := []string{
		"uid", "creator_id", "title", "description", "location",
		"start_ts", "end_ts", "all_day", "timezone",
		"recurrence_rule", "recurrence_end_ts", "reminders", "payload",
	}
	placeholderValues := []any{
		create.UID, create.CreatorID, create.Title, create.Description, create.Location,
		create.StartTs, create.EndTs, create.AllDay, create.Timezone,
		create.RecurrenceRule, create.RecurrenceEndTs, create.Reminders, create.Payload,
	}

	// Add optional timestamps
	if create.CreatedTs != 0 {
		fields = append(fields, "created_ts")
		placeholderValues = append(placeholderValues, create.CreatedTs)
	}
	if create.UpdatedTs != 0 {
		fields = append(fields, "updated_ts")
		placeholderValues = append(placeholderValues, create.UpdatedTs)
	}

	stmt := `INSERT INTO schedule (` + strings.Join(fields, ", ") + `)
		VALUES (` + placeholders(len(placeholderValues)) + `)`

	result, err := d.db.ExecContext(ctx, stmt, placeholderValues...)
	if err != nil {
		return nil, fmt.Errorf("failed to create schedule: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}
	create.ID = int32(id)

	// Fetch the created schedule to get timestamps
	if err := d.db.QueryRowContext(ctx, "SELECT created_ts, updated_ts, row_status FROM schedule WHERE id = ?", create.ID).Scan(
		&create.CreatedTs,
		&create.UpdatedTs,
		&create.RowStatus,
	); err != nil {
		return nil, fmt.Errorf("failed to fetch created schedule: %w", err)
	}

	return create, nil
}

func (d *DB) ListSchedules(ctx context.Context, find *store.FindSchedule) ([]*store.Schedule, error) {
	where, args := []string{"1 = 1"}, []any{}

	if v := find.ID; v != nil {
		where, args = append(where, "schedule.id = ?"), append(args, *v)
	}
	if v := find.UID; v != nil {
		where, args = append(where, "schedule.uid = ?"), append(args, *v)
	}
	if v := find.CreatorID; v != nil {
		where, args = append(where, "schedule.creator_id = ?"), append(args, *v)
	}
	if v := find.RowStatus; v != nil {
		where, args = append(where, "schedule.row_status = ?"), append(args, *v)
	}
	if v := find.StartTs; v != nil {
		where, args = append(where, "schedule.start_ts >= ?"), append(args, *v)
	}
	if v := find.EndTs; v != nil {
		where, args = append(where, "schedule.start_ts <= ?"), append(args, *v)
	}

	// Ordering (always by start_ts ascending)
	orderBy := "ORDER BY schedule.start_ts ASC"

	query := `
		SELECT
			id, uid, creator_id, created_ts, updated_ts, row_status,
			title, description, location,
			start_ts, end_ts, all_day, timezone,
			recurrence_rule, recurrence_end_ts, reminders, payload
		FROM schedule
		WHERE ` + strings.Join(where, " AND ") + ` ` + orderBy

	if find.Limit != nil {
		query = fmt.Sprintf("%s LIMIT %d", query, *find.Limit)
		if find.Offset != nil {
			query = fmt.Sprintf("%s OFFSET %d", query, *find.Offset)
		}
	}

	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query schedules: %w", err)
	}
	defer rows.Close()

	list := make([]*store.Schedule, 0)
	for rows.Next() {
		var schedule store.Schedule
		var description, location, recurrenceRule, reminders, payload string
		var endTs, recurrenceEndTs sql.NullInt64

		if err := rows.Scan(
			&schedule.ID,
			&schedule.UID,
			&schedule.CreatorID,
			&schedule.CreatedTs,
			&schedule.UpdatedTs,
			&schedule.RowStatus,
			&schedule.Title,
			&description,
			&location,
			&schedule.StartTs,
			&endTs,
			&schedule.AllDay,
			&schedule.Timezone,
			&recurrenceRule,
			&recurrenceEndTs,
			&reminders,
			&payload,
		); err != nil {
			return nil, fmt.Errorf("failed to scan schedule: %w", err)
		}

		schedule.Description = description
		schedule.Location = location
		if endTs.Valid {
			schedule.EndTs = &endTs.Int64
		}
		if recurrenceRule != "" {
			schedule.RecurrenceRule = &recurrenceRule
		}
		if recurrenceEndTs.Valid {
			schedule.RecurrenceEndTs = &recurrenceEndTs.Int64
		}
		if reminders != "" && reminders != "[]" {
			schedule.Reminders = &reminders
		}
		if payload != "" && payload != "{}" {
			schedule.Payload = &payload
		}

		list = append(list, &schedule)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate schedules: %w", err)
	}

	return list, nil
}

func (d *DB) UpdateSchedule(ctx context.Context, update *store.UpdateSchedule) error {
	set, args := []string{}, []any{}

	if v := update.UID; v != nil {
		set, args = append(set, "uid = ?"), append(args, *v)
	}
	if v := update.RowStatus; v != nil {
		set, args = append(set, "row_status = ?"), append(args, *v)
	}
	if v := update.Title; v != nil {
		set, args = append(set, "title = ?"), append(args, *v)
	}
	if v := update.Description; v != nil {
		set, args = append(set, "description = ?"), append(args, *v)
	}
	if v := update.Location; v != nil {
		set, args = append(set, "location = ?"), append(args, *v)
	}
	if v := update.StartTs; v != nil {
		set, args = append(set, "start_ts = ?"), append(args, *v)
	}
	if v := update.EndTs; v != nil {
		set, args = append(set, "end_ts = ?"), append(args, *v)
	}
	if v := update.AllDay; v != nil {
		set, args = append(set, "all_day = ?"), append(args, *v)
	}
	if v := update.Timezone; v != nil {
		set, args = append(set, "timezone = ?"), append(args, *v)
	}
	if v := update.RecurrenceRule; v != nil {
		set, args = append(set, "recurrence_rule = ?"), append(args, *v)
	}
	if v := update.RecurrenceEndTs; v != nil {
		set, args = append(set, "recurrence_end_ts = ?"), append(args, *v)
	}
	if v := update.Reminders; v != nil {
		set, args = append(set, "reminders = ?"), append(args, *v)
	}
	if v := update.Payload; v != nil {
		set, args = append(set, "payload = ?"), append(args, *v)
	}

	// If no fields to update, return early
	if len(set) == 0 {
		return nil
	}

	args = append(args, update.ID)

	stmt := `UPDATE schedule SET ` + strings.Join(set, ", ") + ` WHERE id = ?`
	result, err := d.db.ExecContext(ctx, stmt, args...)
	if err != nil {
		return fmt.Errorf("failed to update schedule: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("schedule not found")
	}

	return nil
}

func (d *DB) DeleteSchedule(ctx context.Context, delete *store.DeleteSchedule) error {
	stmt := `DELETE FROM schedule WHERE id = ?`
	result, err := d.db.ExecContext(ctx, stmt, delete.ID)
	if err != nil {
		return fmt.Errorf("failed to delete schedule: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("schedule not found")
	}

	return nil
}
