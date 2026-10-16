package boss

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/arkurl/cloak/internal/render"
)

// replyMsg is delivered after the fake "thinking" delay.
type replyMsg struct{}

var (
	userStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	asstStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	toolStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("110"))
	outStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("242"))
	dimStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	headStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
	inputStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
	spinStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
)

// Model is the fake Claude Code screen.
type Model struct {
	width, height int
	entries       []Entry
	live          []Entry
	liveIdx       int
	input         []rune
	focused       bool
	thinking      bool
	pending       string
	scrollUp      int
}

// New creates the boss model with a plausible opening conversation.
func New() Model {
	return Model{entries: initialConversation(), live: liveScript()}
}

// SetSize records the available dimensions.
func (m *Model) SetSize(w, h int) { m.width, m.height = w, h }

// PageLines returns how many conversation lines are visible at once, used by
// the page-up / page-down keys in browse mode.
func (m Model) PageLines() int {
	h := m.height
	if h <= 0 {
		h = 24
	}
	n := h - 5
	if n < 1 {
		n = 1
	}
	return n
}

// Focused reports whether the fake input line has the keyboard.
func (m Model) Focused() bool { return m.focused }

// Focus gives the keyboard to the input line.
func (m *Model) Focus() { m.focused = true }

// Blur returns the keyboard to browse mode.
func (m *Model) Blur() { m.focused = false }

// Thinking reports whether a fake reply is pending.
func (m Model) Thinking() bool { return m.thinking }

// EntryCount returns the number of conversation entries.
func (m Model) EntryCount() int { return len(m.entries) }

// InsertRune appends a character to the input line.
func (m *Model) InsertRune(r rune) {
	if r == '\n' || r == '\r' {
		return
	}
	m.input = append(m.input, r)
}

// Backspace removes the last input character.
func (m *Model) Backspace() {
	if len(m.input) > 0 {
		m.input = m.input[:len(m.input)-1]
	}
}

// ScrollBy moves the viewport; positive scrolls toward older content.
func (m *Model) ScrollBy(n int) {
	m.scrollUp += n
	if m.scrollUp < 0 {
		m.scrollUp = 0
	}
}

// Advance appends the next scripted line so the session looks live. It is a
// no-op while a reply is pending or while the user has scrolled up to review
// older content.
func (m *Model) Advance() {
	if m.thinking || m.scrollUp > 0 || len(m.live) == 0 {
		return
	}
	m.entries = append(m.entries, m.live[m.liveIdx%len(m.live)])
	m.liveIdx++
	if len(m.entries) > 500 {
		m.entries = m.entries[len(m.entries)-500:]
	}
}

// Submit appends the current input as a user message and schedules a reply.
func (m Model) Submit() (Model, tea.Cmd) {
	text := strings.TrimSpace(string(m.input))
	if text == "" {
		return m, nil
	}
	m.entries = append(m.entries, Entry{Role: "user", Text: text})
	m.input = m.input[:0]
	m.thinking = true
	m.pending = text
	m.scrollUp = 0
	return m, tea.Tick(650*time.Millisecond, func(time.Time) tea.Msg { return replyMsg{} })
}

// Update handles the internal reply timer.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if _, ok := msg.(replyMsg); ok && m.thinking {
		m.entries = append(m.entries, replyFor(m.pending)...)
		m.pending = ""
		m.thinking = false
		m.scrollUp = 0
	}
	return m, nil
}

func (m Model) renderLines() []string {
	var out []string
	for _, e := range m.entries {
		out = append(out, renderEntry(e, m.width)...)
	}
	if m.thinking {
		out = append(out, spinStyle.Render("✻ Thinking… (esc to interrupt)"))
	}
	return out
}

func renderEntry(e Entry, width int) []string {
	var prefix string
	var style lipgloss.Style
	switch e.Role {
	case "user":
		prefix, style = "> ", userStyle
	case "tool":
		prefix, style = "⏺ ", toolStyle
	case "output":
		prefix, style = "  ⎿ ", outStyle
	case "info":
		prefix, style = "  ", dimStyle
	default: // assistant
		prefix, style = "", asstStyle
	}
	pw := render.Width(prefix)
	avail := width - pw
	if avail < 1 {
		avail = 1
	}
	wrapped := render.Wrap(e.Text, avail)
	lines := make([]string, 0, len(wrapped))
	for i, ln := range wrapped {
		if i == 0 {
			lines = append(lines, style.Render(prefix+ln))
		} else {
			lines = append(lines, style.Render(strings.Repeat(" ", pw)+ln))
		}
	}
	return lines
}

// View renders the screen at the current size.
func (m Model) View() string {
	w := m.width
	if w <= 0 {
		w = 80
	}

	header := headStyle.Render(" ✻ Claude Code") + dimStyle.Render("   ~/dev/acme-api   main ")
	header = fit(header, w)
	sep := dimStyle.Render(strings.Repeat("─", w))

	// Layout: header + separator + conversation + separator + input + status.
	convH := m.PageLines()
	lines := m.renderLines()

	maxScroll := len(lines) - convH
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.scrollUp > maxScroll {
		m.scrollUp = maxScroll
	}
	top := len(lines) - convH - m.scrollUp
	if top < 0 {
		top = 0
	}
	end := top + convH
	if end > len(lines) {
		end = len(lines)
	}
	window := make([]string, 0, convH)
	for _, ln := range lines[top:end] {
		window = append(window, pad(ln, w))
	}
	for len(window) < convH {
		window = append(window, strings.Repeat(" ", w))
	}

	// Input line.
	cursor := "█"
	var hint string
	if m.focused {
		hint = "   enter to send · esc to browse"
	} else {
		hint = "   i to type · ⌃b to exit"
	}
	// Trim the input by display width so a long line cannot wrap the layout.
	avail := w - render.Width("❯ ") - render.Width(cursor) - render.Width(hint)
	if avail < 1 {
		hint = ""
		avail = w - render.Width("❯ ") - render.Width(cursor)
	}
	if avail < 1 {
		avail = 1
	}
	shownInput := tailWidth(string(m.input), avail)
	var inputLine string
	if m.focused {
		inputLine = inputStyle.Render("❯ ") + shownInput + inputStyle.Render(cursor) + dimStyle.Render(hint)
	} else {
		inputLine = dimStyle.Render("❯ ") + dimStyle.Render(shownInput) + dimStyle.Render(cursor) + dimStyle.Render(hint)
	}
	inputLine = fit(inputLine, w)

	status := dimStyle.Render("  ? for shortcuts   ·   / for commands   ·   ⏵⏵ accept edits on (shift+tab to cycle)")
	status = fit(status, w)

	return strings.Join([]string{header, sep, strings.Join(window, "\n"), sep, inputLine, status}, "\n")
}

// tailWidth returns the trailing part of s that fits within width columns.
func tailWidth(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if render.Width(s) <= width {
		return s
	}
	runes := []rune(s)
	for len(runes) > 0 && render.Width(string(runes)) > width {
		runes = runes[1:]
	}
	return string(runes)
}

// pad right-pads a (possibly styled) string to width using ANSI-aware width.
func pad(s string, width int) string {
	gap := width - lipgloss.Width(s)
	if gap <= 0 {
		return s
	}
	return s + strings.Repeat(" ", gap)
}

// fit truncates a (possibly styled) string to width and pads it to exactly
// width columns, so no line can overflow and wrap the layout.
func fit(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(s) > width {
		s = ansi.Truncate(s, width, "")
	}
	return pad(s, width)
}
