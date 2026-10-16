package boss

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

func TestLongInputDoesNotWrap(t *testing.T) {
	const width = 40
	m := New()
	m.SetSize(width, 10)
	m.Focus()
	for _, r := range "this is a rather long sentence typed into the fake input box" {
		m.InsertRune(r)
	}
	for _, ln := range strings.Split(m.View(), "\n") {
		if w := lipgloss.Width(ln); w > width {
			t.Errorf("line is %d cols wide (max %d): %q", w, width, ln)
		}
	}
}

func TestPageLines(t *testing.T) {
	m := New()
	m.SetSize(80, 24)
	if got := m.PageLines(); got != 19 {
		t.Errorf("PageLines = %d, want 19", got)
	}
	m.SetSize(80, 0) // falls back to a default
	if got := m.PageLines(); got < 1 {
		t.Errorf("PageLines should be at least 1, got %d", got)
	}
}
func TestViewFillsHeight(t *testing.T) {
	m := New()
	m.SetSize(80, 24)
	out := m.View()
	if got := strings.Count(out, "\n") + 1; got != 24 {
		t.Errorf("boss view rendered %d lines, want 24", got)
	}
	if !strings.Contains(out, "shortcuts") {
		t.Error("status line is missing from the view")
	}
}

func TestAdvanceAppendsAndRespectsScroll(t *testing.T) {
	m := New()
	start := m.EntryCount()
	m.Advance()
	if m.EntryCount() != start+1 {
		t.Fatalf("Advance did not append (start=%d now=%d)", start, m.EntryCount())
	}

	// While a reply is pending, Advance must not interleave output.
	m.InsertRune('h')
	m.InsertRune('i')
	m, _ = m.Submit()
	if !m.Thinking() {
		t.Fatal("expected thinking state")
	}
	n := m.EntryCount()
	m.Advance()
	if m.EntryCount() != n {
		t.Error("Advance should be a no-op while thinking")
	}

	// While the user has scrolled up, Advance must not yank the view.
	m.thinking = false
	m.ScrollBy(3)
	n = m.EntryCount()
	m.Advance()
	if m.EntryCount() != n {
		t.Error("Advance should be a no-op while scrolled up")
	}
}
