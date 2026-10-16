// Package epub implements a small, tolerant EPUB 2/3 reader.
//
// It deliberately avoids strict validation so that real-world books that fail
// the spec still open. DRM-protected books are not supported.
package epub

import (
	"archive/zip"
	"fmt"
	"io"
	"path"
	"strings"
)

// Para is a block of text extracted from a chapter.
type Para struct {
	Text  string
	Level int // 0 = normal paragraph, 1..6 = heading level
}

// TocEntry is one navigation entry.
type TocEntry struct {
	Title string
	Href  string // resolved zip path, fragment stripped
	Level int
}

// Chapter is one spine document (a readable unit).
type Chapter struct {
	ID    string
	Href  string // resolved zip path
	Title string
}

// Book is an opened EPUB.
type Book struct {
	Path   string
	Title  string
	Author string
	Lang   string

	rc       *zip.ReadCloser
	chapters []Chapter
	toc      []TocEntry
}

// Open opens an EPUB file and reads its package document.
func Open(p string) (*Book, error) {
	rc, err := zip.OpenReader(p)
	if err != nil {
		return nil, fmt.Errorf("epub: open %s: %w", p, err)
	}
	b := &Book{Path: p, rc: rc}
	if err := b.parse(); err != nil {
		_ = rc.Close()
		return nil, err
	}
	return b, nil
}

// Close releases the underlying archive.
func (b *Book) Close() error {
	if b.rc != nil {
		return b.rc.Close()
	}
	return nil
}

// Chapters returns the spine documents in reading order.
func (b *Book) Chapters() []Chapter { return b.chapters }

// TOC returns navigation entries. It may be empty.
func (b *Book) TOC() []TocEntry { return b.toc }

// NumChapters returns the number of spine documents.
func (b *Book) NumChapters() int { return len(b.chapters) }

// ChapterTitle returns a best-effort title for chapter i.
func (b *Book) ChapterTitle(i int) string {
	if i < 0 || i >= len(b.chapters) {
		return ""
	}
	return b.chapters[i].Title
}

// ChapterParas loads and extracts paragraphs from chapter i.
func (b *Book) ChapterParas(i int) ([]Para, error) {
	if i < 0 || i >= len(b.chapters) {
		return nil, fmt.Errorf("epub: chapter %d out of range", i)
	}
	data, err := b.read(b.chapters[i].Href)
	if err != nil {
		return nil, err
	}
	return extractParas(data), nil
}

func (b *Book) parse() error {
	cdata, err := b.read("META-INF/container.xml")
	if err != nil {
		return err
	}
	opfPath, err := parseContainer(cdata)
	if err != nil {
		return err
	}
	opfPath = cleanZipPath(opfPath)
	opfDir := path.Dir(opfPath)

	odata, err := b.read(opfPath)
	if err != nil {
		return err
	}
	pkg, err := parseOPF(odata)
	if err != nil {
		return err
	}

	if len(pkg.Metadata.Title) > 0 {
		b.Title = strings.TrimSpace(pkg.Metadata.Title[0])
	}
	if len(pkg.Metadata.Creator) > 0 {
		b.Author = strings.TrimSpace(pkg.Metadata.Creator[0])
	}
	if len(pkg.Metadata.Language) > 0 {
		b.Lang = strings.TrimSpace(pkg.Metadata.Language[0])
	}

	manifest := make(map[string]manifestItem, len(pkg.Manifest.Items))
	for _, it := range pkg.Manifest.Items {
		manifest[it.ID] = it
	}

	for _, ref := range pkg.Spine.Itemrefs {
		if strings.EqualFold(strings.TrimSpace(ref.Linear), "no") {
			continue
		}
		it, ok := manifest[ref.IDRef]
		if !ok {
			continue
		}
		b.chapters = append(b.chapters, Chapter{ID: it.ID, Href: resolve(opfDir, it.Href)})
	}
	if len(b.chapters) == 0 {
		for _, it := range pkg.Manifest.Items {
			if isHTMLType(it.MediaType) {
				b.chapters = append(b.chapters, Chapter{ID: it.ID, Href: resolve(opfDir, it.Href)})
			}
		}
	}

	b.toc = b.buildTOC(pkg, manifest, opfDir)
	b.assignTitles()
	return nil
}

// assignTitles maps TOC titles onto chapters, falling back to the file name.
func (b *Book) assignTitles() {
	byHref := make(map[string]string, len(b.toc))
	for _, e := range b.toc {
		if e.Href == "" {
			continue
		}
		if _, ok := byHref[e.Href]; !ok {
			byHref[e.Href] = e.Title
		}
	}
	for i := range b.chapters {
		if t := byHref[b.chapters[i].Href]; t != "" {
			b.chapters[i].Title = t
			continue
		}
		b.chapters[i].Title = chapterFallbackTitle(b.chapters[i].Href)
	}
}

func chapterFallbackTitle(href string) string {
	base := path.Base(href)
	base = strings.TrimSuffix(base, path.Ext(base))
	base = strings.ReplaceAll(base, "_", " ")
	base = strings.ReplaceAll(base, "-", " ")
	base = strings.TrimSpace(base)
	if base == "" {
		return "Chapter"
	}
	return base
}

func (b *Book) read(name string) ([]byte, error) {
	f := b.find(name)
	if f == nil {
		return nil, fmt.Errorf("epub: %q not found in archive", name)
	}
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

func (b *Book) find(name string) *zip.File {
	name = cleanZipPath(name)
	for _, f := range b.rc.File {
		if cleanZipPath(f.Name) == name {
			return f
		}
	}
	lower := strings.ToLower(name)
	for _, f := range b.rc.File {
		if strings.ToLower(cleanZipPath(f.Name)) == lower {
			return f
		}
	}
	return nil
}
