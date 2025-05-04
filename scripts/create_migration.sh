set -euo pipefail

read -p "Enter migration name (e.g. create_npcs_table): " name

if [[ -z "$name" ]]; then
  echo "❌ Migration name cannot be empty" >&2
  exit 1
fi

migration create -ext sql -dir "$(dirname "$0")/../internal/migration/migrations" "$name"