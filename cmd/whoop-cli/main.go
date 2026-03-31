package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/KaishuShito/whoop-cli/internal/airquality"
	"github.com/KaishuShito/whoop-cli/internal/auth"
	"github.com/KaishuShito/whoop-cli/internal/config"
	"github.com/KaishuShito/whoop-cli/internal/display"
	"github.com/KaishuShito/whoop-cli/internal/journal"
	"github.com/KaishuShito/whoop-cli/internal/weather"
	"github.com/KaishuShito/whoop-cli/internal/whoop"
)

var (
	version = "dev"
	commit  = "unknown"
)

func main() {
	os.Exit(run())
}

func run() int {
	projectDir := resolveProjectDir()

	if len(os.Args) < 2 {
		return runToday(projectDir, nil)
	}
	if strings.HasPrefix(os.Args[1], "-") {
		return runToday(projectDir, os.Args[1:])
	}

	switch os.Args[1] {
	case "today":
		return runToday(projectDir, os.Args[2:])
	case "setup":
		return runSetup(projectDir, os.Args[2:])
	case "version":
		return runVersion()
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
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		printUsage()
		return 1
	}
}

func printUsage() {
	fmt.Printf(`whoop-cli — WHOOP data → journal context

Usage:
  whoop-cli                 Show today's terminal dashboard
  whoop-cli today [flags]   Show terminal dashboard or JSON
  whoop-cli setup [flags]   Interactive setup + OAuth authorization
  whoop-cli fetch [flags]   Fetch WHOOP data and write/preview markdown
  whoop-cli weather [flags] Fetch weather data only
  whoop-cli airquality      Fetch air quality data only
  whoop-cli status          Show token status and config
  whoop-cli auth            Run OAuth2 flow to get access tokens
  whoop-cli version         Show version information

Today flags:
  --date YYYY-MM-DD         Date to fetch (default: today)
  --json                    Output raw JSON
  --no-color                Disable ANSI colors

Setup flags:
  --non-interactive         Create .env from environment variables and run auth

Fetch flags:
  --date YYYY-MM-DD         Date to fetch (default: today)
  --days N                  Number of days to fetch (default: 1)
  --format FORMAT           Output format: compact, dashboard, detailed (default: compact)
  --write                   Write to journal file (default: preview only)
  --update                  Replace existing WHOOP section with fresh data
  --prepend                 Insert at top of journal (after header) instead of append
  --json                    Output raw API data as JSON

Weather flags:
  --date YYYY-MM-DD         Date to fetch (default: today)
  --json                    Output raw JSON

Airquality flags:
  --date YYYY-MM-DD         Date to fetch (default: today)
  --json                    Output raw JSON
`)
}

func runVersion() int {
	fmt.Printf("whoop-cli v%s (commit %s)\n", version, commit)
	return 0
}

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

