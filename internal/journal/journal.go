package journal

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kai/whoop-journal/internal/weather"
	"github.com/kai/whoop-journal/internal/whoop"
)

var jst = time.FixedZone("JST", 9*3600)

// FilePath returns the journal file path for a given date.
func FilePath(journalDir, date string) string {
	return filepath.Join(journalDir, date+".md")
}

// FormatCompact produces a concise bullet-point WHOOP section.
func FormatCompact(d *whoop.DayData) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("## WHOOP Daily - %s\n", d.Date))

	if len(d.Recovery) > 0 && d.Recovery[0].Score != nil {
		s := d.Recovery[0].Score
		b.WriteString(fmt.Sprintf("\n**Recovery**: %s %.0f%% (%s)\n", recoveryEmoji(s.RecoveryScore), s.RecoveryScore, recoveryLabel(s.RecoveryScore)))
		b.WriteString(fmt.Sprintf("- HRV: %.0f ms | RHR: %.0f bpm | SpO2: %.1f%%\n", s.HrvRmssdMilli, s.RestingHeartRate, s.Spo2Percentage))
		if s.SkinTempCelsius > 0 {
			b.WriteString(fmt.Sprintf("- Skin Temp: %.1f°C\n", s.SkinTempCelsius))
		}
	}

	if len(d.Sleep) > 0 && d.Sleep[0].Score != nil {
		s := d.Sleep[0].Score
		st := s.StageSummary
		b.WriteString(fmt.Sprintf("\n**Sleep**: %s in bed\n", msToHM(st.TotalInBedTimeMilli)))
		if s.SleepPerformancePercentage > 0 {
			b.WriteString(fmt.Sprintf("- Performance: %.0f%% | Efficiency: %.0f%%\n", s.SleepPerformancePercentage, s.SleepEfficiencyPercentage))
		}
		b.WriteString(fmt.Sprintf("- REM: %s | Deep: %s | Light: %s\n", msToHM(st.TotalRemSleepTimeMilli), msToHM(st.TotalSlowWaveSleepTimeMilli), msToHM(st.TotalLightSleepTimeMilli)))
		b.WriteString(fmt.Sprintf("- Awake: %s | Disturbances: %d\n", msToHM(st.TotalAwakeTimeMilli), st.DisturbanceCount))
	}

	if len(d.Cycles) > 0 && d.Cycles[0].Score != nil {
		s := d.Cycles[0].Score
		b.WriteString(fmt.Sprintf("\n**Strain**: %.1f | %.0f kJ\n", s.Strain, s.Kilojoule))
		b.WriteString(fmt.Sprintf("- Avg HR: %d | Max HR: %d\n", s.AverageHeartRate, s.MaxHeartRate))
	}

	if len(d.Workouts) > 0 {
		b.WriteString(fmt.Sprintf("\n**Workouts** (%d)\n", len(d.Workouts)))
		for _, w := range d.Workouts {
			name := w.SportName
			if name == "" {
				name = "Unknown"
			}
			if w.Score != nil {
				b.WriteString(fmt.Sprintf("- %s: strain %.1f, Avg HR %d\n", name, w.Score.Strain, w.Score.AverageHeartRate))
			}
		}
	}

	if d.Weather != nil || d.AirQuality != nil {
		recoveryScore, sleepPerformance := extractRecoveryAndSleepScores(d)
		risk := weather.CalculateRiskWithAirQuality(d.Weather, d.AirQuality, recoveryScore, sleepPerformance)

		b.WriteString("\n**Environment**\n")
		if d.Weather != nil {
			b.WriteString(fmt.Sprintf("- Weather: %s %s | %.0f°C (体感%.0f°C) | Humidity %.0f%% | Wind %.1fm/s\n",
				d.Weather.WeatherEmoji, d.Weather.WeatherLabel, d.Weather.TemperatureMaxC,
				d.Weather.ApparentTemperatureC, d.Weather.HumidityPercent, d.Weather.WindSpeedMS))
			b.WriteString(fmt.Sprintf("- Pressure: %.0f hPa (%s)",
				d.Weather.PressureHPa, formatPressureChange(d.Weather.PressureChangeHPa)))
			if d.Weather.PressureAlert != "" {
				b.WriteString(" " + d.Weather.PressureAlert)
			}
			b.WriteString("\n")
		}
		if d.AirQuality != nil {
			b.WriteString(fmt.Sprintf("- Air Quality: PM2.5 %.0fμg/m³ (%s) | Ox %.3fppm (%s)\n",
				d.AirQuality.PM25UgM3, d.AirQuality.PM25Level.Emoji,
				d.AirQuality.OxPpm, d.AirQuality.OxLevel.Emoji))
		}
		if d.Weather != nil {
			b.WriteString(fmt.Sprintf("- UV Index: %.0f (%s)\n", d.Weather.UVIndexMax, weather.UVIndexLabel(d.Weather.UVIndexMax)))
		}
		b.WriteString(fmt.Sprintf("- 気象病リスク: %s %s (%d/100)\n", risk.Emoji, risk.Level, risk.Score))
	}

	return b.String()
}

