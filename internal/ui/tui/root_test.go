package tui

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"encoding/json"
	"github.com/aeroknight786/trialm/internal/catalog"
	"github.com/aeroknight786/trialm/internal/core"
	"github.com/aeroknight786/trialm/internal/hw"
	"github.com/aeroknight786/trialm/internal/setup"
	"github.com/aeroknight786/trialm/internal/ui/tui/calprogress"
	"github.com/aeroknight786/trialm/internal/verify"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/ansi"

	"github.com/aeroknight786/trialm/internal/openrouter"
	"github.com/aeroknight786/trialm/internal/store"
	"github.com/aeroknight786/trialm/internal/trial"
)

var update = flag.Bool("update", false, "update TUI frame goldens")

var listingFixture struct {
	sync.Once
	value core.Listing
	err   error
}
var pickerFixture struct {
	sync.Once
	value core.ConfigPicker
	err   error
}

func fixture(t testing.TB) *Root {
	t.Helper()
	s := store.Store{Dir: t.TempDir()}
	p, _ := s.Preferences()
	p.Tool = "codex"
	p.Context = 65536
	p.QualityFloor = "3bit-ok"
	opts := core.Options{Device: "Apple M5", RAM: 24, Runtime: "all", Tuned: true, Context: p.Context, Use: "agent", QualityFloor: p.QualityFloor}
	cfg := SessionConfig(context.Background(), s, p, opts, 3, "test")
	cfg.UpdateNotice = nil
	cfg.ModelServers = nil
	cfg.Tools = []string{"codex", "claude-code"}
	m := New(cfg)
	d, e := cfg.Detect(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	m.detection = d
	listingFixture.Do(func() { listingFixture.value, listingFixture.err = cfg.Find(context.Background(), p) })
	if listingFixture.err != nil {
		t.Fatal(listingFixture.err)
	}
	m.listing = listingFixture.value
	m.listing.Rows = append([]core.Row(nil), listingFixture.value.Rows...)
	pickerFixture.Do(func() { pickerFixture.value, pickerFixture.err = cfg.Picker(context.Background(), d, p) })
	if pickerFixture.err != nil {
		t.Fatal(pickerFixture.err)
	}
	m.picker = pickerFixture.value
	m.picker.Models = append([]core.ConfigModel(nil), pickerFixture.value.Models...)
	m.last = store.TrialEntry{ID: "fixture", At: time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC), Model: "qwen3.6:35b-a3b", Quant: "IQ3_XXS", Runtime: "llamacpp", Band: 60, Report: trial.Report{Requests: 38, LocalSeconds: 840, CloudSeconds: 190, ReadingSeconds: 370, WritingSeconds: 294, BetweenTurnsSeconds: 176, FirstTurn: 10, TypicalTurn: 9.8, SlowestTurn: 41, SlowestCause: "reading", Fidelity: "fp8 (declared), no-logging; upper bound on quality", SpendKnown: true, SpendUSD: .21}}
	m.history = []store.TrialEntry{m.last}
	m.notice = "v0.6.1 available · trialm update"
	setupFrame(m)
	return m
}
func TestViewGoldens(t *testing.T) {
	old := lipgloss.Writer.Profile
	lipgloss.Writer.Profile = colorprofile.TrueColor
	defer func() { lipgloss.Writer.Profile = old }()
	screens := []Screen{First, Paste, Detect, Agents, Calibrate, "calibrating", "calibrate-low-power", "pick-second", "pick-last", "pick-cache", "pick-file", "pick-disk", "pick-grid", "pick-moved", "pick-scrolled", "calibrate-models", "calibrate-models-details", "setup-models", "setup-models-details", "detecting", Help, Recipes, Details, Trial, Summary, Home, Gallery, ClaimCard, Search, Why, Setup, Chat, Ready, Launching, "setup-progress", "setup-paused", "error", "error-details", "setup-details", "ready-details", "setup-disk", "setup-check", "setup-check-later"}
	for _, screen := range screens {
		for _, w := range []int{60, 80, 120} {
			for _, plain := range []bool{false, true} {
				name := fmt.Sprintf("%s-%d-color%t", screen, w, !plain)
				t.Run(name, func(t *testing.T) {
					m := fixture(t)
					m.Update(tea.WindowSizeMsg{Width: w, Height: 24})
					m.stack = []Screen{screen}
					m.detecting = false
					m.modelIndex = 0
					if strings.HasPrefix(string(screen), "pick-") {
						m.stack = []Screen{Recipes}
						switch screen {
						case "pick-second":
							m.selected = 1
						case "pick-last":
							m.selected = len(m.picker.Models) - 1
						case "pick-cache", "pick-file":
							m.gridFocus = true
							for i, x := range m.picker.Models {
								if x.Default.Result.Model.OllamaTag == "qwen3.5:9b" {
									m.selected = i
								}
							}
							m.picker.Models[m.selected].Select("c")
							if screen == "pick-file" {
								m.picker.Models[m.selected].Select("down")
							}
						}
					}

					if screen == "pick-grid" || screen == "pick-moved" || screen == "pick-scrolled" {
						m.gridFocus = true
						for i, model := range m.picker.Models {
							if model.Default.Result.Model.OllamaTag == "qwen3.5:9b" {
								m.selected = i
							}
						}
						if screen != "pick-grid" {
							m.picker.Models[m.selected].Select("down")
							m.picker.Models[m.selected].Select("right")
						}
						if screen == "pick-scrolled" {
							manyFileFixture(m)
							for i := 0; i < 8; i++ {
								m.picker.Models[m.selected].Select("down")
							}
						}
					}
					if strings.Contains(string(screen), "-models") {
						m.otherModels = []hw.ModelServer{{Name: "llama-server", Model: "qwen3.5:9b"}, {Name: "Ollama"}}
						m.expanded = strings.HasSuffix(string(screen), "-details")
						m.stack = []Screen{Calibrate}
						if strings.HasPrefix(string(screen), "setup-") {
							m.stack = []Screen{Setup}
						}
					}
					if screen == "pick-disk" {
						m.stack = []Screen{Recipes}
						for i, x := range m.picker.Models {
							if x.Default.Result.Model.OllamaTag != "qwen3.5:9b" {
								continue
							}
							m.selected = i
							cm := &m.picker.Models[i]
							g := cm.Grid()
							g.Files = append([]core.ConfigFile(nil), g.Files...)
							f := g.Files[cm.File]
							f.LocalPath = "/fixture/model.gguf"
							f.Cells = append([]core.ConfigCell(nil), f.Cells...)
							f.Cells[cm.Column].Measured = true
							f.Cells[cm.Column].Speed.Kind = "measured"
							g.Files[cm.File] = f
							grids := map[string]core.ConfigGrid{}
							for k, v := range cm.Grids {
								grids[k] = v
							}
							grids[cm.Engines[cm.Engine].ID+"/"+cm.CacheTypes[cm.Cache]] = g
							cm.Grids = grids
							cm.OnDisk = true
						}
					}
					if screen == "calibrate-low-power" {
						m.stack = []Screen{Calibrate}
						m.detection.System.Power.LowPower = true
					}
					if screen == "detecting" {
						m.stack = []Screen{Detect}
						m.detecting = true
						m.detected = false
					}
					if screen == "calibrating" {
						m.stack = []Screen{Calibrate}
						m.calibration = &calibrationJob{Model: calprogress.Model{Fraction: .25, ETA: "About 5 min left", Step: 3, Estimates: make([]float64, 9)}}
						m.busy = true
					}
					switch screen {
					case Setup, Ready, Launching, "setup-progress", "setup-paused", "error":
						setupFrame(m)
					}
					if screen == Setup {
						m.setup.update = setup.Update{}
						m.setup.states = nil
						m.setup.logs = nil
						m.setup.journal = setup.Journal{}
					}
					if screen == "setup-progress" {
						m.stack = []Screen{Setup}
						m.busy = true
					}
					if screen == "setup-paused" {
						m.stack = []Screen{Setup}
						m.setup.journal.State = "paused"
						m.message = "Setup paused. Enter resumes."
					}
					if screen == "error" || screen == "error-details" {
						m.stack = []Screen{Home}
						m.fail("Could not open the tool. Press enter to retry or d for details.", errors.New("fixture handshake refused"))
						m.expanded = screen == "error-details"
					}
					if screen == "setup-details" || screen == "setup-disk" {
						m.stack = []Screen{Setup}
						m.setup.update = setup.Update{}
						m.setup.states = nil
						m.setup.journal = setup.Journal{}
						m.expanded = screen == "setup-details"
						m.setup.onDisk = screen == "setup-disk"
						if m.setup.onDisk {
							m.setup.plan.DownloadBytes = 16700000
							m.setup.plan.Steps = append(m.setup.plan.Steps[:1:1], m.setup.plan.Steps[2:]...)
						}
					}
					if screen == "setup-check" || screen == "setup-check-later" {
						m.stack = []Screen{Setup}
						m.busy = true
						m.setup.states = map[int]string{1: "done", 2: "done", 3: "done", 4: "done", 5: "running"}
						m.setup.update = setup.Update{Kind: setup.Check}
						m.setup.checkStarted = time.Unix(1000, 0)
						m.setup.checkNow = m.setup.checkStarted
						m.setup.plan.Predicted.PrefillTPS = 300
						m.setup.plan.ContextTokens = 32768
						m.setup.subCheck = "Reading a 23K-token prompt"
						if screen == "setup-check-later" {
							m.setup.checkNow = m.setup.checkNow.Add(17 * time.Second)
							m.setup.subCheck = "Writing"
						}
					}
					if screen == "ready-details" || screen == "ready-sparkle" {
						m.stack = []Screen{Ready}
						m.expanded = screen == "ready-details"
						if screen == "ready-sparkle" {
							m.sparkle = 1
						}
					}
					m.theme = newPalette(plain, false)
					if plain && screen == "ready-sparkle" {
						m.sparkle = 3
					}
					fraction := 0.0
					if screen == "setup-progress" {
						fraction = float64(m.setup.update.Progress.Done) / float64(m.setup.update.Progress.Total)
					}
					if screen == "calibrating" {
						fraction = m.calibration.Fraction
					}
					if fraction > 0 && !plain {
						frame := m.bar.SetPercent(fraction)()
						for i := 0; i < 120; i++ {
							m.Update(frame)
						}
					}
					m.started = time.Now().Add(-30 * time.Second)
					m.password = huh.NewInput().Title("OpenRouter key").Value(&m.secret).EchoMode(huh.EchoModePassword)
					m.password.Focus()
					m.search.SetValue("qwen")
					m.chat = newChat(context.Background(), nil)
					m.chat.resize(w, 24)
					got := m.View().Content
					if screen == Recipes && !strings.Contains(got, "Qwen3.6") {
						t.Fatal("recipe rows are not visible")
					}
					if plain && strings.Contains(got, "\x1b") {
						t.Fatal("NO_COLOR has an escape")
					}
					for _, line := range strings.Split(got, "\n") {
						if ansi.StringWidth(line) > w {
							t.Fatalf("overflow %d: %q", ansi.StringWidth(line), line)
						}
					}
					path := filepath.Join("testdata", name+".golden")
					if *update {
						if e := os.MkdirAll(filepath.Dir(path), 0755); e != nil {
							t.Fatal(e)
						}
						if e := os.WriteFile(path, []byte(got), 0644); e != nil {
							t.Fatal(e)
						}
					}
					want, e := os.ReadFile(path)
					if e != nil {
						t.Fatal(e)
					}
					if got != strings.ReplaceAll(string(want), "\r\n", "\n") { // a CRLF checkout is not a change
						t.Fatalf("frame changed: %s; use -update\ngot:\n%s\nwant:\n%s", path, got, want)
					}
				})
			}
		}
	}
}
func key(s string) tea.KeyPressMsg {
	switch s {
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	default:
		return tea.KeyPressMsg{Code: []rune(s)[0], Text: s}
	}
}
func TestNavigationAndRating(t *testing.T) {
	m := fixture(t)
	m.stack = []Screen{Recipes}
	m.Update(key("d"))
	if m.screen() != Details {
		t.Fatal(m.screen())
	}
	m.Update(key("esc"))
	if m.screen() != Recipes {
		t.Fatal(m.screen())
	}
	m.stack = []Screen{Summary}
	m.cfg.Preferences.Priority = "quality"
	_, cmd := m.Update(key("1"))
	if cmd == nil || m.screen() != Recipes || m.cfg.Preferences.Priority != "balanced" {
		t.Fatal(m.screen(), m.cfg.Preferences.Priority)
	}
	rows, e := m.cfg.Store.History()
	if e != nil || len(rows) != 1 || rows[0].Rating != 1 {
		t.Fatal(rows, e)
	}
	m.Update(tea.PasteMsg{Content: "llama-server -m untrusted.gguf --evil"})
	if m.screen() == ClaimCard || !strings.Contains(m.View().Content, "Search by model name") {
		t.Fatal(m.View().Content)
	}
	m.theme = newPalette(true, true)
	m.stack = []Screen{First}
	if strings.ContainsAny(m.View().Content, "✓❯·≈━") {
		t.Fatal(m.View().Content)
	}
}
func TestClaimsHaveSourcesAndNeverRank(t *testing.T) {
	for _, c := range Claims() {
		if c.SourceURL == "" || c.SourceName == "" || c.Hardware == "" || c.Recipe == "" || c.Date == "" || c.DateBasis == "" || !strings.Contains("VCS", c.Grade) || len(c.Figures) == 0 {
			t.Fatal(c)
		}
		for _, f := range c.Figures {
			if string(f.Kind) != "claimed" || f.Src == "" {
				t.Fatal(f)
			}
		}
	}
	m := fixture(t)
	for _, r := range m.listing.Rows {
		if r.Result.Kind == "claimed" {
			t.Fatal("claim ranked")
		}
	}
}

