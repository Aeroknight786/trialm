package tui

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"strings"
	"time"

	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"github.com/aeroknight786/trialm/internal/core"
	"github.com/aeroknight786/trialm/internal/harness"
	"github.com/aeroknight786/trialm/internal/hw"
	"github.com/aeroknight786/trialm/internal/openrouter"
	"github.com/aeroknight786/trialm/internal/setup"
	"github.com/aeroknight786/trialm/internal/store"
	"github.com/aeroknight786/trialm/internal/ui/tui/pick"
	"github.com/aeroknight786/trialm/internal/ui/window"
	"github.com/charmbracelet/colorprofile"
)

type Screen string

const (
	Detect    Screen = "detect"
	Agents    Screen = "agents"
	Calibrate Screen = "calibrate"
	First     Screen = "try-first"
	Paste     Screen = "paste"
	Recipes   Screen = "recipes"
	Details   Screen = "details"
	Trial     Screen = "trial"
	Summary   Screen = "summary"
	Home      Screen = "launcher"
	Help      Screen = "help"
	Gallery   Screen = "gallery"
	ClaimCard Screen = "claim"
	Search    Screen = "search"
	Why       Screen = "why"
	Setup     Screen = "setup"
	Chat      Screen = "chat"
	Ready     Screen = "ready"
	Launching Screen = "launch"
)

// Config uses application callbacks so no Charm types escape into the CLI/core.
type Config struct {
	ModelServers    func(context.Context) []hw.ModelServer
	Power           func(context.Context) bool
	Picker          func(context.Context, core.Detection, store.Preferences) (core.ConfigPicker, error)
	LocalPlan       *setup.Plan
	Plan            func(context.Context, core.Row, store.Preferences) (setup.Plan, error)
	Apply           func(context.Context, setup.Plan, core.Sink) (setup.Journal, error)
	Models          func(context.Context, core.Detection, store.Preferences) ([]core.ModelChoice, error)
	Context         context.Context
	Store           store.Store
	Preferences     store.Preferences
	Detect          func(context.Context) (core.Detection, error)
	Find            func(context.Context, store.Preferences) (core.Listing, error)
	Authorize       func(context.Context, string) error
	TrialArgs       func(core.Row, store.Preferences) []string
	Window          func(context.Context, []string) error
	UpdateNotice    func(context.Context) string
	Executable      string
	Start           Screen
	NoColor, ASCII  bool
	DirectArgs      []string
	AutomaticWindow bool
	Process         func(context.Context, string, []string) *exec.Cmd
	WindowArgs      []string
	Headless        bool
	Tools           []string
}
type powerMsg struct {
	low     bool
	servers []hw.ModelServer
}
type serversMsg []hw.ModelServer
type calibrationPresenceMsg bool
type detectDoneMsg struct{}
type pickerMsg struct {
	picker core.ConfigPicker
	err    error
}
type modelsMsg struct {
	tool  string
	start int
	rows  []core.ModelChoice
	err   error
}
type sparkleMsg int
type detectedMsg struct {
	detection core.Detection
	err       error
}
type recipesMsg struct {
	listing core.Listing
	err     error
}
type authorizedMsg struct{ err error }
type headlessReadyMsg struct {
	login *headlessLogin
	err   error
}
type noticeMsg string
type childMsg struct {
	kind Screen
	err  error
}
type pulseMsg time.Time
type windowDoneMsg struct{ err error }

// Root owns every screen and sub-model for the entire interactive session.
type Root struct {
	start                         int // tokens the agent's estimates assume it sends first
	otherModels                   []hw.ModelServer
	gridFocus                     bool
	picker                        core.ConfigPicker
	agents                        []core.AgentChoice
	agentIndex                    int
	firstRun                      bool
	cfg                           Config
	setup                         setupView
	models                        []core.ModelChoice
	modelIndex                    int
	detecting, detected, expanded bool
	sparkle                       int
	rawError                      string
	pendingTrial                  bool
	trialRow                      *core.Row
	trialWindow                   bool
	address                       string
	stack                         []Screen
	width, height                 int
	theme                         palette
	detection                     core.Detection
	listing                       core.Listing
	selected                      int

	secret          string
	password        *huh.Input
	search          textinput.Model
	spin            spinner.Model
	bar             progress.Model
	busy            bool
	message, notice string
	history         []store.TrialEntry
	last            store.TrialEntry
	started         time.Time
	calibration     *calibrationJob
	chat            chatModel
	chatStop        func() error
	Error           error
	meter           window.Meter
	headless        *headlessLogin
	authURL         string
	autoQuality     bool
}

