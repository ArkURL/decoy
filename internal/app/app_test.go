package app

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/arkurl/cloak/internal/config"
	"github.com/arkurl/cloak/internal/epub"
	"github.com/arkurl/cloak/internal/storage"
)

func testBook(t *testing.T) *epub.Book {
	t.Helper()
	files := map[string]string{
		"META-INF/container.xml": `<?xml version="1.0"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
 <rootfiles><rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/></rootfiles>
</container>`,
		"OEBPS/content.opf": `<?xml version="1.0"?>
<package xmlns="http://www.idpf.org/2007/opf" version="3.0" unique-identifier="id">
 <metadata xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>Test Book</dc:title></metadata>
 <manifest>
  <item id="c1" href="c1.xhtml" media-type="application/xhtml+xml"/>
  <item id="c2" href="c2.xhtml" media-type="application/xhtml+xml"/>
  <item id="nav" href="nav.xhtml" media-type="application/xhtml+xml" properties="nav"/>
 </manifest>
 <spine><itemref idref="c1"/><itemref idref="c2"/></spine>
</package>`,
		"OEBPS/nav.xhtml": `<html xmlns="http://www.w3.org/1999/xhtml" xmlns:epub="http://www.idpf.org/2007/ops"><body>
<nav epub:type="toc"><ol><li><a href="c1.xhtml">One</a></li><li><a href="c2.xhtml">Two</a></li></ol></nav></body></html>`,
		"OEBPS/c1.xhtml": `<html><body><h1>One</h1><p>` + longText + `</p></body></html>`,
		"OEBPS/c2.xhtml": `<html><body><h1>Two</h1><p>second chapter</p></body></html>`,
	}
	return openTestBook(t, files)
}

// testEmptyBook returns a book with no readable chapters.
func testEmptyBook(t *testing.T) *epub.Book {
	t.Helper()
	return openTestBook(t, map[string]string{
		"META-INF/container.xml": `<?xml version="1.0"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
 <rootfiles><rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/></rootfiles>
</container>`,
		"OEBPS/content.opf": `<?xml version="1.0"?>
<package xmlns="http://www.idpf.org/2007/opf" version="3.0" unique-identifier="id">
 <metadata xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>Empty</dc:title></metadata>
 <manifest><item id="css" href="s.css" media-type="text/css"/></manifest>
 <spine></spine>
</package>`,
		"OEBPS/s.css": "body{}",
	})
}

// openTestBook writes files into a temp .epub and opens it.
func openTestBook(t *testing.T, files map[string]string) *epub.Book {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "b.epub")
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	for name, body := range files {
		w, _ := zw.Create(name)
		_, _ = w.Write([]byte(body))
	}
	_ = zw.Close()
	_ = f.Close()
	b, err := epub.Open(p)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { b.Close() })
	return b
}

const longText = "第一章 这是一个足够长的段落，用来测试在多种宽度下的换行与滚动行为是否正确，包含中文与 English words 的混排。"

func press(k tea.Key) tea.KeyPressMsg { return tea.KeyPressMsg(k) }

func upd(t *testing.T, m Model, msg tea.Msg) (Model, tea.Cmd) {
	t.Helper()
	nm, cmd := m.Update(msg)
	mm, ok := nm.(Model)
	if !ok {
		t.Fatalf("model type %T", nm)
	}
	return mm, cmd
}

func TestBossKeyToggleAndTitle(t *testing.T) {
	cfg := config.Default()
	m := New(testBook(t), cfg, nil)
	m, _ = upd(t, m, tea.WindowSizeMsg{Width: 60, Height: 20})

	if got := m.View().WindowTitle; got == cfg.FakeTitle {
		t.Fatalf("title should be the real one initially, got %q", got)
	}

	m, _ = upd(t, m, press(tea.Key{Code: 'b', Text: "b"}))
	if m.mode != modeBoss {
		t.Fatal("b did not enter boss mode")
	}
	if got := m.View().WindowTitle; got != cfg.FakeTitle {
		t.Errorf("boss title = %q, want %q", got, cfg.FakeTitle)
	}

	m, _ = upd(t, m, press(tea.Key{Code: 'b', Mod: tea.ModCtrl}))
	if m.mode != modeReader {
		t.Fatal("ctrl+b did not return to reader")
	}
	if got := m.View().WindowTitle; got == cfg.FakeTitle {
		t.Errorf("title not restored: %q", got)
	}
}

