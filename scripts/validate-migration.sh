#!/bin/bash
# Migration validation script for schedule conflict detection
# This script validates migration version consistency across:
# - VERSION file
# - LATEST.sql schema_version
# - Migration file naming

set -euo pipefail

MIGRATION_DIR="store/migration/postgres"
VERSION_FILE="$MIGRATION_DIR/VERSION"
LATEST_SQL="$MIGRATION_DIR/LATEST.sql"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "==================================="
echo "Migration Validation Script"
echo "==================================="
echo

# Read VERSION file
if [ ! -f "$VERSION_FILE" ]; then
    echo -e "${RED}❌ VERSION file not found: $VERSION_FILE${NC}"
    exit 1
fi

VERSION=$(cat "$VERSION_FILE" | tr -d '[:space:]')
echo "📋 VERSION file: $VERSION"

# Extract schema_version from LATEST.sql
if [ ! -f "$LATEST_SQL" ]; then
    echo -e "${RED}❌ LATEST.sql not found: $LATEST_SQL${NC}"
    exit 1
fi

SCHEMA_VERSION=$(grep "schema_version" "$LATEST_SQL" | sed -n "s/.*schema_version.*'\([0-9.]*\)'.*/\1/p" | head -1)
echo "📋 LATEST.sql schema_version: $SCHEMA_VERSION"

# Check if versions match
if [ "$VERSION" != "$SCHEMA_VERSION" ]; then
    echo -e "${RED}❌ Version mismatch! VERSION=$VERSION, LATEST.sql=$SCHEMA_VERSION${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Version consistency check passed${NC}"
echo

# Check for migration file naming convention
echo "Checking migration files..."
MIGRATION_FILE="$MIGRATION_DIR/V${VERSION}__schedule_conflict_constraint.sql"
if [ -f "$MIGRATION_FILE" ]; then
    echo -e "${GREEN}✓ Migration file found: $MIGRATION_FILE${NC}"

    # Check if migration file content is reflected in LATEST.sql
    if grep -q "no_overlapping_schedules" "$LATEST_SQL"; then
        echo -e "${GREEN}✓ Migration constraint present in LATEST.sql${NC}"
    else
        echo -e "${YELLOW}⚠ Warning: no_overlapping_schedules constraint not found in LATEST.sql${NC}"
    fi
else
    echo -e "${YELLOW}⚠ Warning: Migration file not found: $MIGRATION_FILE${NC}"
fi

echo

# Validate SQL syntax if psql is available
if command -v psql &> /dev/null; then
    echo "Validating SQL syntax..."
    if psql --help | grep -q -- "--parse-only"; then
        if psql --parse-only -f "$LATEST_SQL" 2>&1 | grep -q "ERROR"; then
            echo -e "${RED}❌ SQL syntax error in LATEST.sql${NC}"
            psql --parse-only -f "$LATEST_SQL"
            exit 1
        else
            echo -e "${GREEN}✓ SQL syntax validation passed${NC}"
        fi
    fi
else
    echo -e "${YELLOW}⚠ psql not found, skipping SQL syntax validation${NC}"
fi

echo
echo "==================================="
echo -e "${GREEN}✅ All validation checks passed!${NC}"
echo "==================================="
