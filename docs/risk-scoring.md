# Risk Scoring

This file documents the exact composite weather-sensitive health risk logic used by `internal/weather/client.go`.

Read this file when you need the exact thresholds, want to explain the score to a user, or need to debug why a day is marked `Low`, `Moderate`, or `High`.

## Inputs

The score combines three domains:

1. Weather
2. Air quality
3. WHOOP recovery / sleep state

The output is a bounded integer score `0-100` plus a categorical level.

## Formula

Start from `score = 0`.

### Weather contribution

Pressure drop:

- `<= -10 hPa`: `+40`
- `<= -5 hPa`: `+25`
- `<= -3 hPa`: `+10`

Humidity:

- `humidity >= 80%`: `+10`

Rain-sensitive weather codes:

- if the WMO code is one of
  `51,53,55,56,57,61,63,65,66,67,80,81,82,95,96,99`
  then add `+10`

### Air quality contribution

PM2.5:

- `>= 36 ug/m3`: `+15`
- `>= 16 ug/m3`: `+5`

Ox:

- `>= 0.12 ppm`: `+15`
- `>= 0.06 ppm`: `+5`

Note:

- Open-Meteo provides ozone in `μg/m³`
- the CLI converts that value to `ppm` before applying the existing Ox thresholds

### WHOOP state contribution

Recovery:

- `recovery_score < 34`: `+15`
- `34 <= recovery_score <= 66`: `+5`

Sleep:

- `sleep_performance <= 60`: `+10`

## Clamping and Levels

After summing all contributions:

- clamp to `100`

Then map:

- `0-25`: `Low` `🟢`
- `26-50`: `Moderate` `🟡`
- `51-100`: `High` `🔴`

## Interpretation

- `Low`: environment is unlikely to be the main bottleneck.
- `Moderate`: environment may noticeably increase fatigue, headache risk, or irritability.
- `High`: environment is a meaningful stressor and should influence scheduling decisions.

## Worked Examples

### Example A: pressure-drop day with weak recovery

- pressure change `-7 hPa`: `+25`
- humidity `82%`: `+10`
- rain code `61`: `+10`
- recovery score `20`: `+15`
- sleep performance `55`: `+10`

Total: `70` -> `High`

This matches `TestCalculateRisk`.

### Example B: decent WHOOP state, poor environment

- pressure change `-7 hPa`: `+25`
- humidity `82%`: `+10`
- rain code `61`: `+10`
- PM2.5 `18`: `+5`
- Ox `0.08`: `+5`
- recovery score `54`: `+5`
- sleep performance `82`: `+0`

Total: `60` -> `High`

This matches `TestCalculateRiskWithAirQuality`.

## Notes

- The score is intentionally heuristic, not medical advice.
- The score is asymmetric around pressure drops: drops matter much more than rises.
- Weather and air-quality failures can be absent; the scoring function will still compute with whichever sources are present.
