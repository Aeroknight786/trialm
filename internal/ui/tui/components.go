package tui

func (m *Root) helpView() string { return m.theme.dim.Render(m.footer()) }
func selection(i, selected int) string {
	if i == selected {
		return "❯ "
	}
	return "  "
}
