package tui

import (
	"charm.land/lipgloss/v2"
	"fmt"
	"github.com/aeroknight786/trialm/internal/catalog"
	"github.com/aeroknight786/trialm/internal/core"
	"github.com/aeroknight786/trialm/internal/figure"
)

func (m *Root) tryBody() string {
	name := "the model"
	if r, ok := m.row(); ok {
		name = r.Result.Model.Name
	}
	cost := "Free · $1 credit with GitHub or Google"
	var credit figure.Figure
	if m.cfg.Store.Key() != "" && m.cfg.Store.Read("credits.json", &credit) == nil && credit.Unit == "USD" {
		cost = "$" + fmt.Sprintf("%.2f", credit.V) + " of credit left"
	}
	b := "Try " + name + " before you download it\n\n" + m.card("Runs in your own agent and project, paced to this Mac's estimated speed.\n"+cost+" · ZDR · No download") + "\n\n"
	if m.busy && m.pendingTrial {
		return b + "Finish signing in in your browser. Waiting…"
	}
	b += "Run it in\n"
	for _, id := range core.TrialAgents(m.cfg.Preferences) {
		name := toolName(id)
		if id == "generic" {
			name = "Your own agent"
		}
		if id == m.cfg.Preferences.Tool {
			b += m.theme.cloud.Render("["+name+"]") + "  "
		} else {
			b += name + "  "
		}
	}
	return b
}
func (m *Root) summaryBody() string {
	if m.last.ID == "" {
		return "No trials yet."
	}
	r := m.last.Report
	name := m.last.Model
	var size figure.Figure
	if model, ok := catalog.Lookup(m.last.Model); ok {
		name = model.Name
		if q, e := model.Quant(m.last.Quant); e == nil {
			size = figure.Bytes(q.FileBytes, "catalog:file")
		}
	}
	cost := "Cost unknown"
	if r.SpendKnown {
		cost = "Cost $" + fmt.Sprintf("%.2f", r.SpendUSD)
	}
	if m.creditLeft != nil && *m.creditLeft < 0.02 {
		cost += " · your trial credit is used up; setting it up here is free"
	} else if m.creditLeft != nil {
		cost += " · $" + fmt.Sprintf("%.2f", *m.creditLeft) + " of credit left"
	}
	b := "Your trial of " + name + "\n\n" + r.Lines() + "\n" + cost + "\n\nHow did it feel?\n1 Too slow    2 Fine    3 Great\n\nInstall? " + size.String()
	if m.expanded {
		b += "\n" + r.Plain(m.last.Band)
	}
	return b
}
func (m *Root) tiles(labels, values, sub []string) string {
	var rows []string
	width := max(12, (m.width-2)/len(labels))
	for i, label := range labels {
		text := wrap(label, width-2) + "\n" + wrap(values[i], width-2)
		if i < len(sub) {
			text += "\n" + wrap(sub[i], width-2)
		}
		rows = append(rows, text)
	}
	return joinTiles(rows, width)
}
func joinTiles(rows []string, width int) string {
	height := 0
	for _, r := range rows {
		height = max(height, lipgloss.Height(r))
	}
	for i, r := range rows {
		rows[i] = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Width(width).Height(height + 2).Render(r)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, rows...)
}