// Driver preserves the real root through every tea message, with read-only
// snapshots serialized on tea's event loop so the race detector checks it too.
type snapshot chan Screen
type driver struct{ root *Root }

func (d *driver) Init() tea.Cmd  { return d.root.Init() }
func (d *driver) View() tea.View { return d.root.View() }
func (d *driver) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if v, ok := msg.(probeLauncher); ok {
		v.ready <- !d.root.detecting && len(d.root.models) > 0
		return d, nil
	}
	if v, ok := msg.(probePlan); ok {
		v.ready <- d.root.setup.plan.ID != "" && !d.root.busy
		return d, nil
	}
	if v, ok := msg.(probeRows); ok {
		v.ready <- (len(d.root.listing.Rows) > 0 || len(d.root.picker.Models) > 0) && !d.root.busy
		return d, nil
	}
	if ch, ok := msg.(snapshot); ok {
		ch <- d.root.screen()
		return d, nil
	}
	_, cmd := d.root.Update(msg)
	return d, cmd
}
func waitScreen(t *testing.T, p *tea.Program, want Screen) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		ch := make(snapshot, 1)
		p.Send(ch)
		select {
		case got := <-ch:
			if got == want {
				return
			}
		case <-time.After(time.Second):
			t.Fatal("event loop stopped")
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("never reached %s", want)
}
func TestLauncherKeyCount(t *testing.T) {
	for _, state := range []string{"disk-first", "disk", "ready"} {
		t.Run(state, func(t *testing.T) {
			t.Setenv("OPENROUTER_API_KEY", "")
			m := fixture(t)
			home := t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("TRIALM_HOME", m.cfg.Store.Dir)
			t.Setenv("HF_HUB_CACHE", filepath.Join(home, "hf"))
			t.Setenv("OLLAMA_MODELS", filepath.Join(home, "ollama"))
			model, _ := catalog.Lookup("qwen3.5:9b")
			file := filepath.Join(home, ".lmstudio", "models", model.HFGGUFRepo, "Qwen3.5-9B-Q4_K_M.gguf")
			if e := os.MkdirAll(filepath.Dir(file), 0700); e != nil {
				t.Fatal(e)
			}
			if e := os.WriteFile(file, []byte("UI cache fixture"), 0600); e != nil {
				t.Fatal(e)
			}
			if state != "disk-first" {
				if e := m.choice("skipped"); e != nil {
					t.Fatal(e)
				}
			}
			if state == "ready" {
				j := *m.models[0].Journal
				j.Plan.Model, j.Plan.ModelName, j.Plan.Quant = model.ID, model.Name, "Q4_K_M"
				path := filepath.Join(m.cfg.Store.Dir, "setups", j.Plan.Slug, "journal.json")
				b, _ := json.Marshal(j)
				if e := os.MkdirAll(filepath.Dir(path), 0700); e != nil {
					t.Fatal(e)
				}
				if e := os.WriteFile(path, b, 0600); e != nil {
					t.Fatal(e)
				}
			}
			m.cfg.Preferences.FirstRunDone = true
			m = New(m.cfg)
			approvals, launches := 0, 0
			m.cfg.Plan = func(context.Context, core.Row, store.Preferences) (setup.Plan, error) {
				return setup.Plan{ID: "approved", Slug: "local", Harness: "codex"}, nil
			}
			m.cfg.Apply = func(ctx context.Context, p setup.Plan, sink core.Sink) (setup.Journal, error) {
				approvals++
				return setup.Journal{Plan: p, State: "ready", Report: &verify.Report{Ready: true}}, nil
			}
			m.cfg.Process = func(ctx context.Context, exe string, args []string) *exec.Cmd {
				if args[0] != "launch" || args[1] != "codex" {
					t.Errorf("unexpected handoff %v", args)
				}
				launches++
				cmd := exec.CommandContext(ctx, exe, "-test.run=TestTrialHelper")
				cmd.Env = append(os.Environ(), "TRIALM_TUI_HELPER=1", "TRIALM_TUI_LAUNCH=1")
				return cmd
			}
			program := tea.NewProgram(&driver{m}, tea.WithInput(nil), tea.WithOutput(io.Discard), tea.WithoutRenderer(), tea.WithColorProfile(colorprofile.Ascii))
			done := make(chan error, 1)
			go func() { _, e := program.Run(); done <- e }()
			defer program.Kill()
			keys := 0
			waitScreen(t, program, Home)
			deadline := time.Now().Add(5 * time.Second)
			for {
				ch := make(chan bool, 1)
				program.Send(probeLauncher{ch})
				if <-ch {
					break
				}
				if time.Now().After(deadline) {
					t.Fatal("launcher discovery did not complete")
				}
				time.Sleep(time.Millisecond)
			}

			program.Send(key("enter"))
			keys++
			if state != "ready" {
				waitScreen(t, program, Setup)
				deadline := time.Now().Add(5 * time.Second)
				for {
					ch := make(chan bool, 1)
					program.Send(probePlan{ch})
					if <-ch {
						break
					}
					if time.Now().After(deadline) {
						t.Fatal("plan did not arrive")
					}
					time.Sleep(time.Millisecond)
				}
				program.Send(key("enter"))
				keys++
				waitScreen(t, program, Ready)
				time.Sleep(time.Second) // sparkle has finished and Ready still waits
				waitScreen(t, program, Ready)
				program.Send(key("enter"))
				keys++
			}
			// A return to the launcher proves the tool subprocess completed.
			waitScreen(t, program, Home)
			program.Send(key("q"))
			if e := <-done; e != nil {
				t.Fatal(e)
			}
			want := 3

			if state == "ready" {
				want = 1
			}
			if keys != want || launches != 1 || (state != "ready" && approvals != 1) {
				t.Fatalf("keys=%d approvals=%d launches=%d", keys, approvals, launches)
			}
			t.Logf("%s reached Codex in %d keys", state, keys)
		})
	}
}

