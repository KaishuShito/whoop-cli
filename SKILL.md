---
name: whoop-cli
description: >
  WHOOP biometrics + weather + air quality -> daily health context for AI agents.
  Fetches recovery, sleep, strain, workouts from WHOOP; weather and UV from Open-Meteo;
  PM2.5, PM10, ozone, and NO2 from Open-Meteo Air Quality. Produces a terminal dashboard,
  markdown journal output, and JSON. Works great with Obsidian, Logseq, or any markdown-based journal.
  Use when: (1) "how's my recovery?", "sleep data", "WHOOP data", "am I healthy?",
  (2) capacity assessment for task planning,
  (3) "update my journal", "write WHOOP data",
  (4) "what's the weather impact?", "why do I feel bad?", "pressure check",
  (5) backfill past data.
  Don't use when: WHOOP Developer Dashboard account creation outside the CLI flow.
---

# whoop-cli

`whoop-cli` gives agents a compact daily readiness view: recovery, sleep, strain, weather, air quality, and a composite weather-sensitive health risk score.

## Fastest Commands

```bash
cd ~/Develop/whoop-journal
./dist/whoop-cli
./dist/whoop-cli today --json
./dist/whoop-cli fetch --json
./dist/whoop-cli fetch --write --update --prepend
bash skill/scripts/whoop-summary.sh
```

## First-Run Setup

Preferred flow:

```bash
cd ~/Develop/whoop-journal
go build -o ./dist/whoop-cli ./cmd/whoop-cli
./dist/whoop-cli setup
```

What setup does:

1. Opens the WHOOP Developer Dashboard
2. Collects client ID and client secret
3. Collects a journal directory, or allows stdout-only mode
4. Collects coordinates for weather and air quality
5. Writes `.env`
6. Runs OAuth and saves `tokens.json`

Non-interactive mode:

```bash
WHOOP_CLIENT_ID=... \
WHOOP_CLIENT_SECRET=... \
JOURNAL_DIR=~/journal \
WEATHER_LAT=35.6503 \
WEATHER_LON=139.7225 \
./dist/whoop-cli setup --non-interactive
```

## Capacity Assessment

Use `recovery_zone` from `skill/scripts/whoop-summary.sh` to set task intensity:

- `green` (`recovery_score >= 67`): full-capacity day
- `yellow` (`34-66`): mixed day
- `red` (`<34`): protect the day

Also watch:

- `sleep_debt_hours > 1.5`
- `sleep_performance < 70`
- `strain >= 14`

## Weather Risk Interpretation

Use `weather_risk_level` and `weather_risk_score` as the "why do I feel off?" layer:

- `Low` (`0-25`): normal planning is fine
- `Moderate` (`26-50`): simplify the day
- `High` (`51-100`): assume environment is materially affecting the day

If the user feels bad despite decent recovery, prioritize checking:

- `pressure_change_hpa`
- `humidity_pct`
- `weather_label`
- `pm25_ug_m3`
- `ox_ppm`

For the exact thresholds, read:

- `skill/references/risk-scoring.md`
- `docs/risk-scoring.md`

## Gotchas

- Commands should run from the project root if you rely on the local `.env`.
- `JOURNAL_DIR` is the primary key. `VAULT_JOURNAL_DIR` still works as a deprecated alias.
- `--write` without `JOURNAL_DIR` will fail. Plain stdout modes still work.
- WHOOP access tokens refresh automatically on `401`.
- If refresh token rotation fails or tokens are expired in a bad state, rerun `./dist/whoop-cli auth` or `./dist/whoop-cli setup`.
- Today's WHOOP data can be incomplete before sleep is fully scored. Use yesterday for debugging missing recovery or sleep.
- Weather and air quality can degrade independently without blocking WHOOP fetches.

## Health Check

```bash
cd ~/Develop/whoop-journal
bash skill/scripts/health-check.sh
```

## Reference Files

- `docs/spec.md` — command and flag specification
- `docs/whoop-api.md` — WHOOP Developer App setup
- `docs/risk-scoring.md` — OSS-facing risk scoring explanation
- `skill/references/risk-scoring.md` — exact scoring logic used by the skill
- `skill/references/weather-codes.md` — WMO weather labels and emoji
- `skill/references/api-response-schema.md` — JSON field paths
