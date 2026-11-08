package tui

import (
	"fmt"
	"strings"

	"github.com/aeroknight786/trialm/internal/core"
	"github.com/aeroknight786/trialm/internal/engine"
	"github.com/aeroknight786/trialm/internal/figure"
	"github.com/aeroknight786/trialm/internal/predict"
	"github.com/aeroknight786/trialm/internal/ui/tui/pick"
	"github.com/charmbracelet/x/ansi"
	"strconv"
)

func wrap(s string, w int) string { return ansi.Wrap(s, max(1, w), "") }
func strip(s string) string       { return ansi.Strip(s) }
func clip(s string, w int) string { return ansi.Truncate(s, max(1, w), "") }
func estimated(f figure.Figure, local ...bool) string {
	if len(local) > 0 && local[0] && f.Kind == figure.Calibrated {
		return strings.TrimPrefix(f.String(), "~")
	}
	return "~" + strings.TrimPrefix(f.String(), "~")
}
func metric(r predict.Result, v float64, unit string, reading bool) figure.Figure {
	return core.PredictionMetric(r, v, unit, reading)
}
func (m *Root) body() string {
	switch m.screen() {
	case Detect:
		return m.detectBody()
	case Agents:
		return m.agentsBody()
	case Calibrate:
		return m.calibrationBody()
	case First:
		return m.tryBody()
	case Note:
		return "How did the trial go?\n\n" + m.note.View() + "\n\n" + m.theme.dim.Render(wrap("Sends your rating, this note, the model, this Mac's chip and memory, and the timings from the trial. Never your code, prompts or file paths.", max(20, min(m.width, 80)-2)))
	case Paste:
		if m.password == nil {
			return "OpenRouter key"
		}
		return m.authURL + "\n" + m.password.View()
	case Recipes:
		return m.recipes()
	case Details:
		return m.details()
	case Summary:
		return m.summaryBody()
	case Home:
		return m.launcher()
	case Help:
		return "Keys\n\n↑↓ Model    ⏎ Choose    / Search    v Details\nGrid    ↑↓ File    ←→ Context    c Cache    e Engine\n⏎ Try    s Set up    esc Models\nEsc Back    q Later    ctrl-c Pause"
	case Gallery:
		b := "Published claims · unverified by triaLM · never ranked\n"
		for i, c := range Claims() {
			b += selection(i, m.selected) + c.ID + " [" + c.Grade + "]\n  " + c.Hardware + "\n  " + c.Figures[0].String() + "\n"
		}
		return b
	case ClaimCard:

		cs := Claims()
		c := cs[min(max(0, m.selected), len(cs)-1)]
		b := "Claim · " + c.ID + " [" + c.Grade + "]\n" + c.Hardware + "\n" + c.Recipe + "\n"
		for _, f := range c.Figures {
			b += f.String() + "\n"
		}
		return b + "Source: " + c.SourceName + "\n" + c.SourceURL + "\nRecorded " + c.Date + "\nNot simulated yet. Adapter arrives in a later release."
	case Search:
		return m.search.View()
	case Why:
		if len(m.stack) > 1 && m.stack[len(m.stack)-2] == ClaimCard {
			cs := Claims()
			c := cs[min(max(0, m.selected), len(cs)-1)]
			return "Source · " + c.SourceName + "\n" + c.SourceURL + "\n" + c.Date + " · " + c.DateBasis + "\nGrade " + c.Grade + " · unverified by triaLM\n" + c.Hardware
		}
		if row, ok := m.row(); ok {
			r := row.Result
			f := metric(r, r.DecodeTPSDeep, "tok/s", false)
			return fmt.Sprintf("Why these numbers?\n%s · %s\n%s\nRange %s to %s\nSource: %s\nWriting ±%d%% · reading ±%d%%\n%s", r.Model.Name, r.Runtime.Name, estimated(f, m.detection.Target.OS == "darwin"), figure.BandNumber(f.Lo, 26), figure.BandNumber(f.Hi, 26), f.Src, r.Accuracy.DecodePct, r.Accuracy.PrefillPct, strings.Join(r.Warnings, "\n"))
		}
		return "Claims are source-reported figures, not triaLM measurements.\n" + Claims()[0].SourceURL
	case Setup:
		return m.setupBody()
	case Ready:
		return m.readyBody()
	case Launching:
		return "Opening " + toolName(m.cfg.Preferences.Tool) + " on the local model…"
	case Trial:
		if m.address != "" {
			return m.address
		}
		return m.meter.String()
	case Chat:
		return m.chat.view()
	}
	return ""
}
func (m *Root) recipes() string {
	if m.cfg.Picker != nil {
		if m.busy {
			return "Models\n" + m.waiting() + " Loading"
		}
		return pick.View(&m.picker, m.selected, m.width, m.theme.plain, m.gridFocus)
	}
	return "Models\nNo models fit this agent on this machine."
}

