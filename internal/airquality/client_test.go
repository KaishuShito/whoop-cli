package airquality

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFetchDay(t *testing.T) {
	var capturedQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"hourly": {
				"time": [
					"2026-03-16T05:00", "2026-03-16T06:00", "2026-03-16T12:00",
					"2026-03-16T22:00", "2026-03-16T23:00"
				],
				"pm2_5": [99, 12, 18, 24, 77],
				"pm10": [99, 20, 28, 32, 77],
				"ozone": [999, 40, 60, 80, 777],
				"nitrogen_dioxide": [999, 12, 18, 24, 777]
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

	if !strings.Contains(capturedQuery, "latitude=35.6503") {
		t.Fatalf("query should include latitude, got %q", capturedQuery)
	}
	if !strings.Contains(capturedQuery, "longitude=139.7225") {
		t.Fatalf("query should include longitude, got %q", capturedQuery)
	}
	if !strings.Contains(capturedQuery, "hourly=pm2_5%2Cpm10%2Cozone%2Cnitrogen_dioxide") {
		t.Fatalf("query should request expected variables, got %q", capturedQuery)
	}
	if data.PM25UgM3 != 18 {
		t.Fatalf("PM2.5 = %.1f, want 18", data.PM25UgM3)
	}
	if data.PM10UgM3 != 26.666666666666668 {
		t.Fatalf("PM10 = %.3f, want average daytime value", data.PM10UgM3)
	}
	if data.OzoneUgM3 != 60 {
		t.Fatalf("ozone ug/m3 = %.1f, want 60", data.OzoneUgM3)
	}
	if data.NO2UgM3 != 18 {
		t.Fatalf("NO2 ug/m3 = %.1f, want 18", data.NO2UgM3)
	}
	if data.PM25Level.Emoji != "🟡" {
		t.Fatalf("unexpected PM2.5 level: %+v", data.PM25Level)
	}
	if data.OxLevel.Emoji != "🟢" {
		t.Fatalf("unexpected ozone level: %+v", data.OxLevel)
	}
}

func TestFetchDay_NoDaytimeData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"hourly": {
				"time": ["2026-03-16T01:00", "2026-03-16T23:00"],
				"pm2_5": [1, 2],
				"pm10": [1, 2],
				"ozone": [1, 2],
				"nitrogen_dioxide": [1, 2]
			}
		}`))
	}))
	defer server.Close()

	client := NewClient(35.6503, 139.7225)
	client.SetBaseURL(server.URL)

	_, err := client.FetchDay(context.Background(), "2026-03-16")
	if err == nil || !strings.Contains(err.Error(), "no daytime air quality data") {
		t.Fatalf("expected daytime data error, got %v", err)
	}
}

func TestLevels(t *testing.T) {
	if got := PM25Level(10); got.Emoji != "🟢" {
		t.Fatalf("PM25 good = %+v", got)
	}
	if got := PM25Level(20); got.Emoji != "🟡" {
		t.Fatalf("PM25 moderate = %+v", got)
	}
	if got := PM25Level(40); got.Emoji != "🔴" {
		t.Fatalf("PM25 bad = %+v", got)
	}

	if got := OxLevel(0.03); got.Emoji != "🟢" {
		t.Fatalf("Ox good = %+v", got)
	}
	if got := OxLevel(0.08); got.Emoji != "🟡" {
		t.Fatalf("Ox moderate = %+v", got)
	}
	if got := OxLevel(0.15); got.Emoji != "🔴" {
		t.Fatalf("Ox bad = %+v", got)
	}
}
