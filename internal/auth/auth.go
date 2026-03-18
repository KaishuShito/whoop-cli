package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/kai/whoop-journal/internal/config"
)

const (
	authURL  = "https://api.prod.whoop.com/oauth/oauth2/auth"
	tokenURL = "https://api.prod.whoop.com/oauth/oauth2/token"
	scopes   = "read:recovery read:sleep read:cycles read:workout read:profile read:body_measurement offline"
)

type Tokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
	SavedAt      int64  `json:"saved_at"`
}

func RunOAuthFlow(cfg config.Config) (*Tokens, error) {
	state := randomState()
	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	// Parse port from redirect URI
	u, err := url.Parse(cfg.RedirectURI)
	if err != nil {
		return nil, fmt.Errorf("invalid redirect URI: %w", err)
	}
	port := u.Port()
	if port == "" {
		port = "8080"
	}

	// Start local callback server
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state {
			http.Error(w, "state mismatch", http.StatusBadRequest)
			errCh <- fmt.Errorf("state mismatch")
			return
		}
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "no code", http.StatusBadRequest)
			errCh <- fmt.Errorf("no authorization code in callback")
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<html><body><h2>WHOOP Auth Success!</h2><p>You can close this tab.</p></body></html>`)
		codeCh <- code
	})

	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return nil, fmt.Errorf("listen on port %s: %w", port, err)
	}
	server := &http.Server{Handler: mux}
	go server.Serve(ln)
	defer server.Shutdown(context.Background())

	// Build auth URL and open browser
	authParams := url.Values{
		"client_id":     {cfg.ClientID},
		"redirect_uri":  {cfg.RedirectURI},
		"response_type": {"code"},
		"scope":         {scopes},
		"state":         {state},
	}
	authFullURL := authURL + "?" + authParams.Encode()

	fmt.Println("Opening browser for WHOOP authorization...")
	openBrowser(authFullURL)
	fmt.Printf("If browser didn't open, visit:\n%s\n\n", authFullURL)
	fmt.Println("Waiting for callback...")

	// Wait for code or error
	var code string
	select {
	case code = <-codeCh:
	case err := <-errCh:
		return nil, err
	case <-time.After(120 * time.Second):
		return nil, fmt.Errorf("timeout waiting for OAuth callback (120s)")
	}

	// Exchange code for tokens
	fmt.Println("Exchanging code for tokens...")
	tokens, err := exchangeCode(cfg, code)
	if err != nil {
		return nil, err
	}
	tokens.SavedAt = time.Now().Unix()

	if err := SaveTokens(cfg.TokenFile, tokens); err != nil {
		return nil, fmt.Errorf("save tokens: %w", err)
	}

	return tokens, nil
}

func exchangeCode(cfg config.Config, code string) (*Tokens, error) {
	data := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"client_id":     {cfg.ClientID},
		"client_secret": {cfg.ClientSecret},
		"redirect_uri":  {cfg.RedirectURI},
	}

	resp, err := http.Post(tokenURL, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("token exchange request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("token exchange failed (%d): %s", resp.StatusCode, string(body))
	}

	var tokens Tokens
	if err := json.Unmarshal(body, &tokens); err != nil {
		return nil, fmt.Errorf("parse token response: %w", err)
	}
	return &tokens, nil
}

func RefreshTokens(cfg config.Config, tokens *Tokens) (*Tokens, error) {
	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {tokens.RefreshToken},
		"client_id":     {cfg.ClientID},
		"client_secret": {cfg.ClientSecret},
		"scope":         {"offline"},
	}

	resp, err := http.Post(tokenURL, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("refresh request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("refresh failed (%d): %s", resp.StatusCode, string(body))
	}

	var newTokens Tokens
	if err := json.Unmarshal(body, &newTokens); err != nil {
		return nil, fmt.Errorf("parse refresh response: %w", err)
	}
	newTokens.SavedAt = time.Now().Unix()

	if err := SaveTokens(cfg.TokenFile, &newTokens); err != nil {
		return nil, fmt.Errorf("save refreshed tokens: %w", err)
	}
	return &newTokens, nil
}

func LoadTokens(path string) (*Tokens, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var tokens Tokens
	if err := json.Unmarshal(data, &tokens); err != nil {
		return nil, err
	}
	return &tokens, nil
}

func SaveTokens(path string, tokens *Tokens) error {
	data, err := json.MarshalIndent(tokens, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func randomState() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	default:
		return
	}
	cmd.Start()
}
