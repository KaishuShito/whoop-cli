package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/kai/whoop-journal/internal/airquality"
)

const defaultBaseURL = "https://api.open-meteo.com/v1/forecast"

type Client struct {
	baseURL    string
	httpClient *http.Client
	lat        float64
	lon        float64
	timezone   string
}

type DailyData struct {
	Date                 string
	WeatherCode          int
	WeatherLabel         string
	WeatherEmoji         string
	TemperatureMaxC      float64
	TemperatureMinC      float64
	HumidityPercent      float64
	PressureHPa          float64
	PressureChangeHPa    float64
	UVIndexMax           float64
	ApparentTemperatureC float64
	WindSpeedMS          float64
	PressureAlert        string
}

type RiskScore struct {
	Score int
	Level string
	Emoji string
}

type apiResponse struct {
	Daily struct {
		Time           []string  `json:"time"`
		WeatherCode    []int     `json:"weather_code"`
		TemperatureMax []float64 `json:"temperature_2m_max"`
		TemperatureMin []float64 `json:"temperature_2m_min"`
		UVIndexMax     []float64 `json:"uv_index_max"`
	} `json:"daily"`
	Hourly struct {
		Time                []string  `json:"time"`
		SurfacePressure     []float64 `json:"surface_pressure"`
		RelativeHumidity    []float64 `json:"relative_humidity_2m"`
		ApparentTemperature []float64 `json:"apparent_temperature"`
		WindSpeed10M        []float64 `json:"wind_speed_10m"`
	} `json:"hourly"`
}

type hourlySummary struct {
	PressureHPa          float64
	HumidityPercent      float64
	ApparentTemperatureC float64
	WindSpeedMS          float64
}

func NewClient(lat, lon float64) *Client {
	return &Client{
		baseURL: defaultBaseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		lat:      lat,
		lon:      lon,
		timezone: "Asia/Tokyo",
	}
}

func (c *Client) SetBaseURL(baseURL string) {
	c.baseURL = baseURL
}

func (c *Client) FetchDay(ctx context.Context, date string) (*DailyData, error) {
	targetDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, fmt.Errorf("parse date: %w", err)
	}
	prevDate := targetDate.AddDate(0, 0, -1)

	reqURL, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse base url: %w", err)
	}

	q := reqURL.Query()
	q.Set("latitude", strconv.FormatFloat(c.lat, 'f', 4, 64))
	q.Set("longitude", strconv.FormatFloat(c.lon, 'f', 4, 64))
	q.Set("start_date", prevDate.Format("2006-01-02"))
	q.Set("end_date", targetDate.Format("2006-01-02"))
	q.Set("daily", "weather_code,temperature_2m_max,temperature_2m_min,uv_index_max")
	q.Set("hourly", "surface_pressure,relative_humidity_2m,apparent_temperature,wind_speed_10m")
	q.Set("timezone", c.timezone)
	reqURL.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("weather api %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var apiResp apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, err
	}

	targetIndex := -1
	prevIndex := -1
	for i, day := range apiResp.Daily.Time {
		switch day {
		case date:
			targetIndex = i
		case prevDate.Format("2006-01-02"):
			prevIndex = i
		}
	}
	if targetIndex == -1 {
		return nil, fmt.Errorf("target date %s missing from weather response", date)
	}

	targetSummary, err := aggregateHourlyForDate(apiResp, date)
	if err != nil {
		return nil, err
	}

	prevPressure := targetSummary.PressureHPa
	if prevIndex != -1 {
		if prevSummary, err := aggregateHourlyForDate(apiResp, prevDate.Format("2006-01-02")); err == nil {
			prevPressure = prevSummary.PressureHPa
		}
	}

	label, emoji := WeatherDescription(apiResp.Daily.WeatherCode[targetIndex])
	change := targetSummary.PressureHPa - prevPressure

	data := &DailyData{
		Date:                 date,
		WeatherCode:          apiResp.Daily.WeatherCode[targetIndex],
		WeatherLabel:         label,
		WeatherEmoji:         emoji,
		TemperatureMaxC:      safeDailyFloat(apiResp.Daily.TemperatureMax, targetIndex),
		TemperatureMinC:      safeDailyFloat(apiResp.Daily.TemperatureMin, targetIndex),
		HumidityPercent:      targetSummary.HumidityPercent,
		PressureHPa:          targetSummary.PressureHPa,
		PressureChangeHPa:    change,
		UVIndexMax:           safeDailyFloat(apiResp.Daily.UVIndexMax, targetIndex),
		ApparentTemperatureC: targetSummary.ApparentTemperatureC,
		WindSpeedMS:          targetSummary.WindSpeedMS,
	}
	if change <= -5 {
		data.PressureAlert = "⚠️ 気圧急低下"
	} else if change >= 5 {
		data.PressureAlert = "⚠️ 気圧急上昇"
	}

	return data, nil
}

func CalculateRisk(env *DailyData, recoveryScore, sleepPerformance float64) RiskScore {
	return CalculateRiskWithAirQuality(env, nil, recoveryScore, sleepPerformance)
}

