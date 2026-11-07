// Package chrome composes the shared single-line identity and key hints.
package chrome

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

func Brand(text string, plain bool) string {
	if plain {
		return text
	}
	colors := lipgloss.Blend1D(len([]rune(text)), lipgloss.Color("#3fb950"), lipgloss.Color("#39c5cf"), lipgloss.Color("#d2a8ff"))
	var b strings.Builder
	for i, r := range []rune(text) {
		b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(colors[i]).Render(string(r)))
	}
	return b.String()
}

// Footer wraps whole key hints. The spaces inside a hint stay together.
func Footer(hints string, width int) string {
	var lines []string
	line := ""
	for _, key := range strings.Split(hints, "  ") {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if line != "" && ansi.StringWidth(line)+3+ansi.StringWidth(key) > width {
			lines = append(lines, line)
			line = ""
		}
		if line != "" {
			line += "   "
		}
		line += key
	}
	return strings.Join(append(lines, line), "\n")
}
