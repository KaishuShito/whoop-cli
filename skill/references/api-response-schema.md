# WHOOP API v2 Response Schema

Read this file when you need to parse `--json` output or write custom analysis.

## DayData (top-level)

```json
{
  "date": "2026-03-18",
  "cycles": [...],
  "recovery": [...],
  "sleep": [...],
  "workouts": [...]
}
```

## Recovery

```
.recovery[0].score.recovery_score    float  0-100 (Green >=67, Yellow >=34, Red <34)
.recovery[0].score.hrv_rmssd_milli   float  Heart Rate Variability in ms
.recovery[0].score.resting_heart_rate float  Resting heart rate in bpm
.recovery[0].score.spo2_percentage   float  Blood oxygen %
.recovery[0].score.skin_temp_celsius float  Skin temperature
.recovery[0].score.user_calibrating  bool   True during first ~30 days
```

## Sleep

```
.sleep[0].score.stage_summary.total_in_bed_time_milli          int64  Total time in bed
.sleep[0].score.stage_summary.total_rem_sleep_time_milli       int64  REM sleep
.sleep[0].score.stage_summary.total_slow_wave_sleep_time_milli int64  Deep sleep (SWS)
.sleep[0].score.stage_summary.total_light_sleep_time_milli     int64  Light sleep
.sleep[0].score.stage_summary.total_awake_time_milli           int64  Awake time
.sleep[0].score.stage_summary.sleep_cycle_count                int    Number of cycles
.sleep[0].score.stage_summary.disturbance_count                int    Wake-ups
.sleep[0].score.sleep_needed.baseline_milli                    int64  Base need
.sleep[0].score.sleep_needed.need_from_sleep_debt_milli        int64  Debt accumulation
.sleep[0].score.sleep_needed.need_from_recent_strain_milli     int64  Strain-driven need
.sleep[0].score.sleep_performance_percentage                   float  % of need met
.sleep[0].score.sleep_efficiency_percentage                    float  Sleep vs in-bed %
.sleep[0].score.sleep_consistency_percentage                   float  Schedule regularity
.sleep[0].score.respiratory_rate                               float  Breaths per minute
```

All durations are in milliseconds. Divide by 3600000 for hours, 60000 for minutes.

## Cycle (Day Strain)

```
.cycles[0].score.strain              float  0-21 scale
.cycles[0].score.kilojoule           float  Calories burned
.cycles[0].score.average_heart_rate  int    Daily avg HR
.cycles[0].score.max_heart_rate      int    Daily max HR
```

## Workout

```
.workouts[N].sport_name                         string  Activity name
.workouts[N].score.strain                        float   Workout strain
.workouts[N].score.average_heart_rate            int
.workouts[N].score.max_heart_rate                int
.workouts[N].score.kilojoule                     float
.workouts[N].score.distance_meter                float?  null if not tracked
.workouts[N].score.zone_durations.zone_zero_milli  int64  HR zone 0 (rest)
  ... zone_one through zone_five_milli
```