func (m *Root) details() string {
	row, ok := m.row()
	if !ok {
		return "No recipe selected."
	}
	r := row.Result
	mem := r.Memory
	b := r.Model.Name + " · " + r.Quant.Name + " · " + r.Runtime.Name + "\n"
	b += "Memory " + m.progressBar(mem.TotalBytes/max(1, mem.BudgetBytes)) + "\n" + estimated(figure.Exact(mem.TotalBytes/(1<<30), "GB", figure.Simulated, "memory:plan")) + " of " + figure.Exact(mem.BudgetBytes/(1<<30), "GB", figure.Measured, "hardware:budget").String() + " · " + mem.Level + "\n"
	if r.Timing != nil {
		read := r.Scenario.FollowUpTTFT.Seconds()
		write := max(0, r.Scenario.FollowUpTurn.Seconds()-read)
		b += "Turn " + estimated(metric(r, r.Scenario.FollowUpTurn.Seconds(), "s", false), m.detection.Target.OS == "darwin") + "\n" + m.split(read, write, 0)
	}
	flash := "off"
	if r.Settings.FlashAttention {
		flash = "on: keeps prompt reading efficient"
	}
	b += fmt.Sprintf("\n✓ flash attention, %d slot(s) · %s\n· KV cache · %s · %s at this context\n· YaRN · not needed: native %dk ≥ %dk\n· spec · not simulated yet (v0.6)\n", r.Settings.Slots, flash, r.Settings.KVCacheType, mem.Level, r.Model.NativeCtx/1024, r.Settings.ContextTokens/1024)
	b += strings.Join(r.TuneSteps, "\n")
	if mem.Level == predict.FitOK {
		b = m.theme.good.Render("✓ fits") + "\n" + b
	} else if mem.Level == predict.FitNo {
		b = m.theme.blocked.Render("✗ blocked") + "\n" + b
	} else {
		b = m.theme.caveat.Render("caveat: "+mem.Level) + "\n" + b
	}
	f := metric(r, r.Timing.DecodeTPS(r.Settings.ContextTokens), "t/s", false)
	b += "\nRange " + figure.BandNumber(f.Lo, 26) + " to " + figure.BandNumber(f.Hi, 26) + " t/s\nSource: " + f.Src + "\n" + r.Accuracy.Basis
	args, e := engine.Args(r.Runtime.ID, map[string]string{"binary": "llama-server", "model": "<model>", "name": r.Model.OllamaTag, "port": "8080", "ctx": strconv.Itoa(r.Settings.ContextTokens), "ngl": "99", "fa": "on", "kv": r.Settings.KVCacheType, "host": "127.0.0.1"})
	if e == nil {
		b += "\nArgv " + fmt.Sprint(args)
	}
	return b
}
func (m *Root) progressBar(v float64) string {
	v = max(0, min(1, v))
	w := max(8, min(36, m.width-8))
	if m.theme.plain || m.theme.ascii {
		n := int(float64(w) * v)
		return "[" + strings.Repeat("=", n) + strings.Repeat("-", w-n) + "]"
	}
	if m.busy {
		return m.bar.View()
	}
	return m.bar.ViewAs(v)
}
func (m *Root) split(read, write, fixed float64) string {
	f := func(v float64) string { return figure.Band(v, 30, "s", figure.Simulated, "predict:turn").String() }
	return "Reading " + f(read) + " · Writing " + f(write) + " · Other " + f(fixed) + "\n"
}
