package ops_intelligence

import (
	"time"
)

type Predictor interface {
	Predict(h []AppointmentHistory) NoShowRisk
}

type defaultPredictor struct{}

func NewPredictor() Predictor {
	return &defaultPredictor{}
}

func (p *defaultPredictor) Predict(h []AppointmentHistory) NoShowRisk {
	var risk Score
	risk.BaseScore = 0.1 // Baseline risk

	if len(h) == 0 {
		return NoShowRisk{
			RiskScore: 0.2,
			RiskLevel: RiskLow,
			Factors:   []string{"New patient, no previous history"},
		}
	}

	noShowCount := 0
	lastStatus := ""
	var lastTime time.Time
	factors := []string{}

	for i, appt := range h {
		if appt.Status == "no_show" {
			noShowCount++
		}
		if i == 0 {
			lastStatus = appt.Status
			lastTime = appt.StartTime
		}
	}

	// Factor 1: Previous no-shows
	if noShowCount > 0 {
		risk.BaseScore += float64(noShowCount) * 0.2
		factors = append(factors, "Patient has history of missed visits")
	}

	// Factor 2: Last attendance status
	if lastStatus == "no_show" {
		risk.BaseScore += 0.3
		factors = append(factors, "Last appointment was a no-show")
	}

	// Factor 3: Time gap since last visit
	timeGap := time.Since(lastTime)
	if timeGap > 180*24*time.Hour { // More than 6 months
		risk.BaseScore += 0.1
		factors = append(factors, "Long gap since last visit")
	}

	// Cap the score
	if risk.BaseScore > 1.0 {
		risk.BaseScore = 0.95
	}

	level := RiskLow
	if risk.BaseScore > 0.7 {
		level = RiskHigh
	} else if risk.BaseScore > 0.4 {
		level = RiskMedium
	}

	return NoShowRisk{
		RiskScore: risk.BaseScore,
		RiskLevel: level,
		Factors:   factors,
	}
}

type Score struct {
	BaseScore float64
}
