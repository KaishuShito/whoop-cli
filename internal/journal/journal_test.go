package journal

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kai/whoop-journal/internal/whoop"
)

// --- test fixtures ---

func fullDayData() *whoop.DayData {
	return &whoop.DayData{
		Date: "2026-03-16",
		Cycles: []whoop.Cycle{{
			ID: 1, Score: &whoop.CycleScore{
				Strain: 9.4, Kilojoule: 8963, AverageHeartRate: 68, MaxHeartRate: 177,
			},
		}},
		Recovery: []whoop.Recovery{{
			Score: &whoop.RecoveryScore{
				RecoveryScore: 54, RestingHeartRate: 53, HrvRmssdMilli: 24.8,
				Spo2Percentage: 95.8, SkinTempCelsius: 34.4,
			},
		}},
		Sleep: []whoop.Sleep{{
			Score: &whoop.SleepScore{
				StageSummary: whoop.StageSummary{
					TotalInBedTimeMilli:         27561950,
					TotalAwakeTimeMilli:         2554100,
					TotalLightSleepTimeMilli:    17284590,
					TotalSlowWaveSleepTimeMilli: 6370210,
					TotalRemSleepTimeMilli:      1353050,
					SleepCycleCount:             2,
					DisturbanceCount:            18,
				},
				SleepNeeded: whoop.SleepNeeded{
					BaselineMilli:             28383913,
					NeedFromSleepDebtMilli:    4078438,
					NeedFromRecentStrainMilli: 292966,
				},
				RespiratoryRate:            14.9,
				SleepPerformancePercentage: 82,
				SleepEfficiencyPercentage:  91,
				SleepConsistencyPercentage: 76,
			},
		}},
		Workouts: []whoop.Workout{{
			SportName: "Running",
			Score: &whoop.WorkoutScore{
				Strain: 6.5, AverageHeartRate: 116, MaxHeartRate: 177,
				ZoneDurations: whoop.ZoneDurations{
					ZoneZeroMilli: 694000, ZoneOneMilli: 941990,
				},
			},
		}},
	}
}

func emptyDayData() *whoop.DayData {
	return &whoop.DayData{Date: "2026-03-16"}
}

func recoveryOnlyData() *whoop.DayData {
	return &whoop.DayData{
		Date: "2026-03-16",
		Recovery: []whoop.Recovery{{
			Score: &whoop.RecoveryScore{RecoveryScore: 80, HrvRmssdMilli: 50, RestingHeartRate: 48, Spo2Percentage: 97},
		}},
	}
}

func nilScoreData() *whoop.DayData {
	return &whoop.DayData{
		Date:     "2026-03-16",
		Cycles:   []whoop.Cycle{{ID: 1, Score: nil}},
		Recovery: []whoop.Recovery{{Score: nil}},
		Sleep:    []whoop.Sleep{{Score: nil}},
		Workouts: []whoop.Workout{{SportName: "Running", Score: nil}},
	}
}

// --- msToHM tests ---

func TestMsToHM(t *testing.T) {
	tests := []struct {
		ms   int64
		want string
	}{
		{0, "0m"},
		{60000, "1m"},
		{3600000, "1h 00m"},
		{5400000, "1h 30m"},
		{27561950, "7h 39m"},
		{1353050, "22m"},
		{120000, "2m"},
	}
	for _, tt := range tests {
		got := msToHM(tt.ms)
		if got != tt.want {
			t.Errorf("msToHM(%d) = %q, want %q", tt.ms, got, tt.want)
		}
	}
}

// --- recovery helpers ---

func TestRecoveryEmoji(t *testing.T) {
	tests := []struct {
		score float64
		emoji string
		label string
	}{
		{80, "🟢", "Green"},
		{67, "🟢", "Green"},
		{54, "🟡", "Yellow"},
		{34, "🟡", "Yellow"},
		{20, "🔴", "Red"},
		{0, "🔴", "Red"},
	}
	for _, tt := range tests {
		if got := recoveryEmoji(tt.score); got != tt.emoji {
			t.Errorf("recoveryEmoji(%.0f) = %q, want %q", tt.score, got, tt.emoji)
		}
		if got := recoveryLabel(tt.score); got != tt.label {
			t.Errorf("recoveryLabel(%.0f) = %q, want %q", tt.score, got, tt.label)
		}
	}
}

// --- format tests ---

