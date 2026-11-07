package tui

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/aeroknight786/trialm/internal/core"
	"github.com/aeroknight786/trialm/internal/figure"
	"github.com/aeroknight786/trialm/internal/setup"
	"github.com/aeroknight786/trialm/internal/setup/download"
	"github.com/aeroknight786/trialm/internal/verify"
)

func setupFrame(m *Root) {
	p := setup.Plan{OS: "darwin", ID: "fixture-plan", Slug: "qwen-local", ModelName: "Qwen3.6 35B-A3B", Quant: "IQ3_XXS", RuntimeName: "llama.cpp", Harness: "codex", ContextTokens: 40960, DownloadBytes: 13200000000, Prompts: setup.Prompts{Approvals: 1}, Untouched: []string{"Codex config", "shell profile", "other models"}, Predicted: verify.Predicted{DecodeTPS0: 44, DecodeTPSTarget: 36, PrefillTPS: 800}, DecodeErrorPct: 60, PrefillErrorPct: 70,
		Steps: []setup.Step{{ID: "engine", Kind: setup.Install, Title: "Install llama.cpp b12182", Request: &download.Request{Size: 16700000}}, {Kind: setup.Fetch, Title: "Qwen3.6-35B-A3B.gguf", Request: &download.Request{Size: 13200000000}}, {Kind: setup.Write, Title: "Save owned launcher"}, {Kind: setup.Start, Title: "llama-server :8080 · in-process bridge"}, {Kind: setup.Check, Title: "speed · greeting · tool call · follow-up"}}}
	report := &verify.Report{Ready: true, DecodeTPS: map[int]float64{0: 39.8, 40960: 31.2}, PrefillTPS: 655, Checks: []verify.Check{{Name: "harness", OK: true, Detail: "greeting · tool call · follow-up"}}}
	progress := download.Progress{Done: 8100000000, Total: 13200000000, BytesPerSec: 47000000, ETA: 109 * time.Second}
	j := setup.Journal{State: "ready", Plan: p, Report: report, Steps: []setup.Record{{State: "done"}, {State: "done"}, {Progress: progress}, {}, {}}}
	m.setup = setupView{plan: p, journal: j, report: report, states: map[int]string{1: "done", 2: "running"}, update: setup.Update{Index: 2, Total: 5, Kind: setup.Fetch, Progress: &progress}, logs: []string{"sha256 checked", "archive extracted", "download resumed", "range accepted", "8.1 GB received"}}
	// The Ready fixture and recipe table share one calibrated result.
	m.listing.Rows[0].Result.Kind = "calibrated"
	m.models = []core.ModelChoice{{Row: m.listing.Rows[0], Journal: &j, Speed: figure.Band(21, 20, "tok/s", figure.Measured, "ready")}, {Row: m.listing.Rows[1], Speed: figure.Band(29, 30, "tok/s", figure.Simulated, "prediction")}}
	for i := range m.models {
		m.models[i].Reply = metric(m.models[i].Row.Result, m.models[i].Row.Result.Scenario.FollowUpTurn.Seconds(), "s", false)
	}
	m.models[1].Row.LocalPath = "/fixture/model.gguf"
	m.models = append(m.models, core.ModelChoice{Row: m.listing.Rows[2], Reply: metric(m.listing.Rows[2].Result, m.listing.Rows[2].Result.Scenario.FollowUpTurn.Seconds(), "s", false), Speed: figure.Band(26, 30, "tok/s", figure.Simulated, "prediction"), Download: figure.Bytes(12000000000, "catalog")})
}
func TestSetupEventsAndPause(t *testing.T) {
	m := fixture(t)
	m.theme = newPalette(true, true)
	m.stack = []Screen{Setup}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m.cfg.Apply = func(ctx context.Context, p setup.Plan, sink core.Sink) (setup.Journal, error) {
		sink(core.Event{Type: "setup_progress", Data: setup.Update{Index: 3, Total: 5, Kind: setup.Fetch, Progress: &download.Progress{Done: 61, Total: 100}}})
		<-ctx.Done()
		j := setup.Journal{State: "paused", Plan: p, Steps: []setup.Record{{State: "done"}, {State: "done"}, {Progress: download.Progress{Done: 61, Total: 100}}, {}, {}}}
		sink(core.Event{Type: "setup_paused", Data: j})
		return j, ctx.Err()
	}
	m.cfg.Context = ctx
	next := m.applySetup()
	v := next().(setupEvent)
	next = m.setupEvent(v)
	if !strings.Contains(m.View().Content, "of") || !strings.Contains(m.footer(), "Pause") {
		t.Fatal(m.View().Content)
	}
	m.Update(key("ctrl+c"))
	for next != nil {
		v = next().(setupEvent)
		next = m.setupEvent(v)
	}
	if m.setup.journal.State != "paused" || m.busy {
		t.Fatal(m.setup.journal.State)
	}
	if !strings.Contains(m.View().Content, "2 of 5 done, download 61%") {
		t.Fatal(m.View().Content)
	}
}