type probeLauncher struct{ ready chan bool }
type probePlan struct{ ready chan bool }
type probeRows struct{ ready chan bool }

func TestTrialHelper(t *testing.T) {
	if os.Getenv("TRIALM_TUI_HELPER") != "1" {
		return
	}
	if os.Getenv("TRIALM_TUI_LAUNCH") == "1" {
		return
	}
	r, e := http.Post(os.Getenv("TRIALM_TUI_ENDPOINT"), "application/json", strings.NewReader(`{"messages":[{"role":"user","content":"hello"}]}`))
	if e != nil {
		t.Fatal(e)
	}
	r.Body.Close()
	e = store.Default().SaveTrial(store.TrialEntry{ID: "child", At: time.Now(), Model: "fake", Report: trial.Report{Requests: 1}})
	if e != nil {
		t.Fatal(e)
	}
}

func TestMain(m *testing.M) {
	if os.Getenv("TRIALM_CALIBRATION_HELPER") == "1" {
		fmt.Println(`{"v":1,"type":"step","step":2,"total":4,"message":"bench fixture","eta_s":45,"step_estimates_s":[15,15,15,15]}`)
		fmt.Println(`{"v":1,"type":"result","complete":true,"message":"done"}`)
		os.Exit(0)
	}
	os.Exit(m.Run())
}
func TestCalibrationScreenAndPersistence(t *testing.T) {
	m := fixture(t)
	t.Setenv("TRIALM_HOME", m.cfg.Store.Dir)
	t.Setenv("TRIALM_CALIBRATION_HELPER", "1")
	m.stack = []Screen{Calibrate}
	m.detecting = false
	m.cfg.Executable = os.Args[0]
	m.Update(key("s"))
	var prefs map[string]any
	if e := m.cfg.Store.Read("prefs.json", &prefs); e != nil || prefs["calibration_offer"] != "skipped" {
		t.Fatal(prefs, e)
	}
	m.stack = []Screen{Calibrate}
	m.busy = false
	m.Update(key("enter"))
	job := m.calibration
	if job == nil {
		t.Fatal("calibration did not start")
	}
	for {
		v := job.next()().(calibrationEventMsg)
		m.Update(v)
		if v.done {
			if v.err != nil {
				t.Fatal(v.err)
			}
			break
		}
		if v.event.Step == 2 && (!strings.Contains(m.View().Content, "left") || job.Fraction != .25) {
			t.Fatal(job, m.View().Content)
		}
	}
	if m.screen() != Recipes {
		t.Fatal(m.screen())
	}
	_ = m.cfg.Store.Read("prefs.json", &prefs)
	if prefs["calibration_offer"] != "accepted" {
		t.Fatal(prefs)
	}
}
func TestFooterBindings(t *testing.T) {
	m := fixture(t)
	m.stack = []Screen{Recipes}
	for _, want := range []string{"↑↓ Model", "⏎ Choose", "/ Search", "v Details", "q Quit"} {
		if !strings.Contains(m.footer(), want) {
			t.Fatal(m.footer())
		}
	}
}
func TestNoColorOverridesForce(t *testing.T) {
	t.Setenv("NO_COLOR", "yes")
	t.Setenv("FORCE_COLOR", "1")
	if len(programOptions()) != 1 {
		t.Fatal("NO_COLOR not applied")
	}
	m := fixture(t)
	m.theme = newPalette(true, false)
	m.stack = []Screen{Recipes}
	m.Update(tea.ColorProfileMsg{Profile: colorprofile.TrueColor})
	if strings.Contains(m.View().Content, "\x1b") {
		t.Fatal("colour leaked")
	}
}

