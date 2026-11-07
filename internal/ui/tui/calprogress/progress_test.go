package calprogress

import (
	"math"
	"testing"
	"time"

	"github.com/aeroknight786/trialm/internal/calib"
)

func TestProgressBetweenEventsAndOverrun(t *testing.T) {
	now := time.Unix(1000, 0)
	m := New([]calib.Step{{EstimatedSeconds: 100}, {EstimatedSeconds: 200}, {EstimatedSeconds: 100}}, now)
	for _, seconds := range []int{1, 2, 50, 95, 100, 500} {
		m.Tick(now.Add(time.Duration(seconds) * time.Second))
		want := float64(min(seconds, 95)) / 400
		if math.Abs(m.Fraction-want) > 1e-9 {
			t.Fatalf("%ds got %g want %g", seconds, m.Fraction, want)
		}
	}
	// Step 2 starts late. Its progress uses its own estimate and start time.
	now = now.Add(500 * time.Second)
	m.Record(calib.Event{Type: "step", Step: 2, Total: 3}, now)
	if m.Fraction != .25 || m.ETA != "About 5 min left" {
		t.Fatal(m)
	}
	m.Tick(now.Add(30 * time.Second))
	if m.Fraction != .325 {
		t.Fatal(m)
	}
	// Repeated messages for the same step must not restart its clock.
	m.Record(calib.Event{Type: "step", Step: 2, Total: 3}, now.Add(30*time.Second))
	m.Tick(now.Add(40 * time.Second))
	if m.Fraction != .35 {
		t.Fatal(m)
	}
	// Finished measurements can move the estimate forward. A delayed pulse
	// or event can never undo that correction.
	m.Record(calib.Event{Type: "progress", Step: 2, Total: 3, ETA: 100}, now.Add(41*time.Second))
	if m.Fraction != .75 {
		t.Fatal(m)
	}
	m.Tick(now.Add(10 * time.Second))
	if m.Fraction != .75 {
		t.Fatal(m)
	}
	m.Record(calib.Event{Type: "step", Step: 3, Total: 3}, now.Add(50*time.Second))
	m.Tick(now.Add(100 * time.Second))
	if m.Fraction != .875 || m.ETA != "Under a minute left" {
		t.Fatal(m)
	}
	m.Record(calib.Event{Type: "result", Complete: true}, now.Add(101*time.Second))
	if m.Fraction != 1 {
		t.Fatal(m)
	}
}

func TestProgressAcceptsChildPlanAndResumedSteps(t *testing.T) {
	now := time.Unix(1000, 0)
	m := New(nil, now)
	if m.ETA != "Estimating time left" {
		t.Fatal(m)
	}
	m.Record(calib.Event{Type: "start", StepEstimates: []float64{10, 30, 60}}, now)
	m.Record(calib.Event{Type: "step", Step: 1, Total: 3, Message: "Using finished rows", ETA: 90}, now)
	if math.Abs(m.Fraction-.1) > 1e-9 {
		t.Fatal(m)
	}
	m.Record(calib.Event{Type: "step", Step: 2, Total: 3}, now)
	m.Tick(now.Add(15 * time.Second))
	if m.Fraction != .25 {
		t.Fatal(m)
	}
}
