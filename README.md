# Whoop CLI

**Your body's data in your AI agent's context.**

Whoop CLI pulls recovery, sleep, strain, workouts, weather, air quality, and a daily risk score into a single agent-friendly journal entry or JSON payload.

## What It Does

Turn WHOOP API data plus environmental context into a daily summary your CLI, notes, and AI skills can actually use.

## Demo

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

Japanese labels are used by default for weather and air-quality output, and the formatter layer is easy to customize for your own labels or locale.

## Quick Start

1. Create a WHOOP developer app at [developer-dashboard.whoop.com](https://developer-dashboard.whoop.com) with redirect URI `http://localhost:8080/callback` and scopes `read:recovery read:sleep read:cycles read:workout read:profile read:body_measurement offline`.
2. Clone this repo and copy the example config:

```bash
git clone https://github.com/KaishuShito/whoop-cli.git
cd whoop-cli
cp .env.example .env
```

3. Fill in `.env` with your WHOOP credentials and journal path.
4. Build and authenticate:

```bash
go build -o ./dist/whoop-cli ./cmd/whoop-cli
./dist/whoop-cli auth
./dist/whoop-cli status
```

5. Fetch your first report:

```bash
./dist/whoop-cli fetch
./dist/whoop-cli fetch --write --update --prepend
```

## Features

- WHOOP API v2 support for recovery, sleep, cycles, and workouts
- Agent-friendly JSON output via `fetch --json`
- Journal-ready markdown output in `compact`, `dashboard`, and `detailed` formats
- Weather enrichment via Open-Meteo
- Air quality enrichment via Japan's そらまめ君 API
- Daily weather sensitivity risk score that blends body state and environment
- Automatic OAuth token refresh on 401
- Duplicate-safe journal writing with prepend and update modes
- Included macOS `launchd` template for background automation
- Zero third-party Go dependencies

## Skills

Whoop CLI is designed to work nicely as a Claude Code Skill.

1. Keep this repo somewhere stable, for example `~/Develop/whoop-cli`
2. Build the binary: `go build -o ./dist/whoop-cli ./cmd/whoop-cli`
3. Expose the included skill metadata from [`SKILL.md`](./SKILL.md)
4. Trigger it in Claude Code with `/whoop`

Useful commands behind the skill:

```bash
./dist/whoop-cli fetch --json
./dist/whoop-cli fetch --write --update --prepend
bash skill/scripts/whoop-summary.sh --date 2026-03-18
```

## Architecture

```text
             +----------------------+
             |      WHOOP API       |
             | recovery/sleep/etc.  |
             +----------+-----------+
                        |
                        v
 +---------------+   +----------------------+   +------------------+
 | Open-Meteo    |-->|     whoop-cli        |-->| Markdown journal |
 | weather       |   | fetch / status / auth|   | or stdout JSON   |
 +---------------+   | risk scoring         |   +------------------+
                     +----------+-----------+
                                ^
                                |
                     +----------+-----------+
                     |  そらまめ君 API       |
                     |  air quality         |
                     +----------------------+
```

## Configuration

All configuration lives in `.env`.

| Variable | Required | Default | Description |
|---|---|---|---|
| `WHOOP_CLIENT_ID` | Yes | none | WHOOP developer app client ID |
| `WHOOP_CLIENT_SECRET` | Yes | none | WHOOP developer app client secret |
| `WHOOP_REDIRECT_URI` | No | `http://localhost:8080/callback` | OAuth callback used by `auth` |
| `VAULT_JOURNAL_DIR` | Yes | none | Directory where `YYYY-MM-DD.md` files are written |
| `WEATHER_ENABLED` | No | `true` | Enable Open-Meteo enrichment |
| `WEATHER_LAT` | No | `35.6503` | Latitude for weather lookup |
| `WEATHER_LON` | No | `139.7225` | Longitude for weather lookup |
| `AIRQUALITY_ENABLED` | No | `true` | Enable air quality enrichment |
| `AIRQUALITY_STATION_CODE` | No | `13103010` | そらまめ monitoring station code |

Other generated files:

- `.env`: local config, never commit
- `tokens.json`: OAuth tokens, auto-refreshed, never commit
- `dist/`: compiled binaries

## API Sources

- WHOOP API v2
  Used for recovery, sleep, cycles, workouts, and body metrics via OAuth 2.0.
- Open-Meteo
  Used for weather code, temperature, humidity, pressure, wind, and UV index.
- そらまめ君 API
  Used for PM2.5 and oxidant data from Japan's public air quality monitoring network.

## Commands

```bash
./dist/whoop-cli auth
./dist/whoop-cli status
./dist/whoop-cli fetch
./dist/whoop-cli fetch --format dashboard
./dist/whoop-cli fetch --date 2026-03-16 --write --update --prepend
./dist/whoop-cli fetch --days 7 --json
./dist/whoop-cli weather --json
./dist/whoop-cli airquality --json
```

## Automation

The included macOS automation template lives at [`launchd/com.kai.whoop-cli.daily.plist`](./launchd/com.kai.whoop-cli.daily.plist).

- Label: `com.kai.whoop-cli.daily`
- Schedule: hourly plus run-at-load
- Command: `./dist/whoop-cli fetch --write --update --prepend --format compact`
- Logs: `~/Library/Logs/whoop-cli/`

Install or remove it with:

```bash
bash scripts/install-launchd.sh
bash scripts/uninstall-launchd.sh
```

## Contributing

Contributions are welcome.

1. Fork the repo
2. Create a branch
3. Run `go test ./...`
4. Open a PR with a clear description of the change

If you change API integrations or formatters, include an example output snippet in the PR.

## License

MIT. See [`LICENSE`](./LICENSE).
