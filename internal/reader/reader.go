// Package reader renders a chapter and handles navigation.
package reader

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/arkurl/decoy/internal/config"
	"github.com/arkurl/decoy/internal/epub"
	"github.com/arkurl/decoy/internal/render"
	"github.com/arkurl/decoy/internal/storage"
)

var (
	barStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Background(lipgloss.Color("236"))
	bodyStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("253"))
	headStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("117")).Bold(true)
	statusStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	noticeStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
	tocSelStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("117")).Bold(true)
	tocHeadStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
)

// Model is the reading view.
type Model struct {
	book *epub.Book
	keys config.KeyMap

	width, height int
	spine         int
	paras         []epub.Para
	layout        *render.Layout
	offset        int

	showTOC   bool
	tocCursor int
	notice    string // transient boundary hint shown in the status bar

	// lineCounts caches the line count of every chapter at cachedWidth, so the
	// page indicator does not re-lay-out the whole book on every render.
	lineCounts  []int
	cachedWidth int

	err error
}

// New creates a reader positioned at the start of the book.
func New(book *epub.Book, keys config.KeyMap) Model {
	m := Model{book: book, keys: keys}
	m.loadChapter(0, render.Anchor{})
	return m
}

// SetSize records the viewport and reflows, keeping the reading position.
func (m *Model) SetSize(w, h int) {
	m.width, m.height = w, h
	m.rebuild()
	m.ensureLineCounts()
}

// ShowTOC reports whether the table of contents overlay is open.
func (m Model) ShowTOC() bool { return m.showTOC }

// SpineIndex returns the index of the chapter currently shown.
func (m Model) SpineIndex() int { return m.spine }

// Position returns the current (width-independent) reading position.
func (m Model) Position() storage.Position {
	a := render.Anchor{}
	if m.layout != nil {
		a = m.layout.AnchorOf(m.offset)
	}
	return storage.Position{SpineIndex: m.spine, Para: a.Para, LineInPara: a.LineInPara}
}

// Restore jumps to a saved position.
func (m *Model) Restore(pos storage.Position) {
	m.loadChapter(pos.SpineIndex, render.Anchor{Para: pos.Para, LineInPara: pos.LineInPara})
}

// Progress returns overall progress in the range 0..1, based on the page
// indicator so the percentage and the page number always agree.
func (m Model) Progress() float64 {
	cur, total := m.Page()
	if total <= 1 {
		return 0
	}
	f := float64(cur-1) / float64(total-1)
	if f < 0 {
		return 0
	}
	if f > 1 {
		return 1
	}
	return f
}

func (m *Model) contentWidth() int {
	w := m.width - 4
	if w < 1 {
		w = 1
	}
	return w
}

func (m Model) contentHeight() int {
	h := m.height - 2
	if h < 1 {
		h = 1
	}
	return h
}

// ensureLineCounts lays out every chapter once at the current width and caches
// the line counts. It is only recomputed when the width changes.
func (m *Model) ensureLineCounts() {
	w := m.contentWidth()
	if m.lineCounts != nil && m.cachedWidth == w {
		return
	}
	n := m.book.NumChapters()
	counts := make([]int, n)
	for i := 0; i < n; i++ {
		paras, err := m.book.ChapterParas(i)
		if err != nil {
			counts[i] = 1
			continue
		}
		counts[i] = render.CountLines(paras, w)
	}
	m.lineCounts = counts
	m.cachedWidth = w
}

// pagesFor returns how many pages chapter i occupies at the current height.
func (m Model) pagesFor(i, height int) int {
	lines := 1
	if i >= 0 && i < len(m.lineCounts) {
		lines = m.lineCounts[i]
	}
	if lines < 1 {
		lines = 1
	}
	return (lines + height - 1) / height
}

// Page returns the current page and the total number of pages.
func (m Model) Page() (current, total int) {
	n := m.book.NumChapters()
	if n == 0 {
		return 1, 1
	}
	h := m.contentHeight()
	before := 0
	for i := 0; i < m.spine; i++ {
		before += m.pagesFor(i, h)
	}
	total = before
	for i := m.spine; i < n; i++ {
		total += m.pagesFor(i, h)
	}
	current = before + m.offset/h + 1
	if current < 1 {
		current = 1
	}
	if current > total {
		current = total
	}
	return current, total
}

