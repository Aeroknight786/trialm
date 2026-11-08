package tui

import (
	tea "charm.land/bubbletea/v2"
	"context"
	"fmt"
	"github.com/aeroknight786/trialm/internal/core"
	"github.com/aeroknight786/trialm/internal/hw"
	"github.com/aeroknight786/trialm/internal/setup"
	"github.com/aeroknight786/trialm/internal/store"
	"github.com/aeroknight786/trialm/internal/verify"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestFirstRunKeysAndLowPower(t *testing.T) {
	m := fixture(t)
	m.stack = []Screen{Detect}
	m.detecting = false
	m.Update(detectDoneMsg{})
	if m.screen() != Agents {
		t.Fatal(m.screen())
	}
	m.Update(key("enter"))
	if m.screen() != Calibrate {
		t.Fatal(m.screen())
	}
	m.detection.System = &hw.System{Power: hw.Power{LowPower: true}}
	m.Update(key("enter"))
	if m.calibration != nil || m.message != "Turn off Low Power Mode to calibrate" || m.screen() != Calibrate {
		t.Fatal(m.message)
	}
	m.detection.System.Power.LowPower = false
	m.Update(key("s"))
	if m.screen() != Home {
		t.Fatal(m.screen())
	}
	// A new user's pick is not on this Mac yet: Enter tries it.
	m.busy, m.detecting, m.modelIndex = false, false, 2
	m.models[2].Journal, m.models[2].Row.LocalPath = nil, ""
	m.Update(key("enter"))
	if m.screen() != First {
		t.Fatal(m.screen())
	}
	t.Log("Detect advances automatically; Agents Enter + Later s + Try Enter + sign-in Enter = 4 keys + 1 browser click to the trial handoff")
}
func TestPickerCachedDrawUnder150ms(t *testing.T) {
	m := fixture(t)
	m.stack = []Screen{Recipes}
	m.width, m.height = 80, 40
	start := time.Now()
	for i := 0; i < 50; i++ {
		if !strings.Contains(m.View().Content, "KV cache") {
			t.Fatal("missing grid")
		}
	}
	elapsed := time.Since(start) / 50
	if elapsed >= 150*time.Millisecond {
		t.Fatal(elapsed)
	}
	t.Log("cached draw", elapsed)
}

func TestFirstRunLocalSixKeys(t *testing.T) {
	m := fixture(t)
	m.stack = []Screen{Detect}
	m.firstRun = true
	m.detecting = false
	m.Update(detectDoneMsg{})
	m.Update(key("enter"))
	m.Update(key("s"))
	m.busy = false
	m.stack = []Screen{Recipes}
	m.selected = 0
	// Clone the selected table before adding a file fixture to it.
	model := m.picker.Models[0]
	grid := model.Grid()
	grid.Files = append([]core.ConfigFile(nil), grid.Files...)
	f := grid.Files[model.File]
	f.Cells = append([]core.ConfigCell(nil), f.Cells...)
	f.Cells[model.Column].Row.LocalPath = "/scratch/model.gguf"
	grid.Files[model.File] = f
	grids := map[string]core.ConfigGrid{}
	for k, v := range model.Grids {
		grids[k] = v
	}
	grids[model.Engines[model.Engine].ID+"/"+model.CacheTypes[model.Cache]] = grid
	model.Grids = grids
	m.picker.Models[0] = model
	expected, _ := model.Cell()
	p := m.setup.plan
	m.cfg.Plan = func(_ context.Context, row core.Row, _ store.Preferences) (setup.Plan, error) {
		if row.Result.Settings.ContextTokens != expected.Context || row.Result.Settings.KVCacheType != expected.Row.Result.Settings.KVCacheType {
			t.Fatal("lost selected cell")
		}
		return p, nil
	}
	m.cfg.Apply = func(context.Context, setup.Plan, core.Sink) (setup.Journal, error) {
		return setup.Journal{Plan: p, State: "ready", Report: &verify.Report{Ready: true}}, nil
	}
	m.Update(key("enter"))
	_, cmd := m.Update(key("s"))
	m.Update(cmd())
	_, cmd = m.Update(key("enter"))
	m.Update(cmd())
	_, launch := m.Update(sparkleMsg(3))
	if m.screen() != Ready || launch != nil {
		t.Fatal("Ready did not wait for a key", m.screen())
	}
	_, launch = m.Update(key("enter"))
	if m.screen() != Launching || launch == nil {
		t.Fatal("six keys did not reach the local tool handoff", m.screen())
	}
	t.Log("Agents Enter + Later s + Choose Enter + Set up s + approval Enter + Open Enter = 6 keys")
}
func TestTryKeepsCellAcrossAgentSwitch(t *testing.T) {
	m := fixture(t)
	m.stack = []Screen{Recipes}
	m.cfg.Preferences.PreferredTools = []string{"codex", "claude-code"}
	m.Update(key("enter"))
	m.Update(key("enter"))
	before := *m.trialRow
	m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if m.cfg.Preferences.Tool != "claude-code" || m.trialRow.Result.Settings.ContextTokens != before.Result.Settings.ContextTokens || m.trialRow.Result.Settings.KVCacheType != before.Result.Settings.KVCacheType {
		t.Fatal("agent switch changed the configuration")
	}
	t.Setenv("OPENROUTER_API_KEY", "present")
	if strings.Contains(m.footer(), "sign-in") || strings.Contains(m.footer(), "Paste key") {
		t.Fatal(m.footer())
	}
}

func TestGenericServerHelper(t *testing.T) {
	if os.Getenv("TRIALM_GENERIC_HELPER") != "1" {
		t.Skip("child only")
	}
	fmt.Println(`{"type":"progress","data":{"endpoint":"http://127.0.0.1:12345"}}`)
	_, _ = io.Copy(io.Discard, os.Stdin)
}
func TestOwnAgentShowsOneAddress(t *testing.T) {
	m := fixture(t)
	m.cfg.Preferences.Tool = "generic"
	m.stack = []Screen{Recipes}
	var args []string
	m.cfg.Process = func(ctx context.Context, exe string, a []string) *exec.Cmd {
		args = append([]string(nil), a...)
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=TestGenericServerHelper")
		cmd.Env = append(os.Environ(), "TRIALM_GENERIC_HELPER=1")
		return cmd
	}
	m.try(false)
	cmd := m.startTrial()
	msg := cmd().(chatStartedMsg)
	if msg.err != nil {
		t.Fatal(msg.err)
	}
	m.Update(msg)
	if m.body() != "http://127.0.0.1:12345" {
		t.Fatal(m.body())
	}
	if !strings.Contains(strings.Join(args, " "), "--serve --json") || !strings.Contains(strings.Join(args, " "), "--for generic") {
		t.Fatal(args)
	}
	done := m.finishChat()().(childMsg)
	if done.err != nil {
		t.Fatal(done.err)
	}
}

func TestFreshRunFirstTokenFourKeys(t *testing.T) {
	base := fixture(t)
	s := store.Store{Dir: t.TempDir()}
	t.Setenv("TRIALM_HOME", s.Dir)
	t.Setenv("OPENROUTER_API_KEY", "")
	first := make(chan struct{}, 1)
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `data: {"choices":[{"delta":{"content":"Hello"}}]}`)
		select {
		case first <- struct{}{}:
		default:
		}
	}))
	defer up.Close()
	cfg := base.cfg
	cfg.Store = s
	cfg.Headless = false // CI runners without a display would sign in by pasting a code
	cfg.Preferences.FirstRunDone = false
	cfg.NoColor = true
	cfg.Detect = func(context.Context) (core.Detection, error) { return base.detection, nil }
	cfg.Models = func(context.Context, core.Detection, store.Preferences) ([]core.ModelChoice, error) {
		rows := append([]core.ModelChoice(nil), base.models...)
		for i := range rows {
			rows[i].Journal = nil
			rows[i].Row.LocalPath = ""
		}
		return rows, nil
	}
	cfg.Picker = func(context.Context, core.Detection, store.Preferences) (core.ConfigPicker, error) {
		return base.picker, nil
	}
	cfg.Authorize = func(context.Context, string, bool) error { return s.SaveKey("fake-browser-key") }
	cfg.Process = func(ctx context.Context, _ string, _ []string) *exec.Cmd {
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=TestTrialHelper")
		cmd.Env = append(os.Environ(), "TRIALM_TUI_HELPER=1", "TRIALM_TUI_ENDPOINT="+up.URL)
		return cmd
	}
	m := New(cfg)
	program := tea.NewProgram(&driver{m}, tea.WithInput(nil), tea.WithOutput(io.Discard), tea.WithoutRenderer())
	done := make(chan error, 1)
	go func() { _, e := program.Run(); done <- e }()
	defer program.Kill()
	waitScreen(t, program, Agents)
	program.Send(key("enter"))
	waitScreen(t, program, Calibrate)
	program.Send(key("s"))
	waitScreen(t, program, Home)
	deadline := time.Now().Add(5 * time.Second)
	for {
		ch := make(chan bool, 1)
		program.Send(probeLauncher{ch})
		if <-ch {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("picks never ready")
		}
		time.Sleep(time.Millisecond)
	}
	program.Send(key("enter"))
	waitScreen(t, program, First)
	program.Send(key("enter"))
	select {
	case <-first:
	case <-time.After(5 * time.Second):
		t.Fatal("no first fake trial token")
	}
	waitScreen(t, program, Summary)
	program.Send(key("q"))
	if e := <-done; e != nil {
		t.Fatal(e)
	}
	t.Log("4 keys to first trial token; one browser authorization simulated by fake callback")
}

