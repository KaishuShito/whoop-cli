#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
WJ="$PROJECT_DIR/dist/whoop-cli"
CONFIG_FILE="$PROJECT_DIR/skill/config.json"
ENV_FILE="$PROJECT_DIR/.env"
TOKENS_FILE="$PROJECT_DIR/tokens.json"
TARGET_DATE="$(TZ=Asia/Tokyo date -v-1d +%F 2>/dev/null || python3 - <<'PY'
from datetime import datetime, timedelta, timezone
print((datetime.now(timezone(timedelta(hours=9))) - timedelta(days=1)).strftime("%Y-%m-%d"))
PY
)"

failures=0

check_file() {
  local label="$1"
  local path="$2"
  if [[ -e "$path" ]]; then
    echo "$label=ok path=$path"
  else
    echo "$label=missing path=$path"
    failures=$((failures + 1))
  fi
}

run_check() {
  local label="$1"
  shift
  if output="$("$@" 2>&1)"; then
    echo "$label=ok"
  else
    echo "$label=failed"
    echo "$output" >&2
    failures=$((failures + 1))
  fi
}

check_file "skill_config" "$CONFIG_FILE"
check_file "env_file" "$ENV_FILE"
check_file "tokens_file" "$TOKENS_FILE"

if [[ ! -x "$WJ" ]]; then
  echo "binary=missing path=$WJ"
  echo "hint=run_go_build"
  exit 1
fi
echo "binary=ok path=$WJ"

cd "$PROJECT_DIR"

run_check "status_check" "$WJ" status
run_check "whoop_fetch" "$WJ" fetch --json --date "$TARGET_DATE"
run_check "weather_fetch" "$WJ" weather --json --date "$TARGET_DATE"
run_check "airquality_fetch" "$WJ" airquality --json --date "$TARGET_DATE"

if [[ "$failures" -gt 0 ]]; then
  echo "health=degraded failures=$failures"
  exit 1
fi

echo "health=ok date=$TARGET_DATE"
