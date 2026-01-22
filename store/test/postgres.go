package test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	testUser            = "testuser"
	testPassword        = "testpassword"
	StableMemosVersion  = "0.23.0" // Stable version for migration testing
)

// MemosContainerConfig holds configuration for running a Memos container.
type MemosContainerConfig struct {
	Driver  string // "sqlite" or "postgres"
	DSN     string // Database connection string
	DataDir string // Data directory for SQLite
	Version string // Memos version tag ("0.23.0", "local", etc.)
}

// MemosContainer represents a running Memos container.
type MemosContainer struct {
	container testcontainers.Container
}

// Terminate stops the Memos container.
func (mc *MemosContainer) Terminate(ctx context.Context) error {
	return mc.container.Terminate(ctx)
}

// GetPostgresDSN returns a DSN for PostgreSQL testing.
// It uses testcontainers to create a fresh PostgreSQL instance for each test.
func GetPostgresDSN(t *testing.T) string {
	// Check if a custom DSN is provided via environment variable
	if dsn := os.Getenv("POSTGRES_TEST_DSN"); dsn != "" {
		return dsn
	}

	// Use testcontainers for automated testing
	pgContainer, err := postgres.Run(t.Context(),
		"postgres:16-alpine",
		testcontainers.WithImage("postgres:16-alpine"),
		postgres.WithDatabase("memos_test"),
		postgres.WithUsername(testUser),
		postgres.WithPassword(testPassword),
	)
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}

	// Store container for cleanup
	t.Cleanup(func() {
		if err := pgContainer.Terminate(t.Context()); err != nil {
			t.Logf("failed to terminate postgres container: %v", err)
		}
	})

	connStr, err := pgContainer.ConnectionString(t.Context(), "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

	return connStr
}

// GetDedicatedPostgresDSN creates a dedicated PostgreSQL container for migration testing.
// It returns the DSN, container hostname, and a cleanup function.
func GetDedicatedPostgresDSN(t *testing.T) (dsn string, containerHost string, cleanup func()) {
	pgContainer, err := postgres.Run(t.Context(),
		"postgres:16-alpine",
		postgres.WithDatabase("memos_test"),
		postgres.WithUsername(testUser),
		postgres.WithPassword(testPassword),
	)
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}

	connStr, err := pgContainer.ConnectionString(t.Context(), "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

	// Get the container's host IP for internal networking
	host, err := pgContainer.Host(t.Context())
	if err != nil {
		t.Fatalf("failed to get container host: %v", err)
	}

	// Get the mapped port
	port, err := pgContainer.MappedPort(t.Context(), "5432")
	if err != nil {
		t.Fatalf("failed to get container port: %v", err)
	}

	containerHost = fmt.Sprintf("%s:%s", host, port.Port())

	return connStr, containerHost, func() {
		if err := pgContainer.Terminate(t.Context()); err != nil {
			t.Logf("failed to terminate postgres container: %v", err)
		}
	}
}

// StartMemosContainer starts a Memos container with the given configuration.
// For version="local", it builds from the local Dockerfile.
func StartMemosContainer(ctx context.Context, cfg MemosContainerConfig) (*MemosContainer, error) {
	image := "hrygo/memos:" + cfg.Version
	if cfg.Version == "local" {
		image = "memos-test:local"
	}

	// Build environment variables map
	env := map[string]string{
		"MEMOS_DRIVER": cfg.Driver,
	}

	switch cfg.Driver {
	case "sqlite":
		if cfg.DataDir == "" {
			return nil, fmt.Errorf("DataDir is required for SQLite")
		}
		env["MEMOS_DATA"] = cfg.DataDir
	case "postgres":
		if cfg.DSN == "" {
			return nil, fmt.Errorf("DSN is required for PostgreSQL")
		}
		env["MEMOS_DSN"] = cfg.DSN
	}

	// Create and start the container
	req := testcontainers.ContainerRequest{
		Image:       image,
		Env:         env,
		ExposedPorts: []string{"5230/tcp"},
		WaitingFor:  wait.ForLog("start HTTP server").WithStartupTimeout(30 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	return &MemosContainer{container: container}, nil
}

// TerminateContainers is a no-op for PostgreSQL (containers are self-cleaning via t.Cleanup).
// Kept for compatibility with existing test infrastructure.
func TerminateContainers() {
	// No-op: containers are cleaned up via t.Cleanup
}
