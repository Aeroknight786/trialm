package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
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
	n := len(m.otherModels)
	b := m.theme.caveat.Render(fmt.Sprintf("%d Other Model%s Loaded | Expect Slower Speeds", n, map[bool]string{true: "s"}[n > 1]))
	for _, s := range m.otherModels {
		if m.expanded {
			b += "\n" + m.theme.dim.Render(strings.TrimSuffix(s.Name+" · "+s.Model, " · "))
		}
	}
	return b
}