func TestSelectedCellReachesRealSetupArgv(t *testing.T) {
	m := fixture(t)
	t.Setenv("TRIALM_HOME", m.cfg.Store.Dir)
	model := &m.picker.Models[0]
	model.Select("c")
	cell, ok := model.Cell()
	if !ok {
		t.Fatal("no selected cell")
	}
	p, err := m.cfg.Plan(context.Background(), cell.Row, m.cfg.Preferences)
	if err != nil {
		t.Fatal(err)
	}
	if p.ContextTokens != cell.Context || p.KVCacheType != cell.Row.Result.Settings.KVCacheType || p.Quant != cell.Row.Result.Quant.Name {
		t.Fatal("plan changed the selected configuration", p)
	}
	found := false
	for _, step := range p.Steps {
		if step.ID == "server" {
			text := strings.Join(step.Argv, " ")
			if !strings.Contains(text, "-c "+strconv.Itoa(cell.Context)) || !strings.Contains(text, "-ctk "+p.KVCacheType) || !strings.Contains(text, "-ctv "+p.KVCacheType) {
				t.Fatal(step.Argv)
			}
			found = true
		}
	}
	if !found {
		t.Fatal("no server argv")
	}
}

func TestPowerModeCanBeTurnedOffOnTheCard(t *testing.T) {
	m := fixture(t)
	m.stack = []Screen{Calibrate}
	m.cfg.Power = func(context.Context) bool { return true }
	_, cmd := m.Update(key("enter"))
	m.Update(cmd())
	if m.screen() != Calibrate || m.busy || m.message != "Turn off Low Power Mode to calibrate" {
		t.Fatal(m.screen(), m.message)
	}
	m.cfg.Power = func(context.Context) bool { return false }
	m.cfg.Executable = os.Args[0]
	t.Setenv("TRIALM_CALIBRATION_HELPER", "1")
	_, cmd = m.Update(key("enter"))
	m.Update(cmd())
	if m.calibration == nil || m.detection.System.Power.LowPower {
		t.Fatal("power mode remained stale")
	}
	m.calibration.stop()
	m.calibration = nil
}
func TestDetectionRetainsDiskModelsAndReturningSetups(t *testing.T) {
	for _, ready := range []bool{false, true} {
		m := fixture(t)
		m.stack = []Screen{Detect}
		m.firstRun = true
		rows := append([]core.ModelChoice(nil), m.models...)
		for i := range rows {
			if !ready {
				rows[i].Journal = nil
			}
			rows[i].Row.LocalPath = "/scratch/model.gguf"
		}
		m.Update(modelsMsg{tool: m.cfg.Preferences.Tool, rows: rows})
		if ready {
			if m.screen() != Home {
				t.Fatal("returning setup did not open the launcher")
			}
		} else if !strings.Contains(m.detectBody(), rows[0].Row.Result.Model.Name) {
			t.Fatal("detected files were discarded")
		}
	}
}