func BenchmarkCachedView(b *testing.B) {
	m := fixture(b)
	m.stack = []Screen{Recipes}
	m.width = 120
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.View()
	}
}
func TestASCIIConsole(t *testing.T) {
	if !asciiTerminal("windows", func(string) string { return "" }) || asciiTerminal("windows", func(k string) string {
		if k == "WT_SESSION" {
			return "yes"
		}
		return ""
	}) || !asciiTerminal("linux", func(k string) string {
		if k == "TERM" {
			return "dumb"
		}
		return ""
	}) {
		t.Fatal("console glyph selection")
	}
}

func TestQualityDefaultFollowsGPU(t *testing.T) {
	for _, gb := range []float64{16, 48} {
		p, _ := (store.Store{Dir: t.TempDir()}).Preferences()
		p.Tool = "codex"
		m := New(Config{Store: store.Store{Dir: t.TempDir()}, Preferences: p})
		d := core.Detection{}
		d.Target.GPUMemBytes = gb * (1 << 30)
		m.Update(detectedMsg{detection: d})
		want := "3bit-ok"
		if gb > 24 {
			want = "4bit+"
		}
		if m.cfg.Preferences.QualityFloor != want {
			t.Fatal(gb, m.cfg.Preferences)
		}
	}
}
func TestHeadlessCodeAgainstFake(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/keys" {
			fmt.Fprint(w, `{"key":"fake-secret"}`)
		} else {
			fmt.Fprint(w, `{"data":{"total_credits":10,"total_usage":1}}`)
		}
	}))
	defer srv.Close()
	s := store.Store{Dir: t.TempDir()}
	h, e := beginHeadless(s, openrouter.Client{BaseURL: srv.URL}, srv.URL+"/auth")
	if e != nil {
		t.Fatal(e)
	}
	if strings.Contains(h.URL, h.verifier) || !strings.Contains(h.URL, "S256") {
		t.Fatal("invalid public authorization URL")
	}
	if e = h.finish(context.Background(), "http://127.0.0.1/callback?code=one-use-code"); e != nil {
		t.Fatal(e)
	}
	t.Setenv("OPENROUTER_API_KEY", "")
	if s.Key() != "fake-secret" {
		t.Fatal("key not saved")
	}
}

