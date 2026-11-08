package tui

import (
	"context"
	"fmt"
	"math"
	"os/exec"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/aeroknight786/trialm/internal/catalog"
	"github.com/aeroknight786/trialm/internal/core"
	"github.com/aeroknight786/trialm/internal/engine"
	"github.com/aeroknight786/trialm/internal/figure"
	"github.com/aeroknight786/trialm/internal/hw"
	"github.com/aeroknight786/trialm/internal/setup"
	"github.com/aeroknight786/trialm/internal/store"
	"github.com/aeroknight786/trialm/internal/ui/tui/chrome"
	"github.com/aeroknight786/trialm/internal/verify"
)

type adminRequest struct {
	ctx  context.Context
	argv []string
	done chan error
}

type setupEvent struct {
	servers []hw.ModelServer
	event   core.Event
	journal setup.Journal
	err     error
	done    bool
}
type setupView struct {
	plan                   setup.Plan
	journal                setup.Journal
	report                 *verify.Report
	update                 setup.Update
	states                 map[int]string
	logs                   []string
	events                 chan setupEvent
	cancel                 context.CancelFunc
	onDisk                 bool
	checkStarted, checkNow time.Time
	subCheck               string
}

func (m *Root) planSetup() tea.Cmd {
	row, ok := m.row()
	if m.screen() == Summary {
		if m.trialRow != nil {
			row, ok = *m.trialRow, true
		}
		if model, found := catalog.Lookup(m.last.Model); found {
			row.Result.Model = model
			row.Result.Quant.Name = m.last.Quant
			row.Result.Runtime, _ = engine.Get(m.last.Runtime)
			if m.last.Context > 0 {
				row.Result.Settings.ContextTokens = m.last.Context
				row.Result.Settings.KVCacheType = m.last.KV
			}
		}
	}
	if !ok || m.cfg.Plan == nil {
		return nil
	}
	m.push(Setup)
	m.busy = true
	m.setup = setupView{states: map[int]string{}, onDisk: row.LocalPath != ""}
	return func() tea.Msg {
		p, e := m.cfg.Plan(m.cfg.Context, row, m.cfg.Preferences)
		executor, _ := (core.Session{}).SetupExecutor()
		var j setup.Journal
		if executor != nil {
			j, _ = executor.Status(p.Slug)
		}
		return setupEvent{servers: m.readServers(), journal: j, event: core.Event{Type: "setup_plan", Data: map[string]any{"plan": p}}, err: e}
	}
}
func (m *Root) applySetup() tea.Cmd {
	ctx, cancel := context.WithCancel(m.cfg.Context)
	m.setup.checkStarted = time.Time{}
	m.setup.cancel = cancel
	m.setup.events = make(chan setupEvent, 32)
	events := m.setup.events
	plan := m.setup.plan
	m.busy = true
	go func() {
		j, e := m.cfg.Apply(ctx, plan, func(ev core.Event) { events <- setupEvent{event: ev} })
		events <- setupEvent{journal: j, err: e, done: true}
		close(events)
		cancel()
	}()
	return m.nextSetup()
}
func (m *Root) nextSetup() tea.Cmd {
	ch := m.setup.events
	return func() tea.Msg { v := <-ch; return v }
}
func (m *Root) setupEvent(v setupEvent) tea.Cmd {
	if v.done {
		m.busy = false
		m.setup.checkStarted = time.Time{}
		m.setup.cancel = nil
		m.setup.journal = v.journal
		if v.journal.Report != nil {
			m.setup.report = v.journal.Report
		}
		if v.err != nil {
			m.fail("Setup needs another try. Press enter to resume or d for details.", v.err)
		} else if v.journal.State == "ready" && v.journal.Report != nil && v.journal.Report.Ready {
			m.replace(Ready)
			m.sparkle = 0
			return tea.Tick(150*time.Millisecond, func(time.Time) tea.Msg { return sparkleMsg(1) })
		}
		return nil
	}
	if v.err != nil {
		m.busy = false
		m.fail("Setup could not be planned. Press esc to choose another model or d for details.", v.err)
		return nil
	}
	switch v.event.Type {
	case "setup_admin":
		request := v.event.Data.(adminRequest)
		cmd := exec.CommandContext(request.ctx, request.argv[0], request.argv[1:]...)
		return tea.ExecProcess(cmd, func(err error) tea.Msg { request.done <- err; return setupEvent{} })
	case "setup_plan":
		m.otherModels = v.servers
		data, _ := v.event.Data.(map[string]any)
		m.setup.plan, _ = data["plan"].(setup.Plan)
		m.setup.journal = v.journal
		m.busy = false
	case "setup_status", "setup_paused":
		m.setup.journal, _ = v.event.Data.(setup.Journal)
	case "setup_step", "setup_progress", "setup_log":
		u, _ := v.event.Data.(setup.Update)
		if m.setup.states == nil {
			m.setup.states = map[int]string{}
		}
		if u.Index > 0 && v.event.Type != "setup_log" {
			m.setup.update = u
			m.setup.states[u.Index] = u.State
		}
		if u.Kind == setup.Check && u.State == "running" && m.setup.checkStarted.IsZero() {
			m.setup.checkStarted = time.Now()
			m.setup.checkNow = m.setup.checkStarted
			m.setup.subCheck = "Writing"
			return tea.Batch(m.nextSetup(), tea.Tick(time.Second, func(t time.Time) tea.Msg { return pulseMsg(t) }))
		}
		m.setup.logs = append(m.setup.logs, u.Lines...)
		if len(m.setup.logs) > 5 {
			m.setup.logs = m.setup.logs[len(m.setup.logs)-5:]
		}
	case "setup_check":
		m.setup.subCheck, _ = v.event.Data.(string)
	case "setup_ready":
		m.setup.report, _ = v.event.Data.(*verify.Report)
	}
	if m.setup.events != nil {
		return tea.Batch(m.nextSetup(), m.animateProgress())
	}
	return nil
}
func (m *Root) launchSetup(j setup.Journal) tea.Cmd {
	m.push(Launching)
	argv := []string{"launch", m.cfg.Preferences.Tool, "--setup", j.Plan.Slug, "--here", "--plain"}
	if m.cfg.Preferences.Tool == "generic" {
		argv = append(argv, "--serve")
	} else {
		argv = append(argv, "--tui-handoff")
	}
	cmd := m.child(argv)
	m.busy = true
	return tea.ExecProcess(cmd, func(e error) tea.Msg { return childMsg{kind: Launching, err: e} })
}
func (m *Root) animateProgress() tea.Cmd {
	if m.theme.plain || m.theme.ascii {
		return nil
	}
	if u := m.setup.update; u.Progress != nil && u.Progress.Total > 0 {
		return m.bar.SetPercent(float64(u.Progress.Done) / float64(u.Progress.Total))
	}
	return nil
}
func (m *Root) setupBody() string {
	p := m.setup.plan
	if p.ID == "" {
		return m.card(m.waiting() + " Planning your setup")
	}
	b := "Setting up " + p.ModelName + " for " + toolName(m.cfg.Preferences.Tool) + "\n"
	if p.DownloadUnknown {
		b += "\nInstaller download size unknown\n"
	}
	if !m.busy && m.setup.journal.State != "ready" {
		b += "\nInstall? " + p.CountsLine(m.setup.onDisk) + "\n"
	}
	if p.Prompts.Passwords > 0 {
		b += "\nNeeds your Mac password once\n"
	}
	checkSuffix := ""
	if m.busy && !m.setup.checkStarted.IsZero() {
		left := core.ReadyDuration(p) - m.setup.checkNow.Sub(m.setup.checkStarted)
		seconds := int(math.Ceil(max(time.Minute, left).Seconds()))
		checkSuffix += fmt.Sprintf(" · About %d min", seconds/60)
		if seconds%60 != 0 {
			checkSuffix += fmt.Sprintf(" %d s", seconds%60)
		}
		checkSuffix += " left"
	}
	line := func(title, state string) {
		mark := "·"
		if state == "done" {
			mark = m.theme.good.Render("✓")
		} else if state != "" {
			mark = m.waiting()
		}
		b += "\n" + mark + " " + title
	}
	checkLine := func(title, state string) {
		line(title+checkSuffix, state)
		if m.busy && state == "running" {
			b += "\n  " + m.theme.dim.Render(m.setup.subCheck)
		}
	}
	if m.expanded {
		if m.setup.onDisk {
			line("Model already on disk", "done")
		}
		for i, s := range p.Steps {
			if s.Kind == setup.Check {
				if warning := m.modelWarning(); warning != "" {
					b += "\n\n" + warning
				}
				checkLine(s.Description(true), m.setup.states[i+1])
			} else {
				line(s.Description(true), m.setup.states[i+1])
			}
		}
		b += "\n" + strings.Join(append(append([]string{}, m.setup.logs...), p.Warnings...), "\n")
		return m.card(b)
	}
	download, started, checked := "", "", ""
	hasDownload := false
	for i, s := range p.Steps {
		state := m.setup.states[i+1]
		switch s.Kind {
		case setup.Install, setup.RunInstaller:
			if s.Package == "ollama-model" {
				continue
			}
			title := "Install the engine"
			if s.Request != nil {
				title += " (" + figure.Band(float64(s.Request.Size)/1e6, 20, "MB", figure.Simulated, "engine:asset").String() + ")"
			}
			line(title, state)
		case setup.Fetch:
			hasDownload = true
			if state != "" {
				download = state
			}
		case setup.Start:
			if s.ID != "bridge" && state != "" {
				started = state
			}
		case setup.Check:
			checked = state
		case setup.Calibrate:
			title := "Calibrate"
			if u := m.setup.update; u.Kind == setup.Calibrate {
				title += " " + figure.Exact(u.Fraction*100, "%", figure.Measured, "setup:calibrate").String()
			}
			line(title, state)
		case setup.Admin:
			line(s.Title, state)
		}
	}
	if m.setup.onDisk {
		line("Model already on disk", "done")
	} else if hasDownload {
		line("Download the model", download)
		if u := m.setup.update; u.Kind == setup.Fetch && u.Progress != nil && u.Progress.Total > 0 {
			progress := u.Progress
			b += "\n  " + m.bar.ViewAs(float64(progress.Done)/float64(progress.Total))
			b += "\n  " + figure.Exact(float64(progress.Done)/1e9, "", figure.Measured, "download:progress").Number() + " of " + figure.Exact(float64(progress.Total)/1e9, "GB", figure.Measured, "download:progress").String() + " · " + timeLeft(progress.ETA)
		}
	}
	line("Start and connect "+toolName(m.cfg.Preferences.Tool), started)
	if warning := m.modelWarning(); warning != "" {
		b += "\n\n" + warning
	}
	checkLine("Check speed", checked)
	if m.setup.journal.State == "paused" {
		b += "\n" + m.setup.journal.Summary()
	}
	return m.card(b)
}
func timeLeft(d time.Duration) string {
	if d <= time.Minute {
		return "Under a minute left"
	}
	return fmt.Sprintf("About %.0f min left", math.Ceil(d.Minutes()))
}
func (m *Root) readyBody() string {
	p, r := m.setup.plan, m.setup.report
	b := chrome.Brand("✦ Ready", m.theme.plain || m.theme.ascii) + "\n\n" + p.ModelName + " · " + toolName(p.Harness) + fmt.Sprintf(" · %dK context", p.ContextTokens/1024)
	if r == nil {
		return b
	}
	reply, first, estReply, estFirst := core.ReadyTimes(p, r)
	status := "Check needed"
	for _, c := range r.Checks {
		if c.Name == "harness" && c.OK {
			status = "Works"
		}
	}
	b += "\n\n" + m.tiles([]string{"Per reply", "First reply", toolName(p.Harness)}, []string{estimated(reply), estimated(first), status}, []string{"est. " + strings.TrimPrefix(estReply.Number(), "~"), "est. " + strings.TrimPrefix(estFirst.Number(), "~"), "Chat · Tools · Follow-up"})
	if warning := m.modelWarning(); warning != "" {
		b += "\n\n" + warning
	}
	b += "\n\nNext time: trialm"
	if m.expanded {
		b += "\n\n" + r.Plain() + "\n" + strings.Join(m.setup.logs, "\n") + "\nReply times use measured speeds and the agent's reply length"
	}
	return b
}

func RunSetup(ctx context.Context, p setup.Plan) error {
	s := store.Default()
	prefs, _ := s.Preferences()
	prefs.Context = p.ContextTokens
	if p.Harness != "" {
		prefs.Tool = p.Harness
	}
	cfg := SessionConfig(ctx, s, prefs, core.Options{}, 5, "")
	cfg.Start = Setup
	cfg.LocalPlan = &p
	cfg.UpdateNotice = nil
	return Run(cfg)
}
