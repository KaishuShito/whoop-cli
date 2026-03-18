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
VAULT_JOURNAL_DIR=/tmp/journal
`
	os.WriteFile(filepath.Join(dir, ".env"), []byte(envContent), 0644)

	// Clear any existing env vars
	os.Unsetenv("WHOOP_CLIENT_ID")
	os.Unsetenv("WHOOP_CLIENT_SECRET")
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
}

func TestLoad_MissingRequired(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, ".env"), []byte(""), 0644)

	os.Unsetenv("WHOOP_CLIENT_ID")
	os.Unsetenv("WHOOP_CLIENT_SECRET")
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
VAULT_JOURNAL_DIR=/from/file
`
	os.WriteFile(filepath.Join(dir, ".env"), []byte(envContent), 0644)

	// Set env var — should take precedence
	os.Setenv("WHOOP_CLIENT_ID", "from-env")
	defer os.Unsetenv("WHOOP_CLIENT_ID")
	os.Unsetenv("WHOOP_CLIENT_SECRET")
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
VAULT_JOURNAL_DIR=/tmp/journal
`
	os.WriteFile(filepath.Join(dir, ".env"), []byte(envContent), 0644)

	os.Unsetenv("WHOOP_CLIENT_ID")
	os.Unsetenv("WHOOP_CLIENT_SECRET")
	os.Unsetenv("VAULT_JOURNAL_DIR")

	cfg, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ClientID != "test-id" {
		t.Errorf("failed to parse .env with comments, got %q", cfg.ClientID)
	}
}
