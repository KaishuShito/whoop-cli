# whoop-cli Specification

`whoop-cli` pulls WHOOP biometrics plus local weather and air quality into either:

- a terminal dashboard
- markdown output for journal writing
- JSON for scripts and agents

## Command Map

### `whoop-cli`

Alias for `whoop-cli today`.

Shows a terminal dashboard with ANSI color when stdout is a TTY. Automatically falls back to plain output when piped.

### `whoop-cli today`

Fetch today's WHOOP, weather, and air-quality data and render a terminal dashboard.

Flags:

- `--date YYYY-MM-DD`: fetch a specific day instead of today
- `--json`: output raw JSON instead of the terminal dashboard
- `--no-color`: disable ANSI color

Output modes:

- Default: Unicode box drawing dashboard
- `--json`: the same enriched day payload used by `fetch --json`

### `whoop-cli fetch`

Fetch WHOOP data and render markdown.

Flags:

- `--date YYYY-MM-DD`: fetch a specific end date
- `--days N`: fetch multiple days ending on `--date` or today
- `--format compact|dashboard|detailed`: markdown format
- `--write`: write markdown to `JOURNAL_DIR`
- `--update`: replace an existing WHOOP block in the markdown file
- `--prepend`: insert near the top of the file instead of appending
- `--json`: output raw JSON

Behavior:

- Without `--write`, output goes to stdout.
- With `--write`, `JOURNAL_DIR` must be configured.
- Weather and air quality degrade independently; WHOOP data can still succeed when one source fails.

### `whoop-cli setup`

Interactive first-run setup.

Flow:

1. Opens the WHOOP Developer Dashboard.
2. Prompts for client ID and client secret.
3. Prompts for a journal directory, or allows stdout-only mode.
4. Opens a coordinate finder for weather and air quality.
5. Writes `.env`, backing up any existing file first.
6. Runs OAuth authorization immediately and saves `tokens.json`.

Flags:

- `--non-interactive`: read configuration from environment variables, write `.env`, and run OAuth without prompts

Environment variables consumed in `--non-interactive` mode:

- `WHOOP_CLIENT_ID`
- `WHOOP_CLIENT_SECRET`
- `JOURNAL_DIR` or deprecated `VAULT_JOURNAL_DIR`
- `WEATHER_LAT`
- `WEATHER_LON`

### `whoop-cli weather`

Fetch weather only.

Flags:

- `--date YYYY-MM-DD`
- `--json`

### `whoop-cli airquality`

Fetch air quality only from Open-Meteo Air Quality.

Flags:

- `--date YYYY-MM-DD`
- `--json`

Data source:

- `https://air-quality-api.open-meteo.com/v1/air-quality`
- hourly variables: `pm2_5,pm10,ozone,nitrogen_dioxide`
- daily summary: daytime average across `06:00-22:00`

### `whoop-cli status`

Show token status, current journal directory, client ID prefix/suffix, and build version.

### `whoop-cli auth`

Run the WHOOP OAuth flow using existing `.env` configuration and save `tokens.json`.

### `whoop-cli version`

Print build version and commit:

```text
whoop-cli v0.2.0 (commit abc1234)
```

## Configuration

Configuration lives in `.env`.

Required:

- `WHOOP_CLIENT_ID`
- `WHOOP_CLIENT_SECRET`

Optional:

- `WHOOP_REDIRECT_URI` default: `http://localhost:8080/callback`
- `JOURNAL_DIR`
- deprecated alias: `VAULT_JOURNAL_DIR`
- `WEATHER_ENABLED` default: `true`
- `WEATHER_LAT` default: `35.6503`
- `WEATHER_LON` default: `139.7225`
- `AIRQUALITY_ENABLED` default: `true`

Notes:

- Air quality uses the same `WEATHER_LAT` and `WEATHER_LON` coordinates.
- `JOURNAL_DIR` is only required for `fetch --write`.

## Data Model

The enriched day payload contains:

- WHOOP cycles, recovery, sleep, workouts
- weather summary from Open-Meteo forecast
- air-quality summary from Open-Meteo Air Quality

Air-quality fields include:

- `PM25UgM3`
- `PM10UgM3`
- `OzoneUgM3`
- `OxPpm`
- `NO2UgM3`

The risk-scoring model still evaluates ozone against the same ppm thresholds by converting Open-Meteo ozone `μg/m³` into `ppm`.