func runToday(projectDir string, args []string) int {
	fs := flag.NewFlagSet("today", flag.ExitOnError)
	dateFlag := fs.String("date", "", "Date to fetch (YYYY-MM-DD)")
	jsonFlag := fs.Bool("json", false, "Output raw JSON")
	noColorFlag := fs.Bool("no-color", false, "Disable ANSI colors")
	fs.Parse(args)

	cfg, err := config.Load(projectDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		return 1
	}

	tokens, err := auth.LoadTokens(cfg.TokenFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "no tokens found — run 'whoop-cli setup' or 'whoop-cli auth' first\n")
		return 1
	}

	targetDate, err := resolveTargetDate(*dateFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid date: %v\n", err)
		return 1
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	data, _, err := fetchEnrichedDay(ctx, cfg, targetDate.Format("2006-01-02"), tokens)
	if err != nil {
		fmt.Fprintf(os.Stderr, "today failed: %v\n", err)
		return 1
	}

	if *jsonFlag {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(data)
		return 0
	}

	noColor := *noColorFlag || !isTTY(os.Stdout)
	fmt.Println(display.RenderToday(data, noColor))
	return 0
}

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
	if *writeFlag && cfg.JournalDir == "" {
		fmt.Fprintln(os.Stderr, "journal directory is not configured. Set JOURNAL_DIR (VAULT_JOURNAL_DIR still works as a deprecated alias) or omit --write for stdout preview.")
		return 1
	}

	tokens, err := auth.LoadTokens(cfg.TokenFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "no tokens found — run 'whoop-cli auth' first\n")
		return 1
	}

	endDate, err := resolveTargetDate(*dateFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid date: %v\n", err)
		return 1
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
	default:
		fmt.Fprintf(os.Stderr, "unknown format: %s\n", *formatFlag)
		return 1
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	exitCode := 0
	for _, date := range dates {
		data, newTokens, err := fetchEnrichedDay(ctx, cfg, date, tokens)
		if err != nil {
			fmt.Fprintf(os.Stderr, "fetch %s failed: %v\n", date, err)
			exitCode = 1
			continue
		}
		tokens = newTokens

		if *jsonFlag {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			_ = enc.Encode(data)
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
			continue
		}

		fmt.Println(output)
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

	targetDate, err := resolveTargetDate(*dateFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid date: %v\n", err)
		return 1
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

	targetDate, err := resolveTargetDate(*dateFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid date: %v\n", err)
		return 1
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := airquality.NewClient(aqCfg.Lat, aqCfg.Lon)
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

	fmt.Printf("Coordinates: %.4f, %.4f\n", data.Latitude, data.Longitude)
	fmt.Printf("PM2.5: %.0fμg/m³ (%s %s)\n", data.PM25UgM3, data.PM25Level.Emoji, data.PM25Level.Label)
	fmt.Printf("PM10: %.0fμg/m³\n", data.PM10UgM3)
	fmt.Printf("Ozone: %.0fμg/m³ / %.3fppm (%s %s)\n", data.OzoneUgM3, data.OxPpm, data.OxLevel.Emoji, data.OxLevel.Label)
	fmt.Printf("NO2: %.0fμg/m³\n", data.NO2UgM3)
	return 0
}

func runStatus(projectDir string) int {
	cfg, err := config.Load(projectDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		return 1
	}

	tokens, err := auth.LoadTokens(cfg.TokenFile)
	if err != nil {
		fmt.Println("tokens=none")
		fmt.Println("hint: run 'whoop-cli setup' or 'whoop-cli auth'")
		return 0
	}

	if tokens.SavedAt == 0 {
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
	fmt.Printf("client_id=%s...%s\n", cfg.ClientID[:min(8, len(cfg.ClientID))], cfg.ClientID[max(0, len(cfg.ClientID)-4):])
	fmt.Printf("version=v%s commit=%s\n", version, commit)

	return 0
}

func runSetup(projectDir string, args []string) int {
	fs := flag.NewFlagSet("setup", flag.ExitOnError)
	nonInteractive := fs.Bool("non-interactive", false, "Use environment variables instead of prompts")
	fs.Parse(args)

	existing := readEnvFile(filepath.Join(projectDir, ".env"))
	values, err := collectSetupValues(existing, *nonInteractive)
	if err != nil {
		fmt.Fprintf(os.Stderr, "setup error: %v\n", err)
		return 1
	}

	if err := backupEnvFile(filepath.Join(projectDir, ".env")); err != nil {
		fmt.Fprintf(os.Stderr, "backup .env failed: %v\n", err)
		return 1
	}
	if err := writeEnvFile(filepath.Join(projectDir, ".env"), values); err != nil {
		fmt.Fprintf(os.Stderr, "write .env failed: %v\n", err)
		return 1
	}

	if !*nonInteractive {
		fmt.Println()
		fmt.Println("Step 4/4: Authorize with WHOOP")
		fmt.Println("  → Opening browser for OAuth authorization...")
	}

	lat, _ := configValueFloat(values.Lat)
	lon, _ := configValueFloat(values.Lon)
	cfg := config.Config{
		ClientID:     values.ClientID,
		ClientSecret: values.ClientSecret,
		RedirectURI:  "http://localhost:8080/callback",
		JournalDir:   values.JournalDir,
		TokenFile:    filepath.Join(projectDir, "tokens.json"),
		Weather:      config.WeatherConfig{Enabled: true, Lat: lat, Lon: lon},
		AirQuality:   config.AirQualityConfig{Enabled: true, Lat: lat, Lon: lon},
	}

	if _, err := auth.RunOAuthFlow(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "authorization failed after writing .env: %v\n", err)
		return 1
	}

	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━")
	fmt.Println("✅ Setup complete! Your .env has been created.")
	fmt.Println()
	fmt.Println("Try it:")
	fmt.Println("  whoop-cli          # terminal dashboard")
	fmt.Println("  whoop-cli fetch    # preview markdown output")
	fmt.Println("  whoop-cli fetch --write  # write to your journal")
	fmt.Println("  whoop-cli weather  # check weather & air quality")
	fmt.Println("  whoop-cli status   # verify tokens")
	return 0
}

type setupValues struct {
	ClientID     string
	ClientSecret string
	JournalDir   string
	Lat          string
	Lon          string
}

func collectSetupValues(existing map[string]string, nonInteractive bool) (setupValues, error) {
	defJournal := expandPath("~/journal")
	values := setupValues{
		ClientID:     firstNonEmpty(os.Getenv("WHOOP_CLIENT_ID"), existing["WHOOP_CLIENT_ID"]),
		ClientSecret: firstNonEmpty(os.Getenv("WHOOP_CLIENT_SECRET"), existing["WHOOP_CLIENT_SECRET"]),
		JournalDir:   expandPath(firstNonEmpty(os.Getenv("JOURNAL_DIR"), os.Getenv("VAULT_JOURNAL_DIR"), existing["JOURNAL_DIR"], existing["VAULT_JOURNAL_DIR"])),
		Lat:          firstNonEmpty(os.Getenv("WEATHER_LAT"), existing["WEATHER_LAT"], "35.6503"),
		Lon:          firstNonEmpty(os.Getenv("WEATHER_LON"), existing["WEATHER_LON"], "139.7225"),
	}

	if nonInteractive {
		if values.ClientID == "" || values.ClientSecret == "" {
			return setupValues{}, fmt.Errorf("WHOOP_CLIENT_ID and WHOOP_CLIENT_SECRET are required in --non-interactive mode")
		}
		return values, nil
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Println("🏋️ Whoop CLI Setup")
	fmt.Println("━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	fmt.Println("Step 1/4: WHOOP Developer App")
	fmt.Println("  You need a WHOOP developer app to access your data.")
	fmt.Println("  → Opening https://developer-dashboard.whoop.com ...")
	auth.OpenBrowser("https://developer-dashboard.whoop.com")
	fmt.Println()
	fmt.Println("  Create an app with these settings:")
	fmt.Println("    Redirect URI: http://localhost:8080/callback")
	fmt.Println("    Scopes: read:recovery read:sleep read:cycles read:workout read:profile read:body_measurement offline")
	fmt.Println()
	values.ClientID = prompt(reader, "  Paste your Client ID", values.ClientID)
	values.ClientSecret = prompt(reader, "  Paste your Client Secret", values.ClientSecret)
	if values.ClientID == "" || values.ClientSecret == "" {
		return setupValues{}, fmt.Errorf("client ID and client secret are required")
	}

	fmt.Println()
	fmt.Println("Step 2/4: Journal Directory")
	fmt.Println("  Where should daily entries be written?")
	fmt.Println("  Leave blank for stdout-only mode.")
	values.JournalDir = promptAllowBlank(reader, "  Path ["+defJournal+"]", values.JournalDir)
	values.JournalDir = expandPath(values.JournalDir)

	fmt.Println()
	fmt.Println("Step 3/4: Location (for weather & air quality)")
	fmt.Println("  Enter your coordinates for local weather data.")
	fmt.Println("  Tip: Search your address at https://www.latlong.net/")
	fmt.Println("  → Opening https://www.latlong.net/ ...")
	auth.OpenBrowser("https://www.latlong.net/")
	fmt.Println()
	values.Lat = prompt(reader, "  Latitude", values.Lat)
	values.Lon = prompt(reader, "  Longitude", values.Lon)
	if _, err := configValueFloat(values.Lat); err != nil {
		return setupValues{}, fmt.Errorf("invalid latitude: %w", err)
	}
	if _, err := configValueFloat(values.Lon); err != nil {
		return setupValues{}, fmt.Errorf("invalid longitude: %w", err)
	}

	return values, nil
}

func fetchEnrichedDay(ctx context.Context, cfg config.Config, date string, tokens *auth.Tokens) (*whoop.DayData, *auth.Tokens, error) {
	client := whoop.NewClient(tokens.AccessToken)
	data, err := client.FetchDay(ctx, date)
	if err != nil {
		newTokens, refreshErr := auth.RefreshTokens(cfg, tokens)
		if refreshErr != nil {
			return nil, tokens, fmt.Errorf("%v (refresh also failed: %v)", err, refreshErr)
		}
		tokens = newTokens
		client.SetAccessToken(tokens.AccessToken)
		data, err = client.FetchDay(ctx, date)
		if err != nil {
			return nil, tokens, err
		}
	}

	if cfg.Weather.Enabled {
		weatherClient := weather.NewClient(cfg.Weather.Lat, cfg.Weather.Lon)
		weatherData, weatherErr := weatherClient.FetchDay(ctx, date)
		if weatherErr != nil {
			fmt.Fprintf(os.Stderr, "date=%s weather_status=degraded error=%v\n", date, weatherErr)
		} else {
			data.Weather = weatherData
		}
	}
	if cfg.AirQuality.Enabled {
		airClient := airquality.NewClient(cfg.AirQuality.Lat, cfg.AirQuality.Lon)
		airData, airErr := airClient.FetchDay(ctx, date)
		if airErr != nil {
			fmt.Fprintf(os.Stderr, "date=%s airquality_status=degraded error=%v\n", date, airErr)
		} else {
			data.AirQuality = airData
		}
	}

	return data, tokens, nil
}

func resolveProjectDir() string {
	exe, _ := os.Executable()
	projectDir := filepath.Dir(exe)
	if _, err := os.Stat(filepath.Join(projectDir, ".env")); err != nil {
		projectDir, _ = os.Getwd()
	}
	if _, err := os.Stat(filepath.Join(projectDir, ".env")); err != nil {
		for dir := projectDir; dir != "/"; dir = filepath.Dir(dir) {
			if _, err := os.Stat(filepath.Join(dir, ".env")); err == nil {
				projectDir = dir
				break
			}
		}
	}
	return projectDir
}

func resolveTargetDate(raw string) (time.Time, error) {
	jst := time.FixedZone("JST", 9*3600)
	if raw == "" {
		return time.Now().In(jst), nil
	}
	return time.ParseInLocation("2006-01-02", raw, jst)
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

func isTTY(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func readEnvFile(path string) map[string]string {
	values := map[string]string{}
	data, err := os.ReadFile(path)
	if err != nil {
		return values
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		values[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	return values
}

func backupEnvFile(path string) error {
	if _, err := os.Stat(path); err != nil {
		return nil
	}
	backup := fmt.Sprintf("%s.bak-%s", path, time.Now().Format("20060102150405"))
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return os.WriteFile(backup, data, 0600)
}

func writeEnvFile(path string, values setupValues) error {
	var b strings.Builder
	b.WriteString("WHOOP_CLIENT_ID=" + values.ClientID + "\n")
	b.WriteString("WHOOP_CLIENT_SECRET=" + values.ClientSecret + "\n")
	b.WriteString("WHOOP_REDIRECT_URI=http://localhost:8080/callback\n\n")
	if values.JournalDir != "" {
		b.WriteString("# Optional: markdown journal directory for --write mode.\n")
		b.WriteString("JOURNAL_DIR=" + values.JournalDir + "\n\n")
	}
	b.WriteString("# Optional environment context via Open-Meteo.\n")
	b.WriteString("WEATHER_ENABLED=true\n")
	b.WriteString("WEATHER_LAT=" + values.Lat + "\n")
	b.WriteString("WEATHER_LON=" + values.Lon + "\n\n")
	b.WriteString("# Optional air quality context via Open-Meteo Air Quality.\n")
	b.WriteString("AIRQUALITY_ENABLED=true\n")
	return os.WriteFile(path, []byte(b.String()), 0600)
}

func prompt(reader *bufio.Reader, label, current string) string {
	fmt.Printf("%s: ", label)
	text, _ := reader.ReadString('\n')
	text = strings.TrimSpace(text)
	if text == "" {
		return current
	}
	return text
}

func promptAllowBlank(reader *bufio.Reader, label, current string) string {
	fmt.Printf("%s: ", label)
	text, _ := reader.ReadString('\n')
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	return text
}

func expandPath(path string) string {
	if path == "" {
		return ""
	}
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			if path == "~" {
				return home
			}
			return filepath.Join(home, strings.TrimPrefix(path, "~/"))
		}
	}
	return path
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func configValueFloat(raw string) (float64, error) {
	return strconv.ParseFloat(strings.TrimSpace(raw), 64)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
