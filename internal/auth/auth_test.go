package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoadTokens(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tokens.json")

	tokens := &Tokens{
		AccessToken:  "test-access",
		RefreshToken: "test-refresh",
		ExpiresIn:    3600,
		TokenType:    "Bearer",
		Scope:        "read:recovery",
		SavedAt:      1710000000,
	}

	if err := SaveTokens(path, tokens); err != nil {
		t.Fatal(err)
	}

	// Verify file permissions
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("token file permissions = %o, want 0600", info.Mode().Perm())
	}

	loaded, err := LoadTokens(path)
	if err != nil {
		t.Fatal(err)
	}

	if loaded.AccessToken != tokens.AccessToken {
		t.Errorf("AccessToken = %q, want %q", loaded.AccessToken, tokens.AccessToken)
	}
	if loaded.RefreshToken != tokens.RefreshToken {
		t.Errorf("RefreshToken = %q, want %q", loaded.RefreshToken, tokens.RefreshToken)
	}
	if loaded.SavedAt != tokens.SavedAt {
		t.Errorf("SavedAt = %d, want %d", loaded.SavedAt, tokens.SavedAt)
	}
}

func TestLoadTokens_Missing(t *testing.T) {
	_, err := LoadTokens("/nonexistent/tokens.json")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestLoadTokens_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tokens.json")
	os.WriteFile(path, []byte("not json"), 0644)

	_, err := LoadTokens(path)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestTokensJSONRoundtrip(t *testing.T) {
	// Verify JSON field names match WHOOP API response format
	raw := `{"access_token":"abc","refresh_token":"def","expires_in":3600,"token_type":"Bearer","scope":"offline","saved_at":12345}`
	var tokens Tokens
	if err := json.Unmarshal([]byte(raw), &tokens); err != nil {
		t.Fatal(err)
	}
	if tokens.AccessToken != "abc" {
		t.Errorf("AccessToken = %q, want \"abc\"", tokens.AccessToken)
	}
	if tokens.ExpiresIn != 3600 {
		t.Errorf("ExpiresIn = %d, want 3600", tokens.ExpiresIn)
	}
}
