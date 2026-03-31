package airquality

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
)

const (
	defaultBaseURL      = "https://air-quality-api.open-meteo.com/v1/air-quality"
	defaultTimezone     = "auto"
	ozoneUgM3PerPPM     = 1963.6
	nitrogenUgM3PerPPM  = 1881.0
	daytimeStartHour    = 6
	daytimeEndHour      = 22
	openMeteoHourlyVars = "pm2_5,pm10,ozone,nitrogen_dioxide"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
	lat        float64
	lon        float64
	timezone   string
}

type DailyData struct {
	Date      string
	Latitude  float64
	Longitude float64
	PM25UgM3  float64
	PM10UgM3  float64
	OzoneUgM3 float64
	OxPpm     float64
	NO2UgM3   float64
	NO2Ppm    float64
	PM25Level Level
	OxLevel   Level
}

type Level struct {
	Label string
	Emoji string
}

type apiResponse struct {
	Hourly struct {
		Time            []string  `json:"time"`
		PM25            []float64 `json:"pm2_5"`
		PM10            []float64 `json:"pm10"`
		Ozone           []float64 `json:"ozone"`
		NitrogenDioxide []float64 `json:"nitrogen_dioxide"`
	} `json:"hourly"`
}

func NewClient(lat, lon float64) *Client {
	return &Client{
		baseURL: defaultBaseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		lat:      lat,
		lon:      lon,
		timezone: defaultTimezone,
	}
}

func (c *Client) SetBaseURL(baseURL string) {
	c.baseURL = baseURL
}

func (c *Client) FetchDay(ctx context.Context, date string) (*DailyData, error) {
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return nil, fmt.Errorf("parse date: %w", err)
	}

	reqURL, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse base url: %w", err)
	}

	q := reqURL.Query()
	q.Set("latitude", strconv.FormatFloat(c.lat, 'f', 4, 64))
	q.Set("longitude", strconv.FormatFloat(c.lon, 'f', 4, 64))
	q.Set("start_date", date)
	q.Set("end_date", date)
	q.Set("hourly", openMeteoHourlyVars)
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
		return nil, fmt.Errorf("air quality api %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var apiResp apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, err
	}

	var pm25Sum, pm10Sum, ozoneSum, no2Sum float64
	var count int
	for i, ts := range apiResp.Hourly.Time {
		parsed, err := time.Parse("2006-01-02T15:04", ts)
		if err != nil {
			return nil, fmt.Errorf("parse hourly timestamp %q: %w", ts, err)
		}
		if parsed.Format("2006-01-02") != date {
			continue
		}
		if parsed.Hour() < daytimeStartHour || parsed.Hour() > daytimeEndHour {
			continue
		}
		if !hasIndex(apiResp.Hourly.PM25, i) || !hasIndex(apiResp.Hourly.PM10, i) || !hasIndex(apiResp.Hourly.Ozone, i) || !hasIndex(apiResp.Hourly.NitrogenDioxide, i) {
			continue
		}

		pm25Sum += apiResp.Hourly.PM25[i]
		pm10Sum += apiResp.Hourly.PM10[i]
		ozoneSum += apiResp.Hourly.Ozone[i]
		no2Sum += apiResp.Hourly.NitrogenDioxide[i]
		count++
	}

	if count == 0 {
		return nil, fmt.Errorf("no daytime air quality data for %s", date)
	}

	pm25 := pm25Sum / float64(count)
	ozoneUgM3 := ozoneSum / float64(count)
	ozonePpm := ozoneUgM3 / ozoneUgM3PerPPM
	no2UgM3 := no2Sum / float64(count)

	return &DailyData{
		Date:      date,
		Latitude:  c.lat,
		Longitude: c.lon,
		PM25UgM3:  pm25,
		PM10UgM3:  pm10Sum / float64(count),
		OzoneUgM3: ozoneUgM3,
		OxPpm:     ozonePpm,
		NO2UgM3:   no2UgM3,
		NO2Ppm:    no2UgM3 / nitrogenUgM3PerPPM,
		PM25Level: PM25Level(pm25),
		OxLevel:   OxLevel(ozonePpm),
	}, nil
}

func PM25Level(value float64) Level {
	switch {
	case value >= 36:
		return Level{Label: "Bad", Emoji: "🔴"}
	case value >= 16:
		return Level{Label: "Moderate", Emoji: "🟡"}
	default:
		return Level{Label: "Good", Emoji: "🟢"}
	}
}

func OxLevel(value float64) Level {
	switch {
	case value >= 0.12:
		return Level{Label: "Bad", Emoji: "🔴"}
	case value >= 0.06:
		return Level{Label: "Moderate", Emoji: "🟡"}
	default:
		return Level{Label: "Good", Emoji: "🟢"}
	}
}

func hasIndex[T any](values []T, idx int) bool {
	return idx >= 0 && idx < len(values)
}