func TestAdminPromptReleasesTerminal(t *testing.T) {
	m := fixture(t)
	request := adminRequest{ctx: context.Background(), argv: []string{"sudo", "sysctl", "iogpu.wired_limit_mb=20480"}, done: make(chan error, 1)}
	cmd := m.setupEvent(setupEvent{event: core.Event{Type: "setup_admin", Data: request}})
	if cmd == nil || !strings.Contains(fmt.Sprintf("%T", cmd()), "execMsg") {
		t.Fatal("password prompt did not use Bubble Tea's TTY hand-off")
	}
	// Deliberately inspect only the returned message. Never execute an admin row.
}

func TestSetupShowsUnknownInstallerBytes(t *testing.T) {
	m := fixture(t)
	m.theme = newPalette(true, true)
	m.stack = []Screen{Setup}
	m.setup.plan.DownloadUnknown = true
	if !strings.Contains(m.View().Content, "Installer download size unknown") {
		t.Fatal(m.View().Content)
	}
}

func TestReadyCountdownAndSubChecks(t *testing.T) {
	m := fixture(t)
	m.stack = []Screen{Setup}
	m.theme = newPalette(true, false)
	m.busy = true
	m.setup.plan.Predicted.PrefillTPS = 300
	m.setup.plan.ContextTokens = 32768
	m.setupEvent(setupEvent{event: core.Event{Type: "setup_step", Data: setup.Update{Index: 5, Kind: setup.Check, State: "running"}}})
	start := m.setup.checkStarted
	for _, tc := range []struct {
		elapsed     time.Duration
		left, stage string
	}{{0, "About 2 min left", "Reading a 23K-token prompt"}, {time.Second, "About 1 min 59 s left", "Writing"}, {17 * time.Second, "About 1 min 43 s left", "Follow-up"}, {3 * time.Minute, "About 1 min left", "Checking Claude Code"}} {
		m.setupEvent(setupEvent{event: core.Event{Type: "setup_check", Data: tc.stage}})
		_, tick := m.Update(pulseMsg(start.Add(tc.elapsed)))
		for _, expanded := range []bool{false, true} {
			m.expanded = expanded
			b := strip(m.setupBody())
			if tick == nil || !strings.Contains(b, tc.left) || !strings.Contains(b, tc.stage) {
				t.Fatal(b)
			}
		}
	}
	m.setupEvent(setupEvent{done: true, journal: setup.Journal{State: "ready", Report: &verify.Report{Ready: true}}})
	_, cmd := m.Update(pulseMsg(start.Add(4 * time.Minute)))
	m.Update(sparkleMsg(3))
	if cmd != nil || m.screen() != Ready || m.busy {
		t.Fatal("Ready kept ticking or launched")
	}
	_, quit := m.Update(key("q"))
	if quit == nil || m.screen() != Ready {
		t.Fatal("Later did not leave Ready")
	}
}

func TestPlanOnlyMentionsNeededPassword(t *testing.T) {
	m := fixture(t)
	m.stack = []Screen{Setup}
	m.theme = newPalette(true, false)
	m.setup.journal = setup.Journal{}
	b := m.setupBody()
	if strings.Contains(b, "approval") || strings.Contains(b, "password") || !strings.Contains(b, "Install? 13.2 GB") {
		t.Fatal(b)
	}
	m.setup.plan.Prompts.Passwords = 1
	if b = m.setupBody(); !strings.Contains(b, "Needs your Mac password once") {
		t.Fatal(b)
	}
}
