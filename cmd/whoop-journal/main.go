package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/kai/whoop-journal/internal/auth"
	"github.com/kai/whoop-journal/internal/config"
	"github.com/kai/whoop-journal/internal/journal"
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
  status                  Show token status and config

Fetch flags:
  --date YYYY-MM-DD       Date to fetch (default: yesterday)
  --days N                Number of days to fetch (default: 1)
  --format FORMAT         Output format: compact, dashboard, detailed (default: compact)
  --write                 Write to journal file (default: preview only)
  --prepend               Insert at top of journal (after header) instead of append
  --json                  Output raw API data as JSON`)
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
		endDate = time.Now().In(jst).Add(-24 * time.Hour)
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
			if err := journal.WriteToJournal(cfg.JournalDir, date, output, *prependFlag); err != nil {
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
