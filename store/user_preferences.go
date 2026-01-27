package store

// UserPreferences represents user preferences for AI personalization.
type UserPreferences struct {
	UserID      int32
	Preferences string // JSON string
	CreatedTs   int64
	UpdatedTs   int64
}

// FindUserPreferences specifies the conditions for finding user preferences.
type FindUserPreferences struct {
	UserID *int32
}

// UpsertUserPreferences specifies the data for upserting user preferences.
type UpsertUserPreferences struct {
	UserID      int32
	Preferences string // JSON string
}