// FormatDashboard produces a table-based WHOOP section.
func FormatDashboard(d *whoop.DayData) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("## WHOOP Daily - %s\n", d.Date))

	if len(d.Recovery) > 0 && d.Recovery[0].Score != nil {
		s := d.Recovery[0].Score
		b.WriteString(fmt.Sprintf("\n> %s **Recovery %.0f%%** (%s)\n", recoveryEmoji(s.RecoveryScore), s.RecoveryScore, recoveryLabel(s.RecoveryScore)))
	}

	b.WriteString("\n| Metric | Value |\n|--------|-------|\n")
	if len(d.Recovery) > 0 && d.Recovery[0].Score != nil {
		s := d.Recovery[0].Score
		b.WriteString(fmt.Sprintf("| HRV (RMSSD) | %.0f ms |\n", s.HrvRmssdMilli))
		b.WriteString(fmt.Sprintf("| Resting HR | %.0f bpm |\n", s.RestingHeartRate))
		b.WriteString(fmt.Sprintf("| SpO2 | %.1f%% |\n", s.Spo2Percentage))
		if s.SkinTempCelsius > 0 {
			b.WriteString(fmt.Sprintf("| Skin Temp | %.1f°C |\n", s.SkinTempCelsius))
		}
	}
	if len(d.Cycles) > 0 && d.Cycles[0].Score != nil {
		s := d.Cycles[0].Score
		b.WriteString(fmt.Sprintf("| Day Strain | %.1f |\n", s.Strain))
		b.WriteString(fmt.Sprintf("| Calories | %.0f kJ |\n", s.Kilojoule))
	}

	if len(d.Sleep) > 0 && d.Sleep[0].Score != nil {
		s := d.Sleep[0].Score
		st := s.StageSummary
		total := st.TotalInBedTimeMilli
		b.WriteString("\n**Sleep Breakdown**\n\n")
		b.WriteString("| Stage | Duration | % |\n|-------|----------|---|\n")
		for _, row := range []struct {
			label string
			ms    int64
		}{
			{"REM", st.TotalRemSleepTimeMilli},
			{"Deep (SWS)", st.TotalSlowWaveSleepTimeMilli},
			{"Light", st.TotalLightSleepTimeMilli},
			{"Awake", st.TotalAwakeTimeMilli},
		} {
			pct := float64(0)
			if total > 0 {
				pct = float64(row.ms) / float64(total) * 100
			}
			b.WriteString(fmt.Sprintf("| %s | %s | %.0f%% |\n", row.label, msToHM(row.ms), pct))
		}
		b.WriteString(fmt.Sprintf("| **Total in bed** | **%s** | |\n", msToHM(total)))

		var extras []string
		if s.SleepPerformancePercentage > 0 {
			extras = append(extras, fmt.Sprintf("Performance: %.0f%%", s.SleepPerformancePercentage))
		}
		if s.SleepEfficiencyPercentage > 0 {
			extras = append(extras, fmt.Sprintf("Efficiency: %.0f%%", s.SleepEfficiencyPercentage))
		}
		if s.RespiratoryRate > 0 {
			extras = append(extras, fmt.Sprintf("Resp Rate: %.1f", s.RespiratoryRate))
		}
		if len(extras) > 0 {
			b.WriteString(fmt.Sprintf("\n*%s*\n", strings.Join(extras, " | ")))
		}
	}

	if len(d.Workouts) > 0 {
		b.WriteString("\n**Workouts**\n\n")
		b.WriteString("| Activity | Strain | Avg HR | Max HR |\n|----------|--------|--------|--------|\n")
		for _, w := range d.Workouts {
			name := w.SportName
			if name == "" {
				name = "Unknown"
			}
			if w.Score != nil {
				b.WriteString(fmt.Sprintf("| %s | %.1f | %d | %d |\n", name, w.Score.Strain, w.Score.AverageHeartRate, w.Score.MaxHeartRate))
			}
		}
	}

	return b.String()
}

