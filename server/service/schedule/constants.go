package schedule

// Package-level constants for schedule management.

const (
	// DefaultTimezone is the default timezone for schedule operations when not specified.
	DefaultTimezone = "Asia/Shanghai"

	// MaxInstances is the maximum number of instances to expand for recurring schedules.
	// This prevents excessive memory usage and processing time for schedules with
	// very long recurrence periods.
	MaxInstances = 500

	// MaxIterations is the maximum number of reasoning cycles for the scheduler agent.
	// This prevents infinite loops in the ReAct reasoning process.
	MaxIterations = 5
)
