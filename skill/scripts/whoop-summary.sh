#!/bin/bash
# Quick WHOOP summary for AI agents — outputs key metrics as key=value pairs.
# Usage: bash scripts/whoop-summary.sh [--date YYYY-MM-DD]
set -euo pipefail

WJ_DIR=~/Develop/whoop-journal
WJ=$WJ_DIR/dist/whoop-journal
DATE_FLAG=""
[ "${1:-}" = "--date" ] && DATE_FLAG="--date $2"

JSON=$(cd "$WJ_DIR" && $WJ fetch --json $DATE_FLAG 2>/dev/null)

if [ -z "$JSON" ] || [ "$JSON" = "null" ]; then
  echo "status=no_data"
  exit 0
fi

# Extract key metrics with jq-less parsing (pure bash + python one-liner)
python3 -c "
import json, sys
d = json.loads(sys.stdin.read())

recovery = d.get('recovery', [])
sleep = d.get('sleep', [])
cycles = d.get('cycles', [])
workouts = d.get('workouts', [])

if not any([recovery, sleep, cycles, workouts]):
    print('status=no_data')
    sys.exit(0)

print('status=ok')
print(f'date={d[\"date\"]}')

if recovery and recovery[0].get('score'):
    s = recovery[0]['score']
    print(f'recovery_score={s[\"recovery_score\"]:.0f}')
    print(f'hrv_ms={s[\"hrv_rmssd_milli\"]:.1f}')
    print(f'rhr_bpm={s[\"resting_heart_rate\"]:.0f}')
    print(f'spo2_pct={s[\"spo2_percentage\"]:.1f}')
    rs = s['recovery_score']
    print(f'recovery_zone={\"green\" if rs >= 67 else \"yellow\" if rs >= 34 else \"red\"}')

if sleep and sleep[0].get('score'):
    s = sleep[0]['score']
    st = s['stage_summary']
    total_h = st['total_in_bed_time_milli'] / 3600000
    print(f'sleep_hours={total_h:.1f}')
    print(f'sleep_performance={s[\"sleep_performance_percentage\"]:.0f}')
    print(f'sleep_efficiency={s[\"sleep_efficiency_percentage\"]:.0f}')
    debt_h = s['sleep_needed'].get('need_from_sleep_debt_milli', 0) / 3600000
    print(f'sleep_debt_hours={debt_h:.1f}')

if cycles and cycles[0].get('score'):
    s = cycles[0]['score']
    print(f'strain={s[\"strain\"]:.1f}')
    print(f'calories_kj={s[\"kilojoule\"]:.0f}')

print(f'workout_count={len(workouts)}')
" <<< "$JSON"
