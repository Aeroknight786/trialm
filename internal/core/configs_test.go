package core

import (
	"context"
	"encoding/json"
	"github.com/aeroknight786/trialm/internal/calib"
	"github.com/aeroknight786/trialm/internal/catalog"
	"github.com/aeroknight786/trialm/internal/engine"
	"github.com/aeroknight786/trialm/internal/harness"
	"github.com/aeroknight786/trialm/internal/predict"
	"github.com/aeroknight786/trialm/internal/setup"
	"github.com/aeroknight786/trialm/internal/store"
	"github.com/aeroknight786/trialm/internal/verify"
	"slices"
	"testing"
	"time"
)

func gridFixture(t *testing.T) (catalog.Model, engine.Profile, predict.Target, predict.Scenario) {
	t.Helper()
	m, _ := catalog.Lookup("qwen3.5:9b")
	rt, _ := engine.Get("llamacpp")
	d, e := (Session{}).Detect(context.Background(), Options{Device: "Apple M5", RAM: 24})
	if e != nil {
		t.Fatal(e)
	}
	h, _ := harness.Get("codex")
	return m, rt, d.Target, h.Scenario
}
func TestConfigTables(t *testing.T) {
	for _, name := range []string{"small cache only", "native limit", "many files", "measured"} {
		t.Run(name, func(t *testing.T) {
			m, rt, target, sc := gridFixture(t)
			m.Quants = slices.Clone(m.Quants[:1]) // a copy: appends below must not reach the catalog
			q := m.Quants[0]
			local := map[string]string{q.Name: "on-disk.gguf"}
			var evidence *calib.Calibration
			switch name {
			case "small cache only":
				target.GPUMemBytes = predict.EstimateMemory(m, q, 32768, engine.KVElemBytes("q8_0"), true, "metal", target).TotalBytes + 1
			case "native limit":
				m.NativeCtx = 65536
			case "many files":
				for i := 0; i < 7; i++ {
					x := q
					x.Name = q.Name + string(rune('A'+i))
					m.Quants = append(m.Quants, x)
				}
			case "measured":
				raw, _ := json.Marshal(map[string]string{"Model": m.ID, "Engine": "llamacpp", "Mode": "decode"})
				evidence = &calib.Calibration{Rows: []calib.Row{{Probe: "real", Quant: q.Name, Depth: 65536, Seconds: 1. / 9, Raw: raw}}}
			}
			g := BuildConfigGrid(m, rt, "f16", target, sc, local, evidence)
			largest := 0
			for _, file := range g.Files {
				for _, cell := range file.Cells {
					if cell.Fits {
						largest = max(largest, cell.Context)
					}
				}
			}
			if g.MaxContext != largest {
				t.Fatalf("list context %d has no fitting grid cell %d", g.MaxContext, largest)
			}
			g8 := BuildConfigGrid(m, rt, "q8_0", target, sc, local, evidence)
			switch name {
			case "small cache only":
				if g.Files[0].Cells[0].Fits || !g8.Files[0].Cells[0].Fits {
					t.Fatal("smaller cache did not unlock cell")
				}
			case "native limit":
				if !g.Files[0].Cells[3].PastNative || g.Files[0].Cells[3].Fits {
					t.Fatal("native cap lost")
				}
			case "many files":
				if len(g.Files) != 8 {
					t.Fatal("core truncated files", len(g.Files))
				}
			case "measured":
				if !g.Files[0].Cells[1].Measured || g.Files[0].Cells[3].Measured || g8.Files[0].Cells[1].Measured {
					t.Fatal("measurement leaked to another depth/cache")
				}
				g = BuildConfigGrid(m, rt, "f16", target, sc, nil, evidence)
				if g.Files[0].Cells[1].Measured {
					t.Fatal("absent file measured")
				}
			}
		})
	}
}
func TestPickerSelectionAndArgs(t *testing.T) {
	m, rt, target, sc := gridFixture(t)
	cm := ConfigModel{Default: Row{Result: predict.Predict(predict.Input{Model: m, Runtime: rt, Target: target, Tuned: true, CtxOverride: 65536, Scenario: sc})}, Engines: []engine.Profile{rt}, CacheTypes: []string{"f16", "q8_0", "q4_0"}, Grids: map[string]ConfigGrid{}}
	for _, cache := range cm.CacheTypes {
		cm.Grids[gridKey(rt.ID, cache)] = BuildConfigGrid(m, rt, cache, target, sc, nil, nil)
	}
	cm.Reset()
	before, _ := cm.Cell()
	cm.Select("c")
	after, _ := cm.Cell()
	if before.Context != after.Context || before.Row.Result.Quant.Name != after.Row.Result.Quant.Name {
		t.Fatal("valid selection not retained")
	}
	args := TrialArgs(after.Row, store.Preferences{Tool: "codex"}, Options{})
	found := false
	for i, a := range args {
		if a == "--kv" && args[i+1] == "q8_0" {
			found = true
		}
	}
	if !found {
		t.Fatal(args)
	}
}
func TestFirstRunAgentPersistence(t *testing.T) {
	s := store.Store{Dir: t.TempDir()}
	p, _ := s.Preferences()
	a := FirstRunAgents([]string{"claude-code", "codex"})
	if !a[0].Checked || !a[1].Checked || a[2].Checked {
		t.Fatal(a)
	}
	a[0].Checked = false
	if e := SaveAgents(s, &p, a); e != nil {
		t.Fatal(e)
	}
	got, _ := s.Preferences()
	if got.Tool != "codex" || len(got.PreferredTools) != 1 {
		t.Fatal(got)
	}
}

