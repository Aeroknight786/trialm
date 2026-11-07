package tui

import (
	tea "charm.land/bubbletea/v2"
	"fmt"
	"github.com/aeroknight786/trialm/internal/core"
	"github.com/aeroknight786/trialm/internal/figure"
	"strings"
	"time"
)

func (m *Root) afterDetect() tea.Cmd {
	if len(m.cfg.Tools) == 1 {
		for i := range m.agents {
			m.agents[i].Checked = m.agents[i].ID == m.cfg.Tools[0]
		}
		cmd := m.afterAgents()
		m.message = "Using " + toolName(m.cfg.Tools[0])
		return cmd
	}
	m.replace(Agents)
	return nil
}
func (m *Root) afterAgents() tea.Cmd {
	if e := core.SaveAgents(m.cfg.Store, &m.cfg.Preferences, m.agents); e != nil {
		m.fail("Could not save agents", e)
		return nil
	}
	m.cfg.Preferences.FirstRunDone = true
	if e := m.cfg.Store.SavePreferences(m.cfg.Preferences); e != nil {
		m.fail("Could not save agents", e)
		return nil
	}
	m.replace(Calibrate)
	return tea.Batch(m.scanServers(), func() tea.Msg {
		return calibrationPresenceMsg(core.CurrentCalibration(m.cfg.Context, m.cfg.Store.Dir) != nil)
	})

}
func (m *Root) pickerKey(key string) tea.Cmd {
	if len(m.picker.Models) == 0 {
		if key == "/" {
			return m.openSearch()
		}
		return nil
	}
	if m.gridFocus {
		switch key {
		case "up", "down", "left", "right", "c", "e":
			m.picker.Models[m.selected].Select(key)
		case "enter":
			return m.try(false)
		case "s":
			return m.planSetup()
		}
		return nil
	}
	switch key {
	case "up", "k":
		m.selected = max(0, m.selected-1)
	case "down", "j":
		m.selected = min(len(m.picker.Models)-1, m.selected+1)
	case "v", "d":
		m.push(Details)
	case "enter":
		m.gridFocus = true
	case "/":
		return m.openSearch()
	}
	return nil
}
func (m *Root) detectBody() string {
	b := ""
	if m.detecting {
		for _, s := range []string{"Machine", "Coding tools", "Models on disk", "Engine"} {
			b += "\n" + m.waiting() + " " + s
		}
		return b
	}
	mark := m.theme.good.Render("✓") + "  "
	t := m.detection.Target
	machine := t.Device.Name
	if sys := m.detection.System; sys != nil && sys.AppleChip != "" {
		machine = sys.AppleChip
		if len(sys.GPUs) > 0 {
			machine += fmt.Sprintf(" · %d-core GPU", sys.GPUs[0].Cores)
		}
	}
	b += "\n" + mark + machine + " · " + strings.TrimSuffix(figure.Exact(t.RAMBytes/(1<<30), "GB", figure.Measured, "hardware:ram").Number(), ".0") + " GB memory"
	var names []string
	for _, id := range m.cfg.Tools {
		names = append(names, toolName(id))
	}
	tools := "None found"
	if len(names) > 0 {
		tools = strings.Join(names, ", ")
	}
	b += "\n" + mark + "Coding tools: " + tools
	var local []string
	for _, c := range m.models {
		if c.Row.LocalPath != "" || c.Journal != nil {
			local = append(local, c.Row.Result.Model.Name+" ("+figure.Bytes(c.Row.Result.Quant.FileBytes, "catalog:file").String()+")")
		}
	}
	onDisk := "None yet"
	if len(local) > 0 {
		onDisk = strings.Join(local, ", ")
	}
	b += "\n" + mark + "Already on your Mac: " + onDisk
	var engines []string
	for _, r := range m.detection.Runtimes {
		if n := core.RuntimeTitles[r.ID]; n != "" {
			engines = append(engines, n)
		}
	}
	engine := "None yet · triaLM brings its own"
	if m.detection.SharedEngine {
		engine = "llama.cpp"
	}
	if len(engines) > 0 {
		engine = strings.Join(engines, ", ")
	}
	b += "\n" + mark + "Engine: " + engine
	if m.expanded {
		b += "\n" + strings.Join(t.Notes, "\n")
	}
	b += "\n\n" + m.theme.good.Render(strings.TrimSuffix(figure.Exact(t.GPUMemBytes/(1<<30), "GB", figure.Measured, "hardware:budget").Number(), ".0")+" GB for models")
	return b
}
func (m *Root) agentsBody() string {
	b := "Agents\n"
	for i, a := range m.agents {
		check := "[ ]"
		if a.Checked {
			check = "[✓]"
		}
		desc := "Not installed"
		if a.Detected {
			desc = "Found on this Mac"
		}
		if a.ID == "generic" {
			desc = "One local address for any app"
		}
		line := selection(i, m.agentIndex) + check + " " + a.Name + "  " + desc
		if !a.Detected {
			line = m.theme.dim.Render(line)
		} else if a.Checked {
			line = m.theme.good.Render(line)
		}
		b += "\n" + line
	}
	return b
}
func (m *Root) calibrationBody() string {
	b := m.theme.selected.Bold(true).Render(fmt.Sprintf("✦ Calibrate for Accurate Estimates | ~%d mins | No Downloads", core.CalibrationMinutes()))
	if m.calibration != nil {
		// The bar is the real share of estimated time, so it agrees with "About N min left".
		b += "\n\n" + m.spin.View() + " Calibrating\n" + m.bar.ViewAs(m.calibration.Fraction) + "\n" + m.calibration.ETA + m.theme.dim.Render(fmt.Sprintf(" · %d of %d", m.calibration.Step, len(m.calibration.Estimates)))
	}
	if m.detection.System != nil && m.detection.System.Power.LowPower {
		b += "\n\n" + m.theme.caveat.Render("Low Power Mode | Expect Slower Speeds")
	}
	if warning := m.modelWarning(); warning != "" {
		b += "\n\n" + warning
	}
	return m.card(b)
}

func (m *Root) beginCalibration(low bool) tea.Cmd {
	if low {
		m.message = "Turn off Low Power Mode to calibrate"
		return nil
	}
	if e := m.choice("accepted"); e != nil {
		m.fail("Could not save calibration choice", e)
		return nil
	}
	m.message = ""
	m.busy = true
	m.started = time.Now()
	m.calibration = newCalibration(m.cfg.Context, m.cfg.Executable)
	return tea.Batch(m.calibration.next(), tea.Tick(time.Second, func(t time.Time) tea.Msg { return pulseMsg(t) }))
}
