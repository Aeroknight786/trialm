package core

import (
	"context"
	"fmt"
	"github.com/aeroknight786/trialm/internal/calib"
	"github.com/aeroknight786/trialm/internal/figure"
	"github.com/aeroknight786/trialm/internal/harness"
	"github.com/aeroknight786/trialm/internal/hw"
	"github.com/aeroknight786/trialm/internal/setup"
	"github.com/aeroknight786/trialm/internal/store"
	"github.com/aeroknight786/trialm/internal/verify"
	"math"
	"os"
	"slices"
	"strings"
)

// AgentChoice is shared first-run state; the UI only toggles Checked.
type AgentChoice struct {
	ID, Name          string
	Detected, Checked bool
}

func FirstRunAgents(detected []string) []AgentChoice {
	ids := []string{"claude-code", "codex", "opencode"}
	for _, id := range detected {
		if !slices.Contains(ids, id) {
			ids = append(ids, id)
		}
	}
	ids = append(ids, "generic")
	var out []AgentChoice
	for _, id := range ids {
		name := strings.TrimSuffix(harness.DisplayName(id), " CLI")
		if id == "generic" {
			name = "Your own agent"
		}
		found := slices.Contains(detected, id)
		out = append(out, AgentChoice{id, name, found, found})
	}
	return out
}
func SaveAgents(s store.Store, p *store.Preferences, agents []AgentChoice) error {
	p.PreferredTools = nil
	for _, a := range agents {
		if a.Checked {
			p.PreferredTools = append(p.PreferredTools, a.ID)
		}
	}
	if len(p.PreferredTools) == 0 {
		p.PreferredTools = []string{"generic"}
	}
	p.Tool = p.PreferredTools[0]
	return s.SavePreferences(*p)
}
func CurrentCalibration(ctx context.Context, home string) *calib.Calibration {
	c, e := calib.Load(home)
	if e != nil {
		return nil
	}
	engine, e := calib.FindEngine(home)
	if e != nil {
		return nil
	}
	f, e := calib.MachineFingerprint(ctx, engine)
	if e != nil || f.Hash != c.Fingerprint.Hash {
		return nil
	}
	return c
}
func CalibrationMinutes() int {
	duration := calib.Duration(calib.Plan(false, os.Getenv("TRIALM_CALIBRATE_SPEC") != "0"))
	var lo, hi float64
	if _, e := fmt.Sscanf(duration, "about %f–%f min", &lo, &hi); e == nil {
		return int(math.Round((lo + hi) / 2))
	}
	return 1
}

func TrialAgents(p store.Preferences) []string {
	out := slices.Clone(p.PreferredTools)
	if len(out) == 0 && p.Tool != "" {
		out = append(out, p.Tool)
	}
	if !slices.Contains(out, "generic") {
		out = append(out, "generic")
	}
	return out
}

// ReadyTimes projects the harness's reply length from the report's measured
// rates. These remain calibrated estimates, not observed workflow latencies.
func ReadyTimes(p setup.Plan, r *verify.Report) (reply, first, estReply, estFirst figure.Figure) {
	h, _ := harness.Get(p.Harness)
	sc := harness.OpeningScenario(h.Scenario, p.OS)
	output := float64(sc.OutputTokensPerTurn)
	if output <= 0 {
		output = 300
	}
	speed := 0.
	distance := int(^uint(0) >> 1)
	for depth, v := range r.DecodeTPS {
		if d := abs(depth - p.ContextTokens); d < distance && v > 0 {
			distance, speed = d, v
		}
	}
	short := 0.
	depth := int(^uint(0) >> 1)
	for d, v := range r.DecodeTPS {
		if d < depth && v > 0 {
			depth, short = d, v
		}
	}
	duration := func(read, rate float64) float64 {
		if rate <= 0 {
			return 0
		}
		return read + output/rate
	}
	read := 0.
	if r.PrefillTPS > 0 {
		read = float64(sc.BasePromptTokens) / r.PrefillTPS
	}
	reply = figure.Band(duration(r.FollowUpTTFT, speed), 20, "s", figure.Calibrated, "ready:measured-rates")
	first = figure.Band(duration(read, short), 20, "s", figure.Calibrated, "ready:measured-rates")
	pread := 0.
	if p.Predicted.PrefillTPS > 0 {
		pread = float64(sc.BasePromptTokens) / p.Predicted.PrefillTPS
	}
	estReply = figure.Band(duration(0, p.Predicted.DecodeTPSTarget), p.DecodeErrorPct, "s", figure.Simulated, "setup:prediction")
	estFirst = figure.Band(duration(pread, p.Predicted.DecodeTPS0), p.PrefillErrorPct, "s", figure.Simulated, "setup:prediction")
	return
}

func LowPower(ctx context.Context) bool { return hw.MacPower(ctx).LowPower }
