package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/kai/whoop-journal/internal/airquality"
	"github.com/kai/whoop-journal/internal/auth"
	"github.com/kai/whoop-journal/internal/config"
	"github.com/kai/whoop-journal/internal/journal"
	"github.com/kai/whoop-journal/internal/weather"
	"github.com/kai/whoop-journal/internal/whoop"
)

func main() {
	os.Exit(run())
}

func run() int {
	if len(os.Args) < 2 {
		printUsage()
		return 1
	}

	// Determine project dir (where .env and tokens.json live)
	exe, _ := os.Executable()
	projectDir := filepath.Dir(exe)
	// If running via `go run`, use working directory
	if _, err := os.Stat(filepath.Join(projectDir, ".env")); err != nil {
		projectDir, _ = os.Getwd()
	}
	// Also check parent directories for .env
	if _, err := os.Stat(filepath.Join(projectDir, ".env")); err != nil {
		// Try the directory of the source file
		for dir := projectDir; dir != "/"; dir = filepath.Dir(dir) {
			if _, err := os.Stat(filepath.Join(dir, ".env")); err == nil {
				projectDir = dir
				break
			}
		}
	}

	cmd := os.Args[1]
	switch cmd {
	case "auth":
		return runAuth(projectDir)
	case "fetch":
		return runFetch(projectDir, os.Args[2:])
	case "weather":
		return runWeather(projectDir, os.Args[2:])
	case "airquality":
		return runAirQuality(projectDir, os.Args[2:])
	case "status":
		return runStatus(projectDir)
	case "help", "--help", "-h":
		printUsage()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		printUsage()
		return 1
	}
}

func printUsage() {
	fmt.Println(`whoop-journal — WHOOP data → Obsidian Journal

Commands:
  auth                    Run OAuth2 flow to get access tokens
  fetch [flags]           Fetch WHOOP data and write to journal
  weather [flags]         Fetch weather/environment data only
  airquality [flags]      Fetch air quality data only
  status                  Show token status and config

Fetch flags:
  --date YYYY-MM-DD       Date to fetch (default: today)
  --days N                Number of days to fetch (default: 1)
  --format FORMAT         Output format: compact, dashboard, detailed (default: compact)
  --write                 Write to journal file (default: preview only)
  --update                Replace existing WHOOP section with fresh data
  --prepend               Insert at top of journal (after header) instead of append
  --json                  Output raw API data as JSON

Weather flags:
  --date YYYY-MM-DD       Date to fetch (default: today)
  --json                  Output weather data as JSON

Airquality flags:
  --date YYYY-MM-DD       Date to fetch (default: today)
  --json                  Output air quality data as JSON`)
}

// --- auth ---

func runAuth(projectDir string) int {
	cfg, err := config.Load(projectDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		return 1
	}

	tokens, err := auth.RunOAuthFlow(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "auth failed: %v\n", err)
		return 1
	}

	fmt.Printf("command=auth status=ok expires_in=%d\n", tokens.ExpiresIn)
	return 0
}

// --- fetch ---

