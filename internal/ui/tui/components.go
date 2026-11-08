package tui

import (
	"charm.land/lipgloss/v2"
	"strings"

	"github.com/aeroknight786/trialm/internal/figure"
	"github.com/aeroknight786/trialm/internal/ui/tui/chrome"
)

func (m *Root) helpView() string {
	return m.theme.dim.Render(chrome.Footer(m.theme.glyph(m.footer()), m.width))
}
func selection(i, selected int) string {
	if i == selected {
		return "❯ "
	}
	return "  "
}

func (m *Root) brandLine() string {
	b := m.theme.wordmark()
	t := m.detection.Target
	machine := t.Device.Name
	if sys := m.detection.System; sys != nil && sys.AppleChip != "" {
		machine = sys.AppleChip
	}
	if machine != "" {
		budget := strings.TrimSuffix(figure.Exact(t.GPUMemBytes/(1<<30), "GB", figure.Measured, "hardware:budget").Number(), ".0")
		context := toolName(m.cfg.Preferences.Tool) + " · " + machine + " · " + budget + " GB for models"
		b += m.theme.dim.Render("  " + context)
	}
	if pad := min(m.width, 80) - lipgloss.Width(b) - 7; pad > 1 { // the company mark, so the name sticks
		b += strings.Repeat(" ", pad) + lipgloss.NewStyle().Bold(!m.theme.plain).Render(m.theme.glyph("/mortiϕ"))
	}
	return clip(b, m.width)
}
