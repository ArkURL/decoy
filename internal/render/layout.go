package render

import "github.com/arkurl/decoy/internal/epub"

// CountLines returns how many screen lines the paragraphs occupy at width. It
// matches Build's line count but avoids allocating the lines.
func CountLines(paras []epub.Para, width int) int {
	if width <= 0 {
		width = 1
	}
	n := 0
	for i, p := range paras {
		if i > 0 {
			n++ // blank separator
		}
		n += len(Wrap(p.Text, width))
	}
	if n == 0 {
		return 1
	}
	return n
}

// Line is one screen line produced from a chapter.
type Line struct {
	Text       string
	Para       int // index of the source paragraph
	LineInPara int // line number within the paragraph, -1 for blank separators
	Level      int // heading level (0 = body text)
}

// Anchor identifies a reading position independently of terminal width.
type Anchor struct {
	Para       int
	LineInPara int
}

// Layout is a chapter rendered to screen lines at a fixed width.
type Layout struct {
	Width     int
	Lines     []Line
	paraStart []int // paraStart[p] = index in Lines of paragraph p's first line
}

// Build lays out paragraphs at the given width.
func Build(paras []epub.Para, width int) *Layout {
	l := &Layout{Width: width}
	if width <= 0 {
		width = 1
	}
	l.paraStart = make([]int, len(paras))
	for i, p := range paras {
		if i > 0 {
			l.Lines = append(l.Lines, Line{Text: "", Para: i, LineInPara: -1})
		}
		l.paraStart[i] = len(l.Lines)
		for j, ln := range Wrap(p.Text, width) {
			l.Lines = append(l.Lines, Line{Text: ln, Para: i, LineInPara: j, Level: p.Level})
		}
	}
	if len(l.Lines) == 0 {
		l.Lines = []Line{{Text: "", Para: 0, LineInPara: -1}}
	}
	return l
}

// Len returns the number of lines.
func (l *Layout) Len() int { return len(l.Lines) }

// View returns up to height lines starting at offset, clamped to bounds.
func (l *Layout) View(offset, height int) []Line {
	if offset < 0 {
		offset = 0
	}
	if offset > len(l.Lines) {
		offset = len(l.Lines)
	}
	end := offset + height
	if end > len(l.Lines) {
		end = len(l.Lines)
	}
	return l.Lines[offset:end]
}

// ClampOffset constrains an offset to the valid scroll range.
func (l *Layout) ClampOffset(offset int) int {
	if offset < 0 {
		return 0
	}
	if offset > len(l.Lines) {
		return len(l.Lines)
	}
	return offset
}

// AnchorOf returns the anchor for a line offset.
func (l *Layout) AnchorOf(offset int) Anchor {
	offset = l.ClampOffset(offset)
	if offset >= len(l.Lines) {
		return Anchor{Para: len(l.paraStart) - 1, LineInPara: 0}
	}
	ln := l.Lines[offset]
	return Anchor{Para: ln.Para, LineInPara: ln.LineInPara}
}

// OffsetOf returns the line offset for an anchor, clamped to bounds.
func (l *Layout) OffsetOf(a Anchor) int {
	if len(l.paraStart) == 0 {
		return 0
	}
	p := a.Para
	if p < 0 {
		p = 0
	}
	if p >= len(l.paraStart) {
		return len(l.Lines)
	}
	if a.LineInPara < 0 {
		// A blank separator line directly before the paragraph.
		off := l.paraStart[p] - 1
		if off < 0 {
			off = 0
		}
		return l.ClampOffset(off)
	}
	return l.ClampOffset(l.paraStart[p] + a.LineInPara)
}