func New(cfg Config) *Root {
	if cfg.Context == nil {
		cfg.Context = context.Background()
	}
	if cfg.Executable == "" {
		cfg.Executable, _ = os.Executable()
	}
	start := cfg.Start
	if start == "" {
		start = Home
	}
	m := &Root{autoQuality: cfg.Preferences.QualityFloor == "", cfg: cfg, stack: []Screen{start}, width: 80, height: 24, theme: newPalette(cfg.NoColor, cfg.ASCII), spin: spinner.New(spinner.WithSpinner(spinner.Dot)), bar: progress.New(progress.WithWidth(36), progress.WithDefaultBlend(), progress.WithoutPercentage())}
	m.firstRun = start == Home && !cfg.Preferences.FirstRunDone
	var legacy map[string]json.RawMessage
	_ = cfg.Store.Read("prefs.json", &legacy)
	if _, ok := legacy["calibration_offer"]; ok {
		m.firstRun = false
	}
	m.agents = core.FirstRunAgents(cfg.Tools)
	if m.firstRun {
		m.replace(Detect)
	}
	m.detecting = start == Home
	if cfg.ASCII {
		m.spin = spinner.New(spinner.WithSpinner(spinner.Line))
	}
	m.search = textinput.New()
	m.search.Prompt = "Search models: "
	m.search.CharLimit = 500
	if cfg.LocalPlan != nil {
		m.setup = setupView{plan: *cfg.LocalPlan, states: map[int]string{}}
	}
	return m
}
func Run(cfg Config) error {
	cfg.NoColor = cfg.NoColor || os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb"
	cfg.ASCII = cfg.ASCII || asciiTerminal(nativeOS, os.Getenv)
	m := New(cfg)
	opts := append(programOptions(), tea.WithContext(m.cfg.Context))
	_, err := tea.NewProgram(m, opts...).Run()
	return terminalResult(err, m.Error)
}
func (m *Root) screen() Screen   { return m.stack[len(m.stack)-1] }
func (m *Root) push(s Screen)    { m.stack = append(m.stack, s); m.message, m.rawError = "", "" }
func (m *Root) replace(s Screen) { m.stack[len(m.stack)-1] = s; m.message, m.rawError = "", "" }
func (m *Root) back() {
	if len(m.stack) > 1 {
		m.stack = m.stack[:len(m.stack)-1]
	}
	m.message, m.rawError = "", ""
}
func (m *Root) Init() tea.Cmd {
	cmds := []tea.Cmd{m.tick(), m.scanServers()}
	if m.cfg.WindowArgs != nil {
		return tea.Batch(m.tick(), m.beginWindow(m.cfg.WindowArgs))
	}
	if m.cfg.DirectArgs != nil {
		if m.cfg.Store.Key() != "" || !core.NeedsCloudKey(m.cfg.DirectArgs) {
			return tea.Batch(m.tick(), m.execute(m.cfg.DirectArgs))
		}
		m.replace(First)
	}
	if m.cfg.Detect != nil {
		cmds = append(cmds, func() tea.Msg { d, e := m.cfg.Detect(m.cfg.Context); return detectedMsg{d, e} })
	}
	if m.cfg.UpdateNotice != nil {
		cmds = append(cmds, func() tea.Msg { return noticeMsg(m.cfg.UpdateNotice(m.cfg.Context)) })
	}
	return tea.Batch(cmds...)
}
func (m *Root) find() tea.Cmd {
	if m.cfg.Find == nil && m.cfg.Picker == nil {
		return nil
	}
	m.busy = true
	p := m.cfg.Preferences
	if m.cfg.Picker != nil {
		d := m.detection
		return func() tea.Msg { picker, e := m.cfg.Picker(m.cfg.Context, d, p); return pickerMsg{picker, e} }
	}
	return func() tea.Msg { l, e := m.cfg.Find(m.cfg.Context, p); return recipesMsg{l, e} }
}
func (m *Root) afterKey() tea.Cmd {
	if m.screen() == Paste {
		m.back()
	}
	if m.pendingTrial {
		m.pendingTrial = false
		return m.startTrial()
	}
	m.back()
	return nil
}
func (m *Root) loadModels() tea.Cmd {
	if m.cfg.Models == nil {
		m.detecting = false
		return nil
	}
	m.detecting = true
	d, p := m.detection, m.cfg.Preferences
	return func() tea.Msg {
		rows, e := m.cfg.Models(m.cfg.Context, d, p)
		h, _ := harness.Get(p.Tool)
		return modelsMsg{rows: rows, err: e, tool: p.Tool, start: h.Scenario.BasePromptTokens}
	}
}
func (m *Root) tick() tea.Cmd {
	return m.spin.Tick
}
func (m *Root) fail(sentence string, err error) {
	m.message = sentence
	if err != nil {
		m.rawError = err.Error()
	}
}
func (m *Root) choice(choice string) error {
	p := map[string]json.RawMessage{}
	if e := m.cfg.Store.Read("prefs.json", &p); e != nil && !errors.Is(e, os.ErrNotExist) {
		return e
	}
	p["calibration_offer"], _ = json.Marshal(choice)
	return m.cfg.Store.Write("prefs.json", p)
}
func (m *Root) row() (core.Row, bool) {
	if m.screen() == Home {
		if m.modelIndex < 0 || m.modelIndex >= len(m.models) {
			return core.Row{}, false
		}
		return m.models[m.modelIndex].Row, true
	}
	if m.screen() == First && m.trialRow != nil {
		return *m.trialRow, true
	}
	if len(m.picker.Models) > 0 {
		i := min(m.selected, len(m.picker.Models)-1)
		if c, ok := m.picker.Models[i].Cell(); ok {
			return c.Row, c.Fits
		}
	}
	if len(m.listing.Rows) == 0 {
		return core.Row{}, false
	}
	m.selected = max(0, min(m.selected, len(m.listing.Rows)-1))
	return m.listing.Rows[m.selected], true
}
func (m *Root) try(window bool) tea.Cmd {
	row, ok := m.row()
	if !ok {
		return nil
	}
	if !hasSim(row) {
		m.message = "No SIM for this model: no private cloud host, estimates only"
		return nil
	}
	m.trialRow = &row
	m.trialWindow = window || m.cfg.AutomaticWindow
	m.push(First)
	return nil
}
func (m *Root) startTrial() tea.Cmd {
	if m.cfg.Preferences.Privacy == "predict-only" {
		m.message = "Cloud trials are off in preferences"
		return nil
	}
	row, ok := m.row()
	if m.trialRow != nil {
		row, ok = *m.trialRow, true
	}
	if !ok {
		return nil
	}
	args := m.cfg.TrialArgs(row, m.cfg.Preferences)
	m.push(Trial)
	m.started = time.Now()
	if m.cfg.Preferences.Tool == "generic" {
		return m.startChat(args)
	}
	if m.trialWindow {
		return m.beginWindow(args)
	}
	return m.execute(args)
}
func (m *Root) execute(args []string) tea.Cmd {
	if m.cfg.Preferences.Use == "chat" {
		return m.startChat(args)
	}
	argv := append([]string{"try", "--plain", "--here"}, args...)
	cmd := m.child(argv)
	return tea.ExecProcess(cmd, func(e error) tea.Msg { return childMsg{kind: Trial, err: e} })
}
func (m *Root) child(argv []string) *exec.Cmd {
	cmd := exec.CommandContext(m.cfg.Context, m.cfg.Executable, argv...)
	if m.cfg.Process != nil {
		cmd = m.cfg.Process(m.cfg.Context, m.cfg.Executable, argv)
	}
	if cmd.Env == nil {
		cmd.Env = os.Environ()
	}
	cmd.Env = core.ChildEnv(cmd.Env)
	return cmd
}

