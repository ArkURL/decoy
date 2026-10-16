package epub

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func writeEPUB(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "book.epub")
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return p
}

const containerXMLDoc = `<?xml version="1.0" encoding="utf-8"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`

const ch1 = `<html xmlns="http://www.w3.org/1999/xhtml"><head><title>c1</title></head>
<body><h1>Chapter One</h1><p>Hello world.</p><p>Second paragraph.</p></body></html>`

const ch2 = `<html xmlns="http://www.w3.org/1999/xhtml"><head><title>c2</title></head>
<body><h1>Chapter Two</h1><p>Another chapter.</p></body></html>`

func TestOpenEPUB2(t *testing.T) {
	opf := `<?xml version="1.0" encoding="utf-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="2.0" unique-identifier="bookid">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>Test Book</dc:title>
    <dc:creator>Jane Doe</dc:creator>
    <dc:language>en</dc:language>
  </metadata>
  <manifest>
    <item id="ch1" href="ch1.xhtml" media-type="application/xhtml+xml"/>
    <item id="ch2" href="ch2.xhtml" media-type="application/xhtml+xml"/>
    <item id="ncx" href="toc.ncx" media-type="application/x-dtbncx+xml"/>
  </manifest>
  <spine toc="ncx">
    <itemref idref="ch1"/>
    <itemref idref="ch2"/>
  </spine>
</package>`
	ncx := `<?xml version="1.0" encoding="utf-8"?>
<ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1">
  <navMap>
    <navPoint id="n1" playOrder="1"><navLabel><text>Chapter One</text></navLabel><content src="ch1.xhtml"/></navPoint>
    <navPoint id="n2" playOrder="2"><navLabel><text>Chapter Two</text></navLabel><content src="ch2.xhtml"/></navPoint>
  </navMap>
</ncx>`

	p := writeEPUB(t, map[string]string{
		"META-INF/container.xml": containerXMLDoc,
		"OEBPS/content.opf":      opf,
		"OEBPS/toc.ncx":          ncx,
		"OEBPS/ch1.xhtml":        ch1,
		"OEBPS/ch2.xhtml":        ch2,
	})

	b, err := Open(p)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer b.Close()

	if b.Title != "Test Book" {
		t.Errorf("Title = %q, want Test Book", b.Title)
	}
	if b.Author != "Jane Doe" {
		t.Errorf("Author = %q, want Jane Doe", b.Author)
	}
	if b.NumChapters() != 2 {
		t.Fatalf("NumChapters = %d, want 2", b.NumChapters())
	}
	if got := b.ChapterTitle(0); got != "Chapter One" {
		t.Errorf("ChapterTitle(0) = %q, want Chapter One", got)
	}
	if len(b.TOC()) != 2 {
		t.Fatalf("TOC len = %d, want 2", len(b.TOC()))
	}
	paras, err := b.ChapterParas(0)
	if err != nil {
		t.Fatalf("ChapterParas: %v", err)
	}
	if len(paras) != 3 {
		t.Fatalf("paras = %d want 3: %#v", len(paras), paras)
	}
	if paras[0].Level != 1 || paras[0].Text != "Chapter One" {
		t.Errorf("heading = %#v", paras[0])
	}
	if paras[1].Text != "Hello world." {
		t.Errorf("para[1] = %q", paras[1].Text)
	}
}

func TestOpenEPUB3Nav(t *testing.T) {
	opf := `<?xml version="1.0" encoding="utf-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="3.0" unique-identifier="bookid">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>Nav Book</dc:title>
  </metadata>
  <manifest>
    <item id="ch1" href="ch1.xhtml" media-type="application/xhtml+xml"/>
    <item id="ch2" href="ch2.xhtml" media-type="application/xhtml+xml"/>
    <item id="nav" href="nav.xhtml" media-type="application/xhtml+xml" properties="nav"/>
  </manifest>
  <spine>
    <itemref idref="ch1"/>
    <itemref idref="ch2"/>
  </spine>
</package>`
	nav := `<html xmlns="http://www.w3.org/1999/xhtml" xmlns:epub="http://www.idpf.org/2007/ops">
<body><nav epub:type="toc" id="toc"><ol>
  <li><a href="ch1.xhtml">Chapter One</a></li>
  <li><a href="ch2.xhtml">Chapter Two</a>
    <ol><li><a href="ch2.xhtml#s1">Section</a></li></ol>
  </li>
</ol></nav></body></html>`

	p := writeEPUB(t, map[string]string{
		"META-INF/container.xml": containerXMLDoc,
		"OEBPS/content.opf":      opf,
		"OEBPS/nav.xhtml":        nav,
		"OEBPS/ch1.xhtml":        ch1,
		"OEBPS/ch2.xhtml":        ch2,
	})

	b, err := Open(p)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer b.Close()

	toc := b.TOC()
	if len(toc) != 3 {
		t.Fatalf("TOC len = %d, want 3: %#v", len(toc), toc)
	}
	if toc[0].Title != "Chapter One" {
		t.Errorf("toc[0] = %#v", toc[0])
	}
	if toc[2].Level != 1 {
		t.Errorf("nested level = %d want 1", toc[2].Level)
	}
	if toc[2].Href != "OEBPS/ch2.xhtml" {
		t.Errorf("fragment not stripped: %q", toc[2].Href)
	}
}

func TestChapterFallbackTitle(t *testing.T) {
	if got := chapterFallbackTitle("OEBPS/some_chapter-1.xhtml"); got != "some chapter 1" {
		t.Errorf("got %q", got)
	}
}
