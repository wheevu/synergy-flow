#!/usr/bin/env bash
# ────────────────────────────────────────────────────────────
# SynergyFlow — Reset demo database script
# ────────────────────────────────────────────────────────────
# DANGER: This script drops all application data and re-runs
# migrations + seed data.  It is intended ONLY for the
# production-like demo / staging environment.
#
# Protection: the RESET_DEMO_DB_CONFIRM env variable must be
# set to "yes" before the script will execute.
# ────────────────────────────────────────────────────────────
set -euo pipefail

if [ "${RESET_DEMO_DB_CONFIRM:-}" != "yes" ]; then
  echo "ERROR: This script will DESTROY ALL EXISTING DATA."
  echo "Set RESET_DEMO_DB_CONFIRM=yes to proceed."
  echo ""
  echo "Usage:"
  echo "  RESET_DEMO_DB_CONFIRM=yes ./scripts/reset-demo-db.sh"
  exit 1
fi

COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.prod.yml}"
COMPOSE="docker compose -f $COMPOSE_FILE"

echo "=== SynergyFlow Demo DB Reset ==="
echo "WARNING: Dropping all application data..."
echo ""

# 1. Drop and recreate the schema
echo "--- Dropping public schema ---"
$COMPOSE exec -T postgres psql -U "${POSTGRES_USER:-synergy}" -d "${POSTGRES_DB:-synergyflow}" -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"

# 2. Re-run migrations
echo "--- Re-running migrations ---"
$COMPOSE run --rm migrate

# 3. Seed data is embedded in migration files (002_seed.sql etc.)
echo "--- Seed data included in migrations (002_seed, 003_expand_demo_data, etc.) ---"

echo ""
echo "=== Demo database has been reset to its seeded state ==="
echo "You can now log in with the demo account:"
echo "  Email:    demo@synergyflow.dev"
echo "  Password: password123"
echo ""
echo "Restart the backend to clear in-memory / Redis state if needed:"
echo "  $COMPOSE restart backend worker"
