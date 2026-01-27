package v1

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

// MetricsOverviewResponse represents the overview response of system metrics
type MetricsOverviewResponse struct {
	TotalRequests int64   `json:"total_requests"`
	SuccessRate   float64 `json:"success_rate"`
	AvgLatencyMs  int64   `json:"avg_latency_ms"`
	P50LatencyMs  int64   `json:"p50_latency_ms"`
	P95LatencyMs  int64   `json:"p95_latency_ms"`
	ErrorCount    int64   `json:"error_count"`
	TimeRange     string  `json:"time_range"`
	IsMock        bool    `json:"is_mock"` // indicates if the data is mock
}

// GetMetricsOverview returns the system metrics overview
// GET /api/v1/system/metrics/overview
func (s *APIV1Service) GetMetricsOverview(c echo.Context) error {
	// Parse time range parameter
	timeRange := c.QueryParam("range")
	if timeRange == "" {
		timeRange = "24h"
	}
	_, err := parseTimeRange(timeRange)
	if err != nil {
		slog.Warn("Invalid time range parameter in metrics request", "range", timeRange, "error", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid time range"})
	}

	// TODO: Implement actual metrics query logic
	// Currently returns mock data. IsMock=true indicates mock data.
	// In the future, real data can be fetched from metrics service.
	return c.JSON(http.StatusOK, MetricsOverviewResponse{
		TotalRequests: 0,
		SuccessRate:   0,
		AvgLatencyMs:  0,
		P50LatencyMs:  0,
		P95LatencyMs:  0,
		ErrorCount:    0,
		TimeRange:     timeRange,
		IsMock:        true,
	})
}

// parseTimeRange parses time range string and returns the start time
func parseTimeRange(timeRange string) (time.Time, error) {
	now := time.Now()
	switch timeRange {
	case "1h":
		return now.Add(-1 * time.Hour), nil
	case "24h":
		return now.Add(-24 * time.Hour), nil
	case "7d":
		return now.Add(-7 * 24 * time.Hour), nil
	case "30d":
		return now.Add(-30 * 24 * time.Hour), nil
	default:
		return time.Time{}, fmt.Errorf("invalid time range: %s (valid: 1h, 24h, 7d, 30d)", timeRange)
	}
}
