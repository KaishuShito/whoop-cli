#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
SKILL_DIR="$PROJECT_DIR/skill"
CONFIG_FILE="$SKILL_DIR/config.json"
ENV_FILE="$PROJECT_DIR/.env"

DEFAULT_LOCATION_NAME="Tokyo Hiroo"
DEFAULT_LAT="35.6503"
DEFAULT_LON="139.7225"
DEFAULT_JOURNAL_DIR="$HOME/journal"

mkdir -p "$SKILL_DIR"

read_existing_json_value() {
  local key="$1"
  local file="$2"
  if [[ ! -f "$file" ]]; then
    return 0
  fi
  python3 - "$key" "$file" <<'PY'
import json
import sys

key = sys.argv[1]
path = sys.argv[2]
try:
    with open(path, "r", encoding="utf-8") as f:
        data = json.load(f)
except Exception:
    sys.exit(0)

value = data
for part in key.split("."):
    if isinstance(value, dict) and part in value:
        value = value[part]
    else:
        sys.exit(0)

if value is None:
    sys.exit(0)
print(value)
PY
}

read_existing_env_value() {
  local key="$1"
  local file="$2"
  [[ -f "$file" ]] || return 0
  awk -F= -v k="$key" '$1 == k {print substr($0, index($0, "=") + 1)}' "$file" | tail -n 1
}

prompt_value() {
  local label="$1"
  local default_value="$2"
  local secret="${3:-false}"
  local value=""
  if [[ "$secret" == "true" ]]; then
    read -r -s -p "$label [$default_value]: " value
    echo
  else
    read -r -p "$label [$default_value]: " value
  fi
  if [[ -z "$value" ]]; then
    value="$default_value"
  fi
  printf '%s' "$value"
}

existing_client_id="$(read_existing_json_value "whoop.client_id" "$CONFIG_FILE")"
existing_client_secret="$(read_existing_json_value "whoop.client_secret" "$CONFIG_FILE")"
existing_journal_dir="$(read_existing_json_value "journal.directory" "$CONFIG_FILE")"
existing_location_name="$(read_existing_json_value "location.name" "$CONFIG_FILE")"
existing_lat="$(read_existing_json_value "location.lat" "$CONFIG_FILE")"
existing_lon="$(read_existing_json_value "location.lon" "$CONFIG_FILE")"
if [[ -z "$existing_client_id" ]]; then
  existing_client_id="$(read_existing_env_value "WHOOP_CLIENT_ID" "$ENV_FILE")"
fi
if [[ -z "$existing_client_secret" ]]; then
  existing_client_secret="$(read_existing_env_value "WHOOP_CLIENT_SECRET" "$ENV_FILE")"
fi
if [[ -z "$existing_journal_dir" ]]; then
  existing_journal_dir="$(read_existing_env_value "JOURNAL_DIR" "$ENV_FILE")"
fi
if [[ -z "$existing_journal_dir" ]]; then
  existing_journal_dir="$(read_existing_env_value "VAULT_JOURNAL_DIR" "$ENV_FILE")"
fi
if [[ -z "$existing_lat" ]]; then
  existing_lat="$(read_existing_env_value "WEATHER_LAT" "$ENV_FILE")"
fi
if [[ -z "$existing_lon" ]]; then
  existing_lon="$(read_existing_env_value "WEATHER_LON" "$ENV_FILE")"
fi
CLIENT_ID="$(prompt_value "WHOOP Client ID" "${existing_client_id:-}")"
CLIENT_SECRET="$(prompt_value "WHOOP Client Secret" "${existing_client_secret:-}" true)"
JOURNAL_DIR="$(prompt_value "Journal directory path" "${existing_journal_dir:-$DEFAULT_JOURNAL_DIR}")"
LOCATION_NAME="$(prompt_value "Location name" "${existing_location_name:-$DEFAULT_LOCATION_NAME}")"
LAT="$(prompt_value "Weather latitude" "${existing_lat:-$DEFAULT_LAT}")"
LON="$(prompt_value "Weather longitude" "${existing_lon:-$DEFAULT_LON}")"

python3 - "$CONFIG_FILE" "$CLIENT_ID" "$CLIENT_SECRET" "$JOURNAL_DIR" "$LOCATION_NAME" "$LAT" "$LON" <<'PY'
import json
import sys
from pathlib import Path

config_path = Path(sys.argv[1])
config = {
    "whoop": {
        "client_id": sys.argv[2],
        "client_secret": sys.argv[3],
        "redirect_uri": "http://localhost:8080/callback",
    },
    "journal": {
        "directory": sys.argv[4],
    },
    "location": {
        "name": sys.argv[5],
        "lat": float(sys.argv[6]),
        "lon": float(sys.argv[7]),
    },
    "weather": {
        "enabled": True,
    },
    "airquality": {
        "enabled": True,
    },
}
config_path.write_text(json.dumps(config, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
PY

if [[ -f "$ENV_FILE" ]]; then
  backup="$ENV_FILE.bak.$(date +%Y%m%d%H%M%S)"
  cp "$ENV_FILE" "$backup"
  echo "Backed up existing .env to $backup"
fi

cat > "$ENV_FILE" <<EOF
WHOOP_CLIENT_ID=$CLIENT_ID
WHOOP_CLIENT_SECRET=$CLIENT_SECRET
WHOOP_REDIRECT_URI=http://localhost:8080/callback
JOURNAL_DIR=$JOURNAL_DIR
WEATHER_ENABLED=true
WEATHER_LAT=$LAT
WEATHER_LON=$LON
AIRQUALITY_ENABLED=true
EOF

chmod 600 "$ENV_FILE"

echo
echo "Setup complete."
echo "  skill config: $CONFIG_FILE"
echo "  env file:     $ENV_FILE"
echo
echo "Next steps:"
echo "  cd $PROJECT_DIR"
echo "  go build -o ./dist/whoop-cli ./cmd/whoop-cli"
echo "  ./dist/whoop-cli setup"
echo "  bash skill/scripts/health-check.sh"
