package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	JournalDir   string
	TokenFile    string
}

func Load(projectDir string) (Config, error) {
	loadEnvFile(filepath.Join(projectDir, ".env"))

	cfg := Config{
		ClientID:     getEnvDefault("WHOOP_CLIENT_ID", ""),
		ClientSecret: getEnvDefault("WHOOP_CLIENT_SECRET", ""),
		RedirectURI:  getEnvDefault("WHOOP_REDIRECT_URI", "http://localhost:8080/callback"),
		JournalDir:   getEnvDefault("VAULT_JOURNAL_DIR", ""),
		TokenFile:    filepath.Join(projectDir, "tokens.json"),
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
