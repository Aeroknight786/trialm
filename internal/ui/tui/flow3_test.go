package tui

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/aeroknight786/trialm/internal/calib"
	"github.com/aeroknight786/trialm/internal/core"
	"github.com/aeroknight786/trialm/internal/hw"
	"github.com/aeroknight786/trialm/internal/ui/tui/calprogress"
	"github.com/aeroknight786/trialm/internal/ui/tui/chrome"
	"github.com/charmbracelet/x/ansi"
)

func TestPickerFocusRetentionAndScroll(t *testing.T) {
	m := fixture(t)
	m.stack = []Screen{Home, Recipes}
	for i, model := range m.picker.Models {
		if model.Default.Result.Model.OllamaTag == "qwen3.5:9b" {
			m.selected = i
		}
	}
	index := m.selected
	before := m.picker.Models[index]
	if m.gridFocus || strings.Contains(m.footer(), "File") {
		t.Fatal(m.footer())
	}
	for _, k := range []string{"c", "e", "f", "left", "right", "s"} {
		m.Update(key(k))
	}
	if got := m.picker.Models[index]; got.Cache != before.Cache || got.File != before.File || got.Column != before.Column || m.screen() != Recipes {
		t.Fatal("list edited the grid")
	}
	m.Update(key("enter"))
	if !m.gridFocus || m.screen() != Recipes || !strings.Contains(strip(m.recipes()), "✓") {
		t.Fatal("choose did not focus grid")
	}
	m.Update(key("up"))
	m.Update(key("right"))
	m.Update(key("c"))
	chosen := m.picker.Models[index]
	if m.selected != index || chosen.File == before.File {
		t.Fatal("grid moved the model")
	}
	manyFileFixture(m)
	for i := 0; i < 8; i++ {
		m.Update(key("down"))
	}
	if !strings.Contains(strip(m.recipes()), "Files ") || !strings.Contains(strip(m.recipes()), "↕") {
		t.Fatal(m.recipes())
	}
	m.Update(key("esc"))
	if m.gridFocus || m.screen() != Recipes {
		t.Fatal("escape left picker")
	}
	selected := m.picker.Models[index]
	m.Update(key("up"))
	m.Update(key("down"))
	m.Update(key("enter"))
	got := m.picker.Models[index]
	if got.Cache != selected.Cache || got.File != selected.File || got.Column != selected.Column {
		t.Fatal("lost chosen configuration")
	}
}

func TestFooterHintsAndBrandWidths(t *testing.T) {
	m := fixture(t)
	for _, width := range []int{60, 80, 120} {
		m.width = width
		for _, screen := range []Screen{Detect, Agents, Calibrate, Recipes, First, Paste, Details, Home, Summary, Help, Gallery, ClaimCard, Search, Trial, Chat, Setup, Ready, Launching} {
			m.stack = []Screen{screen}
			for _, grid := range []bool{false, true} {
				m.gridFocus = grid
				footer := strip(m.helpView())
				if width >= 80 && strings.Contains(footer, "\n") {
					t.Fatalf("%s %d %q", screen, width, footer)
				}
				for _, hint := range strings.Split(m.footer(), "  ") {
					if hint = strings.TrimSpace(hint); hint != "" && !strings.Contains(footer, hint) {
						t.Fatalf("split hint %q in %q", hint, footer)
					}
				}
			}
			b := strip(m.brandLine())
			if !strings.HasPrefix(b, "✦ triaLM") || strings.Contains(b, "\n") || ansi.StringWidth(b) > width {
				t.Fatal(b)
			}
		}
	}
	if got := chrome.Footer("/ Search   v Details   esc Back", 20); got != "/ Search   v Details\nesc Back" {
		t.Fatal(got)
	}
}

func TestPlainGridSelectedCell(t *testing.T) {
	m := fixture(t)
	m.stack = []Screen{Recipes}
	m.gridFocus = true
	m.theme = newPalette(true, false)
	cell, _ := m.picker.Models[0].Cell()
	b := m.View().Content
	if !strings.Contains(b, "[~"+strings.TrimPrefix(cell.Speed.String(), "~")+"]") || !strings.Contains(b, "["+cell.Memory.String()+"]") || strings.Contains(b, "\x1b") {
		t.Fatal(b)
	}
}

