# Air Quality Standards

This file explains how PM2.5 and Ox are interpreted in the CLI and what those levels mean in practice.

Read this file when you need to explain `pm25_level`, `ox_level`, or the environmental part of the composite risk score.

## Data Source

Air quality is fetched from Japan's soramame network.

The current skill defaults to station code `13103010`, described in code comments as the closest general station to Hiroo available in the 2026-03-31 station metadata snapshot.

## PM2.5 Interpretation

Project thresholds:

| PM2.5 | CLI level | Emoji | Score impact |
| --- | --- | --- | --- |
| `< 16 ug/m3` | `Good` | `🟢` | `+0` |
| `16-35.9 ug/m3` | `Moderate` | `🟡` | `+5` |
| `>= 36 ug/m3` | `Bad` | `🔴` | `+15` |

Context:

- `16 ug/m3` is treated here as the first practical caution threshold for daily planning.
- `36 ug/m3` is treated as meaningfully bad for sensitive users and contributes strongly to the composite risk score.

## Ox Interpretation

Project thresholds:

| Ox | CLI level | Emoji | Score impact |
| --- | --- | --- | --- |
| `< 0.06 ppm` | `Good` | `🟢` | `+0` |
| `0.06-0.119 ppm` | `Moderate` | `🟡` | `+5` |
| `>= 0.12 ppm` | `Bad` | `🔴` | `+15` |

Context:

- `0.06 ppm` is treated as the point where outdoor sensitivity may start to matter.
- `0.12 ppm` is treated as a materially poor air-quality signal for symptom-aware scheduling.

## How To Use This In Agent Planning

- If both PM2.5 and Ox are elevated, bias toward indoor work, shorter outdoor transitions, and reduced physical load.
- If recovery is already yellow/red, even moderate air-quality readings can be enough to justify a lighter day.
- Air-quality signals are additive, not dominant by themselves. The model is meant for context, not diagnosis.

## Important Caveat

These labels are heuristic planning labels used by the CLI. They are not a substitute for medical advice, public-health alerts, or regulatory compliance reporting.
