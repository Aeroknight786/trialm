package tui

import (
	"fmt"
	"slices"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/aeroknight786/trialm/internal/core"
	"github.com/aeroknight786/trialm/internal/harness"
)

// nextTool is the agent tab switches to on the main screen (the same one
// when there is no other).
// agents are the ones tab moves through: those found on this Mac, then
// "Your own agent" (one local address for any app).
func (m *Root) agentCycle() []string { return append(slices.Clone(m.cfg.Tools), "generic") }
func (m *Root) nextTool() string {
	all := m.agentCycle()
	return all[(slices.Index(all, m.cfg.Preferences.Tool)+1)%len(all)]
}

func toolName(id string) string {
	if id == "generic" {
		return "Your own agent"
	}
	return strings.TrimSuffix(harness.DisplayName(id), " CLI")
}
func (m *Root) card(text string) string {
	border := lipgloss.RoundedBorder()
	if m.theme.ascii {
		border = lipgloss.ASCIIBorder()
	}
	return lipgloss.NewStyle().Border(border).BorderForeground(m.theme.selected.GetForeground()).Padding(0, 1).Width(max(16, min(m.width, 80)-2)).Render(wrap(text, max(10, min(m.width, 80)-6)))
}
func (m *Root) waiting() string {
	if m.theme.plain || m.theme.ascii {
		return "..."
	}
	return m.theme.selected.Render(m.spin.View())
}
func (m *Root) launcher() string {
	b := ""
	if m.detection.System != nil && m.detection.System.Power.LowPower {
		b += m.theme.caveat.Render("  Low Power Mode | Expect Slower Speeds") + "\n"
	}
	if m.detecting {
		lines := []string{"machine", "coding tools", "engines", "models on disk"}
		for i, line := range lines {
			mark := m.waiting()
			if (i < 3 && m.detected) || i == 1 {
				mark = m.theme.good.Render("✓")
			}
			b += "\n  " + mark + " " + line
		}
		return b
	}
	yours := slices.ContainsFunc(m.models, func(c core.ModelChoice) bool { return c.Journal != nil || c.Row.LocalPath != "" })
	title := map[bool]string{true: "Your models", false: "Picks for this Mac"}[yours]
	b += "\n  " + m.theme.dim.Render("Agent") + "  "
	for _, id := range m.agentCycle() {
		b += map[bool]string{true: m.theme.cloud.Render("[" + toolName(id) + "]"), false: m.theme.dim.Render(toolName(id))}[id == m.cfg.Preferences.Tool] + "  "
	}
	b += m.theme.dim.Render(" tab") + "\n"
	b += "\n  " + m.theme.selected.Bold(true).Render(title) + "\n"
	for i, choice := range m.models {
		mark, state, style := "○", choice.Download.String()+" download", m.theme.dim
		speed := estimated(choice.Speed)
		if choice.Journal != nil {
			mark, state, style = "●", "ready", m.theme.good
			speed = choice.Speed.String()
		} else if choice.Row.LocalPath != "" {
			mark, state, style = "◐", "on disk · not set up", m.theme.selected
		}
		if m.detection.Target.OS == "darwin" {
			speed = estimated(choice.Reply, true) + " per reply"
		}
		if choice.Journal == nil && core.HasSim(choice.Row) {
			state = "SIM · " + state
		}
		nameWidth := max(12, min(25, m.width-39))
		line := fmt.Sprintf("%s%s %-*s %10s   %s", selection(i, m.modelIndex), mark, nameWidth, clip(choice.Row.Result.Model.Name, nameWidth), speed, state)
		if i == m.modelIndex && !m.theme.plain {
			style = style.Background(lipgloss.Color("#182b30"))
		}
		b += "  " + style.Render(line) + "\n"
	}
	if m.detection.Target.OS == "darwin" && m.start > 0 {
		b += "  " + m.theme.dim.Render(core.Assumes(m.cfg.Preferences.Tool, m.start)) + "\n"
	}
	b += "\n  " + m.theme.cloud.Render(selection(len(m.models), m.modelIndex)+"✦ Find something faster or smarter")
	if m.notice != "" {
		b += "\n\n  " + m.theme.dim.Render(m.notice)
	}
	return b
}