func TestInterruptMapsToExit130(t *testing.T) {
	m := fixture(t)
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	if cmd == nil || !errors.Is(m.Error, context.Canceled) {
		t.Fatal(m.Error)
	}
	if !errors.Is(terminalResult(tea.ErrInterrupted, m.Error), context.Canceled) {
		t.Fatal("interrupt would not map to exit 130")
	}
}

func TestFirstScreenUsesSavedBudget(t *testing.T) {
	m := fixture(t)
	m.stack = []Screen{First}
	m.cfg.Preferences.BudgetUSD = 5
	if !strings.Contains(m.View().Content, "$0.20–2") {
		t.Fatal(m.View().Content)
	}
}

func TestPastesReachMaskedInputAndChat(t *testing.T) {
	m := fixture(t)
	m.stack = []Screen{First}
	m.Update(key("p"))
	m.Update(tea.PasteMsg{Content: "fake-secret-key"})
	if m.secret != "fake-secret-key" || strings.Contains(m.View().Content, "fake-secret-key") {
		t.Fatal("masked paste did not reach the input safely")
	}
	m.chat = newChat(context.Background(), nil)
	m.stack = []Screen{Chat}
	m.Update(tea.PasteMsg{Content: "hello pasted chat"})
	if m.chat.input.Value() != "hello pasted chat" {
		t.Fatal(m.chat.input.Value())
	}
	m.stack = []Screen{Search}
	m.search.Focus()
	m.Update(tea.PasteMsg{Content: "qwen"})
	if m.search.Value() != "qwen" {
		t.Fatal(m.search.Value())
	}
	m.cfg.Start = Recipes
	m.cfg.Preferences.Model = "llama-server -m file.gguf -ngl 999"
	m = New(m.cfg)
	if m.screen() != Recipes || strings.Contains(m.View().Content, "claim parsing arrives") {
		t.Fatal(m.View().Content)
	}
}