func runFetch(projectDir string, args []string) int {
	fs := flag.NewFlagSet("fetch", flag.ExitOnError)
	dateFlag := fs.String("date", "", "Date to fetch (YYYY-MM-DD)")
	daysFlag := fs.Int("days", 1, "Number of days")
	formatFlag := fs.String("format", "compact", "Format: compact, dashboard, detailed")
	writeFlag := fs.Bool("write", false, "Write to journal")
	updateFlag := fs.Bool("update", false, "Replace existing WHOOP section with fresh data")
	prependFlag := fs.Bool("prepend", false, "Insert at top of journal (after header)")
	jsonFlag := fs.Bool("json", false, "Output raw JSON")
	fs.Parse(args)

	cfg, err := config.Load(projectDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		return 1
	}

	tokens, err := auth.LoadTokens(cfg.TokenFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "no tokens found — run 'whoop-journal auth' first\n")
		return 1
	}

	client := whoop.NewClient(tokens.AccessToken)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Determine dates
	jst := time.FixedZone("JST", 9*3600)
	var endDate time.Time
	if *dateFlag != "" {
		endDate, err = time.ParseInLocation("2006-01-02", *dateFlag, jst)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid date: %v\n", err)
			return 1
		}
	} else {
		endDate = time.Now().In(jst)
	}

	dates := make([]string, *daysFlag)
	for i := range dates {
		d := endDate.Add(-time.Duration(*daysFlag-1-i) * 24 * time.Hour)
		dates[i] = d.Format("2006-01-02")
	}

	formatFn := journal.FormatCompact
	switch *formatFlag {
	case "dashboard":
		formatFn = journal.FormatDashboard
	case "detailed":
		formatFn = journal.FormatDetailed
	case "compact":
		// default
	default:
		fmt.Fprintf(os.Stderr, "unknown format: %s\n", *formatFlag)
		return 1
	}

	exitCode := 0
	weatherClient := weather.NewClient(cfg.Weather.Lat, cfg.Weather.Lon)
	airQualityClient := airquality.NewClient(cfg.AirQuality.StationCode)
	for _, date := range dates {
		data, err := client.FetchDay(ctx, date)
		if err != nil {
			// Try token refresh
			newTokens, refreshErr := auth.RefreshTokens(cfg, tokens)
			if refreshErr != nil {
				fmt.Fprintf(os.Stderr, "fetch %s failed: %v (refresh also failed: %v)\n", date, err, refreshErr)
				exitCode = 1
				continue
			}
			tokens = newTokens
			client.SetAccessToken(tokens.AccessToken)
			data, err = client.FetchDay(ctx, date)
			if err != nil {
				fmt.Fprintf(os.Stderr, "fetch %s failed after refresh: %v\n", date, err)
				exitCode = 1
				continue
			}
		}

		if cfg.Weather.Enabled {
			weatherData, weatherErr := weatherClient.FetchDay(ctx, date)
			if weatherErr != nil {
				fmt.Fprintf(os.Stderr, "date=%s weather_status=degraded error=%v\n", date, weatherErr)
			} else {
				data.Weather = weatherData
			}
		}
		if cfg.AirQuality.Enabled {
			airQualityData, airQualityErr := airQualityClient.FetchDay(ctx, date)
			if airQualityErr != nil {
				fmt.Fprintf(os.Stderr, "date=%s airquality_status=degraded error=%v\n", date, airQualityErr)
			} else {
				data.AirQuality = airQualityData
			}
		}

		if *jsonFlag {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			enc.Encode(data)
			continue
		}

		if !data.HasData() {
			fmt.Fprintf(os.Stderr, "date=%s status=no_data\n", date)
			continue
		}

		output := formatFn(data)

		if *writeFlag {
			if err := journal.WriteToJournal(cfg.JournalDir, date, output, *prependFlag, *updateFlag); err != nil {
				fmt.Fprintf(os.Stderr, "date=%s status=error error=%v\n", date, err)
				exitCode = 1
				continue
			}
			fmt.Printf("date=%s status=written file=%s\n", date, journal.FilePath(cfg.JournalDir, date))
		} else {
			fmt.Println(output)
		}
	}

	return exitCode
}

