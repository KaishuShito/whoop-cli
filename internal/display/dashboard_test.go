package display

import (
	"strings"
	"testing"

	"github.com/KaishuShito/whoop-cli/internal/airquality"
	"github.com/KaishuShito/whoop-cli/internal/weather"
	"github.com/KaishuShito/whoop-cli/internal/whoop"
)

func TestRenderToday(t *testing.T) {
	data := &whoop.DayData{
		Date: "2026-03-31",
		Recovery: []whoop.Recovery{{
			Score: &whoop.RecoveryScore{
				RecoveryScore:    79,
				HrvRmssdMilli:    27,
				RestingHeartRate: 54,
				Spo2Percentage:   93.4,
			},
		}},
		Sleep: []whoop.Sleep{{
			Score: &whoop.SleepScore{
				SleepPerformancePercentage: 84,
				StageSummary: whoop.StageSummary{
					TotalInBedTimeMilli:         8*60*60*1000 + 9*60*1000,
					TotalSlowWaveSleepTimeMilli: 1*60*60*1000 + 59*60*1000,
					TotalRemSleepTimeMilli:      1*60*60*1000 + 14*60*1000,
					TotalLightSleepTimeMilli:    4*60*60*1000 + 23*60*1000,
				},
			},
		}},
		Cycles: []whoop.Cycle{{
			Score: &whoop.CycleScore{
				Strain:           6.3,
				Kilojoule:        5432,
				AverageHeartRate: 62,
				MaxHeartRate:     165,
			},
		}},
		Weather: &weather.DailyData{
			WeatherEmoji:         "🌧️",
			WeatherLabel:         "雨",
			TemperatureMaxC:      18,
			ApparentTemperatureC: 15,
			WindSpeedMS:          15.0,
			HumidityPercent:      88,
			UVIndexMax:           3,
			PressureHPa:          1008,
			PressureChangeHPa:    -10,
			PressureAlert:        "⚠️ 気圧急低下",
			WeatherCode:          61,
		},
		AirQuality: &airquality.DailyData{
			PM25UgM3:  12,
			OzoneUgM3: 45,
			NO2UgM3:   18,
			PM25Level: airquality.PM25Level(12),
			OxLevel:   airquality.OxLevel(45 / 1963.6),
			OxPpm:     45 / 1963.6,
		},
	}

	got := RenderToday(data, true)

	for _, want := range []string{
		"┌",
		"RECOVERY",
		"SLEEP",
		"STRAIN",
		"WEATHER",
		"AIR QUALITY",
		"HEALTH RISK",
		"Take it easy today. Stay hydrated.",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected output to contain %q, got:\n%s", want, got)
		}
	}
}