func TestFormatCompact_FullData(t *testing.T) {
	out := FormatCompact(fullDayData())
	mustContain(t, out, "## WHOOP Daily - 2026-03-16")
	mustContain(t, out, "🟡 54%")
	mustContain(t, out, "HRV: 25 ms")
	mustContain(t, out, "7h 39m in bed")
	mustContain(t, out, "**Strain**: 9.4")
	mustContain(t, out, "Running: strain 6.5")
}

func TestFormatCompact_Empty(t *testing.T) {
	out := FormatCompact(emptyDayData())
	if !strings.HasPrefix(out, "## WHOOP Daily") {
		t.Error("empty data should still have header")
	}
	mustNotContain(t, out, "Recovery")
	mustNotContain(t, out, "Sleep")
	mustNotContain(t, out, "Strain")
}

func TestFormatCompact_NilScores(t *testing.T) {
	// Should not panic with nil Score pointers
	out := FormatCompact(nilScoreData())
	mustContain(t, out, "## WHOOP Daily")
	mustNotContain(t, out, "Recovery")
}

func TestFormatDashboard_FullData(t *testing.T) {
	out := FormatDashboard(fullDayData())
	mustContain(t, out, "| Metric | Value |")
	mustContain(t, out, "| HRV (RMSSD) | 25 ms |")
	mustContain(t, out, "| Stage | Duration | % |")
	mustContain(t, out, "| Running |")
}

func TestFormatDetailed_FullData(t *testing.T) {
	out := FormatDetailed(fullDayData())
	mustContain(t, out, "### Recovery")
	mustContain(t, out, "### Sleep")
	mustContain(t, out, "**Sleep Need**")
	mustContain(t, out, "### Day Strain")
	mustContain(t, out, "### Workouts")
	mustContain(t, out, "HR Zones:")
}

func TestFormatCompact_RecoveryOnly(t *testing.T) {
	out := FormatCompact(recoveryOnlyData())
	mustContain(t, out, "🟢 80%")
	mustNotContain(t, out, "Sleep")
	mustNotContain(t, out, "Strain")
}

func TestFormatCompact_NoSkinTemp(t *testing.T) {
	d := recoveryOnlyData()
	d.Recovery[0].Score.SkinTempCelsius = 0
	out := FormatCompact(d)
	mustNotContain(t, out, "Skin Temp")
}

func TestFormatCompact_EmptySportName(t *testing.T) {
	d := fullDayData()
	d.Workouts[0].SportName = ""
	out := FormatCompact(d)
	mustContain(t, out, "Unknown: strain")
}

// --- findHeaderEnd tests ---

func TestFindHeaderEnd(t *testing.T) {
	tests := []struct {
		name string
		text string
		want int
	}{
		{
			"standard journal",
			"---\ntitle: \"test\"\n---\n# 2026_03_16\n\n## ログ\n",
			len("---\ntitle: \"test\"\n---\n# 2026_03_16\n"),
		},
		{
			"no frontmatter",
			"# Title\n\nContent\n",
			len("# Title\n\nContent\n"),
		},
		{
			"empty",
			"",
			0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findHeaderEnd(tt.text)
			if got != tt.want {
				t.Errorf("findHeaderEnd() = %d, want %d\ntext=%q\nprefix=%q", got, tt.want, tt.text, tt.text[:got])
			}
		})
	}
}

// --- WriteToJournal tests ---

func TestWriteToJournal_NewFile(t *testing.T) {
	dir := t.TempDir()
	err := WriteToJournal(dir, "2026-03-20", "## WHOOP Daily - 2026-03-20\ntest\n", false, false)
	if err != nil {
		t.Fatal(err)
	}

	content, _ := os.ReadFile(filepath.Join(dir, "2026-03-20.md"))
	text := string(content)
	mustContain(t, text, "title: \"2026_03_20\"")
	mustContain(t, text, "## WHOOP Daily - 2026-03-20")
}

func TestWriteToJournal_Append(t *testing.T) {
	dir := t.TempDir()
	existing := "---\ntitle: \"2026_03_20\"\n---\n# 2026_03_20\n\n## ログ\nSome content\n"
	os.WriteFile(filepath.Join(dir, "2026-03-20.md"), []byte(existing), 0644)

	err := WriteToJournal(dir, "2026-03-20", "## WHOOP Daily - 2026-03-20\ntest\n", false, false)
	if err != nil {
		t.Fatal(err)
	}

	content, _ := os.ReadFile(filepath.Join(dir, "2026-03-20.md"))
	text := string(content)
	whoopIdx := strings.Index(text, "## WHOOP Daily")
	logIdx := strings.Index(text, "## ログ")
	if whoopIdx <= logIdx {
		t.Error("append mode: WHOOP should come after existing content")
	}
}