func runWeather(projectDir string, args []string) int {
	fs := flag.NewFlagSet("weather", flag.ExitOnError)
	dateFlag := fs.String("date", "", "Date to fetch (YYYY-MM-DD)")
	jsonFlag := fs.Bool("json", false, "Output raw JSON")
	fs.Parse(args)

	weatherCfg, err := config.LoadWeather(projectDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		return 1
	}
	if !weatherCfg.Enabled {
		fmt.Fprintln(os.Stderr, "weather is disabled (WEATHER_ENABLED=false)")
		return 1
	}

	jst := time.FixedZone("JST", 9*3600)
	targetDate := time.Now().In(jst)
	if *dateFlag != "" {
		targetDate, err = time.ParseInLocation("2006-01-02", *dateFlag, jst)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid date: %v\n", err)
			return 1
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := weather.NewClient(weatherCfg.Lat, weatherCfg.Lon)
	data, err := client.FetchDay(ctx, targetDate.Format("2006-01-02"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "weather fetch failed: %v\n", err)
		return 1
	}

	if *jsonFlag {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(data)
		return 0
	}

	fmt.Printf("Weather: %s %s | %.0f°C (%.0f〜%.0f°C) | Humidity %.0f%%\n",
		data.WeatherEmoji, data.WeatherLabel, data.TemperatureMaxC, data.TemperatureMinC, data.TemperatureMaxC, data.HumidityPercent)
	fmt.Printf("Apparent: %.1f°C | Wind: %.1fm/s\n", data.ApparentTemperatureC, data.WindSpeedMS)
	fmt.Printf("Pressure: %.0f hPa (%s)", data.PressureHPa, formatPressureChange(data.PressureChangeHPa))
	if data.PressureAlert != "" {
		fmt.Printf(" %s", data.PressureAlert)
	}
	fmt.Println()
	fmt.Printf("UV Index: %.0f (%s)\n", data.UVIndexMax, weather.UVIndexLabel(data.UVIndexMax))
	return 0
}

func runAirQuality(projectDir string, args []string) int {
	fs := flag.NewFlagSet("airquality", flag.ExitOnError)
	dateFlag := fs.String("date", "", "Date to fetch (YYYY-MM-DD)")
	jsonFlag := fs.Bool("json", false, "Output raw JSON")
	fs.Parse(args)

	aqCfg, err := config.LoadAirQuality(projectDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		return 1
	}
	if !aqCfg.Enabled {
		fmt.Fprintln(os.Stderr, "air quality is disabled (AIRQUALITY_ENABLED=false)")
		return 1
	}

	jst := time.FixedZone("JST", 9*3600)
	targetDate := time.Now().In(jst)
	if *dateFlag != "" {
		targetDate, err = time.ParseInLocation("2006-01-02", *dateFlag, jst)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid date: %v\n", err)
			return 1
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := airquality.NewClient(aqCfg.StationCode)
	data, err := client.FetchDay(ctx, targetDate.Format("2006-01-02"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "air quality fetch failed: %v\n", err)
		return 1
	}

	if *jsonFlag {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(data)
		return 0
	}

	fmt.Printf("Station: %s\n", data.StationCode)
	fmt.Printf("PM2.5: %.0fμg/m³ (%s %s)\n", data.PM25UgM3, data.PM25Level.Emoji, data.PM25Level.Label)
	fmt.Printf("Ox: %.3fppm (%s %s)\n", data.OxPpm, data.OxLevel.Emoji, data.OxLevel.Label)
	if data.SO2Ppm != nil {
		fmt.Printf("SO2: %.3fppm\n", *data.SO2Ppm)
	}
	if data.NO2Ppm != nil {
		fmt.Printf("NO2: %.3fppm\n", *data.NO2Ppm)
	}
	return 0
}

func formatPressureChange(change float64) string {
	switch {
	case change < 0:
		return fmt.Sprintf("▼%.0f hPa", -change)
	case change > 0:
		return fmt.Sprintf("▲%.0f hPa", change)
	default:
		return "±0 hPa"
	}
}

// --- status ---

func runStatus(projectDir string) int {
	cfg, err := config.Load(projectDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		return 1
	}

	tokens, err := auth.LoadTokens(cfg.TokenFile)
	if err != nil {
		fmt.Println("tokens=none")
		fmt.Println("hint: run 'whoop-journal auth'")
		return 0
	}

	if tokens.SavedAt == 0 {
		// Legacy tokens without saved_at — estimate from file mtime
		if info, err := os.Stat(cfg.TokenFile); err == nil {
			tokens.SavedAt = info.ModTime().Unix()
		}
	}
	savedAt := time.Unix(tokens.SavedAt, 0)
	expiresAt := savedAt.Add(time.Duration(tokens.ExpiresIn) * time.Second)
	expired := time.Now().After(expiresAt)

	fmt.Printf("tokens=present saved_at=%s expires_at=%s expired=%v\n",
		savedAt.Format(time.RFC3339), expiresAt.Format(time.RFC3339), expired)
	fmt.Printf("journal_dir=%s\n", cfg.JournalDir)
	fmt.Printf("client_id=%s...%s\n", cfg.ClientID[:8], cfg.ClientID[len(cfg.ClientID)-4:])

	return 0
}
