package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/aeroknight786/trialm/internal/figure"
	"github.com/aeroknight786/trialm/internal/hw"
)

func (m *Root) readServers() []hw.ModelServer {
	if m.cfg.ModelServers == nil {
		return nil
	}
	return m.cfg.ModelServers(m.cfg.Context)
}
func (m *Root) scanServers() tea.Cmd {
	return func() tea.Msg { return serversMsg(m.readServers()) }
}
func (m *Root) modelWarning() string {
	if len(m.otherModels) == 0 {
		return ""
	}
	b := m.theme.caveat.Render(fmt.Sprintf("%d Other Models Loaded | Expect Slower Speeds", len(m.otherModels)))
	if m.expanded {
		for _, server := range m.otherModels {
			b += "\n" + m.theme.dim.Render(server.Name+" · "+figure.Bytes(int64(server.MemoryBytes), "process:rss").String()+" memory")
		}
	}
	return b
}
