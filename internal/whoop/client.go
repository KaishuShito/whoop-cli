package whoop

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/kai/whoop-journal/internal/airquality"
	"github.com/kai/whoop-journal/internal/weather"
)

const apiBase = "https://api.prod.whoop.com/developer/v2"

type Client struct {
	httpClient  *http.Client
	accessToken string
}

func NewClient(accessToken string) *Client {
	return &Client{
		httpClient:  &http.Client{Timeout: 30 * time.Second},
		accessToken: accessToken,
	}
}

func (c *Client) SetAccessToken(token string) {
	c.accessToken = token
}

// --- API response types ---

type PaginatedResponse[T any] struct {
	Records   []T    `json:"records"`
	NextToken string `json:"next_token"`
}

type Cycle struct {
	ID             int64       `json:"id"`
	UserID         int64       `json:"user_id"`
	Start          string      `json:"start"`
	End            string      `json:"end"`
	TimezoneOffset string      `json:"timezone_offset"`
	ScoreState     string      `json:"score_state"`
	Score          *CycleScore `json:"score"`
}

type CycleScore struct {
	Strain           float64 `json:"strain"`
	Kilojoule        float64 `json:"kilojoule"`
	AverageHeartRate int     `json:"average_heart_rate"`
	MaxHeartRate     int     `json:"max_heart_rate"`
}

type Recovery struct {
	CycleID    int64          `json:"cycle_id"`
	SleepID    string         `json:"sleep_id"`
	ScoreState string         `json:"score_state"`
	Score      *RecoveryScore `json:"score"`
}

type RecoveryScore struct {
	UserCalibrating  bool    `json:"user_calibrating"`
	RecoveryScore    float64 `json:"recovery_score"`
	RestingHeartRate float64 `json:"resting_heart_rate"`
	HrvRmssdMilli    float64 `json:"hrv_rmssd_milli"`
	Spo2Percentage   float64 `json:"spo2_percentage"`
	SkinTempCelsius  float64 `json:"skin_temp_celsius"`
}

type Sleep struct {
	ID             string      `json:"id"`
	CycleID        int64       `json:"cycle_id"`
	Start          string      `json:"start"`
	End            string      `json:"end"`
	TimezoneOffset string      `json:"timezone_offset"`
	Nap            bool        `json:"nap"`
	ScoreState     string      `json:"score_state"`
	Score          *SleepScore `json:"score"`
}

type SleepScore struct {
	StageSummary               StageSummary `json:"stage_summary"`
	SleepNeeded                SleepNeeded  `json:"sleep_needed"`
	RespiratoryRate            float64      `json:"respiratory_rate"`
	SleepPerformancePercentage float64      `json:"sleep_performance_percentage"`
	SleepConsistencyPercentage float64      `json:"sleep_consistency_percentage"`
	SleepEfficiencyPercentage  float64      `json:"sleep_efficiency_percentage"`
}

type StageSummary struct {
	TotalInBedTimeMilli         int64 `json:"total_in_bed_time_milli"`
	TotalAwakeTimeMilli         int64 `json:"total_awake_time_milli"`
	TotalLightSleepTimeMilli    int64 `json:"total_light_sleep_time_milli"`
	TotalSlowWaveSleepTimeMilli int64 `json:"total_slow_wave_sleep_time_milli"`
	TotalRemSleepTimeMilli      int64 `json:"total_rem_sleep_time_milli"`
	SleepCycleCount             int   `json:"sleep_cycle_count"`
	DisturbanceCount            int   `json:"disturbance_count"`
}

type SleepNeeded struct {
	BaselineMilli             int64 `json:"baseline_milli"`
	NeedFromSleepDebtMilli    int64 `json:"need_from_sleep_debt_milli"`
	NeedFromRecentStrainMilli int64 `json:"need_from_recent_strain_milli"`
	NeedFromRecentNapMilli    int64 `json:"need_from_recent_nap_milli"`
}

