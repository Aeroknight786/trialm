package setup_test

import (
	"context"
	"os"
	"testing"

	"github.com/aeroknight786/trialm/internal/calib"
	"github.com/aeroknight786/trialm/internal/core"
	"github.com/aeroknight786/trialm/internal/engine"
	"github.com/aeroknight786/trialm/internal/figure"
	"github.com/aeroknight786/trialm/internal/harness"
)

// TestSharedRecordingE2E inspects a real setup produced in a scratch home by
// the CLI. It never downloads a model or runs a GPU engine.
func TestSharedRecordingE2E(t *testing.T) {
	root := os.Getenv("TRIALM_RECORDING_E2E_HOME")
	if root == "" {
		t.Skip("set TRIALM_RECORDING_E2E_HOME to an already Ready CPU setup")
	}
	t.Setenv("TRIALM_HOME", root)
	ctx := context.Background()
	s := core.Session{}
	d, err := s.Detect(ctx, core.Options{})
	if err != nil {
		t.Fatal(err)
	}
	h, _ := harness.Get("claude-code")
	choices, err := s.Models(ctx, core.Options{Backend: "cpu"}, d.Target, h)
	if err != nil {
		t.Fatal(err)
	}
	c, err := calib.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	bench, err := calib.FindEngine(root)
	if err != nil || bench != engine.SharedBinary(root, "llama-bench", d.Target.OS) {
		t.Fatal(bench, err)
	}
	fp, err := calib.MachineFingerprint(ctx, bench)
	if err != nil || fp.Hash != c.Fingerprint.Hash {
		t.Fatal("fingerprint mismatch", err)
	}
	for _, choice := range choices {
		if choice.Journal == nil || choice.Row.Result.Model.ID != "qwen3.5-9b" {
			continue
		}
		if choice.Journal.Plan.Backend != "cpu" {
			t.Fatal("proof must use CPU")
		}
		if choice.Speed.Kind != figure.Measured || choice.Speed.V <= 0 {
			t.Fatal(choice.Speed)
		}
		if choice.Row.Result.Kind != "calibrated" {
			t.Fatal("saved real samples did not reach predictor", choice.Row.Result.Kind)
		}
		real := 0
		for _, row := range c.Rows {
			if row.Probe == "real" {
				real++
			}
		}
		if real == 0 {
			t.Fatal("no saved real measurements")
		}
		t.Logf("CPU Ready row: writing %s (%s), reply %s (%s); %d saved real rows; fingerprint %s", choice.Speed, choice.Speed.Kind, choice.Reply, choice.Reply.Kind, real, fp.Hash)
		return
	}
	t.Fatal("Ready CPU Qwen3.5 9B row missing")
}
