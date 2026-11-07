package setup

import (
	"fmt"
	"strings"

	"github.com/aeroknight786/trialm/internal/figure"
	"github.com/aeroknight786/trialm/internal/harness"
)

// Overview is the short Mac setup card. Steps retains the executable detail.
func (p Plan) Overview() string {
	var lines []string
	for _, step := range p.Steps {
		if step.ID == "engine" || step.Kind == Admin {
			title := step.Title
			if step.Request != nil {
				title += " (" + figure.Bytes(step.Request.Size, "setup:asset").String() + ")"
			}
			lines = append(lines, title)
		}
	}
	return strings.Join(append(lines, "Start the engine", "Connect "+strings.TrimSuffix(harness.DisplayName(p.Harness), " CLI"), "Check speed and tools"), "\n")
}
func (p Plan) UntouchedSummary() string {
	if p.OS == "darwin" {
		return "your " + strings.TrimSuffix(harness.DisplayName(p.Harness), " CLI") + " settings, your shell, other models"
	}
	return strings.Join(p.Untouched, ", ")
}

// CountsLine is the download size shared by compact and detailed setup cards.
func (p Plan) CountsLine(_ bool) string {
	size := figure.Bytes(max(0, p.DownloadBytes-p.OnDisk()), "setup:plan")
	if size.Unit == "MB" {
		size = figure.Band(size.V, 20, size.Unit, size.Kind, size.Src)
	}
	return size.String()
}

func (s Step) Description(expanded bool) string {
	text := s.Title
	if expanded && len(s.Argv) > 0 {
		text += "\n  " + fmt.Sprintf("%q", s.Argv)
	}
	return text
}
