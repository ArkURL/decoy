package render

import (
	"reflect"
	"testing"

	"github.com/arkurl/decoy/internal/epub"
)

func TestWrap(t *testing.T) {
	tests := []struct {
		name  string
		text  string
		width int
		want  []string
	}{
		{"simple", "hello world", 5, []string{"hello", "world"}},
		{"long word", "abcdefgh", 3, []string{"abc", "def", "gh"}},
		{"cjk", "你好世界你好世界", 4, []string{"你好", "世界", "你好", "世界"}},
		{"emoji", "😀😀", 2, []string{"😀", "😀"}},
		{"explicit newline", "a\nb", 10, []string{"a", "b"}},
		{"mixed", "hi 你好 there", 7, []string{"hi 你好", "there"}},
		{"empty", "", 5, []string{""}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Wrap(tt.text, tt.width)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Wrap(%q,%d) = %#v, want %#v", tt.text, tt.width, got, tt.want)
			}
			for _, ln := range got {
				if w := Width(ln); w > tt.width {
					t.Errorf("line %q width %d > %d", ln, w, tt.width)
				}
			}
		})
	}
}

func TestCountLinesMatchesBuild(t *testing.T) {
	paras := []epub.Para{
		{Text: "hello world foo bar baz"},
		{Text: "第二段中文内容，需要换行。"},
		{Text: "x"},
	}
	for _, w := range []int{5, 10, 20, 40} {
		if got, want := CountLines(paras, w), Build(paras, w).Len(); got != want {
			t.Errorf("width %d: CountLines=%d, Build=%d", w, got, want)
		}
	}
}

func TestLayoutAnchorRoundTrip(t *testing.T) {
	paras := []epub.Para{
		{Text: "第一章 这是一个很长的中文段落，需要换行显示才能测试布局是否正确。", Level: 1},
		{Text: "Second paragraph with some english words that should wrap as well."},
		{Text: "第三段。"},
	}
	for _, width := range []int{10, 20, 40} {
		l := Build(paras, width)
		for off := 0; off < l.Len(); off++ {
			a := l.AnchorOf(off)
			got := l.OffsetOf(a)
			if got != off {
				t.Errorf("width=%d off=%d: AnchorOf->%+v OffsetOf->%d", width, off, a, got)
			}
		}
	}
}

func TestLayoutResizeKeepsParagraph(t *testing.T) {
	paras := []epub.Para{
		{Text: "para one"},
		{Text: "para two has more words to wrap around depending on width"},
	}
	narrow := Build(paras, 12)
	wide := Build(paras, 40)

	// Anchor somewhere inside paragraph 1 at narrow width.
	off := narrow.paraStart[1] + 1
	a := narrow.AnchorOf(off)
	wideOff := wide.OffsetOf(a)
	if ln := wide.Lines[wideOff]; ln.Para != 1 {
		t.Errorf("after resize landed in para %d, want 1", ln.Para)
	}
}