func TestFormPreservesExplicitValues(t *testing.T) {
	m := fixture(t)
	cfg := m.cfg
	cfg.Preferences.Model = "qwen3:8b"
	cfg.Preferences.Tool = "zed"
	cfg.Preferences.Context = 98304
	cfg.Preferences.BudgetUSD = 3.5
	cfg.ASCII = true
	cfg.NoColor = true
	m = New(cfg)
	p := m.cfg.Preferences
	if p.Model != "qwen3:8b" || p.Tool != "zed" || p.Context != 98304 || p.BudgetUSD != 3.5 {
		t.Fatal(p)
	}
	m.busy = true
	m.stack = []Screen{First}
	for _, r := range m.View().Content {
		if r > 127 {
			t.Fatalf("ASCII spinner/frame contains %q", r)
		}
	}
	m.busy = false
	m.stack = []Screen{Recipes}
	m.listing.Rows = nil
	if strings.Contains(m.footer(), "enter") || strings.Contains(m.footer(), "window") {
		t.Fatal(m.footer())
	}
}

func TestFailedPasteReturnsToKeyChoices(t *testing.T) {
	m := fixture(t)
	m.cfg.Authorize = func(context.Context, string) error { return errors.New("private upstream failure") }
	m.stack = []Screen{First}
	m.Update(key("p"))
	m.Update(tea.PasteMsg{Content: "fake-secret"})
	_, cmd := m.Update(key("enter"))
	m.Update(cmd())
	if m.screen() != First || m.password != nil || strings.Contains(m.View().Content, "fake-secret") {
		t.Fatal(m.screen(), m.View().Content)
	}
}