func TestCalibrationPulseInterpolatesWithoutEvents(t *testing.T) {
	m := fixture(t)
	m.stack = []Screen{Calibrate}
	m.theme = newPalette(true, false)
	now := time.Unix(1000, 0)
	m.calibration = &calibrationJob{Model: calprogress.New([]calib.Step{{EstimatedSeconds: 20}, {EstimatedSeconds: 20}}, now)}
	before := m.View().Content
	m.Update(pulseMsg(now.Add(time.Second)))
	after := m.View().Content
	if m.calibration.Fraction != .025 || before == after {
		t.Fatal(before, after)
	}
	m.Update(pulseMsg(now.Add(100 * time.Second)))
	if m.calibration.Fraction != .475 {
		t.Fatal(m.calibration.Fraction)
	}
}

func TestOtherModelsWarnAndNeverBlock(t *testing.T) {
	m := fixture(t)
	m.stack = []Screen{Calibrate}
	m.cfg.Power = nil
	servers := []hw.ModelServer{{Name: "llama-server", Model: "qwen3.5:9b"}, {Name: "Ollama"}}
	m.cfg.ModelServers = func(context.Context) []hw.ModelServer { return servers }
	m.Update(m.scanServers()())
	if !strings.Contains(strip(m.body()), "2 Other Models Loaded | Expect Slower Speeds") {
		t.Fatal(m.body())
	}
	m.Update(key("v"))
	if !strings.Contains(m.body(), "Ollama") || !strings.Contains(m.body(), "llama-server · qwen3.5:9b") {
		t.Fatal(m.body())
	}
	// Completion of the asynchronous preflight still starts calibration.
	m.cfg.Executable = "missing-test-executable"
	_, cmd := m.Update(key("enter"))
	m.Update(cmd())
	if m.calibration == nil {
		t.Fatal("warning blocked calibration")
	}
	m.calibration.stop()
	m.calibration = nil
	m.stack = []Screen{Setup}
	m.expanded = false
	b := strip(m.setupBody())
	if strings.Index(b, "Other Models Loaded") > strings.Index(b, "Check speed") {
		t.Fatal(b)
	}
}

func TestGridDirectionsSkipUnfittingCells(t *testing.T) {
	m := core.ConfigModel{File: 0, Column: 0, Engines: fixture(t).picker.Models[0].Engines[:1], CacheTypes: []string{"f16"}, Grids: map[string]core.ConfigGrid{}}
	g := core.ConfigGrid{Contexts: []int{1, 2, 3}, Files: []core.ConfigFile{
		{Cells: []core.ConfigCell{{Fits: true}, {}, {Fits: true}}},
		{Cells: []core.ConfigCell{{}, {}, {}}},
		{Cells: []core.ConfigCell{{Fits: true}, {}, {Fits: true}}},
	}}
	m.Grids[m.Engines[0].ID+"/f16"] = g
	for _, step := range []struct {
		key          string
		file, column int
	}{{"right", 0, 2}, {"down", 2, 2}, {"down", 2, 2}, {"left", 2, 0}, {"up", 0, 0}, {"left", 0, 0}} {
		m.Select(step.key)
		if m.File != step.file || m.Column != step.column {
			t.Fatal(step, m.File, m.Column)
		}
	}
}

// Repeated synthetic variants exercise the scroll path beyond today's short
// fitting catalog. They share immutable cell predictions.
func manyFileFixture(m *Root) {
	cm := &m.picker.Models[m.selected]
	g := cm.Grid()
	f := g.Files[cm.File]
	g.Files = nil
	for i := 0; i < 8; i++ {
		row := f
		row.Name += string(rune('A' + i))
		g.Files = append(g.Files, row)
	}
	grids := make(map[string]core.ConfigGrid)
	for k, grid := range cm.Grids {
		grids[k] = grid
	}
	grids[cm.Engines[cm.Engine].ID+"/"+cm.CacheTypes[cm.Cache]] = g
	cm.Grids = grids
	cm.File = 0
}

// A model without a private cloud host shows no SIM, offers no trial key
// and says why when t is pressed anyway.
func TestNoSimMeansNoTrial(t *testing.T) {
	m := fixture(t)
	m.stack, m.detecting, m.modelIndex = []Screen{Home}, false, 2
	if !strings.Contains(m.footer(), "t Cloud trial") || !strings.Contains(strip(m.launcher()), "SIM · ") {
		t.Fatal(m.footer())
	}
	m.models[2].Row.Result.Model.OpenRouterID = ""
	if strings.Contains(m.footer(), "t Cloud trial") || strings.Count(strip(m.launcher()), "SIM · ") != 1 {
		t.Fatal(m.footer(), m.launcher())
	}
	m.Update(key("t"))
	if m.screen() != Home || !strings.Contains(m.message, "No SIM") {
		t.Fatal(m.screen(), m.message)
	}
}
