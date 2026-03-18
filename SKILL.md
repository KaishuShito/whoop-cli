---
name: whoop-journal
description: >
  WHOOP API v2から生体データ（Recovery, Sleep, Strain, Workout）を取得し、
  Obsidian Journalに記録するGo CLIラッパー。1時間間隔でlaunchdが自動更新中。
  Use when: (1) 「WHOOPデータを見せて」「今日のリカバリーは？」「睡眠データ確認」、
  (2) 「WHOOPを更新して」「Journalに最新データを書き込んで」、
  (3) タスク設計時にユーザーの体調・睡眠状態を参照したい（recovery/HRV/sleep debt）、
  (4) 「WHOOPの状態は？」「デーモン動いてる？」（運用監視）、
  (5) 過去のWHOOPデータをバックフィルしたい（--days N）。
  Don't use when: WHOOP Developer Dashboardの設定変更、OAuth再認証（手動で auth コマンド実行）。
---

# whoop-journal

Go CLI: `~/Develop/whoop-journal/dist/whoop-journal`

## Quick Reference

```bash
WJ=~/Develop/whoop-journal/dist/whoop-journal

# Get today's data as JSON (for programmatic use)
$WJ fetch --json

# Preview today's data (human-readable)
$WJ fetch

# Write/update today's journal
$WJ fetch --write --update --prepend

# Specific date
$WJ fetch --date 2026-03-16 --write --update --prepend

# Backfill last 7 days
$WJ fetch --days 7 --write --update --prepend

# Check daemon & token status
$WJ status

# Check daemon is running
launchctl list | grep whoop
```

## Using WHOOP Data for Task Design

When planning the user's day or assessing capacity, fetch today's JSON and interpret:

```bash
~/Develop/whoop-journal/dist/whoop-journal fetch --json
```

### Key Metrics for AI Agent Decision-Making

| Metric | Field (JSON path) | Green | Yellow | Red |
|--------|-------------------|-------|--------|-----|
| Recovery | `.recovery[0].score.recovery_score` | >=67 | 34-66 | <34 |
| HRV | `.recovery[0].score.hrv_rmssd_milli` | >user baseline | near baseline | well below |
| RHR | `.recovery[0].score.resting_heart_rate` | low/stable | elevated | significantly elevated |
| Sleep Perf | `.sleep[0].score.sleep_performance_percentage` | >=85 | 70-84 | <70 |
| Sleep Debt | `.sleep[0].score.sleep_needed.need_from_sleep_debt_milli` | <30min | 30-90min | >90min |
| Day Strain | `.cycles[0].score.strain` | context-dependent | - | - |

### Recommendations Based on Recovery

- **Green (>=67%)**: Full capacity. Schedule demanding tasks, deep work blocks.
- **Yellow (34-66%)**: Moderate. Mix deep work with lighter tasks. Suggest breaks.
- **Red (<34%)**: Low capacity. Essential tasks only. Suggest early end to day.

Sleep debt `need_from_sleep_debt_milli` >1h: recommend earlier wind-down, lighter afternoon.

## Output Formats

`compact` (default): bullet points. `dashboard`: tables. `detailed`: full report with sleep need.

## Daemon

Hourly via launchd (`com.kai.whoop-journal.daily`). Writes today's data with `--write --update --prepend`.

```bash
# Logs
tail -f ~/Library/Logs/whoop-journal/daily.log

# Manual trigger
launchctl kickstart -k gui/$UID/com.kai.whoop-journal.daily

# Reinstall
cd ~/Develop/whoop-journal && bash scripts/install-launchd.sh
```

## Troubleshooting

| Symptom | Fix |
|---------|-----|
| `no tokens found` | Run `$WJ auth` (opens browser) |
| `refresh failed` | Refresh token expired. Run `$WJ auth` |
| `no_data` for today | WHOOP hasn't scored yet (check after waking) |
| Daemon not running | `bash ~/Develop/whoop-journal/scripts/install-launchd.sh` |
