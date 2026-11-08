package tui

import (
	tea "charm.land/bubbletea/v2"
	"context"
	"github.com/aeroknight786/trialm/internal/core"
	"github.com/aeroknight786/trialm/internal/setup"
	"github.com/aeroknight786/trialm/internal/store"
	"os"
	"testing"
	"time"
)

func TestFlowPTYHelper(t *testing.T) {
	if os.Getenv("TRIALM_FLOW_PTY") != "1" {
		t.Skip("PTY driver only")
	}
	m := fixture(t)
	m.stack = []Screen{Summary}
	m.cfg.Detect = nil
	m.cfg.Models = nil
	m.cfg.UpdateNotice = nil
	m.cfg.NoColor = true
	m.theme = newPalette(true, false)
	m.detecting = false
	p := m.setup.plan
	report := m.setup.report
	p.ContextTokens = 65536
	report.DecodeTPS[65536] = report.DecodeTPS[40960]
	m.cfg.Plan = func(context.Context, core.Row, store.Preferences) (setup.Plan, error) { return p, nil }
	progressUpdate := m.setup.update
	m.cfg.Apply = func(ctx context.Context, p setup.Plan, sink core.Sink) (setup.Journal, error) {
		sink(core.Event{Type: "setup_step", Data: setup.Update{Index: 1, Total: 5, Kind: setup.Install, State: "done"}})
		u := progressUpdate
		u.State = "running"
		sink(core.Event{Type: "setup_progress", Data: u})
		select {
		case <-time.After(2 * time.Second):
		case <-ctx.Done():
			return setup.Journal{Plan: p, State: "paused"}, ctx.Err()
		}
		return setup.Journal{Plan: p, State: "ready", Report: report}, nil
	}
	if _, err := tea.NewProgram(m, tea.WithContext(m.cfg.Context)).Run(); err != nil {
		t.Fatal(err)
	}
}