func (m *Model) loadChapter(i int, anchor render.Anchor) {
	n := m.book.NumChapters()
	if n == 0 {
		// Still build a layout so the view never sees a nil layout.
		m.err = fmt.Errorf("这本书没有可阅读的章节")
		m.paras = nil
		m.layout = nil
		m.rebuild()
		return
	}
	if i < 0 {
		i = 0
	}
	if i >= n {
		i = n - 1
	}
	paras, err := m.book.ChapterParas(i)
	if err != nil {
		m.err = err
		paras = nil
	}
	m.spine = i
	m.paras = paras
	m.layout = nil
	m.rebuild()
	m.offset = m.layout.OffsetOf(anchor)
}

func (m *Model) rebuild() {
	if m.layout != nil {
		anchor := m.layout.AnchorOf(m.offset)
		m.layout = render.Build(m.paras, m.contentWidth())
		m.offset = m.layout.OffsetOf(anchor)
		return
	}
	m.layout = render.Build(m.paras, m.contentWidth())
}

func (m Model) maxTop() int {
	if m.layout == nil {
		return 0
	}
	t := m.layout.Len() - m.contentHeight()
	if t < 0 {
		return 0
	}
	return t
}

// scrollLines moves the viewport by n lines, clamped to the current chapter.
// Fine-grained scrolling never crosses a chapter boundary: for books split into
// many short chapters, rolling over here would yank the reader to a new chapter
// on a single keypress. It sets a boundary notice so the keypress is not silent.
func (m *Model) scrollLines(n int) {
	if n == 0 {
		return
	}
	maxTop := m.maxTop()
	target := m.offset + n
	if target < 0 {
		m.offset = 0
		if m.offset == 0 {
			m.notice = "已到本章开头 · p 上一页"
		}
		return
	}
	if target > maxTop {
		m.offset = maxTop
		m.notice = "已到本章末尾 · n 下一页"
		return
	}
	m.offset = target
}

// scrollPage moves the viewport by n lines and flows across chapter boundaries:
// paging past the bottom opens the next chapter, paging above the top opens the
// previous chapter pinned to its last page.
func (m *Model) scrollPage(n int) {
	if n == 0 {
		return
	}
	maxTop := m.maxTop()
	if n > 0 {
		if m.offset >= maxTop {
			if m.spine < m.book.NumChapters()-1 {
				m.loadChapter(m.spine+1, render.Anchor{})
			} else {
				m.notice = "已经是最后一页了"
			}
			return
		}
		m.offset += n
		if m.offset > maxTop {
			m.offset = maxTop
		}
		return
	}
	if m.offset <= 0 {
		if m.spine > 0 {
			m.loadChapter(m.spine-1, render.Anchor{})
			m.offset = m.maxTop()
		} else {
			m.notice = "已经是第一页了"
		}
		return
	}
	m.offset += n
	if m.offset < 0 {
		m.offset = 0
	}
}

func (m *Model) nextChapter() {
	if m.spine < m.book.NumChapters()-1 {
		m.loadChapter(m.spine+1, render.Anchor{})
	}
}

func (m *Model) prevChapter() {
	if m.spine > 0 {
		m.loadChapter(m.spine-1, render.Anchor{})
	}
}

func (m Model) findSpine(href string) int {
	for i, c := range m.book.Chapters() {
		if c.Href == href {
			return i
		}
	}
	return -1
}

