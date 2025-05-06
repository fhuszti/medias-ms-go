set -euo pipefail

read -p "Enter migration name (e.g. create_npcs_table): " name

if [[ -z "$name" ]]; then
  echo "❌ Migration name cannot be empty" >&2
  exit 1
fi

DIR="$(cd "$(dirname "$0")/../internal/migration/migrations" && pwd)"

if command -v migrate &>/dev/null; then
  MIGRATE_CMD=migrate
else
  echo "ℹ️  migrate CLI not found, running via go run…" >&2
  MIGRATE_CMD="go run github.com/golang-migrate/migrate/v4/cmd/migrate@v4.17.1"
fi

$MIGRATE_CMD create -ext sql -dir "$DIR" "$name"