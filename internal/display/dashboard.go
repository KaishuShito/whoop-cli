package display

import (
	"fmt"
	"strings"

	"github.com/KaishuShito/whoop-cli/internal/weather"
	"github.com/KaishuShito/whoop-cli/internal/whoop"
)

const innerWidth = 43

const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorRed    = "\033[31m"
	colorBold   = "\033[1m"
)

func RenderToday(d *whoop.DayData, noColor bool) string {
	var lines []string
	lines = append(lines, formatLine(colorize(fmt.Sprintf("🏋️ Whoop CLI — %s", d.Date), colorBold, noColor)))
	lines = append(lines, divider())

	recoveryScore, recoveryLine, vitalsLine := recoverySection(d, noColor)
	lines = append(lines, formatLine(recoveryLine))
	lines = append(lines, formatLine(vitalsLine))
	lines = append(lines, divider())

	sleepLine, sleepDetail := sleepSection(d)
	lines = append(lines, formatLine(sleepLine))
	lines = append(lines, formatLine(sleepDetail))
	lines = append(lines, divider())

	strainLine, strainDetail := strainSection(d)
	lines = append(lines, formatLine(strainLine))
	lines = append(lines, formatLine(strainDetail))
	lines = append(lines, divider())

	weatherLine, weatherDetail, pressureLine := weatherSection(d)
	lines = append(lines, formatLine(weatherLine))
	lines = append(lines, formatLine(weatherDetail))
	lines = append(lines, formatLine(pressureLine))
	lines = append(lines, divider())

	airLine, airDetail := airQualitySection(d)
	lines = append(lines, formatLine(airLine))
	lines = append(lines, formatLine(airDetail))
	lines = append(lines, divider())

	sleepPerformance := extractSleepPerformance(d)
	risk := weather.CalculateRiskWithAirQuality(d.Weather, d.AirQuality, recoveryScore, sleepPerformance)
	riskLine, reasonLine, actionLine := riskSection(d, risk, noColor)
	lines = append(lines, formatLine(riskLine))
	lines = append(lines, formatLine(reasonLine))
	lines = append(lines, formatLine(actionLine))

	return topBorder() + "\n" + strings.Join(lines, "\n") + "\n" + bottomBorder()
}

func recoverySection(d *whoop.DayData, noColor bool) (float64, string, string) {
	if len(d.Recovery) == 0 || d.Recovery[0].Score == nil {
		return 0, "RECOVERY        no data", "HRV  -   RHR  -   SpO2  -"
	}
	s := d.Recovery[0].Score
	score := s.RecoveryScore
	return score,
		fmt.Sprintf("RECOVERY        %s", colorRecovery(fmt.Sprintf("%s %.0f%%", recoveryEmoji(score), score), score, noColor)),
		fmt.Sprintf("HRV  %.0fms   RHR  %.0fbpm   SpO2  %.1f%%", s.HrvRmssdMilli, s.RestingHeartRate, s.Spo2Percentage)
}

func sleepSection(d *whoop.DayData) (string, string) {
	if len(d.Sleep) == 0 || d.Sleep[0].Score == nil {
		return "SLEEP           no data", "Deep -  REM -  Light -"
	}
	s := d.Sleep[0].Score
	st := s.StageSummary
	return fmt.Sprintf("SLEEP           %s  (%.0f%% perf)", msToHM(st.TotalInBedTimeMilli), s.SleepPerformancePercentage),
		fmt.Sprintf("Deep %s  REM %s  Light %s", msToHM(st.TotalSlowWaveSleepTimeMilli), msToHM(st.TotalRemSleepTimeMilli), msToHM(st.TotalLightSleepTimeMilli))
}

func strainSection(d *whoop.DayData) (string, string) {
	if len(d.Cycles) == 0 || d.Cycles[0].Score == nil {
		return "STRAIN          no data", "Avg HR -   Max HR -"
	}
	s := d.Cycles[0].Score
	return fmt.Sprintf("STRAIN          %.1f    %.0f kJ", s.Strain, s.Kilojoule),
		fmt.Sprintf("Avg HR %d   Max HR %d", s.AverageHeartRate, s.MaxHeartRate)
}

func weatherSection(d *whoop.DayData) (string, string, string) {
	if d.Weather == nil {
		return "WEATHER         no data", "💨 -   💧 -   ☀️ UV -", "Pressure  -"
	}
	w := d.Weather
	line1 := fmt.Sprintf("WEATHER         %s %s  %.0f°C (feels %.0f°C)", w.WeatherEmoji, w.WeatherLabel, w.TemperatureMaxC, w.ApparentTemperatureC)
	line2 := fmt.Sprintf("💨 %.1fm/s   💧 %.0f%%   ☀️ UV %.0f", w.WindSpeedMS, w.HumidityPercent, w.UVIndexMax)
	line3 := fmt.Sprintf("Pressure  %.0f hPa (%s%.0f)", w.PressureHPa, pressureArrow(w.PressureChangeHPa), abs(w.PressureChangeHPa))
	if w.PressureAlert != "" {
		line3 += " " + strings.TrimPrefix(w.PressureAlert, "⚠️ ")
	}
	return line1, line2, line3
}

