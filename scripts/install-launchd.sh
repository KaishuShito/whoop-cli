#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
LAUNCH_AGENTS_DIR="$HOME/Library/LaunchAgents"
LOG_DIR="$HOME/Library/Logs/whoop-cli"
LABEL="com.kai.whoop-cli.daily"
TEMPLATE="$PROJECT_DIR/launchd/${LABEL}.plist"
TARGET="$LAUNCH_AGENTS_DIR/${LABEL}.plist"

mkdir -p "$LAUNCH_AGENTS_DIR" "$LOG_DIR" "$PROJECT_DIR/dist"

# Check .env
if [[ ! -f "$PROJECT_DIR/.env" ]]; then
  echo "[error] .env not found. Copy .env.example to .env and configure."
  exit 1
fi
chmod 600 "$PROJECT_DIR/.env"

# Check tokens
if [[ ! -f "$PROJECT_DIR/tokens.json" ]]; then
  echo "[error] tokens.json not found. Run './dist/whoop-cli auth' first."
  exit 1
fi

# Check required keys
for key in WHOOP_CLIENT_ID WHOOP_CLIENT_SECRET VAULT_JOURNAL_DIR; do
  value="$(awk -F= -v k="$key" '$1==k {sub(/^[ \t]+/, "", $2); print $2}' "$PROJECT_DIR/.env" | head -n1)"
  if [[ -z "${value:-}" ]]; then
    echo "[error] .env missing $key"
    exit 1
  fi
done

# Build
echo "Building whoop-cli..."
(cd "$PROJECT_DIR" && go build -o ./dist/whoop-cli ./cmd/whoop-cli)
echo "Built: $PROJECT_DIR/dist/whoop-cli"

# Generate plist from template
sed "s|__PROJECT_DIR__|$PROJECT_DIR|g" "$TEMPLATE" > "$TARGET"
echo "Installed plist: $TARGET"

# Load
launchctl bootout "gui/$UID" "$TARGET" >/dev/null 2>&1 || true
launchctl bootstrap "gui/$UID" "$TARGET"

echo ""
echo "whoop-cli daemon installed."
echo "  Schedule: hourly + on load"
echo "  Logs: $LOG_DIR/"
echo ""

# Run once now to verify
echo "Running initial fetch..."
launchctl kickstart -k "gui/$UID/$LABEL"

echo ""
echo "Check logs:"
echo "  tail -f $LOG_DIR/daily.log"
echo "  tail -f $LOG_DIR/daily.err.log"
