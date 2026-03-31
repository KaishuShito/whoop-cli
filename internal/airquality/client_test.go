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
		w.Header().Set("Content-Type", "application/json;charset=SJIS")
		_, _ = w.Write([]byte(`[
			{"SKT_CD":"13103010","SKT_DATE":"2026/03/16","SKT_TIME":"08","PM2_5":"12","OX":"0.044","SO2":"","NO2":"0.004"},
			{"SKT_CD":"13103010","SKT_DATE":"2026/03/16","SKT_TIME":"09","PM2_5":"18","OX":"0.055","SO2":"","NO2":"0.005"},
			{"SKT_CD":"13103010","SKT_DATE":"2026/03/16","SKT_TIME":"10","PM2_5":"24","OX":"0.034","SO2":"0.001","NO2":"0.006"},
			{"SKT_CD":"13103010","SKT_DATE":"2026/03/16","SKT_TIME":"11","PM2_5":"-1","OX":"-1","SO2":"","NO2":""},
			{"SKT_CD":"13103010","SKT_DATE":"2026/03/15","SKT_TIME":"10","PM2_5":"99","OX":"0.200","SO2":"0.100","NO2":"0.100"}
		]`))
	}))
	defer server.Close()

	client := NewClient("13103010")
	client.SetBaseURL(server.URL)

	data, err := client.FetchDay(context.Background(), "2026-03-16")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(capturedQuery, "Start_YM=202603") {
		t.Fatalf("query should include Start_YM, got %q", capturedQuery)
	}
	if !strings.Contains(capturedQuery, "TDFKN_CD=13") {
		t.Fatalf("query should include prefecture code, got %q", capturedQuery)
	}
	if !strings.Contains(capturedQuery, "SKT_CD=13103010") {
		t.Fatalf("query should include station code, got %q", capturedQuery)
	}
	if data.PM25UgM3 != 18 {
		t.Fatalf("PM2.5 = %.1f, want 18", data.PM25UgM3)
	}
	if data.OxPpm != 0.055 {
		t.Fatalf("Ox = %.3f, want 0.055", data.OxPpm)
	}
	if data.SO2Ppm == nil || *data.SO2Ppm != 0.001 {
		t.Fatalf("SO2 = %v, want 0.001", data.SO2Ppm)
	}
	if data.NO2Ppm == nil || *data.NO2Ppm != 0.005 {
		t.Fatalf("NO2 = %v, want 0.005", data.NO2Ppm)
	}
	if data.PM25Level.Emoji != "🟡" || data.OxLevel.Emoji != "🟢" {
		t.Fatalf("unexpected levels: %+v", data)
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