func CalculateRiskWithAirQuality(env *DailyData, aq *airquality.DailyData, recoveryScore, sleepPerformance float64) RiskScore {
	score := 0

	if env != nil {
		switch {
		case env.PressureChangeHPa <= -10:
			score += 40
		case env.PressureChangeHPa <= -5:
			score += 25
		case env.PressureChangeHPa <= -3:
			score += 10
		}

		if env.HumidityPercent >= 80 {
			score += 10
		}
		if IsRainCode(env.WeatherCode) {
			score += 10
		}
	}

	if aq != nil {
		switch {
		case aq.PM25UgM3 >= 36:
			score += 15
		case aq.PM25UgM3 >= 16:
			score += 5
		}

		switch {
		case aq.OxPpm >= 0.12:
			score += 15
		case aq.OxPpm >= 0.06:
			score += 5
		}
	}

	switch {
	case recoveryScore > 0 && recoveryScore < 34:
		score += 15
	case recoveryScore >= 34 && recoveryScore <= 66:
		score += 5
	}

	if sleepPerformance > 0 && sleepPerformance <= 60 {
		score += 10
	}

	if score > 100 {
		score = 100
	}

	risk := RiskScore{Score: score}
	switch {
	case score >= 51:
		risk.Level = "High"
		risk.Emoji = "🔴"
	case score >= 26:
		risk.Level = "Moderate"
		risk.Emoji = "🟡"
	default:
		risk.Level = "Low"
		risk.Emoji = "🟢"
	}
	return risk
}

func WeatherDescription(code int) (label, emoji string) {
	switch code {
	case 0:
		return "晴れ", "☀️"
	case 1:
		return "おおむね晴れ", "🌤️"
	case 2:
		return "晴れ時々曇り", "⛅"
	case 3:
		return "曇り", "☁️"
	case 45, 48:
		return "霧", "🌫️"
	case 51, 53, 55, 56, 57:
		return "霧雨", "🌦️"
	case 61, 63, 65, 66, 67, 80, 81, 82:
		return "雨", "🌧️"
	case 71, 73, 75, 77, 85, 86:
		return "雪", "❄️"
	case 95, 96, 99:
		return "雷雨", "⛈️"
	default:
		return "不明", "🌡️"
	}
}

func IsRainCode(code int) bool {
	switch code {
	case 51, 53, 55, 56, 57, 61, 63, 65, 66, 67, 80, 81, 82, 95, 96, 99:
		return true
	default:
		return false
	}
}

func UVIndexLabel(index float64) string {
	switch {
	case index < 3:
		return "Low"
	case index < 6:
		return "Moderate"
	case index < 8:
		return "High"
	case index < 11:
		return "Very High"
	default:
		return "Extreme"
	}
}

func aggregateHourlyForDate(apiResp apiResponse, date string) (hourlySummary, error) {
	var pressureSum float64
	var pressureCount int
	var humiditySum float64
	var humidityCount int
	var apparentSum float64
	var apparentCount int
	var windSum float64
	var windCount int

	for i, ts := range apiResp.Hourly.Time {
		if !strings.HasPrefix(ts, date) {
			continue
		}
		hour := parseHour(ts)
		if i < len(apiResp.Hourly.SurfacePressure) {
			pressureSum += apiResp.Hourly.SurfacePressure[i]
			pressureCount++
		}
		if i < len(apiResp.Hourly.RelativeHumidity) {
			humiditySum += apiResp.Hourly.RelativeHumidity[i]
			humidityCount++
		}
		if hour >= 6 && hour <= 22 {
			if i < len(apiResp.Hourly.ApparentTemperature) {
				apparentSum += apiResp.Hourly.ApparentTemperature[i]
				apparentCount++
			}
			if i < len(apiResp.Hourly.WindSpeed10M) {
				windSum += apiResp.Hourly.WindSpeed10M[i]
				windCount++
			}
		}
	}

	if pressureCount == 0 {
		return hourlySummary{}, fmt.Errorf("no hourly pressure data for %s", date)
	}
	if humidityCount == 0 {
		return hourlySummary{}, fmt.Errorf("no hourly humidity data for %s", date)
	}
	if apparentCount == 0 {
		return hourlySummary{}, fmt.Errorf("no daytime apparent temperature data for %s", date)
	}
	if windCount == 0 {
		return hourlySummary{}, fmt.Errorf("no daytime wind data for %s", date)
	}

	return hourlySummary{
		PressureHPa:          pressureSum / float64(pressureCount),
		HumidityPercent:      humiditySum / float64(humidityCount),
		ApparentTemperatureC: apparentSum / float64(apparentCount),
		WindSpeedMS:          windSum / float64(windCount),
	}, nil
}

func safeDailyFloat(values []float64, idx int) float64 {
	if idx < 0 || idx >= len(values) {
		return 0
	}
	return values[idx]
}

func parseHour(ts string) int {
	parts := strings.Split(ts, "T")
	if len(parts) != 2 || len(parts[1]) < 2 {
		return -1
	}
	hour, err := strconv.Atoi(parts[1][:2])
	if err != nil {
		return -1
	}
	return hour
}
