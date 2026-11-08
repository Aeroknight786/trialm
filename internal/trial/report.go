package trial

import (
	"fmt"
	"sort"
	"strings"

	"github.com/aeroknight786/trialm/internal/figure"
	"github.com/aeroknight786/trialm/internal/sim"
)

type Report struct {
	SpendKnown                                                                                   bool
	Requests, CloudBound, Reloads, ContextPeak, Overflows                                        int
	LocalSeconds, CloudSeconds, ReadingSeconds, WritingSeconds, LoadSeconds, BetweenTurnsSeconds float64
	FirstTurn, TypicalTurn, SlowestTurn                                                          float64
	SlowestCause, Fidelity                                                                       string
	CachedTokens, PromptTokens                                                                   int
	SpendUSD                                                                                     float64
	// First and the follow-up average, as the user waited for them.
	First, FollowUps Turn
	FollowUpCount    int
	Tools            []sim.ToolGroup `json:",omitempty"`
}

// Turn is one request's tokens and time; Seconds includes Waited in line.
type Turn struct {
	In, New, Out    int
	Seconds, Waited float64
	Cancelled       bool `json:",omitempty"`
}

// Estimate is what triaLM showed before the trial and the sizes it assumed.
type Estimate struct {
	FirstIn, FirstOut, ReplyIn, ReplyNew, ReplyOut int
	FirstSec, ReplySec                             float64
}

func Summarize(records []sim.Record, spend float64, fidelity string) Report {
	r := Report{Requests: len(records), SpendUSD: max(0, spend), SpendKnown: spend >= 0, Fidelity: fidelity}
	var turns []float64
	for i, s := range records {
		local := max(0, s.TotalSec-s.CloudLagSec)
		r.LocalSeconds += local + s.QueueSec
		r.CloudSeconds += s.CloudSec
		r.LoadSeconds += s.LoadSec
		r.ReadingSeconds += s.PrefillSec
		r.WritingSeconds += max(0, local-s.LoadSec-s.PrefillSec)
		if s.CloudLagSec > 0 {
			r.CloudBound++
		}
		if s.Cold {
			r.Reloads++
		}
		r.ContextPeak = max(r.ContextPeak, s.PromptTokens+s.OutputTokens)
		if s.ContextError != "" || s.Truncated != "" {
			r.Overflows++
		}
		r.PromptTokens += s.PromptTokens
		r.CachedTokens += s.CachedTokens
		t := Turn{In: s.PromptTokens, New: s.PrefillTokens, Out: s.OutputTokens, Seconds: s.TotalSec + s.QueueSec, Waited: s.QueueSec, Cancelled: s.Cancelled}
		if i == 0 {
			r.FirstTurn, r.First, r.Tools = s.TotalSec, t, s.Tools
		} else {
			f := &r.FollowUps
			f.In, f.New, f.Out, f.Seconds, f.Waited = f.In+t.In, f.New+t.New, f.Out+t.Out, f.Seconds+t.Seconds, f.Waited+t.Waited
			turns = append(turns, s.TotalSec)
			r.BetweenTurnsSeconds += max(0, s.At.Sub(records[i-1].At).Seconds()-records[i-1].TotalSec)
		}
		if s.TotalSec > r.SlowestTurn {
			r.SlowestTurn = s.TotalSec
			r.SlowestCause = "writing"
			if s.CloudLagSec > 0 {
				r.SlowestCause = "cloud-bound"
			} else if s.LoadSec > s.PrefillSec {
				r.SlowestCause = "model load"
			} else if s.PrefillSec > local/2 {
				r.SlowestCause = "reading"
			}
		}
	}
	sort.Float64s(turns)
	if n := len(turns); n > 0 {
		r.TypicalTurn = turns[n/2]
		f := &r.FollowUps
		r.FollowUpCount, f.In, f.New, f.Out, f.Seconds, f.Waited = n, f.In/n, f.New/n, f.Out/n, f.Seconds/float64(n), f.Waited/float64(n)
	}
	return r
}

