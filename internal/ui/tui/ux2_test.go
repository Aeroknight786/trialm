package tui

import (
	"context"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/aeroknight786/trialm/internal/core"
	"github.com/aeroknight786/trialm/internal/figure"
	"github.com/aeroknight786/trialm/internal/store"
)

func TestPickerToolDisplayName(t *testing.T) {
	m := fixture(t)
	m.stack = []Screen{Recipes}
	if b := strip(m.recipes()); !strings.Contains(b, "Models") || !strings.Contains(b, "KV cache") || !strings.Contains(b, "t/s when full") {
		t.Fatal(b)
	}
}

func TestMacToolCycleRefreshesReply(t *testing.T) {
	m := fixture(t)
	m.detecting = false
	m.stack = []Screen{Home}
	m.cfg.Models = func(_ context.Context, _ core.Detection, p store.Preferences) ([]core.ModelChoice, error) {
		if p.Tool != "claude-code" {
			t.Fatalf("refresh used %q", p.Tool)
		}
		choices := append([]core.ModelChoice(nil), m.models...)
		choices[0].Reply = figure.Band(42, 60, "s", figure.Simulated, "selected:claude-code")
		return choices, nil
	}
	// tab keeps the list on screen while the new agent's list is made
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if cmd == nil || m.cfg.Preferences.Tool != "claude-code" || m.detecting {
		t.Fatal("tool cycle did not refresh in place")
	}
	msg := cmd().(modelsMsg)
	before := len(m.models)
	m.Update(modelsMsg{tool: "codex", rows: nil})
	if len(m.models) != before {
		t.Fatal("stale result replaced the list")
	}
	m.Update(msg)
	if m.detecting || m.models[0].Reply.V != 42 {
		t.Fatal(m.models)
	}
	// back on an agent shown before, the list comes from memory
	m.cfg.Models = func(context.Context, core.Detection, store.Preferences) ([]core.ModelChoice, error) {
		t.Fatal("recomputed a list it had")
		return nil, nil
	}
	m.cfg.Preferences.Tool = "codex"
	m.ranked["claude-code"] = msg
	m.cfg.Preferences.Tool = "claude-code"
	if v, ok := m.rank(false)().(modelsMsg); !ok || v.rows[0].Reply.V != 42 {
		t.Fatal("cached list not reused")
	}
}
