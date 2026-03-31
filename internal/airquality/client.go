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
	defaultBaseURL     = "https://soramame.env.go.jp/soramame/api/data_search"
	DefaultStationCode = "13103010" // Closest general station to Hiroo in 2026-03-31 station metadata: 港区高輪
)

type Client struct {
	baseURL     string
	httpClient  *http.Client
	stationCode string
}

type DailyData struct {
	Date        string
	StationCode string
	PM25UgM3    float64
	OxPpm       float64
	SO2Ppm      *float64
	NO2Ppm      *float64
	PM25Level   Level
	OxLevel     Level
}

type Level struct {
	Label string
	Emoji string
}

type apiRecord struct {
	StationCode string `json:"SKT_CD"`
	Date        string `json:"SKT_DATE"`
	Time        string `json:"SKT_TIME"`
	PM25        string `json:"PM2_5"`
	Ox          string `json:"OX"`
	SO2         string `json:"SO2"`
	NO2         string `json:"NO2"`
}

func NewClient(stationCode string) *Client {
	if stationCode == "" {
		stationCode = DefaultStationCode
	}
	return &Client{
		baseURL: defaultBaseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		stationCode: stationCode,
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

	reqURL, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse base url: %w", err)
	}

	q := reqURL.Query()
	q.Set("Start_YM", targetDate.Format("200601"))
	q.Set("End_YM", targetDate.Format("200601"))
	q.Set("TDFKN_CD", stationPrefectureCode(c.stationCode))
	q.Set("SKT_CD", c.stationCode)
	q.Set("REQUEST_DATA", "PM2_5,OX,SO2,NO2")
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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var records []apiRecord
	if err := json.Unmarshal(body, &records); err != nil {
		return nil, err
	}

	targetAPI := targetDate.Format("2006/01/02")
	var pm25Sum, so2Sum, no2Sum float64
	var pm25Count, so2Count, no2Count int
	var oxMax float64
	var oxCount int

	for _, record := range records {
		if record.Date != targetAPI {
			continue
		}

		if value, ok := parseValue(record.PM25); ok {
			pm25Sum += value
			pm25Count++
		}
		if value, ok := parseValue(record.Ox); ok {
			if oxCount == 0 || value > oxMax {
				oxMax = value
			}
			oxCount++
		}
		if value, ok := parseValue(record.SO2); ok {
			so2Sum += value
			so2Count++
		}
		if value, ok := parseValue(record.NO2); ok {
			no2Sum += value
			no2Count++
		}
	}

	if pm25Count == 0 && oxCount == 0 {
		return nil, fmt.Errorf("no air quality data for %s station=%s", date, c.stationCode)
	}

	data := &DailyData{
		Date:        date,
		StationCode: c.stationCode,
		PM25Level:   PM25Level(0),
		OxLevel:     OxLevel(0),
	}
	if pm25Count > 0 {
		data.PM25UgM3 = pm25Sum / float64(pm25Count)
		data.PM25Level = PM25Level(data.PM25UgM3)
	}
	if oxCount > 0 {
		data.OxPpm = oxMax
		data.OxLevel = OxLevel(data.OxPpm)
	}
	if so2Count > 0 {
		v := so2Sum / float64(so2Count)
		data.SO2Ppm = &v
	}
	if no2Count > 0 {
		v := no2Sum / float64(no2Count)
		data.NO2Ppm = &v
	}

	return data, nil
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

func parseValue(raw string) (float64, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "-1" {
		return 0, false
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, false
	}
	return value, true
}

func stationPrefectureCode(stationCode string) string {
	if len(stationCode) >= 2 {
		return stationCode[:2]
	}
	return ""
}
