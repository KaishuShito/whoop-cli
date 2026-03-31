# whoop-journal

WHOOP API v2 から日次データを取得し、Obsidian Journal に自動記録する Go CLI ツール。

## Architecture

```
whoop-journal/
├── cmd/whoop-journal/main.go      # CLI entry point (auth/fetch/status)
├── internal/
│   ├── auth/                      # OAuth 2.0 + token auto-refresh
│   ├── config/                    # .env loader + validation
│   ├── journal/                   # 3 output formats + journal file I/O
│   ├── weather/                   # Open-Meteo client + risk scoring
│   └── whoop/                     # WHOOP API v2 client
├── launchd/                       # macOS daemon (daily at 09:00 JST)
├── scripts/
│   ├── install-launchd.sh         # Build + install daemon
│   └── uninstall-launchd.sh       # Remove daemon
├── .env                           # Credentials (gitignored)
└── tokens.json                    # OAuth tokens (gitignored, auto-refreshed)
```

## Setup

### 1. WHOOP Developer App

1. [developer-dashboard.whoop.com](https://developer-dashboard.whoop.com) でアプリを作成
2. Redirect URI に `http://localhost:8080/callback` を追加
3. スコープ: `read:recovery read:sleep read:cycles read:workout read:profile read:body_measurement offline`

### 2. Configure

```bash
cp .env.example .env
# Edit .env with your credentials:
#   WHOOP_CLIENT_ID=...
#   WHOOP_CLIENT_SECRET=...
#   VAULT_JOURNAL_DIR=/path/to/Obsidian/01_Projects/Journal
#   WEATHER_LAT=35.6503
#   WEATHER_LON=139.7225
#   WEATHER_ENABLED=true
#   AIRQUALITY_ENABLED=true
#   AIRQUALITY_STATION_CODE=13103010
```

### 3. Build & Auth

```bash
go build -o ./dist/whoop-journal ./cmd/whoop-journal
./dist/whoop-journal auth    # Opens browser for OAuth
./dist/whoop-journal status  # Verify tokens
```

### 4. Install Daemon (auto-run daily at 09:00)

```bash
bash scripts/install-launchd.sh
```

## Usage

```bash
# Preview yesterday's data (default: compact format)
whoop-journal fetch

# Choose format
whoop-journal fetch --format compact     # Bullet points (default)
whoop-journal fetch --format dashboard   # Tables
whoop-journal fetch --format detailed    # Full report with sleep need

# Write to journal (prepend = insert after header)
whoop-journal fetch --write --prepend

# Specific date
whoop-journal fetch --date 2026-03-16 --write --prepend

# Backfill last 7 days
whoop-journal fetch --days 7 --write --prepend

# Raw JSON (for AI agents / piping)
whoop-journal fetch --json

# Weather only (for debugging)
whoop-journal weather
whoop-journal weather --date 2026-03-16 --json

# Air quality only (for debugging)
whoop-journal airquality
whoop-journal airquality --date 2026-03-16 --json

# Token & config status
whoop-journal status
```

## Output Formats

### compact (default)

```markdown
## WHOOP Daily - 2026-03-17

**Recovery**: 🟢 82% (Green)
- HRV: 31 ms | RHR: 54 bpm | SpO2: 99.3%
- Skin Temp: 34.0°C

**Sleep**: 8h 51m in bed
- Performance: 71% | Efficiency: 76%
- REM: 46m | Deep: 1h 47m | Light: 4h 12m
- Awake: 2h 05m | Disturbances: 19

**Strain**: 0.3 | 2990 kJ
- Avg HR: 56 | Max HR: 100

**Environment**
- Weather: 🌧️ 雨 | 18°C (体感15°C) | Humidity 78% | Wind 5.2m/s
- Pressure: 1006 hPa (▼7 hPa) ⚠️ 気圧急低下
- Air Quality: PM2.5 18μg/m³ (🟡) | Ox 0.034ppm (🟢)
- UV Index: 5 (Moderate)
- 気象病リスク: 🟡 Moderate (42/100)
```

### dashboard

Table-based with sleep breakdown percentages.

### detailed

Full report including Sleep Need (baseline, debt, strain), workout HR zones, respiratory rate.

## Data Flow

```
WHOOP API v2 (api.prod.whoop.com)
    │
    ├── /cycle         → Day strain, calories, HR
    ├── /recovery      → Recovery score, HRV, RHR, SpO2, skin temp
    ├── /activity/sleep → Sleep stages, performance, efficiency
    └── /activity/workout → Sport, strain, HR zones
Open-Meteo (api.open-meteo.com)
    │
    ├── daily             → weather_code, temp max/min, UV max
    └── hourly            → surface pressure, relative humidity
    │
    ▼
Go CLI (whoop-journal fetch)
    │  - JST date → UTC range conversion
    │  - Auto token refresh on 401
    │  - Weather fetch degrades gracefully on API failure
    │  - Duplicate write protection
    ▼
Obsidian Journal (01_Projects/Journal/YYYY-MM-DD.md)
    │  --prepend: insert after # header
    │  --append:  add at end of file
    ▼
Daily automated via launchd (09:00 JST)
```

## Daemon

| Item | Value |
|------|-------|
| Label | `com.kai.whoop-journal.daily` |
| Schedule | Every day at 09:00 + on load |
| Command | `fetch --write --prepend --format compact` |
| Logs | `~/Library/Logs/whoop-journal/daily.log` |
| Error logs | `~/Library/Logs/whoop-journal/daily.err.log` |

```bash
# Check status
launchctl list | grep whoop

# View logs
tail -f ~/Library/Logs/whoop-journal/daily.log

# Manual trigger
launchctl kickstart -k gui/$UID/com.kai.whoop-journal.daily

# Uninstall
bash scripts/uninstall-launchd.sh
```

## Token Management

- OAuth tokens are stored in `tokens.json` (0600 permissions)
- Access tokens expire after 1 hour
- Automatic refresh via refresh token on 401 response
- Both access and refresh tokens are replaced on each refresh
- Re-run `whoop-journal auth` if refresh token expires

## Tests

```bash
go test ./... -v
```

22 unit tests covering:
- Config loading, env var precedence, validation
- Token save/load, JSON roundtrip, file permissions
- All 3 format outputs with full/empty/nil data
- Journal prepend/append/duplicate protection
- `findHeaderEnd` frontmatter parsing

## Dependencies

- Go 1.25+
- Zero external dependencies (stdlib only)