func TestUnparsedClaimNeverShowsCuratedSources(t *testing.T) {
	m := fixture(t)
	m.stack = []Screen{Recipes}
	m.Update(tea.PasteMsg{Content: "llama-server -m untrusted.gguf"})
	if m.screen() == ClaimCard || strings.Contains(m.View().Content, "claim parsing arrives") {
		t.Fatal(m.View().Content)
	}
}

func TestDetectionCompletesWhileHelpIsOpen(t *testing.T) {
	m := fixture(t)
	m.stack = []Screen{Home}
	m.Update(key("?"))
	if m.screen() != Help {
		t.Fatal(m.screen())
	}
	_, cmd := m.Update(detectedMsg{detection: m.detection})
	if cmd == nil {
		t.Fatal("discovery was lost behind help")
	}
	m.Update(cmd())
	m.Update(key("esc"))
	if m.screen() != Home || m.detecting || len(m.models) == 0 {
		t.Fatal(m.screen(), m.detecting)
	}
}
func TestLauncherKeysAndLazyCloudKey(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "")
	m := fixture(t)
	m.stack = []Screen{Home}
	m.modelIndex = 0
	m.detecting = false
	m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if m.cfg.Preferences.Tool != "Claude Code" && m.cfg.Preferences.Tool != "claude-code" {
		t.Fatal(m.cfg.Preferences.Tool)
	}
	m.Update(key("t"))
	if m.screen() != First {
		t.Fatal("trial did not offer the key", m.screen())
	}
	m.Update(key("esc"))
	if m.screen() != Home {
		t.Fatal(m.screen())
	}
	m.modelIndex = len(m.models)
	m.detecting = false
	m.Update(key("enter"))
	if m.screen() != Recipes {
		t.Fatal(m.screen())
	}
	m.busy = false
	m.Update(key("n"))
	if m.screen() != Recipes {
		t.Fatal(m.screen())
	}
}
func TestErrorDetailsAndReducedMotion(t *testing.T) {
	m := fixture(t)
	m.stack = []Screen{Setup}
	m.busy = true
	m.Update(setupEvent{done: true, err: errors.New("fixture raw failure"), journal: setup.Journal{State: "failed"}})
	if strings.Contains(m.View().Content, "fixture raw failure") || !strings.Contains(m.View().Content, "Press enter") {
		t.Fatal(m.View().Content)
	}
	m.Update(key("d"))
	if !strings.Contains(m.View().Content, "fixture raw failure") {
		t.Fatal(m.View().Content)
	}
	m.push(Home)
	if strings.Contains(m.View().Content, "fixture raw failure") {
		t.Fatal("stale error")
	}
	m.theme = newPalette(true, false)
	m.cfg.Detect = nil
	m.cfg.Models = nil
	m.cfg.UpdateNotice = nil
	if m.tick() == nil {
		t.Fatal("calibration spinner needs ticks with NO_COLOR")
	}
	m.stack = []Screen{Ready}
	_, cmd := m.Update(sparkleMsg(1))
	if cmd != nil || m.sparkle != 3 {
		t.Fatal("NO_COLOR sparkle animates")
	}
}

