---
name: whoop-cli
description: >
  WHOOP biometrics + weather + air quality -> daily health context for AI agents.
  Fetches recovery, sleep, strain from WHOOP API; pressure, temperature, humidity,
  UV from Open-Meteo; PM2.5 and Ox from Japan's soramame network.
  Calculates composite health risk score (気象病リスク).
  Use when: (1) "how's my recovery?", "sleep data", "WHOOP data", "am I healthy?",
  (2) capacity assessment for task planning (green/yellow/red zones),
  (3) "update my journal", "write WHOOP data",
  (4) "what's the weather impact?", "why do I feel bad?", "pressure check",
  (5) backfill past data (--days N).
  Don't use when: WHOOP Developer Dashboard setup, OAuth re-auth.
---

# whoop-cli

WHOOP CLI gives agents a compact daily readiness view: recovery, sleep, strain, weather, air quality, and a composite weather-sensitive health risk score.

Keep this file lean. Use it for the fastest commands and decision rules, then open the reference files only when you need the exact thresholds or schemas.

## Quick Data Access

Run the summary script first. It returns a low-token `key=value` snapshot that already includes WHOOP + weather + air quality.

```bash
cd ~/Develop/whoop-journal
bash skill/scripts/whoop-summary.sh
bash skill/scripts/whoop-summary.sh --date 2026-03-18
```

For raw JSON or narrower debugging:

```bash
cd ~/Develop/whoop-journal
./dist/whoop-journal fetch --json
./dist/whoop-journal weather --json
./dist/whoop-journal airquality --json
./dist/whoop-journal fetch --date 2026-03-18 --json
./dist/whoop-journal fetch --days 7 --write --update --prepend
```

For journal write:

```bash
cd ~/Develop/whoop-journal
./dist/whoop-journal fetch --write --update --prepend
```

## Capacity Assessment

Use `recovery_zone` from `skill/scripts/whoop-summary.sh` to set task intensity:

- `green` (`recovery_score >= 67`): full-capacity day. Good for deep work, deadlines, intense calls, training, or stacked commitments.
- `yellow` (`34-66`): mixed day. Keep one hard block, then switch to lighter execution, admin, or async work.
- `red` (`<34`): protect the day. Prioritize essentials, shorten meetings, reduce cognitive switching, and bias toward recovery.

Also watch:

- `sleep_debt_hours > 1.5`: treat the afternoon as degraded even if recovery looks acceptable.
- `sleep_performance < 70`: expect lower focus and slower task switching.
- `strain >= 14`: avoid pairing a high-strain body day with cognitively intense scheduling unless recovery is strong.

## Weather Risk Interpretation

Use `weather_risk_level` and `weather_risk_score` as the "why do I feel off?" layer:

- `Low` (`0-25`): normal planning is fine. No extra mitigation needed.
- `Moderate` (`26-50`): simplify the day. Hydrate, reduce context switching, and avoid stacking physically and cognitively demanding work.
- `High` (`51-100`): assume weather sensitivity is materially affecting the day. Prefer recovery-supportive routines, short work blocks, fewer meetings, and lower pressure commitments.

If the user feels bad despite decent recovery, prioritize checking:

- `pressure_change_hpa`
- `humidity_pct`
- `weather_label`
- `pm25_ug_m3`
- `ox_ppm`

For the exact scoring model, read `skill/references/risk-scoring.md`.

## Setup

If `skill/config.json` does not exist, treat this as first-run setup.

Preferred skill flow:

1. Use `AskUserQuestion` to collect:
   `WHOOP Client ID`, `WHOOP Client Secret`, `Journal directory path`, and location.
2. Use Tokyo Hiroo defaults if the user does not specify location details:
   `lat=35.6503`, `lon=139.7225`, `station_code=13103010`.
3. Save those answers to `skill/config.json`.
4. Materialize `.env` from that config so the CLI can run.

Local interactive fallback:

```bash
cd ~/Develop/whoop-journal
bash skill/scripts/setup.sh
go build -o ./dist/whoop-journal ./cmd/whoop-journal
./dist/whoop-journal auth
bash skill/scripts/health-check.sh
```

`skill/scripts/setup.sh` writes:

- `skill/config.json`
- `.env`

After setup, `./dist/whoop-journal auth` is still required once to create `tokens.json`.

## Gotchas

- CLI commands should run from the project root. The config loader reads `.env` from the repo directory.
- Today's WHOOP data may be incomplete before sleep is scored. For reliable interpretation, use yesterday when debugging missing recovery or sleep.
- Access tokens refresh automatically on 401. If refresh token rotation fails, rerun `./dist/whoop-journal auth`.
- WHOOP uses Developer API v2. Old v1 endpoints are not relevant here.
- `--write` without `--update` will fail if a WHOOP block already exists in the journal file.
- Dates are interpreted in JST for both CLI flags and journal file naming.
- `skill/config.json` is for skill-side setup convenience. The CLI itself still runs from `.env` + `tokens.json`.
- Weather and air quality are allowed to degrade independently. A WHOOP fetch can still succeed even if one environmental source is temporarily unavailable.

## Daemon Status

```bash
launchctl list | grep whoop
tail -5 ~/Library/Logs/whoop-journal/daily.log
tail -5 ~/Library/Logs/whoop-journal/daily.err.log
```

Reinstall:

```bash
cd ~/Develop/whoop-journal
bash scripts/install-launchd.sh
```

Health check:

```bash
cd ~/Develop/whoop-journal
bash skill/scripts/health-check.sh
```

## Reference Files

- `skill/references/risk-scoring.md` — Exact risk-score logic, thresholds, and scoring examples.
- `skill/references/weather-codes.md` — WMO weather code to Japanese label / emoji mapping used by this project.
- `skill/references/air-quality-standards.md` — PM2.5 / Ox interpretation and health context.
- `skill/references/api-response-schema.md` — Full JSON field paths for `fetch --json` parsing.
