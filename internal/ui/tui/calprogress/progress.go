// Package calprogress interpolates calibration's per-step estimates.
package calprogress

import (
	"fmt"
	"math"
	"time"

	"github.com/aeroknight786/trialm/internal/calib"
)

type Model struct {
	Fraction  float64
	ETA       string
	Step      int
	Estimates []float64
	started   time.Time
}

func New(steps []calib.Step, now time.Time) Model {
	m := Model{Step: 1, started: now}
	for _, step := range steps {
		m.Estimates = append(m.Estimates, step.EstimatedSeconds)
	}
	m.Tick(now)
	return m
}

func (m *Model) Record(ev calib.Event, now time.Time) {
	if len(ev.StepEstimates) > 0 {
		m.Estimates = ev.StepEstimates
	}
	if ev.Step > m.Step {
		m.Step, m.started = ev.Step, now
	}
	if ev.Total == len(m.Estimates) && (ev.Type == "progress" || ev.Message == "Using finished rows") {
		total := 0.
		for _, estimate := range m.Estimates {
			total += estimate
		}
		if total > 0 {
			m.Fraction = max(m.Fraction, 1-ev.ETA/total)
		}
	}
	if ev.Complete && ev.Type == "result" {
		m.Fraction = 1
	}
	m.Tick(now)
}

func (m *Model) Tick(now time.Time) {
	finished, total := 0., 0.
	for i, estimate := range m.Estimates {
		total += estimate
		if i < m.Step-1 {
			finished += estimate
		}
	}
	if total <= 0 {
		m.ETA = "Estimating time left"
		return
	}
	if m.Step > 0 && m.Step <= len(m.Estimates) {
		finished += min(max(0, now.Sub(m.started).Seconds()), .95*m.Estimates[m.Step-1])
	}
	m.Fraction = max(m.Fraction, min(1, finished/total))
	left := total * (1 - m.Fraction)
	m.ETA = "Under a minute left"
	if left > 60 {
		m.ETA = fmt.Sprintf("About %d min left", int(math.Ceil(left/60)))
	}
}
