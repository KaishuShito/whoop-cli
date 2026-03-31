package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	JournalDir   string
	TokenFile    string
	Weather      WeatherConfig
	AirQuality   AirQualityConfig
}

type WeatherConfig struct {
	Enabled bool
	Lat     float64
	Lon     float64
}

type AirQualityConfig struct {
	Enabled     bool
	StationCode string
}

func Load(projectDir string) (Config, error) {
	loadEnvFile(filepath.Join(projectDir, ".env"))

	weatherCfg, err := loadWeatherConfig(projectDir)
	if err != nil {
		return Config{}, err
	}
	airQualityCfg, err := loadAirQualityConfig(projectDir)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		ClientID:     getEnvDefault("WHOOP_CLIENT_ID", ""),
		ClientSecret: getEnvDefault("WHOOP_CLIENT_SECRET", ""),
		RedirectURI:  getEnvDefault("WHOOP_REDIRECT_URI", "http://localhost:8080/callback"),
		JournalDir:   getEnvDefault("VAULT_JOURNAL_DIR", ""),
		TokenFile:    filepath.Join(projectDir, "tokens.json"),
		Weather:      weatherCfg,
		AirQuality:   airQualityCfg,
	}

	var errs []string
	if cfg.ClientID == "" {
		errs = append(errs, "WHOOP_CLIENT_ID is required")
	}
	if cfg.ClientSecret == "" {
		errs = append(errs, "WHOOP_CLIENT_SECRET is required")
	}
	if cfg.JournalDir == "" {
		errs = append(errs, "VAULT_JOURNAL_DIR is required")
	}
	if len(errs) > 0 {
		return cfg, fmt.Errorf("config validation failed: %s", strings.Join(errs, "; "))
	}
	return cfg, nil
}

func LoadWeather(projectDir string) (WeatherConfig, error) {
	loadEnvFile(filepath.Join(projectDir, ".env"))
	return loadWeatherConfig(projectDir)
}

func LoadAirQuality(projectDir string) (AirQualityConfig, error) {
	loadEnvFile(filepath.Join(projectDir, ".env"))
	return loadAirQualityConfig(projectDir)
}

func loadWeatherConfig(projectDir string) (WeatherConfig, error) {
	lat, err := getEnvFloatDefault("WEATHER_LAT", 35.6503)
	if err != nil {
		return WeatherConfig{}, fmt.Errorf("invalid WEATHER_LAT: %w", err)
	}
	lon, err := getEnvFloatDefault("WEATHER_LON", 139.7225)
	if err != nil {
		return WeatherConfig{}, fmt.Errorf("invalid WEATHER_LON: %w", err)
	}
	enabled, err := getEnvBoolDefault("WEATHER_ENABLED", true)
	if err != nil {
		return WeatherConfig{}, fmt.Errorf("invalid WEATHER_ENABLED: %w", err)
	}

	return WeatherConfig{
		Enabled: enabled,
		Lat:     lat,
		Lon:     lon,
	}, nil
}

func loadAirQualityConfig(projectDir string) (AirQualityConfig, error) {
	enabled, err := getEnvBoolDefault("AIRQUALITY_ENABLED", true)
	if err != nil {
		return AirQualityConfig{}, fmt.Errorf("invalid AIRQUALITY_ENABLED: %w", err)
	}

	return AirQualityConfig{
		Enabled:     enabled,
		StationCode: getEnvDefault("AIRQUALITY_STATION_CODE", "13103010"),
	}, nil
}

func loadEnvFile(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		if os.Getenv(k) == "" {
			os.Setenv(k, v)
		}
	}
}

func getEnvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvFloatDefault(key string, def float64) (float64, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0, err
	}
	return f, nil
}

func getEnvBoolDefault(key string, def bool) (bool, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return false, err
	}
	return b, nil
}