func airQualitySection(d *whoop.DayData) (string, string) {
	if d.AirQuality == nil {
		return "AIR QUALITY     no data", "Ozone -  NO₂ -"
	}
	aq := d.AirQuality
	return fmt.Sprintf("AIR QUALITY     PM2.5 %.0fμg/m³  %s", aq.PM25UgM3, aq.PM25Level.Emoji),
		fmt.Sprintf("Ozone %.0fμg/m³  NO₂ %.0fμg/m³", aq.OzoneUgM3, aq.NO2UgM3)
}

func riskSection(d *whoop.DayData, risk weather.RiskScore, noColor bool) (string, string, string) {
	riskValue := fmt.Sprintf("%s %s (%d/100)", risk.Emoji, risk.Level, risk.Score)
	switch risk.Level {
	case "High":
		riskValue = colorize(riskValue, colorRed, noColor)
	case "Moderate":
		riskValue = colorize(riskValue, colorYellow, noColor)
	default:
		riskValue = colorize(riskValue, colorGreen, noColor)
	}
	return "HEALTH RISK     " + riskValue,
		"⚠️ " + riskReason(d),
		"→ " + riskRecommendation(risk.Level)
}

func riskReason(d *whoop.DayData) string {
	var reasons []string
	if d.Weather != nil {
		if d.Weather.PressureChangeHPa <= -5 {
			reasons = append(reasons, "Pressure drop")
		}
		if d.Weather.HumidityPercent >= 80 {
			reasons = append(reasons, "high humidity")
		}
		if weather.IsRainCode(d.Weather.WeatherCode) {
			reasons = append(reasons, "rain")
		}
	}
	if d.AirQuality != nil {
		if d.AirQuality.PM25UgM3 >= 16 {
			reasons = append(reasons, "elevated PM2.5")
		}
		if d.AirQuality.OxPpm >= 0.06 {
			reasons = append(reasons, "elevated ozone")
		}
	}
	if len(d.Recovery) > 0 && d.Recovery[0].Score != nil && d.Recovery[0].Score.RecoveryScore < 34 {
		reasons = append(reasons, "low recovery")
	}
	if len(d.Sleep) > 0 && d.Sleep[0].Score != nil && d.Sleep[0].Score.SleepPerformancePercentage <= 60 {
		reasons = append(reasons, "poor sleep")
	}
	if len(reasons) == 0 {
		return "No major environment flags detected"
	}
	if len(reasons) > 3 {
		reasons = reasons[:3]
	}
	return strings.Join(reasons, " + ")
}

func riskRecommendation(level string) string {
	switch level {
	case "High":
		return "Take it easy today. Stay hydrated."
	case "Moderate":
		return "Pace yourself today. Keep water nearby."
	default:
		return "Normal load looks fine today."
	}
}

func extractSleepPerformance(d *whoop.DayData) float64 {
	if len(d.Sleep) == 0 || d.Sleep[0].Score == nil {
		return 0
	}
	return d.Sleep[0].Score.SleepPerformancePercentage
}

func recoveryEmoji(score float64) string {
	switch {
	case score >= 67:
		return "🟢"
	case score >= 34:
		return "🟡"
	default:
		return "🔴"
	}
}

func colorRecovery(text string, score float64, noColor bool) string {
	switch {
	case score >= 67:
		return colorize(text, colorGreen, noColor)
	case score >= 34:
		return colorize(text, colorYellow, noColor)
	default:
		return colorize(text, colorRed, noColor)
	}
}

func colorize(text, color string, noColor bool) string {
	if noColor {
		return text
	}
	return color + text + colorReset
}

func topBorder() string {
	return "┌" + strings.Repeat("─", innerWidth) + "┐"
}

func bottomBorder() string {
	return "└" + strings.Repeat("─", innerWidth) + "┘"
}

func divider() string {
	return "├" + strings.Repeat("─", innerWidth) + "┤"
}

func formatLine(text string) string {
	return "│ " + padRight(text, innerWidth-1) + "│"
}

func padRight(text string, width int) string {
	visible := []rune(stripANSI(text))
	if len(visible) >= width {
		return text
	}
	return text + strings.Repeat(" ", width-len(visible))
}

func msToHM(ms int64) string {
	totalMinutes := ms / 60000
	hours := totalMinutes / 60
	minutes := totalMinutes % 60
	if hours == 0 {
		return fmt.Sprintf("%dm", minutes)
	}
	return fmt.Sprintf("%dh %02dm", hours, minutes)
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func pressureArrow(v float64) string {
	switch {
	case v < 0:
		return "▼"
	case v > 0:
		return "▲"
	default:
		return "±"
	}
}

func stripANSI(text string) string {
	var b strings.Builder
	inEscape := false
	for i := 0; i < len(text); i++ {
		ch := text[i]
		if inEscape {
			if (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') {
				inEscape = false
			}
			continue
		}
		if ch == 0x1b {
			inEscape = true
			continue
		}
		b.WriteByte(ch)
	}
	return b.String()
}