func TestIdleTriggersBoss(t *testing.T) {
	m := New(testBook(t), config.Default(), nil)
	m, _ = upd(t, m, tea.WindowSizeMsg{Width: 60, Height: 20})
	m, _ = upd(t, m, idleMsg{m.idle.token})
	if m.mode != modeBoss {
		t.Fatal("idle did not enter boss mode")
	}
}

func TestBossInputDoesNotToggle(t *testing.T) {
	m := New(testBook(t), config.Default(), nil)
	m, _ = upd(t, m, tea.WindowSizeMsg{Width: 60, Height: 20})
	m, _ = upd(t, m, press(tea.Key{Code: 'b', Text: "b"})) // enter boss
	m, _ = upd(t, m, press(tea.Key{Code: 'i', Text: "i"})) // focus input
	if !m.boss.Focused() {
		t.Fatal("input not focused")
	}
	m, _ = upd(t, m, press(tea.Key{Code: 'b', Text: "b"})) // typing b
	if m.mode != modeBoss {
		t.Fatal("typing b in input toggled mode")
	}
	m, cmd := upd(t, m, press(tea.Key{Code: tea.KeyEnter}))
	if cmd == nil || !m.boss.Thinking() {
		t.Fatal("enter did not submit and schedule a reply")
	}
}

func TestBossBackEscapesFromInput(t *testing.T) {
	m := New(testBook(t), config.Default(), nil)
	m, _ = upd(t, m, tea.WindowSizeMsg{Width: 60, Height: 20})
	m, _ = upd(t, m, press(tea.Key{Code: 'b', Text: "b"}))
	m, _ = upd(t, m, press(tea.Key{Code: 'i', Text: "i"}))
	if !m.boss.Focused() {
		t.Fatal("input not focused")
	}
	m, _ = upd(t, m, press(tea.Key{Code: 'b', Mod: tea.ModCtrl}))
	if m.mode != modeReader {
		t.Fatal("ctrl+b did not escape from a focused boss input")
	}
}

func TestPagingFlowsAcrossChapters(t *testing.T) {
	m := New(testBook(t), config.Default(), nil)
	m, _ = upd(t, m, tea.WindowSizeMsg{Width: 40, Height: 20})

	// Both chapters fit on one page, so a page down must open the next chapter.
	m, _ = upd(t, m, press(tea.Key{Code: tea.KeySpace, Text: " "}))
	if got := m.reader.SpineIndex(); got != 1 {
		t.Fatalf("space did not page into the next chapter, spine=%d", got)
	}

	m, _ = upd(t, m, press(tea.Key{Code: tea.KeyPgUp}))
	if got := m.reader.SpineIndex(); got != 0 {
		t.Fatalf("pgup did not page back to the previous chapter, spine=%d", got)
	}
}

func TestPgDownArrowsPageWithinLongChapter(t *testing.T) {
	m := New(testBook(t), config.Default(), nil)
	m, _ = upd(t, m, tea.WindowSizeMsg{Width: 20, Height: 6})
	start := m.reader.SpineIndex()
	// A tall chapter at a narrow width has several pages; paging should move
	// the viewport without leaving chapter 0 on the first press.
	m, _ = upd(t, m, press(tea.Key{Code: tea.KeyPgDown}))
	if m.reader.SpineIndex() != start {
		t.Fatalf("first page down left chapter %d unexpectedly", start)
	}
}

func TestBossAutoAdvance(t *testing.T) {
	m := New(testBook(t), config.Default(), nil)
	m, _ = upd(t, m, tea.WindowSizeMsg{Width: 60, Height: 20})

	before := m.boss.EntryCount()
	m, cmd := upd(t, m, press(tea.Key{Code: 'b', Text: "b"}))
	if cmd == nil {
		t.Fatal("entering boss should arm the auto-update timer")
	}
	m, cmd = upd(t, m, autoMsg{m.auto.token})
	if m.boss.EntryCount() != before+1 {
		t.Fatalf("auto update did not append an entry (%d -> %d)", before, m.boss.EntryCount())
	}
	if cmd == nil {
		t.Fatal("auto update should re-arm the timer")
	}

	// After leaving boss mode a stale tick must not change anything.
	m, _ = upd(t, m, press(tea.Key{Code: 'b', Mod: tea.ModCtrl}))
	after := m.boss.EntryCount()
	m, _ = upd(t, m, autoMsg{m.auto.token})
	if m.boss.EntryCount() != after {
		t.Error("stale auto tick changed boss content")
	}
}

