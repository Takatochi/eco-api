#!/usr/bin/env bash
set -euo pipefail

# ---------------------------------------------------------------------------
# run.sh — start eco-api for local development
#
# Usage:
#   ./scripts/run.sh           # start DB in Docker, run API with go run
#   ./scripts/run.sh --docker  # start full stack with docker compose
#   ./scripts/run.sh --down    # stop and remove all containers + volumes
# ---------------------------------------------------------------------------

ROOT="$(cd "$(dirname "$0")/.." && pwd)"

log()  { echo "[run] $*"; }
die()  { echo "[run] ERROR: $*" >&2; exit 1; }

require() {
  command -v "$1" &>/dev/null || die "'$1' not found — please install it"
}

# ── flags ──────────────────────────────────────────────────────────────────
MODE="local"
case "${1:-}" in
  --docker) MODE="docker" ;;
  --down)   MODE="down"   ;;
  --help|-h)
    sed -n '/^# Usage/,/^# ---/p' "$0" | grep -v '^#.*---' | sed 's/^# //'
    exit 0
    ;;
esac

# ── down ───────────────────────────────────────────────────────────────────
if [[ "$MODE" == "down" ]]; then
  require docker
  log "stopping all containers and removing volumes..."
  docker compose -f "$ROOT/docker-compose.yml" down -v
  log "done."
  exit 0
fi

# ── full docker stack ───────────────────────────────────────────────────────
if [[ "$MODE" == "docker" ]]; then
  require docker
  log "building and starting full stack (DB + blockchain + API + simulators)..."
  docker compose -f "$ROOT/docker-compose.yml" up -d --build
  log ""
  log "API:     http://localhost:8080"
  log "Swagger: http://localhost:8080/swagger/index.html"
  log "pgAdmin: http://localhost:5050  (user@domain.com / 1234)"
  log ""
  log "to stop: $0 --down"
  exit 0
fi

# ── local go run (default) ─────────────────────────────────────────────────
require go
require docker

# Start only the DB container if it is not already healthy.
DB_CONTAINER="eco_db"
if ! docker ps --format '{{.Names}}' | grep -q "^${DB_CONTAINER}$"; then
  log "starting postgres container..."
  docker compose -f "$ROOT/docker-compose.yml" up -d db
fi

log "waiting for postgres to be ready..."
for i in $(seq 1 20); do
  if docker exec "$DB_CONTAINER" pg_isready -U eco -d eco -q 2>/dev/null; then
    break
  fi
  if [[ "$i" -eq 20 ]]; then
    die "postgres did not become ready in time"
  fi
  sleep 1
done

# Build env from .env if present, but let existing env vars take priority.
if [[ -f "$ROOT/.env" ]]; then
  log "loading .env"
  set -o allexport
  # shellcheck source=/dev/null
  source "$ROOT/.env"
  set +o allexport
fi

export DATABASE_URL="${DATABASE_URL:-postgres://eco:eco@localhost:5432/eco?sslmode=disable}"
export PORT="${PORT:-8080}"

log "starting API on :${PORT}  (DATABASE_URL=${DATABASE_URL})"
log "press Ctrl-C to stop"
log ""
cd "$ROOT" && go run .
