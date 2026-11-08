package ui

import (
	"fmt"
	"io"
	"strings"
)

// OwnAgentCard is what serving a model to "your own agent" shows: where to
// point an app, in one card.
func OwnAgentCard(w io.Writer, name, model, endpoint string) {
	root := strings.TrimRight(endpoint, "/")
	rows := [][2]string{{"OpenAI", root + "/v1"}, {"Anthropic", root}, {"Ollama", root}, {"Model", model}, {"API key", "trialm (any value works)"}}
	width := 0
	for _, r := range rows {
		width = max(width, 11+len([]rune(r[1])))
	}
	fmt.Fprintf(w, "✦ %s is ready for your own agent\n\n╭%s╮\n", name, strings.Repeat("─", width+2))
	for _, r := range rows {
		line := fmt.Sprintf("%-11s%s", r[0], r[1])
		fmt.Fprintf(w, "│ %s%s │\n", line, strings.Repeat(" ", width-len([]rune(line))))
	}
	fmt.Fprintf(w, "╰%s╯\n\n", strings.Repeat("─", width+2))
}
