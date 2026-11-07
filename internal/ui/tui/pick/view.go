// Package pick draws immutable predictor tables supplied by core.
package pick

import (
	"charm.land/lipgloss/v2"
	"fmt"
	"github.com/aeroknight786/trialm/internal/core"
	"github.com/aeroknight786/trialm/internal/figure"
	"github.com/charmbracelet/x/ansi"
	"math"
	"strings"
)

func Context(n int) string {
	if n >= 1048576 {
		return fmt.Sprintf("%dM", n/1048576)
	}
	return fmt.Sprintf("%dK", n/1024)
}
func number(f figure.Figure) string   { return f.Number() }
func estimate(f figure.Figure) string { return "~" + strings.TrimPrefix(f.String(), "~") }
func family(name string) (string, string) {
	switch {
	case strings.HasPrefix(name, "Qwen"):
		return "Qwen", "#39c5cf"
	case strings.HasPrefix(name, "Gemma"):
		return "Gemma", "#58a6ff"
	case strings.HasPrefix(name, "gpt-oss"):
		return "gpt-oss", "#ffa657"
	case strings.HasPrefix(name, "Granite"):
		return "Granite", "#d2a8ff"
	case strings.HasPrefix(name, "Llama"):
		return "Llama", "#ff7b72"
	}
	f, _, _ := strings.Cut(name, " ")
	return f, "#39c5cf"
}
func View(p *core.ConfigPicker, index, width, height int, plain bool) string {
	if p == nil || len(p.Models) == 0 {
		return "Models\nNo models fit this agent on this machine."
	}
	index = max(0, min(index, len(p.Models)-1))
	m := &p.Models[index]
	g := m.Grid()
	cell, ok := m.Cell()
	if !ok {
		return "Models\nNo configuration fits."
	}
	fam, col := family(m.Default.Result.Model.Name)
	style := func(text, color string) string {
		if plain {
			return text
		}
		return lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render(text)
	}
	dim := func(s string) string { return style(s, "#8b949e") }
	budget := strings.TrimSuffix(figure.Exact(p.Target.GPUMemBytes/(1<<30), "GB", figure.Measured, "hardware:budget").Number(), ".0") + " GB"
	b := fmt.Sprintf("Models  %s · %s · %s for models · %d of %d\n", strings.TrimSuffix(p.Harness.Name, " CLI"), p.Target.Device.Name, budget, index+1, len(p.Models))
	start := min(max(0, index-3), max(0, len(p.Models)-7))
	nameWidth := max(12, min(27, width-38))
	for i := start; i < min(start+7, len(p.Models)); i++ {
		x := p.Models[i]
		r := x.Default.Result
		cursor, mark := " ", "○"
		if i == index {
			cursor = "❯"
		}
		size := figure.Bytes(r.Quant.FileBytes, "catalog:file").String()
		if x.OnDisk {
			mark = "●"
			size = "on disk"
		}
		speed := estimate(core.PredictionMetric(r, r.DecodeTPS0, "t/s", false))
		if x.EmptyMeasured {
			speed = strings.TrimPrefix(speed, "~")
		}
		line := fmt.Sprintf("%s %s %-*s %-11s %-9s %s", cursor, style(mark, map[bool]string{true: "#3fb950", false: "#8b949e"}[x.OnDisk]), nameWidth, ansi.Truncate(r.Model.Name, nameWidth, ""), "up to "+Context(x.MaxContext), speed, size)
		marker := "│"
		if i-start == min(6, index*7/max(1, len(p.Models))) {
			marker = "█"
		}
		b += ansi.Truncate(line, max(1, width-2), "") + " " + dim(marker) + "\n"
	}
	tag := style("estimate", "#d29922")
	if cell.Measured {
		tag = style("measured", "#3fb950")
	}
	badge := fam
	if !plain {
		badge = lipgloss.NewStyle().Foreground(lipgloss.Color(col)).Reverse(true).Bold(true).Render(" " + fam + " ")
	}
	panel := badge + "  " + m.Default.Result.Model.Name + "  " + tag + "\n"
	toggles := func(labels []string, selected int) string {
		var s []string
		for i, l := range labels {
			if i == selected {
				if plain {
					l = "[" + l + "]"
				} else {
					l = lipgloss.NewStyle().Foreground(lipgloss.Color(col)).Reverse(true).Render(" " + l + " ")
				}
			}
			s = append(s, l)
		}
		return strings.Join(s, " ")
	}
	panel += "KV cache  " + toggles(m.CacheTypes, m.Cache) + "\n"
	var engines []string
	for _, e := range m.Engines {
		name := e.Name
		if e.ID == "llamacpp" {
			name = "llama.cpp"
		}
		engines = append(engines, name)
	}
	panel += "Engine    " + toggles(engines, m.Engine) + "\n\n"
	columns := len(g.Contexts)
	if width < 76 {
		columns = min(3, columns)
	}
	colStart := max(0, m.Column-columns+1)
	labelWidth := 20
	if width < 76 {
		labelWidth = 18
	}
	cellWidth := max(8, (width-6-labelWidth)/max(1, columns))
	row := func(labels []string, selected int, blocked []bool) string {
		var parts []string
		for i, l := range labels {
			w := cellWidth
			if i == 0 {
				w = labelWidth
			}
			if i == selected && plain {
				l = "[" + l + "]"
			}
			l = ansi.Truncate(l, w, "")
			s := lipgloss.NewStyle().Width(w)
			if i == selected && !plain {
				s = s.Foreground(lipgloss.Color("#0d1117")).Background(lipgloss.Color(col)).Bold(true)
			} else if !plain && i < len(blocked) && blocked[i] {
				s = s.Foreground(lipgloss.Color("#a05a5a"))
			}
			parts = append(parts, s.Render(l))
		}
		return strings.Join(parts, "")
	}
	labels := []string{"t/s when full"}
	for _, c := range g.Contexts[colStart : colStart+columns] {
		labels = append(labels, Context(c))
	}
	panel += dim(row(labels, -1, nil))
	if columns < len(g.Contexts) {
		panel += "\n→ more"
	}
	panel += "\n"
	visible := min(5, max(1, (height-19)/2))
	offset := min(max(0, m.File-visible+1), max(0, len(g.Files)-visible))
	for i := offset; i < min(offset+visible, len(g.Files)); i++ {
		f := g.Files[i]
		top, bottom := []string{f.Name}, []string{f.Size.String()}
		if f.LocalPath != "" {
			bottom[0] += style(" ● on disk", "#3fb950")
		}
		blocked := []bool{false}
		selected := -1
		for j := colStart; j < colStart+columns; j++ {
			c := f.Cells[j]
			a, b := number(c.Speed)+" t/s", c.Memory.String()
			if c.PastNative {
				a, b = "—", "max "+Context(m.Default.Result.Model.NativeCtx)
			} else if !c.Fits {
				a = "✗"
			}
			top = append(top, a)
			bottom = append(bottom, b)
			blocked = append(blocked, !c.Fits)
			if i == m.File && j == m.Column {
				selected = j - colStart + 1
			}
		}
		panel += row(top, selected, blocked) + "\n" + row(bottom, selected, blocked) + "\n"
	}
	if len(g.Files) > visible {
		panel += dim(fmt.Sprintf("Files %d–%d of %d ↕", offset+1, min(offset+visible, len(g.Files)), len(g.Files))) + "\n"
	}
	if g.Folded > 0 {
		panel += dim(fmt.Sprintf("%d larger files don't fit", g.Folded)) + "\n"
	}
	panel += "\nSpeed   " + number(cell.Empty) + " t/s empty → " + number(cell.Speed) + " t/s at " + Context(cell.Context) + map[bool]string{true: "    First reply  ", false: "\nFirst reply  "}[width >= 76] + estimate(cell.First) + "\n"
	fraction := cell.Memory.V / max(1, p.Target.GPUMemBytes/(1<<30))
	full := min(20, max(0, int(math.Round(20*fraction))))
	text := cell.Memory.Number() + " of " + budget
	if fraction > .92 {
		text = style(text, "#d29922")
	}
	panel += "Memory  " + style(strings.Repeat("█", full), col) + dim(strings.Repeat("░", 20-full)) + "  " + text
	border := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Width(max(20, width-2)).BorderForeground(lipgloss.Color(col))
	b += "\n" + border.Render(panel)
	if plain {
		b = ansi.Strip(b)
	}
	return b
}

const Keys = "↑↓ Model   ←→ Context   f File   c Cache   e Engine   ⏎ Try   s Set up   / Search   v Details   esc Back"
