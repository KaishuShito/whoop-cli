#!/usr/bin/env bash
# Quick WHOOP + weather + air quality summary for AI agents.
# Usage: bash skill/scripts/whoop-summary.sh [--date YYYY-MM-DD]
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
WJ="$PROJECT_DIR/dist/whoop-journal"

DATE_FLAG=()
if [[ "${1:-}" == "--date" ]]; then
  if [[ -z "${2:-}" ]]; then
    echo "usage: bash skill/scripts/whoop-summary.sh [--date YYYY-MM-DD]" >&2
    exit 1
  fi
  DATE_FLAG=(--date "$2")
fi

if [[ ! -x "$WJ" ]]; then
  echo "status=error"
  echo "error=binary_missing"
  echo "hint=run_go_build"
  exit 1
fi

JSON="$(
  cd "$PROJECT_DIR" &&
    "$WJ" fetch --json "${DATE_FLAG[@]}" 2>/dev/null
)"

if [[ -z "$JSON" || "$JSON" == "null" ]]; then
  echo "status=no_data"
  exit 0
fi

python3 -c "
import json
import sys

d = json.loads(sys.stdin.read())

recovery = d.get('recovery', [])
sleep = d.get('sleep', [])
cycles = d.get('cycles', [])
workouts = d.get('workouts', [])
weather = d.get('weather')
air = d.get('air_quality')

if not any([recovery, sleep, cycles, workouts, weather, air]):
    print('status=no_data')
    sys.exit(0)

def recovery_zone(score):
    if score >= 67:
        return 'green'
    if score >= 34:
        return 'yellow'
    return 'red'

def calc_risk(env, aq, recovery_score, sleep_performance):
    score = 0
    if env:
        pressure_change = env.get('PressureChangeHPa', 0) or 0
        humidity = env.get('HumidityPercent', 0) or 0
        weather_code = env.get('WeatherCode', 0) or 0
        if pressure_change <= -10:
            score += 40
        elif pressure_change <= -5:
            score += 25
        elif pressure_change <= -3:
            score += 10
        if humidity >= 80:
            score += 10
        if weather_code in {51, 53, 55, 56, 57, 61, 63, 65, 66, 67, 80, 81, 82, 95, 96, 99}:
            score += 10

    if aq:
        pm25 = aq.get('PM25UgM3', 0) or 0
        ox = aq.get('OxPpm', 0) or 0
        if pm25 >= 36:
            score += 15
        elif pm25 >= 16:
            score += 5
        if ox >= 0.12:
            score += 15
        elif ox >= 0.06:
            score += 5

    if recovery_score > 0 and recovery_score < 34:
        score += 15
    elif 34 <= recovery_score <= 66:
        score += 5

    if sleep_performance > 0 and sleep_performance <= 60:
        score += 10

    score = min(score, 100)
    if score >= 51:
        return score, 'High', '🔴'
    if score >= 26:
        return score, 'Moderate', '🟡'
    return score, 'Low', '🟢'

print('status=ok')
print(f'date={d.get(\"date\", \"\")}')

recovery_score = 0.0
sleep_performance = 0.0

if recovery and recovery[0].get('score'):
    s = recovery[0]['score']
    recovery_score = s.get('recovery_score', 0) or 0
    print(f'recovery_score={recovery_score:.0f}')
    print(f'hrv_ms={s.get(\"hrv_rmssd_milli\", 0):.1f}')
    print(f'rhr_bpm={s.get(\"resting_heart_rate\", 0):.0f}')
    print(f'spo2_pct={s.get(\"spo2_percentage\", 0):.1f}')
    print(f'recovery_zone={recovery_zone(recovery_score)}')

if sleep and sleep[0].get('score'):
    s = sleep[0]['score']
    st = s.get('stage_summary', {})
    total_h = (st.get('total_in_bed_time_milli', 0) or 0) / 3600000
    sleep_performance = s.get('sleep_performance_percentage', 0) or 0
    print(f'sleep_hours={total_h:.1f}')
    print(f'sleep_performance={sleep_performance:.0f}')
    print(f'sleep_efficiency={s.get(\"sleep_efficiency_percentage\", 0):.0f}')
    debt_h = (s.get('sleep_needed', {}).get('need_from_sleep_debt_milli', 0) or 0) / 3600000
    print(f'sleep_debt_hours={debt_h:.1f}')

if cycles and cycles[0].get('score'):
    s = cycles[0]['score']
    print(f'strain={s.get(\"strain\", 0):.1f}')
    print(f'calories_kj={s.get(\"kilojoule\", 0):.0f}')

print(f'workout_count={len(workouts)}')

if weather:
    print(f'weather_label={weather.get(\"WeatherLabel\", \"\")}')
    print(f'weather_emoji={weather.get(\"WeatherEmoji\", \"\")}')
    print(f'temperature_max_c={weather.get(\"TemperatureMaxC\", 0):.1f}')
    print(f'temperature_min_c={weather.get(\"TemperatureMinC\", 0):.1f}')
    print(f'apparent_temperature_c={weather.get(\"ApparentTemperatureC\", 0):.1f}')
    print(f'humidity_pct={weather.get(\"HumidityPercent\", 0):.1f}')
    print(f'pressure_hpa={weather.get(\"PressureHPa\", 0):.1f}')
    print(f'pressure_change_hpa={weather.get(\"PressureChangeHPa\", 0):.1f}')
    print(f'pressure_alert={weather.get(\"PressureAlert\", \"\") or \"none\"}')
    print(f'uv_index={weather.get(\"UVIndexMax\", 0):.1f}')
    print(f'wind_speed_ms={weather.get(\"WindSpeedMS\", 0):.1f}')

if air:
    print(f'pm25_ug_m3={air.get(\"PM25UgM3\", 0):.1f}')
    pm25_level = (air.get('PM25Level') or {})
    print(f'pm25_level={pm25_level.get(\"Label\", \"\")}')
    print(f'pm25_emoji={pm25_level.get(\"Emoji\", \"\")}')
    print(f'ox_ppm={air.get(\"OxPpm\", 0):.3f}')
    ox_level = (air.get('OxLevel') or {})
    print(f'ox_level={ox_level.get(\"Label\", \"\")}')
    print(f'ox_emoji={ox_level.get(\"Emoji\", \"\")}')
    print(f'air_station_code={air.get(\"StationCode\", \"\")}')

risk_score, risk_level, risk_emoji = calc_risk(weather, air, recovery_score, sleep_performance)
print(f'weather_risk_score={risk_score}')
print(f'weather_risk_level={risk_level}')
print(f'weather_risk_emoji={risk_emoji}')
" <<< "$JSON"
