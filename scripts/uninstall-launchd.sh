#!/usr/bin/env bash
set -euo pipefail

LABEL="com.kai.whoop-cli.daily"
TARGET="$HOME/Library/LaunchAgents/${LABEL}.plist"

launchctl bootout "gui/$UID" "$TARGET" >/dev/null 2>&1 || true
rm -f "$TARGET"

echo "Uninstalled $LABEL"
echo "Logs remain at ~/Library/Logs/whoop-cli/"