func TestCloudAuthorizationResumesSelectedTrial(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "")
	m := fixture(t)
	m.stack = []Screen{Home}
	m.detecting = false
	m.modelIndex = 1
	m.cfg.Headless = false
	m.cfg.Authorize = func(ctx context.Context, key string) error { return m.cfg.Store.SaveKey("fixture-key") }
	var selected string
	m.cfg.TrialArgs = func(row core.Row, p store.Preferences) []string { selected = row.Result.Model.ID; return nil }
	_, cmd := m.Update(key("t"))
	if cmd != nil || m.screen() != First {
		t.Fatal(m.screen())
	}
	_, cmd = m.Update(key("enter"))
	_, cmd = m.Update(cmd())
	if m.screen() != Trial || selected != m.models[1].Row.Result.Model.ID || cmd == nil {
		t.Fatal(m.screen(), selected)
	}
	// Inspect the handoff without starting a cloud request.
	if !strings.Contains(fmt.Sprintf("%T", cmd()), "execMsg") {
		t.Fatal("no TTY handoff")
	}
}

func TestWindowFailureKeepsLauncherUsable(t *testing.T) {
	m := fixture(t)
	m.stack = []Screen{Home, Trial}
	m.busy = true
	m.Update(windowDoneMsg{err: errors.New("fixture raw window failure")})
	if m.busy || m.screen() != Home || strings.Contains(m.View().Content, "fixture raw window failure") {
		t.Fatal(m.View().Content)
	}
	m.Update(key("d"))
	if !strings.Contains(m.View().Content, "fixture raw window failure") {
		t.Fatal(m.View().Content)
	}
}

func TestCalibratedPredictionsKeepApproximationMarker(t *testing.T) {
	m := fixture(t)
	r := m.listing.Rows[0].Result
	r.Accuracy.DecodePct = 10
	r.Kind = "calibrated"
	if f := metric(r, 21, "tok/s", false); estimated(f) != "~21.0 tok/s" || f.Kind != "calibrated" {
		t.Fatal(f, estimated(f))
	}
}

func TestNoColorRetainsUnicodeBoxes(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	t.Setenv("TERM", "xterm-256color")
	m := New(Config{NoColor: os.Getenv("NO_COLOR") != "", ASCII: asciiTerminal("linux", os.Getenv)})
	card := m.card("triaLM")
	if strings.Contains(card, "\x1b") || !strings.Contains(card, "╭") || !strings.Contains(card, "│") || m.theme.glyph("✓") != "✓" || m.theme.wordmark() != "✦ triaLM" {
		t.Fatal(card)
	}
	for _, goos := range []string{"linux", "windows"} {
		t.Setenv("TERM", "dumb")
		m = New(Config{NoColor: true, ASCII: asciiTerminal(goos, os.Getenv)})
		if strings.Contains(m.card("triaLM"), "╭") || !strings.Contains(m.card("triaLM"), "+") {
			t.Fatal(goos, m.card("triaLM"))
		}
	}
}