func TestJDoesNotLeaveChapter(t *testing.T) {
	m := New(testBook(t), config.Default(), nil)
	m, _ = upd(t, m, tea.WindowSizeMsg{Width: 40, Height: 20})
	// Chapter 0 fits on one page, so a line scroll must stay put rather than
	// silently jumping to the next chapter.
	m, _ = upd(t, m, press(tea.Key{Code: 'j', Text: "j"}))
	if got := m.reader.SpineIndex(); got != 0 {
		t.Fatalf("j jumped to chapter %d instead of staying put", got)
	}
	if !strings.Contains(m.reader.View(), "本章末尾") {
		t.Errorf("expected an end-of-chapter notice, got:\n%s", m.reader.View())
	}

	// A page key is still allowed to move on.
	m, _ = upd(t, m, press(tea.Key{Code: 'n', Text: "n"}))
	if got := m.reader.SpineIndex(); got != 1 {
		t.Fatalf("n did not page forward, spine=%d", got)
	}
}

func TestNandPPageKeys(t *testing.T) {
	m := New(testBook(t), config.Default(), nil)
	m, _ = upd(t, m, tea.WindowSizeMsg{Width: 40, Height: 20})
	m, _ = upd(t, m, press(tea.Key{Code: 'n', Text: "n"}))
	if got := m.reader.SpineIndex(); got != 1 {
		t.Fatalf("n did not page forward, spine=%d", got)
	}
	m, _ = upd(t, m, press(tea.Key{Code: 'p', Text: "p"}))
	if got := m.reader.SpineIndex(); got != 0 {
		t.Fatalf("p did not page back, spine=%d", got)
	}
}

func TestHeaderShowsPageNumbers(t *testing.T) {
	m := New(testBook(t), config.Default(), nil)
	m, _ = upd(t, m, tea.WindowSizeMsg{Width: 60, Height: 8})
	cur, total := m.reader.Page()
	if cur < 1 || total < cur {
		t.Fatalf("page indicator out of range: %d/%d", cur, total)
	}
	if !strings.Contains(m.reader.View(), "1/") {
		t.Errorf("header does not show the page number:\n%s", m.reader.View())
	}
}

func TestEmptyBookDoesNotPanic(t *testing.T) {
	m := New(testEmptyBook(t), config.Default(), nil)
	m, _ = upd(t, m, tea.WindowSizeMsg{Width: 40, Height: 10})
	_ = m.View() // must not panic on a nil layout
	m, _ = upd(t, m, press(tea.Key{Code: 'j', Text: "j"}))
	m, _ = upd(t, m, press(tea.Key{Code: 'n', Text: "n"}))
	_ = m.View()
}

func TestPeriodicSavePersistsPosition(t *testing.T) {
	store := storage.NewEmpty(filepath.Join(t.TempDir(), "progress.json"))
	book := testBook(t)
	m := New(book, config.Default(), store)
	m, _ = upd(t, m, tea.WindowSizeMsg{Width: 40, Height: 20})
	m, _ = upd(t, m, press(tea.Key{Code: 'n', Text: "n"})) // page to chapter 1
	m, _ = upd(t, m, saveMsg{})

	pos, ok := store.Get(book.Path)
	if !ok {
		t.Fatal("periodic save did not persist a position")
	}
	if pos.SpineIndex != 1 {
		t.Errorf("saved spine = %d, want 1", pos.SpineIndex)
	}
}

func TestReaderTOCAndResize(t *testing.T) {
	m := New(testBook(t), config.Default(), nil)
	m, _ = upd(t, m, tea.WindowSizeMsg{Width: 40, Height: 12})
	m, _ = upd(t, m, press(tea.Key{Code: 't', Text: "t"}))
	if !m.reader.ShowTOC() {
		t.Fatal("t did not open the TOC")
	}
	// Resize must not panic and should preserve the position.
	m, _ = upd(t, m, tea.WindowSizeMsg{Width: 100, Height: 30})
	if pos := m.reader.Position(); pos.SpineIndex != 0 {
		t.Errorf("spine = %d, want 0", pos.SpineIndex)
	}
}