func (m *Root) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case tea.ColorProfileMsg:
		if !m.theme.plain && v.Profile >= colorprofile.ANSI256 {
			m.theme = m.theme.rich()
		}
	case tea.WindowSizeMsg:
		m.width = max(20, v.Width)
		m.height = max(8, v.Height)
		m.bar.SetWidth(max(8, min(48, m.width-6)))
		m.chat.resize(m.width, m.height)

	case progress.FrameMsg:
		var cmd tea.Cmd
		bar, cmd := m.bar.Update(v)
		m.bar = bar
		return m, cmd
	case sparkleMsg:
		m.sparkle = int(v)
		if m.theme.plain || m.theme.ascii {
			m.sparkle = 3
		}
		if m.sparkle < 3 {
			return m, tea.Tick(150*time.Millisecond, func(time.Time) tea.Msg { return sparkleMsg(int(v) + 1) })
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(v)
		return m, cmd
	case setupEvent:
		return m, m.setupEvent(v)
	case powerMsg:
		m.busy = false
		m.otherModels = v.servers
		if m.detection.System != nil {
			m.detection.System.Power.LowPower = v.low
		}
		return m, m.beginCalibration(v.low)
	case serversMsg:
		m.otherModels = v
	case calibrationPresenceMsg:
		if bool(v) && m.screen() == Calibrate && !m.busy {
			m.replace(Recipes)
			return m, m.find()
		}
	case detectDoneMsg:
		if m.screen() == Detect && !m.detecting {
			return m, m.afterDetect()
		}
	case pickerMsg:
		m.busy = false
		if v.err != nil {
			m.fail("Could not load models", v.err)
			return m, nil
		}
		m.picker = v.picker
		m.selected = 0
		m.gridFocus = false
		if m.firstRun {
			for i, x := range m.picker.Models {
				if x.OnDisk && m.picker.Models[i].SelectOnDisk() {
					m.selected = i
					break
				}
			}
		}
	case modelsMsg:
		if v.tool != m.cfg.Preferences.Tool {
			return m, nil
		}
		m.detecting = false
		if v.err == nil {
			m.models, m.start = v.rows, v.start
		}
		if m.firstRun && m.screen() == Detect && v.err == nil {
			for _, x := range v.rows {
				if x.Journal != nil {
					m.firstRun = false
					m.replace(Home)
					return m, nil
				}
			}
			return m, tea.Tick(1500*time.Millisecond, func(time.Time) tea.Msg { return detectDoneMsg{} })
		}
		if v.err != nil {
			m.fail("Could not load your models. Press / to find a recipe or d for details.", v.err)
		} else {
			m.models = v.rows
			m.modelIndex = min(m.modelIndex, len(m.models))
			m.modelIndex = max(0, m.modelIndex)
		}
	case detectedMsg:
		m.detection = v.detection
		m.detected = true
		if m.autoQuality {
			m.cfg.Preferences.QualityFloor = "3bit-ok"
			if v.detection.Target.GPUMemBytes > 24*(1<<30) {
				m.cfg.Preferences.QualityFloor = "4bit+"
			}
		}
		if v.err != nil {
			m.detecting = false
			m.fail("Could not read this machine. Press / to find a recipe or d for details.", v.err)
			return m, nil
		}
		if m.cfg.Start == "" || m.cfg.Start == Home {
			return m, m.loadModels()
		}
		if m.screen() == Recipes {
			return m, m.find()
		}
	case recipesMsg:
		m.busy = false
		if v.err != nil {
			m.fail("That did not finish. Press esc to go back or d for details.", v.err)
			return m, nil
		}
		m.listing = v.listing
		m.selected = 0
	case headlessReadyMsg:
		m.busy = false
		if v.err != nil {
			m.message = "Cannot start sign-in. Press p to paste a key."
			return m, nil
		}
		m.headless = v.login
		m.authURL = v.login.URL
		m.push(Paste)
		m.password = huh.NewInput().Title("Authorization code").Value(&m.secret).EchoMode(huh.EchoModePassword)
		m.password.WithWidth(max(24, m.width-4))
		m.password.WithKeyMap(huh.NewDefaultKeyMap())
		m.password.WithTheme(m.formTheme())
		return m, m.password.Focus()
	case authorizedMsg:
		m.busy = false
		m.secret = ""
		m.headless = nil
		m.authURL = ""
		if v.err != nil {
			if m.screen() == Paste {
				m.back()
				m.password = nil
			}
			m.message = "Could not connect OpenRouter. Press p to paste a key or try again."
			return m, nil
		}
		if m.cfg.DirectArgs != nil {
			m.replace(Trial)
			return m, m.execute(m.cfg.DirectArgs)
		}
		return m, m.afterKey()
	case noticeMsg:
		m.notice = string(v)
	case windowDoneMsg:
		if v.err != nil {
			m.busy = false
			m.back()
			m.fail("Could not open a trial window. Press q to exit or d for details.", v.err)
			return m, nil
		}
		return m, func() tea.Msg { return childMsg{kind: Trial} }
	case childMsg:
		m.busy = false
		if (v.kind == Launching || v.kind == Setup) && v.err == nil {
			m.stack = []Screen{Home}
			return m, m.loadModels()
		}
		if v.err != nil {
			m.back()
			m.fail("That did not finish. Press esc to go back or d for details.", v.err)
			return m, nil
		}
		if v.kind == Trial {
			m.history, _ = m.cfg.Store.History()
			if len(m.history) > 0 {
				m.last = m.history[len(m.history)-1]
				m.replace(Summary)
			} else {
				m.back()
				m.message = "Trial finished. No requests were recorded."
			}
		}
	case calibrationEventMsg:
		if m.calibration == nil {
			return m, nil // stopped; the job is being drained
		}
		if v.done {
			m.busy = false
			m.calibration = nil
			if m.screen() == Calibrate {
				m.replace(Recipes)
			}
			if v.err != nil {
				m.fail("Calibration stopped · v Details", v.err)
			}
			if m.screen() == Recipes {
				return m, m.find()
			}

			return m, m.loadModels()
		}
		m.message = ""
		m.calibration.Record(v.event, time.Now())
		return m, m.calibration.next()
	case pulseMsg:
		m.setup.checkNow = time.Time(v)
		if m.calibration != nil {
			m.calibration.Tick(time.Time(v))
		}
		if m.screen() == Trial {
			m.meter = window.ReadMeter(m.cfg.Store)
		}
		if m.calibration != nil || m.screen() == Trial || (m.busy && !m.setup.checkStarted.IsZero()) {
			return m, tea.Tick(time.Second, func(t time.Time) tea.Msg { return pulseMsg(t) })
		}
	case tea.PasteMsg:
		if m.screen() == Paste && m.busy {
			return m, nil
		}
		if m.screen() == Paste && m.password != nil {
			_, cmd := m.password.Update(v)
			return m, cmd
		}
		if m.screen() == Chat && !m.chat.busy {
			var cmd tea.Cmd
			m.chat.input, cmd = m.chat.input.Update(v)
			return m, cmd
		}
		if m.screen() == Search && !core.IsClaimText(v.Content) {
			var cmd tea.Cmd
			m.search, cmd = m.search.Update(v)
			return m, cmd
		}
		if m.screen() == Recipes || m.screen() == Home || m.screen() == Search {
			m.message = "Search by model name"
			return m, nil
		}
	case chatStartedMsg:
		m.busy = false
		if v.err != nil {
			m.back()
			m.fail("That did not finish. Press esc to go back or d for details.", v.err)
			return m, nil
		}
		if m.cfg.Preferences.Tool == "generic" {
			m.address = v.endpoint
			m.chatStop = v.stop
			m.replace(Trial)
			return m, nil
		}
		m.chat = newChat(m.cfg.Context, v.client)
		m.chatStop = v.stop
		m.replace(Chat)
		m.chat.resize(m.width, m.height)
		return m, nil
	case chatResponseMsg:
		m.message, m.rawError = "", ""
		if v.err != nil {
			m.fail("The reply stopped. Type a message to retry or press d for details.", v.err)
		}
		return m, m.chat.receive(v)
	case tea.KeyPressMsg:
		key := v.String()
		if key == "ctrl+c" {
			if m.setup.cancel != nil {
				m.setup.cancel()
				m.message = "Pausing setup…"
				return m, nil
			}
			if m.headless != nil {
				m.headless.close()
				m.headless = nil
			}
			if m.calibration != nil {
				job := m.calibration
				m.calibration = nil
				m.Error = context.Canceled
				return m, tea.Sequence(func() tea.Msg { job.stop(); return nil }, tea.Interrupt)
			}
			if m.chatStop != nil {
				stop := m.chatStop
				m.chatStop = nil
				m.Error = context.Canceled
				return m, tea.Sequence(func() tea.Msg { _ = stop(); return nil }, tea.Interrupt)
			}
			m.Error = context.Canceled
			return m, tea.Interrupt
		}
		if (key == "d" || key == "v") && (m.screen() == Detect || m.screen() == Home || m.screen() == Calibrate || m.screen() == Setup || m.screen() == Ready || m.rawError != "") {
			m.expanded = !m.expanded
			return m, nil
		}
		if m.screen() == Trial && m.address != "" && key == "esc" {
			m.address = ""
			return m, m.finishChat()
		}
		if m.screen() == Chat {
			if key == "esc" {
				return m, m.finishChat()
			}
			return m, m.chat.update(v)
		}
		if m.screen() == Paste && m.busy {
			return m, nil
		}
		if m.screen() == Paste {
			if key == "esc" {
				if m.headless != nil {
					m.headless.close()
					m.headless = nil
				}
				m.authURL = ""
				m.secret = ""
				m.back()
				return m, nil
			}
			if key == "enter" {
				if m.headless != nil {
					login := m.headless
					secret := m.secret
					m.busy = true
					return m, func() tea.Msg { return authorizedMsg{login.finish(m.cfg.Context, secret)} }
				}
				return m, m.authorize(m.secret)
			}
			_, cmd := m.password.Update(msg)
			return m, cmd
		}
		if m.screen() == Search {
			if key == "esc" {
				m.back()
				return m, nil
			}
			if key == "enter" {
				if core.IsClaimText(m.search.Value()) {
					m.message = "Search by model name"
					return m, nil
				}
				m.cfg.Preferences.Model = m.search.Value()
				m.replace(Recipes)
				return m, m.find()
			}
			var c tea.Cmd
			m.search, c = m.search.Update(msg)
			return m, c
		}
		if key == "?" && m.screen() == Home {
			m.push(Help)
			return m, nil
		}
		if key == "q" && !m.busy {
			return m, tea.Quit
		}
		if key == "esc" {
			if m.screen() == Recipes && m.gridFocus {
				m.gridFocus = false
				return m, nil
			}
			if !m.busy {
				m.back()
			}
			return m, nil
		}
		if m.busy {
			return m, nil
		}
		switch m.screen() {
		case Detect:
			if key == "enter" && !m.detecting {
				return m, m.afterDetect()
			}
		case Agents:
			switch key {
			case "up":
				m.agentIndex = max(0, m.agentIndex-1)
			case "down":
				m.agentIndex = min(len(m.agents)-1, m.agentIndex+1)
			case "space", " ":
				m.agents[m.agentIndex].Checked = !m.agents[m.agentIndex].Checked
			case "enter":
				return m, m.afterAgents()
			}
		case Calibrate:
			if key == "s" {
				m.choice("skipped")
				m.replace(Recipes)
				return m, m.find()
			}
			if key == "enter" {
				if m.cfg.Power != nil || m.cfg.ModelServers != nil {
					m.busy = true
					low := m.detection.System != nil && m.detection.System.Power.LowPower
					return m, func() tea.Msg {
						if m.cfg.Power != nil {
							low = m.cfg.Power(m.cfg.Context)
						}
						return powerMsg{low: low, servers: m.readServers()}
					}
				}
				return m, m.beginCalibration(m.detection.System != nil && m.detection.System.Power.LowPower)
			}

		case First:
			switch key {
			case "enter":
				if m.cfg.Store.Key() != "" {
					return m, m.startTrial()
				}
				m.pendingTrial = true
				return m, m.authorize("")
			case "tab":
				tools := core.TrialAgents(m.cfg.Preferences)
				for i, id := range tools {
					if id == m.cfg.Preferences.Tool {
						m.cfg.Preferences.Tool = tools[(i+1)%len(tools)]
						break
					}
				}
			case "p":
				m.pendingTrial = true
				m.push(Paste)
				m.password = huh.NewInput().Title("OpenRouter key").Value(&m.secret).EchoMode(huh.EchoModePassword)
				m.password.WithTheme(m.formTheme())
				m.password.WithWidth(max(24, m.width-4))
				m.password.WithKeyMap(huh.NewDefaultKeyMap())
				return m, m.password.Focus()
			}
		case Home:
			switch key {
			case "s", "enter":
				if key == "s" || m.detecting {
					return m, nil
				}
				if m.modelIndex >= len(m.models) {
					m.push(Recipes)
					return m, m.find()
				}
				choice := m.models[m.modelIndex]
				if choice.Journal != nil {
					return m, m.launchSetup(*choice.Journal)
				}
				return m, m.planSetup()
			case "up", "k":
				m.modelIndex = max(0, m.modelIndex-1)
			case "down", "j":
				m.modelIndex = min(len(m.models), m.modelIndex+1)
			case "tab":
				m.cfg.Preferences.Tool = m.nextTool()
				if m.detection.Target.OS == "darwin" {
					return m, m.loadModels()
				}
			case "t":
				return m, m.try(false)
			case "/":
				return m, m.openSearch()
			case "c":
				m.push(Gallery)
			}
		case Recipes:
			return m, m.pickerKey(key)
		case Details:
			switch key {
			case "enter":
				return m, m.try(false)
			case "w":
				return m, m.try(true)
			case "s":
				return m, m.planSetup()
			case "/":
				return m, m.openSearch()
			case "?":
				m.push(Why)
			case "c":
				m.push(Gallery)
			case " ", "space":
				if m.cfg.Preferences.Boosts == "off" {
					m.cfg.Preferences.Boosts = "auto"
				} else {
					m.cfg.Preferences.Boosts = "off"
				}
				return m, m.find()
			}
		case Summary:
			switch key {
			case "1", "2", "3":
				m.last.Rating = int(key[0] - '0')
				if e := m.cfg.Store.SaveTrial(m.last); e != nil {
					m.fail("Could not save that choice. Press enter to retry or d for details.", e)
					return m, nil
				}
				if key == "1" {
					m.firstRun = false
					m.cfg.Preferences.Priority = store.TowardSpeed(m.cfg.Preferences.Priority)
					if e := m.cfg.Store.SavePreferences(m.cfg.Preferences); e != nil {
						m.fail("Could not save that choice. Press enter to retry or d for details.", e)
						return m, nil
					}
					m.replace(Recipes)
					return m, m.find()
				}
				m.message = "Rating saved."
			case "t", "m":
				m.replace(Recipes)
				return m, m.find()
			case "s", "enter":
				return m, m.planSetup()
			}
		case Setup:
			if (key == "enter" || key == "y" || key == "Y") && m.setup.plan.ID != "" && m.cfg.Apply != nil {
				return m, m.applySetup()
			}
			if key == "n" {
				m.back()
			}
		case Ready:
			if key == "u" {
				m.busy = true
				return m, tea.ExecProcess(m.child([]string{"setup", m.setup.journal.Plan.Slug, "--undo", "--plain"}), func(e error) tea.Msg { return childMsg{kind: Setup, err: e} })
			}
			if key == "enter" {
				return m, m.launchSetup(m.setup.journal)
			}
		case Gallery:
			if key == "down" || key == "j" {
				m.selected = min(m.selected+1, len(Claims())-1)
			}
			if key == "up" || key == "k" {
				m.selected = max(0, m.selected-1)
			}
			if key == "enter" || key == "?" {

				m.push(ClaimCard)
			}
		case ClaimCard:
			if key == "?" {
				m.push(Why)
			}
		}
	}
	return m, nil
}
func (m *Root) openSearch() tea.Cmd { m.push(Search); return m.search.Focus() }
func (m *Root) authorize(secret string) tea.Cmd {
	if m.cfg.Headless && m.screen() == First && secret == "" {
		m.busy = true
		return func() tea.Msg {
			h, e := beginHeadless(m.cfg.Store, openrouter.Client{}, "")
			return headlessReadyMsg{h, e}
		}
	}
	if m.cfg.Authorize == nil {
		return nil
	}
	m.busy = true
	return func() tea.Msg { return authorizedMsg{m.cfg.Authorize(m.cfg.Context, secret)} }
}
func (m *Root) footer() string {
	switch m.screen() {
	case Detect:
		return "⏎ Continue    v Details    q Quit"
	case Agents:
		return "⏎ Continue    space Toggle    ↑↓ Move"
	case Calibrate:
		if m.busy {
			return "ctrl-c Pause   v Details"
		}
		return "⏎ Now   s Later (run trialm calibrate)   v Details"
	case Recipes:
		if m.cfg.Picker != nil {
			if r, _ := m.row(); m.gridFocus && !hasSim(r) {
				return strings.Replace(pick.GridKeys, "⏎ Try   ", "", 1)
			}
			if m.gridFocus {
				return pick.GridKeys
			}
			return pick.ListKeys
		}
	}
	if m.screen() == Setup {
		if m.busy {
			return "ctrl-c Pause    v Details"
		}
		return "⏎ Install    esc Back    v Details"
	}
	if m.screen() == Ready {
		return "⏎ Open " + toolName(m.cfg.Preferences.Tool) + " now    q Later    v Details"
	}
	if m.busy || (m.screen() == Chat && m.chat.busy) {
		return "ctrl-c stop"
	}
	if m.screen() == Recipes && len(m.listing.Rows) == 0 {
		return "/ Search    esc Back    q Quit"
	}
	switch m.screen() {
	case First:
		tool := toolName(m.cfg.Preferences.Tool)
		if m.cfg.Preferences.Tool == "generic" {
			tool = "Your own agent"
		}
		hint := "⏎ Try in " + tool
		if m.cfg.Store.Key() == "" {
			hint += "    p Paste key"
		}
		return hint + "    tab Agent    esc Back"
	case Paste:
		return "⏎ Connect    esc Back"

	case Recipes:
		return "enter try  s set up  d details  w window  n needs  / search"
	case Details:
		return "⏎ Try    s Set up    ? Sources    esc Back    q Quit"
	case Home:
		hint := ""
		if next := m.nextTool(); next != m.cfg.Preferences.Tool {
			hint = "    tab " + toolName(next)
		}
		if r, ok := m.row(); !ok || hasSim(r) {
			hint += "    t Cloud trial"
		}
		return "⏎ Run in " + toolName(m.cfg.Preferences.Tool) + hint + "    / Search    ? Help"
	case Summary:
		return "⏎ Install    m Other models    q Later"
	case Gallery:
		return "↑↓ move  enter claim  esc back  q quit"
	case ClaimCard:

		return "? sources  esc back  q quit"
	case Search:
		return "enter search  esc back"
	case Trial:
		return "ctrl-c stop"
	case Chat:
		return "enter send  esc finish  ctrl-c stop"
	default:
		return "esc back  q quit"
	}
}
func (m *Root) View() tea.View {
	body := m.brandLine() + "\n" + m.body()
	if (m.busy || (m.screen() == Chat && m.chat.busy)) && m.calibration == nil && m.screen() != Home && m.screen() != Trial && m.screen() != Setup {
		body += "\n" + m.waiting() + " Working…"
	}
	if m.message != "" {
		body += "\n" + m.theme.caveat.Render(m.message)
	}
	if m.expanded && m.rawError != "" {
		body += "\n" + wrap(m.rawError, m.width-4)
	}
	body += "\n\n" + m.helpView()
	if m.theme.plain {
		body = strip(body)
	}
	return tea.NewView(m.fit(m.theme.glyph(body)))
}
func (m *Root) fit(s string) string {
	// All cells are composed before clipping; terminal controls and pasted secrets
	// never enter this view. ANSI-aware clipping keeps the renderer at the margin.
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		if m.screen() == Recipes || m.screen() == Chat {
			lines[i] = clip(l, m.width)
		} else {
			lines[i] = wrap(l, m.width)
		}
	}
	return strings.Join(lines, "\n")
}

var _ tea.Model = (*Root)(nil)

func (m *Root) beginWindow(args []string) tea.Cmd {
	m.busy = true
	return tea.Batch(func() tea.Msg { return windowDoneMsg{m.cfg.Window(m.cfg.Context, args)} }, tea.Tick(time.Second, func(t time.Time) tea.Msg { return pulseMsg(t) }))
}

func terminalResult(err, modelError error) error {
	if errors.Is(err, tea.ErrInterrupted) {
		return context.Canceled
	}
	if err != nil {
		return err
	}
	return modelError
}