// FormatDetailed produces a full report with all data points.
func FormatDetailed(d *whoop.DayData) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("## WHOOP Daily Report - %s\n", d.Date))

	if len(d.Recovery) > 0 && d.Recovery[0].Score != nil {
		s := d.Recovery[0].Score
		b.WriteString(fmt.Sprintf("\n### Recovery %s %.0f%%\n\n", recoveryEmoji(s.RecoveryScore), s.RecoveryScore))
		b.WriteString(fmt.Sprintf("- **Recovery Score**: %.0f%% (%s)\n", s.RecoveryScore, recoveryLabel(s.RecoveryScore)))
		b.WriteString(fmt.Sprintf("- **HRV (RMSSD)**: %.1f ms\n", s.HrvRmssdMilli))
		b.WriteString(fmt.Sprintf("- **Resting Heart Rate**: %.0f bpm\n", s.RestingHeartRate))
		b.WriteString(fmt.Sprintf("- **SpO2**: %.1f%%\n", s.Spo2Percentage))
		if s.SkinTempCelsius > 0 {
			b.WriteString(fmt.Sprintf("- **Skin Temperature**: %.1f°C\n", s.SkinTempCelsius))
		}
	}

	if len(d.Sleep) > 0 && d.Sleep[0].Score != nil {
		s := d.Sleep[0].Score
		st := s.StageSummary
		b.WriteString("\n### Sleep\n\n")
		b.WriteString(fmt.Sprintf("- **Total in Bed**: %s\n", msToHM(st.TotalInBedTimeMilli)))
		b.WriteString(fmt.Sprintf("  - REM: %s\n", msToHM(st.TotalRemSleepTimeMilli)))
		b.WriteString(fmt.Sprintf("  - Deep (SWS): %s\n", msToHM(st.TotalSlowWaveSleepTimeMilli)))
		b.WriteString(fmt.Sprintf("  - Light: %s\n", msToHM(st.TotalLightSleepTimeMilli)))
		b.WriteString(fmt.Sprintf("  - Awake: %s\n", msToHM(st.TotalAwakeTimeMilli)))
		b.WriteString(fmt.Sprintf("  - Sleep Cycles: %d\n", st.SleepCycleCount))
		b.WriteString(fmt.Sprintf("  - Disturbances: %d\n", st.DisturbanceCount))

		b.WriteString("\n**Quality Metrics**\n")
		if s.SleepPerformancePercentage > 0 {
			b.WriteString(fmt.Sprintf("- Performance: %.0f%%\n", s.SleepPerformancePercentage))
		}
		if s.SleepEfficiencyPercentage > 0 {
			b.WriteString(fmt.Sprintf("- Efficiency: %.0f%%\n", s.SleepEfficiencyPercentage))
		}
		if s.SleepConsistencyPercentage > 0 {
			b.WriteString(fmt.Sprintf("- Consistency: %.0f%%\n", s.SleepConsistencyPercentage))
		}
		if s.RespiratoryRate > 0 {
			b.WriteString(fmt.Sprintf("- Respiratory Rate: %.1f breaths/min\n", s.RespiratoryRate))
		}

		n := s.SleepNeeded
		if n.BaselineMilli > 0 {
			b.WriteString("\n**Sleep Need**\n")
			b.WriteString(fmt.Sprintf("- Baseline: %s\n", msToHM(n.BaselineMilli)))
			b.WriteString(fmt.Sprintf("- From Sleep Debt: %s\n", msToHM(n.NeedFromSleepDebtMilli)))
			b.WriteString(fmt.Sprintf("- From Strain: %s\n", msToHM(n.NeedFromRecentStrainMilli)))
			b.WriteString(fmt.Sprintf("- From Naps: %s\n", msToHM(n.NeedFromRecentNapMilli)))
		}
	}

	if len(d.Cycles) > 0 && d.Cycles[0].Score != nil {
		s := d.Cycles[0].Score
		b.WriteString("\n### Day Strain\n\n")
		b.WriteString(fmt.Sprintf("- **Strain**: %.1f\n", s.Strain))
		b.WriteString(fmt.Sprintf("- **Calories**: %.0f kJ\n", s.Kilojoule))
		b.WriteString(fmt.Sprintf("- **Average Heart Rate**: %d bpm\n", s.AverageHeartRate))
		b.WriteString(fmt.Sprintf("- **Max Heart Rate**: %d bpm\n", s.MaxHeartRate))
	}

	if len(d.Workouts) > 0 {
		b.WriteString("\n### Workouts\n")
		for i, w := range d.Workouts {
			name := w.SportName
			if name == "" {
				name = "Unknown"
			}
			b.WriteString(fmt.Sprintf("\n**%d. %s**\n", i+1, name))
			if w.Score != nil {
				b.WriteString(fmt.Sprintf("- Strain: %.1f\n", w.Score.Strain))
				b.WriteString(fmt.Sprintf("- Avg HR: %d | Max HR: %d\n", w.Score.AverageHeartRate, w.Score.MaxHeartRate))
				if w.Score.DistanceMeter != nil {
					b.WriteString(fmt.Sprintf("- Distance: %.0f m\n", *w.Score.DistanceMeter))
				}
				z := w.Score.ZoneDurations
				b.WriteString(fmt.Sprintf("- HR Zones: Z0=%s, Z1=%s, Z2=%s, Z3=%s, Z4=%s, Z5=%s\n",
					msToHM(z.ZoneZeroMilli), msToHM(z.ZoneOneMilli), msToHM(z.ZoneTwoMilli),
					msToHM(z.ZoneThreeMilli), msToHM(z.ZoneFourMilli), msToHM(z.ZoneFiveMilli)))
			}
		}
	}

	return b.String()
}

