package calibrationcli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/aeroknight786/trialm/internal/hw"
	"github.com/aeroknight786/trialm/internal/paths"
	"math"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/aeroknight786/trialm/internal/calib"
	"github.com/aeroknight786/trialm/internal/setup"
	"github.com/aeroknight786/trialm/internal/setup/calibration"
	"github.com/aeroknight786/trialm/internal/terminal"
	"runtime"
)

func Run(args []string) error {
	fs := flag.NewFlagSet("calibrate", flag.ContinueOnError)
	quick := fs.Bool("quick", false, "the two default hybrid and sliding MoE picks")
	spec := fs.Bool("spec", os.Getenv("TRIALM_CALIBRATE_SPEC") != "0", "include the speculative verify sweep (on by default; --spec=false or TRIALM_CALIBRATE_SPEC=0 skips it)")
	plan := fs.Bool("plan", false, "print the grid and estimated M5 seconds without running a benchmark")
	refit := fs.Bool("refit", false, "rebuild fits from saved raw rows without hardware or llama-bench")
	check := fs.String("check", "", "check an existing local GGUF against calibration (about 1–2 min; no model downloads)")
	status := fs.Bool("status", false, "show calibration coverage and fingerprint status")
	jsonOut := fs.Bool("json", false, "emit NDJSON events")
	pos, e := parseInterspersed(fs, args)
	if e != nil {
		return e
	}
	checkSet := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "check" {
			checkSet = true
		}
	})
	if checkSet && *check == "" {
		return errors.New("--check needs a local GGUF path")
	}
	if len(pos) > 0 {
		return errors.New("usage: trialm calibrate [--quick] [--spec] [--status | --plan | --refit | --check model.gguf] [--json]")
	}
	selected := 0
	for _, v := range []bool{*status, *plan, *refit, *check != ""} {
		if v {
			selected++
		}
	}
	if selected > 1 {
		return errors.New("choose one of --status, --plan, --refit or --check")
	}
	if *quick && *check != "" {
		return errors.New("--quick does not apply to --check")
	}
	home := trialmHome()
	if *plan {
		steps := calib.Plan(*quick, *spec)
		if *jsonOut {
			return json.NewEncoder(os.Stdout).Encode(map[string]any{"v": 1, "type": "calibration_plan", "estimated_seconds": calib.EstimatedSeconds(steps), "estimate_basis": "Apple M5, llama-bench 11960; includes cold synthetic generation; allow about 25% variation", "steps": steps})
		}
		fmt.Fprintf(out(), "Calibration plan: %d steps, about %.0f seconds (%.0f min) on the M5. Allow about 25%% variation.\n", len(steps), calib.EstimatedSeconds(steps), calib.EstimatedSeconds(steps)/60)
		for i, s := range steps {
			fmt.Fprintf(out(), "[%d/%d] %s · about %.0f s\n", i+1, len(steps), s.Description(), s.EstimatedSeconds)
		}
		fmt.Fprintln(out(), "Estimate from the owner's per-step timings, including generation and warm-up. No hard stop; Ctrl-C saves finished rows for resume.")
		return nil
	}
	if *refit {
		c, e := calib.Load(home)
		if e != nil {
			return e
		}
		if e = rebaseline(c); e != nil {
			return e
		}
		if e = c.Save(home); e != nil {
			return e
		}
		if *jsonOut {
			return json.NewEncoder(os.Stdout).Encode(map[string]any{"v": 1, "type": "calibration_refit", "calibration": c})
		}
		fmt.Fprintf(out(), "Rebuilt %d fits from %d saved rows. Coverage: %s\n", len(c.Fits), len(c.Rows), strings.Join(c.Coverage, ", "))
		for _, f := range c.Fits {
			if f.Rejected != "" {
				fmt.Fprintf(out(), "%s/%s depth %d %s: rejected (%s)\n", f.Class, f.Quant, f.Depth, f.Mode, f.Rejected)
			} else {
				fmt.Fprintf(out(), "%s/%s depth %d %s: about %.2gx speed, ±%d%%\n", f.Class, f.Quant, f.Depth, f.Mode, f.Multiplier, f.BandPct)
			}
		}
		return nil
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	engine, engineErr := calib.FindEngine(home)
	if *status {
		return calibrationStatus(ctx, home, engine, engineErr, *jsonOut)
	}
	if runtime.GOOS == "darwin" {
		engine, engineErr = setup.EnsureCalibrationEngine(ctx, home, func(line string) { fmt.Fprintln(os.Stderr, line) })
	}
	if *check != "" {
		if engineErr != nil {
			return engineErr
		}
		return calibrationCheck(ctx, home, engine, *check, *jsonOut)
	}
	var estimates []float64
	for _, step := range calib.Plan(*quick, *spec) {
		estimates = append(estimates, step.EstimatedSeconds)
	}
	total := 0.
	for _, e := range estimates {
		total += e
	}
	// On a terminal: one line that fills in, like triaLM's calibration card.
	pretty := terminal.IsTTY(os.Stderr)
	emit := func(ev calib.Event) {
		if *jsonOut {
			ev.StepEstimates = estimates
			_ = json.NewEncoder(os.Stdout).Encode(ev)
			return
		}
		if pretty && ev.Type != "result" && ev.Type != "interrupted" {
			if ev.Type == "start" {
				fmt.Fprint(os.Stderr, "For more accurate estimates\n\n")
			}
			if ev.Type == "step" && total > 0 {
				done := int(math.Round(24 * max(0, min(1, 1-ev.ETA/total))))
				fmt.Fprintf(os.Stderr, "\r\x1b[2K  %s%s  %d of %d · %s left", strings.Repeat("█", done), strings.Repeat("░", 24-done), ev.Step, ev.Total, calib.Remaining(ev.ETA))
			}
			return
		}
		if pretty {
			fmt.Fprintln(os.Stderr)
		}
		switch ev.Type {
		case "start":
			fmt.Fprintln(os.Stderr, ev.Message)
		case "step":
			fmt.Fprintf(os.Stderr, "[%d/%d] %s · %s · ETA %s\n", ev.Step, ev.Total, ev.Probe, ev.Message, calib.Remaining(ev.ETA))
		case "result":
			if pretty {
				fmt.Fprintln(os.Stderr, "\n✓ Calibrated · estimates now fit this Mac")
				return
			}
			fmt.Fprintf(os.Stderr, "%s. Coverage: %s. MLX is not calibrated.\n", ev.Message, strings.Join(ev.Coverage, ", "))
		case "interrupted":
			if pretty {
				fmt.Fprintln(os.Stderr, "\nPaused · finished steps are kept · trialm calibrate resumes")
				return
			}
			fmt.Fprintln(os.Stderr, "Stopped. Finished rows are in", filepath.Join(home, "calib.json"))
		}
	}
	fail := func(e error) error {
		if *jsonOut {
			emit(calib.Event{Version: 1, Type: "error", Message: e.Error()})
		}
		return e
	}
	if engineErr != nil {
		return fail(engineErr)
	}
	fp, e := calib.MachineFingerprint(ctx, engine)
	if e != nil {
		return fail(e)
	}
	t, _, e := calibrationTarget(ctx)
	if e != nil {
		return fail(e)
	}
	if t.CPUOnly {
		return fail(errors.New("synthetic GPU calibration needs a detected GPU; use trialm detect --probe for CPU bandwidth"))
	}
	baseline, backend := calibration.Baseline(t)
	if hw.MacPower(ctx).LowPower {
		return fail(errors.New("Low Power Mode is on: it slows the GPU, so this calibration would record the wrong speed for this Mac. Turn it off in System Settings → Battery, then run trialm calibrate"))
	}
	_, e = calib.Run(ctx, calib.RunOptions{Backend: backend, Home: home, Engine: engine, Fingerprint: fp, Quick: *quick, Spec: *spec, Baseline: baseline, Event: emit})
	if errors.Is(e, context.Canceled) {
		return fail(fmt.Errorf("calibration interrupted; finished rows kept: %w", e))
	}
	if e != nil {
		return fail(e)
	}
	return nil
}
func calibrationStatus(ctx context.Context, home, engine string, engineErr error, jsonOut bool) error {
	c, e := calib.Load(home)
	state := "missing"
	reason := "Run trialm calibrate for tighter numbers. MLX is not covered."
	if e == nil {
		state = "stale"
		reason = "Machine or llama.cpp build changed. Run trialm calibrate again."
		if engineErr != nil {
			reason = engineErr.Error()
		} else if fp, fe := calib.MachineFingerprint(ctx, engine); fe != nil {
			reason = fe.Error()
		} else if fp.Hash == c.Fingerprint.Hash {
			state = "current"
			reason = "Synthetic fit; real-model coverage is not established. Use trialm calibrate --check with a local GGUF."
			if len(c.Validations) > 0 {
				misses := 0
				for _, v := range c.Validations {
					if !v.InsideBand {
						misses++
					}
				}
				reason = fmt.Sprintf("Synthetic fit; %d real-model validation rows saved (%d outside band). These checks do not establish general coverage.", len(c.Validations), misses)
			}
			if !c.Usable() {
				state = "incomplete"
				reason = "No usable fit yet. Run trialm calibrate --refit to inspect the held-out checks."
			}
		}
	} else if !errors.Is(e, os.ErrNotExist) {
		state = "invalid"
		reason = e.Error()
	}
	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(map[string]any{"v": 1, "type": "calibration_status", "status": state, "message": reason, "calibration": c})
	}
	fmt.Fprintf(out(), "Calibration: %s. %s\n", state, reason)
	if c != nil {
		fmt.Fprintf(out(), "Run: %s · %.0f seconds · %d raw samples · complete: %t\nCoverage: %s\n", c.Started.Format("2006-01-02"), c.WallSeconds, len(c.Rows), c.Complete, strings.Join(c.Coverage, ", "))
		for _, n := range c.Notes {
			fmt.Fprintln(out(), n)
		}
	}
	return nil
}

// MaybeOffer remains the legacy CLI hook; calibration now belongs to setup.
func MaybeOffer([]string) error { return nil }

// rebaseline recomputes the stored baselines for this machine and refits.
func rebaseline(c *calib.Calibration) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	t, _, e := calibrationTarget(ctx)
	if e != nil {
		return e
	}
	f, _ := calibration.Baseline(t)
	c.Rebaseline(f)
	return nil
}

// Upgrade refits a calibration saved by an older speed model, keeping its
// measurements; it runs once, quietly, after an update.
func Upgrade() {
	home, e := paths.Home()
	if e != nil {
		return
	}
	c, e := calib.Load(home)
	if e != nil || !c.NeedsRebaseline {
		return
	}
	if rebaseline(c) == nil {
		_ = c.Save(home)
	}
}