// Lines sets what the trial took beside what triaLM estimated, so a slow
// trial shows its cause: the prompt sizes differ, or requests waited.
func (r Report) Lines(e *Estimate) string {
	var b strings.Builder
	row := func(label string, in, fresh, out int, time ...string) {
		fmt.Fprintf(&b, "%-15s %-6s %-6s %-5s %s\n", label, sim.Kilo(in), sim.Kilo(fresh), sim.Kilo(out), time[0])
		for _, t := range time[1:] {
			fmt.Fprintf(&b, "%36s%s\n", "", t)
		}
	}
	took := func(t Turn, note string) []string {
		out := []string{sim.FormatDur(t.Seconds) + note}
		if t.Waited >= 1 {
			out = append(out, sim.FormatDur(t.Waited)+" of it in line")
		}
		return out
	}
	fmt.Fprintf(&b, "%-15s %-6s %-6s %-5s %s\n", "", "in", "new", "out", "time")
	note := ""
	if r.First.Cancelled {
		note = ", cancelled"
	}
	row("First request", r.First.In, r.First.New, r.First.Out, took(r.First, note)...)
	if r.FollowUpCount > 0 {
		f := r.FollowUps
		row(fmt.Sprintf("Follow-ups (%d)", r.FollowUpCount), f.In, f.New, f.Out, took(f, " average")...)
	}
	if e != nil {
		row("Our estimate", e.FirstIn, e.FirstIn, e.FirstOut, sim.FormatDur(e.FirstSec)+" first reply")
		row("", e.ReplyIn, e.ReplyNew, e.ReplyOut, sim.FormatDur(e.ReplySec)+" per reply")
	}
	if n := sim.ToolTokens(r.Tools); n > 0 {
		fmt.Fprintf(&b, "\nTool definitions: %s of the first %s\n%s\n", sim.Kilo(n), sim.Kilo(r.First.In), sim.ToolsLine(r.Tools, 3))
	}
	return b.String()
}
func (r Report) Plain(band int) string {
	var b strings.Builder
	number := func(v float64) string {
		return figure.Band(v, band, "s", figure.Simulated, "trial:local-schedule").String()
	}
	measured := func(v float64, unit string) string {
		return figure.Exact(v, unit, figure.Measured, "trial:observed").String()
	}
	fmt.Fprintf(&b, "Trial summary · %s\n", r.Fidelity)
	fmt.Fprintf(&b, "  %d requests · local %s · cloud %s · %d cloud-bound moments\n", r.Requests, number(r.LocalSeconds), measured(r.CloudSeconds, "s"), r.CloudBound)
	fmt.Fprintf(&b, "  time split: reading %s · writing %s · load %s · between turns (tools/user) %s\n", number(r.ReadingSeconds), number(r.WritingSeconds), number(r.LoadSeconds), measured(r.BetweenTurnsSeconds, "s"))
	fmt.Fprintf(&b, "  first turn %s · typical follow-up %s · slowest %s (%s)\n", number(r.FirstTurn), number(r.TypicalTurn), number(r.SlowestTurn), r.SlowestCause)
	reuse := 0
	if r.PromptTokens > 0 {
		reuse = 100 * r.CachedTokens / r.PromptTokens
	}
	fmt.Fprintf(&b, "  cache reuse %d%% · loads %d · context peak %d tokens · overflows %d\n", reuse, r.Reloads, r.ContextPeak, r.Overflows)
	if r.SpendKnown {
		fmt.Fprintf(&b, "  spend %s\n", measured(r.SpendUSD, "USD"))
	} else {
		fmt.Fprintln(&b, "  spend unknown · no endpoint usage prices")
	}
	return b.String()
}

// EventData keeps observed cloud measurements separate from local simulation.
func (r Report) EventData(band int) map[string]any {
	local := func(v float64) figure.Figure {
		return figure.Band(v, band, "s", figure.Simulated, "trial:local-schedule")
	}
	observed := func(v float64, unit string) figure.Figure {
		return figure.Exact(v, unit, figure.Measured, "trial:observed")
	}
	d := map[string]any{"requests": observed(float64(r.Requests), "count"), "local_s": local(r.LocalSeconds), "cloud_s": observed(r.CloudSeconds, "s"), "cloud_bound": observed(float64(r.CloudBound), "count"), "reading_s": local(r.ReadingSeconds), "writing_s": local(r.WritingSeconds), "load_s": local(r.LoadSeconds), "between_turns_s": observed(r.BetweenTurnsSeconds, "s"), "first_turn_s": local(r.FirstTurn), "typical_turn_s": local(r.TypicalTurn), "slowest_turn_s": local(r.SlowestTurn), "slowest_cause": r.SlowestCause, "fidelity": r.Fidelity, "cache_reused_tokens": observed(float64(r.CachedTokens), "tokens"), "loads": observed(float64(r.Reloads), "count"), "context_peak_tokens": observed(float64(r.ContextPeak), "tokens"), "overflows": observed(float64(r.Overflows), "count"), "spend_known": r.SpendKnown}
	if r.SpendKnown {
		d["spend_usd"] = observed(r.SpendUSD, "USD")
	} else {
		d["spend_usd"] = nil
	}
	return d
}
