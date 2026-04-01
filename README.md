# whoop-cli

**Your body's data in your terminal, your journal, and your agent context.**

`whoop-cli` pulls WHOOP recovery, sleep, strain, workouts, weather, air quality, and a composite risk score into:

- a beautiful terminal dashboard
- markdown output for your daily journal
- JSON for automation and agents

Works with any markdown journal. It fits especially well with Obsidian, Logseq, or plain markdown files.

## Claude Code / OpenClaw Skill

whoop-cli ships as a **Claude Code Skill** out of the box. Install the skill and your AI agent can check your health context before planning your day:

```
> /whoop How's my recovery?

🟢 Recovery 79%. HRV 27ms. Sleep 8h09m (84% perf).
Weather risk 🔴 High (60/100) — pressure dropped 10 hPa.
→ Body is fine but environment is rough. Go easy on deep work.
```

The skill file is at [`SKILL.md`](./SKILL.md). Helper scripts in [`skill/scripts/`](./skill/scripts/) give agents quick key=value access without reading full JSON.

See [docs/spec.md](./docs/spec.md) for the full CLI reference.

## Install

### Homebrew

Placeholder for future tap support:

```bash
brew install KaishuShito/tap/whoop-cli
```

### Go install

```bash
go install github.com/KaishuShito/whoop-cli/cmd/whoop-cli@latest
```

### Binary

Download a prebuilt binary from GitHub Releases once releases are published.

## Prerequisites

- Go 1.22+
- a WHOOP membership

## Quick Start

```bash
git clone https://github.com/KaishuShito/whoop-cli.git
cd whoop-cli
go build -o ./dist/whoop-cli ./cmd/whoop-cli
./dist/whoop-cli setup
```

After setup:

```bash
./dist/whoop-cli
./dist/whoop-cli fetch
./dist/whoop-cli fetch --write --update --prepend
./dist/whoop-cli status
```

## What `setup` Does

`whoop-cli setup` is the recommended first-run flow.

It will:

1. Open the WHOOP Developer Dashboard
2. Ask for your WHOOP client ID and client secret
3. Ask where markdown entries should be written, or allow stdout-only mode
4. Open a coordinate finder for weather and air quality
5. Create `.env`, backing up any existing file
6. Immediately run OAuth authorization and save `tokens.json`

## WHOOP Developer App Setup

You need a WHOOP developer app before authorization works.

Use these settings:

- Redirect URI: `http://localhost:8080/callback`
- Scopes:
  `read:recovery read:sleep read:cycles read:workout read:profile read:body_measurement offline`

For the full step-by-step guide, see [docs/whoop-api.md](./docs/whoop-api.md).

## Terminal Dashboard

Running `whoop-cli` with no arguments is the same as `whoop-cli today`.

Features:

- ANSI colors when stdout is a TTY
- automatic no-color mode when piped
- `--no-color` for plain output
- `--json` for script-friendly output

Examples:

```bash
./dist/whoop-cli
./dist/whoop-cli today
./dist/whoop-cli today --json
./dist/whoop-cli today --no-color
```

## Commands

```bash
./dist/whoop-cli                  # terminal dashboard
./dist/whoop-cli today           # terminal dashboard
./dist/whoop-cli today --json    # enriched JSON
./dist/whoop-cli fetch           # markdown preview
./dist/whoop-cli fetch --write   # write markdown to JOURNAL_DIR
./dist/whoop-cli weather         # weather only
./dist/whoop-cli airquality      # air quality only
./dist/whoop-cli status          # token + config status
./dist/whoop-cli auth            # OAuth only
./dist/whoop-cli version         # build version
```

Detailed CLI behavior lives in [docs/spec.md](./docs/spec.md).

## Configuration

Everything is configured through `.env`.

Required:

- `WHOOP_CLIENT_ID`
- `WHOOP_CLIENT_SECRET`

Optional:

- `WHOOP_REDIRECT_URI` default: `http://localhost:8080/callback`
- `JOURNAL_DIR` for `fetch --write`
- deprecated alias: `VAULT_JOURNAL_DIR`
- `WEATHER_ENABLED` default: `true`
- `WEATHER_LAT` default: `35.6503`
- `WEATHER_LON` default: `139.7225`
- `AIRQUALITY_ENABLED` default: `true`

Notes:

- air quality reuses the same `WEATHER_LAT` and `WEATHER_LON`
- if `JOURNAL_DIR` is unset, stdout-only workflows still work

See [.env.example](./.env.example).

## Environmental Data Sources

### Weather

Weather comes from Open-Meteo forecast data.

Used fields:

- weather code
- temperature
- humidity
- pressure
- wind
- UV index

### Air Quality

Air quality comes from Open-Meteo Air Quality:

- global coverage
- no API key required
- same ecosystem as the weather integration

Used hourly variables:

- `pm2_5`
- `pm10`
- `ozone`
- `nitrogen_dioxide`

The CLI calculates a daytime average across `06:00-22:00`.

To preserve the existing risk-scoring model, ozone is converted from Open-Meteo `μg/m³` into `ppm` before applying the Ox thresholds.

## Risk Scoring

The composite health risk score combines:

- weather stressors
- air quality
- WHOOP recovery and sleep state

For the exact thresholds and worked examples, see:

- [docs/risk-scoring.md](./docs/risk-scoring.md)
- [skill/references/risk-scoring.md](./skill/references/risk-scoring.md)

## Skill Reference

Full skill documentation: [`SKILL.md`](./SKILL.md)

Helper scripts in [`skill/scripts/`](./skill/scripts/):

- `whoop-summary.sh` — key=value health summary (fast, low-context)
- `setup.sh` — interactive first-run wizard
- `health-check.sh` — verify token + API connectivity

Reference docs in [`skill/references/`](./skill/references/):

- `risk-scoring.md` — composite score algorithm
- `weather-codes.md` — WMO weather code table
- `air-quality-standards.md` — PM2.5/Ozone thresholds

## Packaging and Release

Project-level release tooling includes:

- `.goreleaser.yml` for multi-platform release builds
- `Makefile` for local build/test/install/release workflows

Examples:

```bash
make build
make test
make install
make setup
```

## Automation (macOS)

Run whoop-cli automatically every morning with `launchd`:

```bash
bash scripts/install-launchd.sh   # builds binary + installs daily plist
bash scripts/uninstall-launchd.sh # removes the plist
```

The template plist is in [`launchd/`](./launchd/). Edit the paths inside to match your setup before installing.

## Development

```bash
go test ./...
make build
```

## License

MIT. See [LICENSE](./LICENSE).