// Update handles reader key presses. Global keys are handled by the caller.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	kp, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}
	key := kp.String()

	if m.showTOC {
		switch {
		case key == "esc" || m.keys.Has(config.TOC, key):
			m.showTOC = false
		case m.keys.Has(config.Down, key):
			if m.tocCursor < len(m.book.TOC())-1 {
				m.tocCursor++
			}
		case m.keys.Has(config.Up, key):
			if m.tocCursor > 0 {
				m.tocCursor--
			}
		case key == "enter":
			entries := m.book.TOC()
			if m.tocCursor >= 0 && m.tocCursor < len(entries) {
				if idx := m.findSpine(entries[m.tocCursor].Href); idx >= 0 {
					m.loadChapter(idx, render.Anchor{})
				}
			}
			m.showTOC = false
		}
		return m, nil
	}

	m.notice = ""
	switch {
	case m.keys.Has(config.Down, key):
		m.scrollLines(1)
	case m.keys.Has(config.Up, key):
		m.scrollLines(-1)
	case m.keys.Has(config.PageDown, key):
		m.scrollPage(m.contentHeight())
	case m.keys.Has(config.PageUp, key):
		m.scrollPage(-m.contentHeight())
	case m.keys.Has(config.HalfDown, key):
		m.scrollLines(m.contentHeight() / 2)
	case m.keys.Has(config.HalfUp, key):
		m.scrollLines(-m.contentHeight() / 2)
	case m.keys.Has(config.NextChapter, key):
		m.nextChapter()
	case m.keys.Has(config.PrevChapter, key):
		m.prevChapter()
	case m.keys.Has(config.Top, key):
		m.offset = 0
	case m.keys.Has(config.Bottom, key):
		m.offset = m.maxTop()
	case m.keys.Has(config.TOC, key):
		m.showTOC = true
		m.tocCursor = 0
	}
	return m, nil
}

// View renders the reading screen.
func (m Model) View() string {
	if m.width <= 0 || m.height <= 0 || m.layout == nil {
		return "loading…"
	}
	if m.showTOC {
		return m.viewTOC()
	}

	ch := m.contentHeight()
	var b strings.Builder
	b.WriteString(m.header())
	b.WriteByte('\n')

	lines := m.layout.View(m.offset, ch)
	for _, ln := range lines {
		b.WriteString(m.renderLine(ln))
		b.WriteByte('\n')
	}
	for i := len(lines); i < ch; i++ {
		b.WriteByte('\n')
	}
	b.WriteString(m.status())
	return b.String()
}

func (m Model) header() string {
	title := m.book.ChapterTitle(m.spine)
	if m.err != nil {
		title = "⚠ " + m.err.Error()
	}
	cur, total := m.Page()
	right := fmt.Sprintf("%d/%d  %3d%% ", cur, total, int(m.Progress()*100+0.5))
	left := " " + truncate(title, m.width-render.Width(right)-2)
	gap := m.width - render.Width(left) - render.Width(right)
	if gap < 1 {
		gap = 1
	}
	return barStyle.Render(left + strings.Repeat(" ", gap) + right)
}

func (m Model) renderLine(ln render.Line) string {
	if strings.TrimSpace(ln.Text) == "" {
		return ""
	}
	if ln.Level > 0 {
		return "  " + headStyle.Render(ln.Text)
	}
	return "  " + bodyStyle.Render(ln.Text)
}

func (m Model) status() string {
	if m.notice != "" {
		return noticeStyle.Render(truncate(" "+m.notice, m.width))
	}
	left := " j/k 滚动 · n/space 下页 · p 上页 · J/K 章节 · t 目录 · b 摸鱼 · ? 帮助 · q 退出"
	return statusStyle.Render(truncate(left, m.width))
}

func (m Model) viewTOC() string {
	entries := m.book.TOC()
	var b strings.Builder
	b.WriteString(tocHeadStyle.Render(" 目录") + statusStyle.Render("   enter 跳转 · esc 关闭"))
	b.WriteByte('\n')

	visible := m.height - 1
	if visible < 1 {
		visible = 1
	}
	start := 0
	if m.tocCursor >= visible {
		start = m.tocCursor - visible + 1
	}
	for i := start; i < len(entries) && i < start+visible; i++ {
		label := strings.Repeat("  ", entries[i].Level) + entries[i].Title
		if i == m.tocCursor {
			b.WriteString(tocSelStyle.Render("▸ " + label))
		} else {
			b.WriteString(bodyStyle.Render("  " + label))
		}
		b.WriteByte('\n')
	}
	return b.String()
}

func truncate(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if render.Width(s) <= width {
		return s
	}
	// Trim whole runes so multi-byte characters (e.g. CJK) are never split.
	runes := []rune(s)
	for len(runes) > 0 && render.Width(string(runes)) > width {
		runes = runes[:len(runes)-1]
	}
	return string(runes)
}
