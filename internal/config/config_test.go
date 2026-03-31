package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_Valid(t *testing.T) {
	dir := t.TempDir()
	envContent := `WHOOP_CLIENT_ID=test-id
WHOOP_CLIENT_SECRET=test-secret
JOURNAL_DIR=/tmp/journal
`
	os.WriteFile(filepath.Join(dir, ".env"), []byte(envContent), 0644)

	// Clear any existing env vars
	os.Unsetenv("WHOOP_CLIENT_ID")
	os.Unsetenv("WHOOP_CLIENT_SECRET")
	os.Unsetenv("JOURNAL_DIR")
	os.Unsetenv("VAULT_JOURNAL_DIR")

	cfg, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ClientID != "test-id" {
		t.Errorf("ClientID = %q, want \"test-id\"", cfg.ClientID)
	}
	if cfg.ClientSecret != "test-secret" {
		t.Errorf("ClientSecret = %q, want \"test-secret\"", cfg.ClientSecret)
	}
	if cfg.JournalDir != "/tmp/journal" {
		t.Errorf("JournalDir = %q, want \"/tmp/journal\"", cfg.JournalDir)
	}
	if cfg.RedirectURI != "http://localhost:8080/callback" {
		t.Errorf("RedirectURI = %q, want default", cfg.RedirectURI)
	}
	if cfg.TokenFile != filepath.Join(dir, "tokens.json") {
		t.Errorf("TokenFile = %q", cfg.TokenFile)
	}
	if !cfg.Weather.Enabled {
		t.Error("Weather.Enabled should default to true")
	}
	if cfg.Weather.Lat != 35.6503 || cfg.Weather.Lon != 139.7225 {
		t.Errorf("unexpected default weather coordinates: %+v", cfg.Weather)
	}
	if !cfg.AirQuality.Enabled {
		t.Error("AirQuality.Enabled should default to true")
	}
	if cfg.AirQuality.Lat != 35.6503 || cfg.AirQuality.Lon != 139.7225 {
		t.Errorf("unexpected default air quality coordinates: %+v", cfg.AirQuality)
	}
}

func TestLoad_MissingRequired(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, ".env"), []byte(""), 0644)

	os.Unsetenv("WHOOP_CLIENT_ID")
	os.Unsetenv("WHOOP_CLIENT_SECRET")
	os.Unsetenv("JOURNAL_DIR")
	os.Unsetenv("VAULT_JOURNAL_DIR")

	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected error for missing required fields")
	}
	if got := err.Error(); got == "" {
		t.Error("error message should not be empty")
	}
}

func TestLoad_EnvOverridesFile(t *testing.T) {
	dir := t.TempDir()
	envContent := `WHOOP_CLIENT_ID=from-file
WHOOP_CLIENT_SECRET=from-file
JOURNAL_DIR=/from/file
`
	os.WriteFile(filepath.Join(dir, ".env"), []byte(envContent), 0644)

	// Set env var — should take precedence
	os.Setenv("WHOOP_CLIENT_ID", "from-env")
	defer os.Unsetenv("WHOOP_CLIENT_ID")
	os.Unsetenv("WHOOP_CLIENT_SECRET")
	os.Unsetenv("JOURNAL_DIR")
	os.Unsetenv("VAULT_JOURNAL_DIR")

	cfg, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ClientID != "from-env" {
		t.Errorf("env var should override .env file, got %q", cfg.ClientID)
	}
}

func TestLoad_NoEnvFile(t *testing.T) {
	dir := t.TempDir()
	// No .env file

	os.Unsetenv("WHOOP_CLIENT_ID")
	os.Unsetenv("WHOOP_CLIENT_SECRET")
	os.Unsetenv("JOURNAL_DIR")
	os.Unsetenv("VAULT_JOURNAL_DIR")

	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected error without .env or env vars")
	}
}

func TestLoad_CommentsAndBlankLines(t *testing.T) {
	dir := t.TempDir()
	envContent := `# This is a comment
WHOOP_CLIENT_ID=test-id

# Another comment
WHOOP_CLIENT_SECRET=test-secret
JOURNAL_DIR=/tmp/journal
`
	os.WriteFile(filepath.Join(dir, ".env"), []byte(envContent), 0644)

	os.Unsetenv("WHOOP_CLIENT_ID")
	os.Unsetenv("WHOOP_CLIENT_SECRET")
	os.Unsetenv("JOURNAL_DIR")
	os.Unsetenv("VAULT_JOURNAL_DIR")

	cfg, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ClientID != "test-id" {
		t.Errorf("failed to parse .env with comments, got %q", cfg.ClientID)
	}
}

func TestLoadWeather(t *testing.T) {
	dir := t.TempDir()
	envContent := `WEATHER_ENABLED=false
WEATHER_LAT=35.1
WEATHER_LON=139.1
`
	os.WriteFile(filepath.Join(dir, ".env"), []byte(envContent), 0644)

	os.Unsetenv("WEATHER_ENABLED")
	os.Unsetenv("WEATHER_LAT")
	os.Unsetenv("WEATHER_LON")

	cfg, err := LoadWeather(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Enabled {
		t.Error("expected weather to be disabled")
	}
	if cfg.Lat != 35.1 || cfg.Lon != 139.1 {
		t.Errorf("unexpected weather config: %+v", cfg)
	}
}

func TestLoadAirQuality(t *testing.T) {
	dir := t.TempDir()
	envContent := `AIRQUALITY_ENABLED=false
WEATHER_LAT=34.7
WEATHER_LON=135.5
`
	os.WriteFile(filepath.Join(dir, ".env"), []byte(envContent), 0644)

	os.Unsetenv("AIRQUALITY_ENABLED")
	os.Unsetenv("WEATHER_LAT")
	os.Unsetenv("WEATHER_LON")

	cfg, err := LoadAirQuality(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Enabled {
		t.Error("expected air quality to be disabled")
	}
	if cfg.Lat != 34.7 || cfg.Lon != 135.5 {
		t.Errorf("unexpected air quality config: %+v", cfg)
	}
}

func TestLoad_DeprecatedVaultJournalDirAlias(t *testing.T) {
	dir := t.TempDir()
	envContent := `WHOOP_CLIENT_ID=test-id
WHOOP_CLIENT_SECRET=test-secret
VAULT_JOURNAL_DIR=/legacy/path
`
	os.WriteFile(filepath.Join(dir, ".env"), []byte(envContent), 0644)

	os.Unsetenv("WHOOP_CLIENT_ID")
	os.Unsetenv("WHOOP_CLIENT_SECRET")
	os.Unsetenv("JOURNAL_DIR")
	os.Unsetenv("VAULT_JOURNAL_DIR")

	cfg, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.JournalDir != "/legacy/path" {
		t.Fatalf("JournalDir = %q, want legacy alias path", cfg.JournalDir)
	}
}
