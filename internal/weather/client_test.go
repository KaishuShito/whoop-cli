package weather

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kai/whoop-journal/internal/airquality"
)

func TestFetchDay(t *testing.T) {
	var capturedQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"daily": {
				"time": ["2026-03-15", "2026-03-16"],
				"weather_code": [3, 61],
				"temperature_2m_max": [17.2, 22.1],
				"temperature_2m_min": [8.3, 10.4],
				"uv_index_max": [4.0, 5.0]
			},
			"hourly": {
				"time": [
					"2026-03-15T00:00", "2026-03-15T12:00",
					"2026-03-16T00:00", "2026-03-16T06:00", "2026-03-16T12:00", "2026-03-16T22:00"
				],
				"surface_pressure": [1018, 1016, 1010, 1010, 1010, 1010],
				"relative_humidity_2m": [60, 70, 85, 75, 80, 85],
				"apparent_temperature": [7, 9, 10, 14, 15, 13],
				"wind_speed_10m": [2.0, 2.5, 1.5, 4.8, 5.2, 5.6]
			}
		}`))
	}))
	defer server.Close()

	client := NewClient(35.6503, 139.7225)
	client.SetBaseURL(server.URL)

	data, err := client.FetchDay(context.Background(), "2026-03-16")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(capturedQuery, "start_date=2026-03-15") {
		t.Fatalf("query should include previous day, got %q", capturedQuery)
	}
	if data.WeatherLabel != "雨" || data.WeatherEmoji != "🌧️" {
		t.Fatalf("unexpected weather description: %+v", data)
	}
	if data.PressureHPa != 1010 {
		t.Fatalf("pressure = %.1f, want 1010", data.PressureHPa)
	}
	if data.PressureChangeHPa != -7 {
		t.Fatalf("pressure change = %.1f, want -7", data.PressureChangeHPa)
	}
	if data.PressureAlert == "" {
		t.Fatal("expected pressure alert")
	}
	if data.HumidityPercent != 81.25 {
		t.Fatalf("humidity = %.2f, want 81.25", data.HumidityPercent)
	}
	if data.ApparentTemperatureC != 14 {
		t.Fatalf("apparent temp = %.1f, want 14", data.ApparentTemperatureC)
	}
	if data.WindSpeedMS != 5.2 {
		t.Fatalf("wind speed = %.1f, want 5.2", data.WindSpeedMS)
	}
}

func TestCalculateRisk(t *testing.T) {
	env := &DailyData{
		PressureChangeHPa: -7,
		HumidityPercent:   82,
		WeatherCode:       61,
	}
	risk := CalculateRisk(env, 20, 55)
	if risk.Score != 70 {
		t.Fatalf("score = %d, want 70", risk.Score)
	}
	if risk.Level != "High" || risk.Emoji != "🔴" {
		t.Fatalf("unexpected risk level: %+v", risk)
	}
}

func TestCalculateRiskWithAirQuality(t *testing.T) {
	env := &DailyData{
		PressureChangeHPa: -7,
		HumidityPercent:   82,
		WeatherCode:       61,
	}
	aq := &airquality.DailyData{
		PM25UgM3: 18,
		OxPpm:    0.08,
	}
	risk := CalculateRiskWithAirQuality(env, aq, 54, 82)
	if risk.Score != 60 {
		t.Fatalf("score = %d, want 60", risk.Score)
	}
	if risk.Level != "High" || risk.Emoji != "🔴" {
		t.Fatalf("unexpected risk level: %+v", risk)
	}
}

func TestUVIndexLabel(t *testing.T) {
	tests := []struct {
		input float64
		want  string
	}{
		{1, "Low"},
		{5, "Moderate"},
		{7, "High"},
		{10, "Very High"},
		{12, "Extreme"},
	}
	for _, tt := range tests {
		if got := UVIndexLabel(tt.input); got != tt.want {
			t.Fatalf("UVIndexLabel(%.1f) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