type Workout struct {
	ID             string        `json:"id"`
	Start          string        `json:"start"`
	End            string        `json:"end"`
	TimezoneOffset string        `json:"timezone_offset"`
	SportName      string        `json:"sport_name"`
	SportID        int           `json:"sport_id"`
	ScoreState     string        `json:"score_state"`
	Score          *WorkoutScore `json:"score"`
}

type WorkoutScore struct {
	Strain           float64       `json:"strain"`
	AverageHeartRate int           `json:"average_heart_rate"`
	MaxHeartRate     int           `json:"max_heart_rate"`
	Kilojoule        float64       `json:"kilojoule"`
	DistanceMeter    *float64      `json:"distance_meter"`
	ZoneDurations    ZoneDurations `json:"zone_durations"`
}

type ZoneDurations struct {
	ZoneZeroMilli  int64 `json:"zone_zero_milli"`
	ZoneOneMilli   int64 `json:"zone_one_milli"`
	ZoneTwoMilli   int64 `json:"zone_two_milli"`
	ZoneThreeMilli int64 `json:"zone_three_milli"`
	ZoneFourMilli  int64 `json:"zone_four_milli"`
	ZoneFiveMilli  int64 `json:"zone_five_milli"`
}

// --- Aggregated daily data ---

type DayData struct {
	Date       string                `json:"date"`
	Cycles     []Cycle               `json:"cycles"`
	Recovery   []Recovery            `json:"recovery"`
	Sleep      []Sleep               `json:"sleep"`
	Workouts   []Workout             `json:"workouts"`
	Weather    *weather.DailyData    `json:"weather,omitempty"`
	AirQuality *airquality.DailyData `json:"air_quality,omitempty"`
}

func (d *DayData) HasData() bool {
	return len(d.Cycles) > 0 || len(d.Recovery) > 0 || len(d.Sleep) > 0 || len(d.Workouts) > 0
}

// --- API methods ---

var ErrUnauthorized = fmt.Errorf("unauthorized (token expired)")

func (c *Client) FetchDay(ctx context.Context, date string) (*DayData, error) {
	jst := time.FixedZone("JST", 9*3600)
	dayStart, err := time.ParseInLocation("2006-01-02", date, jst)
	if err != nil {
		return nil, fmt.Errorf("parse date: %w", err)
	}
	dayEnd := dayStart.Add(24 * time.Hour)

	startUTC := dayStart.UTC().Format("2006-01-02T15:04:05.000Z")
	endUTC := dayEnd.UTC().Format("2006-01-02T15:04:05.000Z")

	params := fmt.Sprintf("start=%s&end=%s&limit=10", startUTC, endUTC)

	data := &DayData{Date: date}

	// Cycles — if this returns 401, propagate immediately for token refresh
	var cyclesResp PaginatedResponse[Cycle]
	if err := c.get(ctx, "/cycle?"+params, &cyclesResp); err != nil {
		if strings.Contains(err.Error(), "unauthorized") {
			return nil, ErrUnauthorized
		}
		// Non-auth errors: continue, this endpoint might just be down
	} else {
		data.Cycles = cyclesResp.Records
	}

	// Recovery
	var recoveryResp PaginatedResponse[Recovery]
	if err := c.get(ctx, "/recovery?"+params, &recoveryResp); err == nil {
		data.Recovery = recoveryResp.Records
	}

	// Sleep
	var sleepResp PaginatedResponse[Sleep]
	if err := c.get(ctx, "/activity/sleep?"+params, &sleepResp); err == nil {
		data.Sleep = sleepResp.Records
	}

	// Workouts
	var workoutResp PaginatedResponse[Workout]
	if err := c.get(ctx, "/activity/workout?"+params, &workoutResp); err == nil {
		data.Workouts = workoutResp.Records
	}

	return data, nil
}

func (c *Client) get(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, "GET", apiBase+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return fmt.Errorf("unauthorized (token expired)")
	}
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("API %d: %s", resp.StatusCode, string(body))
	}

	return json.NewDecoder(resp.Body).Decode(out)
}
