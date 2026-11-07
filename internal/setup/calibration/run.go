// Package calibration adapts synthetic calibration to setup events.
package calibration

import (
	"context"
	"github.com/aeroknight786/trialm/internal/calib"
	"github.com/aeroknight786/trialm/internal/engine"
	"github.com/aeroknight786/trialm/internal/hw"
	"github.com/aeroknight786/trialm/internal/predict"
	"github.com/aeroknight786/trialm/internal/setup/download"
	"time"
)

func Run(ctx context.Context, root, goos, selectedBackend string, emit func(string, *download.Progress, string)) error {
	log := func(line string) { emit("setup_log", nil, line) }
	if hw.MacPower(ctx).LowPower {
		log("Calibration skipped · Low Power Mode is on")
		return nil
	}
	binary := engine.SharedBinary(root, "llama-bench", goos)
	if calib.Current(ctx, root, binary) {
		log("Calibration is current for this Mac and engine")
		return nil
	}
	sys, err := hw.Detect(ctx)
	if err != nil {
		return err
	}
	if selectedBackend == "cpu" {
		sys.GPUs = nil
	}
	baseline, backend := Baseline(predict.TargetFromSystem(sys, 0))
	fp, err := calib.MachineFingerprint(ctx, binary)
	if err != nil {
		return err
	}
	_, err = calib.Run(ctx, calib.RunOptions{Home: root, Engine: binary, Fingerprint: fp, Backend: backend, Baseline: baseline, CPUOnly: selectedBackend == "cpu", Spec: true, Event: func(ev calib.Event) {
		progress := download.Progress{Done: int64(max(0, ev.Step-1)), Total: int64(ev.Total), ETA: time.Duration(ev.ETA * float64(time.Second))}
		if ev.Complete {
			progress.Done = progress.Total
		}
		emit("setup_progress", &progress, "")
		log(ev.Message)
	}})
	return err
}

// Baseline predicts a synthetic probe's seconds per token on target t with
// the current predictor; calibration multipliers are relative to it.
func Baseline(t predict.Target) (func(calib.Probe, calib.BenchRow) float64, string) {
	profile, _ := engine.Get("llamacpp")
	timings := map[string]*predict.Timing{}
	backend, _ := profile.ChooseBackend(t.OS, t.Arch, t.HWBackends)
	return func(p calib.Probe, b calib.BenchRow) float64 {
		tm := timings[p.ID()]
		if tm == nil {
			_, _, m, q := p.Layout()
			target := t
			target.Synthetic = true
			r := predict.Predict(predict.Input{Model: m, QuantName: q.Name, Runtime: profile, Target: target, Tuned: true, CtxOverride: 65536, KVCacheType: "f16"})
			tm = r.Timing
			timings[p.ID()] = tm
		}
		if b.Gen > 0 {
			sec := 0.0
			for i := 0; i < b.Gen; i++ {
				sec += tm.DecodeSec(b.Depth + i)
			}
			return sec / float64(b.Gen)
		}
		return tm.PrefillSec(b.Depth, b.Depth+b.Prompt) / float64(b.Prompt)
	}, backend
}
