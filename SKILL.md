---
name: whoop-journal
description: >
  WHOOP生体データ（Recovery, Sleep, Strain, Workout）をGo CLIで取得し、
  Obsidian Journalに自動記録。1時間間隔launchd稼働中。
  Use when: (1) 「リカバリーは？」「睡眠データ」「WHOOPデータ」「体調どう？」、
  (2) タスク設計・タイムボクシング時にユーザーのキャパシティを判断したい、
  (3) 「Journalを更新して」「WHOOPを書き込んで」、
  (4) 「WHOOPの状態は？」「デーモン動いてる？」（運用監視）、
  (5) 過去のデータをバックフィル（--days N）。
  Don't use when: WHOOP Developer Dashboard設定、OAuth再認証。
---

# whoop-journal

## Quick Data Access

Run `scripts/whoop-summary.sh` for key=value summary (fast, no context cost):

```bash
bash ~/.claude/skills/whoop-journal/whoop-journal/scripts/whoop-summary.sh
# or with date
bash ~/.claude/skills/whoop-journal/whoop-journal/scripts/whoop-summary.sh --date 2026-03-18
```

For full JSON: `cd ~/Develop/whoop-journal && ./dist/whoop-journal fetch --json`

For journal write: `cd ~/Develop/whoop-journal && ./dist/whoop-journal fetch --write --update --prepend`

## Capacity Assessment (for task design)

Use recovery_zone from whoop-summary.sh output to calibrate the day:

- **green** (>=67%): Full deep work. Schedule demanding tasks.
- **yellow** (34-66%): Mixed schedule. Alternate deep/light work.
- **red** (<34%): Essentials only. Suggest early wind-down.

`sleep_debt_hours` >1.5: flag it. Recommend lighter afternoon, earlier bedtime.

`sleep_performance` <70%: sleep quality was poor regardless of hours — expect lower focus.

## Gotchas

- **CLI must run from project dir**: Always `cd ~/Develop/whoop-journal` before `./dist/whoop-journal`. The .env is loaded relative to binary location.
- **Today's data is partial until sleep is scored**: Fetching today before ~8AM JST may return only strain/cycle (no recovery/sleep). Yesterday's data is always complete.
- **Token auto-refresh is silent**: Access tokens expire every hour. The CLI refreshes automatically on 401. If refresh token itself expires (rare, ~months), user must run `./dist/whoop-journal auth` manually.
- **API is v2**: v1 endpoints return 404. The CLI uses `api.prod.whoop.com/developer/v2/`.
- **Duplicate protection**: `--write` without `--update` errors if WHOOP section exists. The daemon always uses `--update`.
- **JST dates**: All date parameters and journal filenames use JST. The CLI converts to UTC internally for API calls.

## Daemon Status

```bash
launchctl list | grep whoop    # exit code 0 = last run OK, 1 = last run errored
tail -5 ~/Library/Logs/whoop-journal/daily.log
tail -5 ~/Library/Logs/whoop-journal/daily.err.log
```

Reinstall: `cd ~/Develop/whoop-journal && bash scripts/install-launchd.sh`

## Reference Files

- `references/api-response-schema.md` — Full JSON field paths for all WHOOP data types. Read when parsing `--json` output or writing custom analysis.
