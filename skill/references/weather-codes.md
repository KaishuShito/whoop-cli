# Weather Codes

This file maps the WMO weather codes used by Open-Meteo to the Japanese labels and emoji currently used in this project.

Read this file when you need to explain `weather_label`, interpret a weather-code integer from JSON, or extend the current mapping.

## Current Project Mapping

| WMO code(s) | Japanese label | Emoji | Notes |
| --- | --- | --- | --- |
| `0` | 晴れ | `☀️` | Clear sky |
| `1` | おおむね晴れ | `🌤️` | Mainly clear |
| `2` | 晴れ時々曇り | `⛅` | Partly cloudy |
| `3` | 曇り | `☁️` | Overcast |
| `45, 48` | 霧 | `🌫️` | Fog / depositing rime fog |
| `51, 53, 55, 56, 57` | 霧雨 | `🌦️` | Drizzle family |
| `61, 63, 65, 66, 67, 80, 81, 82` | 雨 | `🌧️` | Rain / showers |
| `71, 73, 75, 77, 85, 86` | 雪 | `❄️` | Snow / snow grains / snow showers |
| `95, 96, 99` | 雷雨 | `⛈️` | Thunderstorm family |
| other | 不明 | `🌡️` | Fallback |

## Rain-Sensitive Codes Used for Risk Scoring

These codes are treated as rain-sensitive in the risk score:

`51,53,55,56,57,61,63,65,66,67,80,81,82,95,96,99`

If today's weather code is in that set, the composite risk score gets `+10`.

## Notes

- The project intentionally collapses many WMO subcodes into simpler Japanese labels for agent-facing summaries.
- If you need a more granular distinction, update both:
  `internal/weather/client.go`
  and this file.
