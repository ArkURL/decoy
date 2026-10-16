// Package render turns chapter paragraphs into wrapped screen lines and
// provides width-independent position mapping.
package render

import (
	"strings"

	"github.com/rivo/uniseg"
)

// Wrap breaks text into lines no wider than width display columns. Explicit
// newlines in the input force a break. It is East-Asian-width and grapheme
// cluster aware, so CJK and emoji wrap correctly.
func Wrap(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}
	var out []string
	for _, logical := range strings.Split(text, "\n") {
		out = append(out, wrapLine(logical, width)...)
	}
	if len(out) == 0 {
		out = []string{""}
	}
	return out
}

type cell struct {
	s string
	w int
}

func wrapLine(s string, width int) []string {
	g := uniseg.NewGraphemes(s)
	var cur []cell
	curW := 0
	spaceIdx := -1
	var out []string

	emit := func(cells []cell) {
		var b strings.Builder
		for _, c := range cells {
			b.WriteString(c.s)
		}
		out = append(out, strings.TrimRight(b.String(), " "))
	}

	for g.Next() {
		c := cell{s: g.Str(), w: g.Width()}
		if c.s == " " {
			if len(cur) == 0 {
				continue // no leading space
			}
			if curW+c.w > width {
				// Line is full: break here and drop the trailing space.
				emit(cur)
				cur, curW, spaceIdx = cur[:0], 0, -1
				continue
			}
			cur = append(cur, c)
			curW += c.w
			spaceIdx = len(cur) - 1
			continue
		}
		if curW+c.w > width && len(cur) > 0 {
			if spaceIdx >= 0 {
				emit(cur[:spaceIdx])
				cur = append([]cell(nil), cur[spaceIdx+1:]...)
				curW, spaceIdx = 0, -1
				for i, cc := range cur {
					curW += cc.w
					if cc.s == " " {
						spaceIdx = i
					}
				}
			} else {
				emit(cur)
				cur = cur[:0]
				curW = 0
			}
		}
		cur = append(cur, c)
		curW += c.w
	}
	if len(cur) > 0 {
		emit(cur)
	}
	if len(out) == 0 {
		out = []string{""}
	}
	return out
}

// Width returns the display width of s in terminal columns.
func Width(s string) int {
	return uniseg.StringWidth(s)
}