// WriteToJournal writes WHOOP data to a journal file.
// If prepend is true, inserts after the frontmatter header. Otherwise appends.
// If update is true, replaces an existing WHOOP section instead of erroring.
func WriteToJournal(journalDir, date, content string, prepend, update bool) error {
	path := FilePath(journalDir, date)

	existing, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read journal: %w", err)
	}

	text := string(existing)

	if strings.Contains(text, "## WHOOP Daily") {
		if update {
			text = removeWhoopSection(text)
		} else {
			return fmt.Errorf("WHOOP section already exists in %s (use --update to replace)", filepath.Base(path))
		}
	}

	if len(existing) == 0 {
		now := time.Now().In(jst)
		dateUnder := strings.ReplaceAll(date, "-", "_")
		header := fmt.Sprintf("---\ntitle: \"%s\"\ntype: journal\ndate: %s\ncreated: %s\ntags: [journal]\n---\n# %s\n",
			dateUnder, date, now.Format("2006-01-02T15:04:05+09:00"), dateUnder)
		text = header
	}

	if prepend {
		headerEnd := findHeaderEnd(text)
		text = text[:headerEnd] + "\n" + content + "\n" + text[headerEnd:]
	} else {
		text = text + "\n" + content + "\n"
	}

	return os.WriteFile(path, []byte(text), 0644)
}

// removeWhoopSection removes an existing ## WHOOP Daily section from text.
// It finds the section start and removes everything until the next ## heading or EOF.
func removeWhoopSection(text string) string {
	lines := strings.Split(text, "\n")
	var out []string
	inWhoop := false
	// Track trailing blank lines before WHOOP section
	trailingBlanks := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "## WHOOP Daily") {
			inWhoop = true
			// Remove trailing blank lines that preceded this section
			for trailingBlanks > 0 && len(out) > 0 && strings.TrimSpace(out[len(out)-1]) == "" {
				out = out[:len(out)-1]
				trailingBlanks--
			}
			continue
		}

		if inWhoop {
			// End of WHOOP section: next ## heading or non-WHOOP content after blank lines
			if strings.HasPrefix(trimmed, "## ") {
				inWhoop = false
				out = append(out, line)
			}
			// Skip all lines within the WHOOP section
			continue
		}

		if trimmed == "" {
			trailingBlanks++
		} else {
			trailingBlanks = 0
		}
		out = append(out, line)
	}

	return strings.Join(out, "\n")
}

func findHeaderEnd(text string) int {
	// Find the end of the first "# " header line
	lines := strings.SplitAfter(text, "\n")
	pos := 0
	inFrontmatter := false
	passedFrontmatter := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "---" {
			if !inFrontmatter {
				inFrontmatter = true
			} else {
				passedFrontmatter = true
			}
		}
		pos += len(line)
		if passedFrontmatter && strings.HasPrefix(trimmed, "# ") {
			return pos
		}
	}
	return len(text)
}

// --- helpers ---

func msToHM(ms int64) string {
	totalMin := ms / 60000
	h := totalMin / 60
	m := totalMin % 60
	if h > 0 {
		return fmt.Sprintf("%dh %02dm", h, m)
	}
	return fmt.Sprintf("%dm", m)
}

func recoveryEmoji(score float64) string {
	if score >= 67 {
		return "🟢"
	}
	if score >= 34 {
		return "🟡"
	}
	return "🔴"
}

func recoveryLabel(score float64) string {
	if score >= 67 {
		return "Green"
	}
	if score >= 34 {
		return "Yellow"
	}
	return "Red"
}

func extractRecoveryAndSleepScores(d *whoop.DayData) (recoveryScore, sleepPerformance float64) {
	if len(d.Recovery) > 0 && d.Recovery[0].Score != nil {
		recoveryScore = d.Recovery[0].Score.RecoveryScore
	}
	if len(d.Sleep) > 0 && d.Sleep[0].Score != nil {
		sleepPerformance = d.Sleep[0].Score.SleepPerformancePercentage
	}
	return recoveryScore, sleepPerformance
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