func TestWriteToJournal_Prepend(t *testing.T) {
	dir := t.TempDir()
	existing := "---\ntitle: \"2026_03_20\"\n---\n# 2026_03_20\n\n## ログ\nSome content\n"
	os.WriteFile(filepath.Join(dir, "2026-03-20.md"), []byte(existing), 0644)

	err := WriteToJournal(dir, "2026-03-20", "## WHOOP Daily - 2026-03-20\ntest\n", true, false)
	if err != nil {
		t.Fatal(err)
	}

	content, _ := os.ReadFile(filepath.Join(dir, "2026-03-20.md"))
	text := string(content)
	whoopIdx := strings.Index(text, "## WHOOP Daily")
	logIdx := strings.Index(text, "## ログ")
	if whoopIdx >= logIdx {
		t.Errorf("prepend mode: WHOOP (pos %d) should come before ログ (pos %d)", whoopIdx, logIdx)
	}
}

func TestWriteToJournal_DuplicateProtection(t *testing.T) {
	dir := t.TempDir()
	existing := "---\ntitle: \"2026_03_20\"\n---\n# 2026_03_20\n\n## WHOOP Daily - 2026-03-20\nalready here\n"
	os.WriteFile(filepath.Join(dir, "2026-03-20.md"), []byte(existing), 0644)

	err := WriteToJournal(dir, "2026-03-20", "## WHOOP Daily - 2026-03-20\nnew\n", false, false)
	if err == nil {
		t.Error("expected error for duplicate WHOOP section")
	}
	mustContain(t, err.Error(), "already exists")
}

func TestWriteToJournal_Update(t *testing.T) {
	dir := t.TempDir()
	existing := "---\ntitle: \"2026_03_20\"\n---\n# 2026_03_20\n\n## WHOOP Daily - 2026-03-20\n\n**Recovery**: old data\n- old line\n\n## ログ\nSome content\n"
	os.WriteFile(filepath.Join(dir, "2026-03-20.md"), []byte(existing), 0644)

	err := WriteToJournal(dir, "2026-03-20", "## WHOOP Daily - 2026-03-20\n\n**Recovery**: NEW data\n", true, true)
	if err != nil {
		t.Fatal(err)
	}

	content, _ := os.ReadFile(filepath.Join(dir, "2026-03-20.md"))
	text := string(content)
	mustContain(t, text, "NEW data")
	mustNotContain(t, text, "old data")
	mustContain(t, text, "## ログ")
	// WHOOP should still be before ログ
	whoopIdx := strings.Index(text, "## WHOOP Daily")
	logIdx := strings.Index(text, "## ログ")
	if whoopIdx >= logIdx {
		t.Errorf("update+prepend: WHOOP (pos %d) should come before ログ (pos %d)", whoopIdx, logIdx)
	}
}

func TestRemoveWhoopSection(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		want   string
	}{
		{
			"middle section",
			"# Title\n\n## WHOOP Daily - 2026-03-20\n\nRecovery data\n- line1\n\n## ログ\ncontent\n",
			"# Title\n## ログ\ncontent\n",
		},
		{
			"end section",
			"# Title\n\n## ログ\ncontent\n\n## WHOOP Daily - 2026-03-20\n\nRecovery data\n",
			"# Title\n\n## ログ\ncontent",
		},
		{
			"no whoop section",
			"# Title\n\n## ログ\ncontent\n",
			"# Title\n\n## ログ\ncontent\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := removeWhoopSection(tt.input)
			if got != tt.want {
				t.Errorf("removeWhoopSection:\ngot:  %q\nwant: %q", got, tt.want)
			}
		})
	}
}

// --- DayData.HasData tests ---

func TestHasData(t *testing.T) {
	if emptyDayData().HasData() {
		t.Error("empty data should return false")
	}
	if !fullDayData().HasData() {
		t.Error("full data should return true")
	}
	if !recoveryOnlyData().HasData() {
		t.Error("recovery-only data should return true")
	}
}

// --- helpers ---

func mustContain(t *testing.T, text, substr string) {
	t.Helper()
	if !strings.Contains(text, substr) {
		t.Errorf("expected output to contain %q, got:\n%s", substr, text[:min(len(text), 300)])
	}
}

func mustNotContain(t *testing.T, text, substr string) {
	t.Helper()
	if strings.Contains(text, substr) {
		t.Errorf("expected output NOT to contain %q", substr)
	}
}