func TestOwnerNineBDepthAnchor(t *testing.T) {
	m, rt, target, sc := gridFixture(t)
	g := BuildConfigGrid(m, rt, "f16", target, sc, nil, nil)
	for _, f := range g.Files {
		if f.Name == "Q4_K_M" {
			empty, full := f.Cells[1].Empty.V, f.Cells[1].Speed.V
			if empty < 17 || empty > 27 || full < 7 || full > 12 {
				t.Fatalf("M5 9B empty %.1f, 64K %.1f", empty, full)
			}
			t.Logf("M5 9B empty %.1f t/s, 64K %.1f t/s", empty, full)
			return
		}
	}
	t.Fatal("missing Q4_K_M")
}

func TestLMStudioPickerSetupRefusesUncontrolledCache(t *testing.T) {
	for _, id := range []string{"lmstudio", "lmstudio-mlx"} {
		rt, _ := engine.Get(id)
		_, err := (Session{}).PlanRecipe(context.Background(), predict.Result{Runtime: rt}, Options{}, "codex")
		if err == nil {
			t.Fatal("accepted an engine whose selected cache cannot be enforced", id)
		}
	}
}

func TestReadyDurationUsesPlanRatesAndSamples(t *testing.T) {
	p := setup.Plan{ContextTokens: 32768, Harness: "claude-code", Predicted: verify.Predicted{DecodeTPS0: 22, DecodeTPSTarget: 15, PrefillTPS: 396}}
	if got := ReadyDuration(p); got != 2*time.Minute {
		t.Fatal(got)
	}
	p.Predicted.PrefillTPS = 150
	if got := ReadyDuration(p); got != 4*time.Minute {
		t.Fatal(got)
	}
	p.ContextTokens = 8192
	if got := ReadyDuration(p); got != time.Minute {
		t.Fatal("duplicate 8K sample", got)
	}
	p.ContextTokens = 131072
	if got := ReadyDuration(p); got <= 4*time.Minute {
		t.Fatal("deep sample missing", got)
	}
	p.BitsPerWeight = 2
	p.Predicted.DecodeTPS0 = 2
	if got := ReadyDuration(p); got <= 20*time.Minute {
		t.Fatal("tool samples missing", got)
	}
}

func TestMLXGridFiles(t *testing.T) {
	m, _, target, sc := gridFixture(t)
	mlx, _ := engine.Get("mlx-lm")
	coder, _ := catalog.Lookup("qwen3:14b")
	if !hasFile(m, mlx) || hasFile(coder, mlx) {
		t.Fatal("MLX is offered only for files with a known repository")
	}
	var names []string
	for _, f := range BuildConfigGrid(m, mlx, "f16", target, sc, nil, nil).Files {
		names = append(names, f.Name)
	}
	if len(names) != 2 || names[0] != "MLX-8bit" || names[1] != "MLX-4bit" {
		t.Fatal(names)
	}
	big, _ := catalog.Lookup("qwen3.6:35b-a3b")
	if g := BuildConfigGrid(big, mlx, "f16", target, sc, nil, nil); len(g.Files) != 0 || g.Folded != 1 {
		t.Fatal("the 20.4 GB MLX file cannot fit 16 GB", g.Files, g.Folded)
	}
	if q, _ := m.Quant("MLX-4bit"); m.MLXRepo(q) != "mlx-community/Qwen3.5-9B-MLX-4bit" {
		t.Fatal(m.MLXRepo(q))
	}
}
