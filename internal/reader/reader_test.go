package reader

import (
	"testing"
	"unicode/utf8"

	"github.com/arkurl/cloak/internal/render"
)

func TestTruncateRuneSafe(t *testing.T) {
	s := "第一章 坚实基础与空中楼阁"
	for w := 1; w <= 24; w++ {
		got := truncate(s, w)
		if !utf8.ValidString(got) {
			t.Fatalf("width %d produced invalid UTF-8: %q", w, got)
		}
		if render.Width(got) > w {
			t.Errorf("width %d: result is %d wide: %q", w, render.Width(got), got)
		}
	}
	if got := truncate(s, 0); got != "" {
		t.Errorf("width 0 should return empty, got %q", got)
	}
	if got := truncate("short", 10); got != "short" {
		t.Errorf("short strings should pass through, got %q", got)
	}
}
