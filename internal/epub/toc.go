package epub

import (
	"bytes"
	"encoding/xml"
	"strings"

	"golang.org/x/net/html"
)

// ncxDoc mirrors an EPUB 2 toc.ncx.
type ncxDoc struct {
	NavMap struct {
		Points []ncxPoint `xml:"navPoint"`
	} `xml:"navMap"`
}

type ncxPoint struct {
	NavLabel struct {
		Text string `xml:"text"`
	} `xml:"navLabel"`
	Content struct {
		Src string `xml:"src,attr"`
	} `xml:"content"`
	Points []ncxPoint `xml:"navPoint"`
}

// buildTOC locates and parses the table of contents. It prefers the EPUB 3 nav
// document and falls back to NCX, then to the spine order.
func (b *Book) buildTOC(pkg *opfPackage, manifest map[string]manifestItem, opfDir string) []TocEntry {
	// EPUB 3 navigation document.
	for _, it := range pkg.Manifest.Items {
		if hasProperty(it.Properties, "nav") {
			if data, err := b.read(resolve(opfDir, it.Href)); err == nil {
				if entries := parseNav(data, opfDir); len(entries) > 0 {
					return entries
				}
			}
			break
		}
	}

	// EPUB 2 NCX, referenced either from the spine or by media type.
	ncxHref := ""
	if pkg.Spine.Toc != "" {
		if it, ok := manifest[pkg.Spine.Toc]; ok {
			ncxHref = it.Href
		}
	}
	if ncxHref == "" {
		for _, it := range pkg.Manifest.Items {
			if strings.EqualFold(strings.TrimSpace(it.MediaType), "application/x-dtbncx+xml") {
				ncxHref = it.Href
				break
			}
		}
	}
	if ncxHref != "" {
		if data, err := b.read(resolve(opfDir, ncxHref)); err == nil {
			if entries := parseNCX(data, opfDir); len(entries) > 0 {
				return entries
			}
		}
	}

	// Last resort: derive from spine order.
	entries := make([]TocEntry, 0, len(b.chapters))
	for _, c := range b.chapters {
		entries = append(entries, TocEntry{Title: chapterFallbackTitle(c.Href), Href: c.Href})
	}
	return entries
}

func hasProperty(props, want string) bool {
	for _, p := range strings.Fields(props) {
		if p == want {
			return true
		}
	}
	return false
}

func parseNCX(data []byte, opfDir string) []TocEntry {
	var doc ncxDoc
	if err := xml.Unmarshal(data, &doc); err != nil {
		return nil
	}
	var out []TocEntry
	var walk func(points []ncxPoint, level int)
	walk = func(points []ncxPoint, level int) {
		for _, p := range points {
			title := strings.TrimSpace(p.NavLabel.Text)
			href := resolve(opfDir, p.Content.Src)
			if title != "" || href != "" {
				out = append(out, TocEntry{Title: title, Href: href, Level: level})
			}
			walk(p.Points, level+1)
		}
	}
	walk(doc.NavMap.Points, 0)
	return out
}

func parseNav(data []byte, opfDir string) []TocEntry {
	doc, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return nil
	}
	nav := findNav(doc)
	if nav == nil {
		return nil
	}
	ol := firstElement(nav, "ol")
	if ol == nil {
		return nil
	}
	var out []TocEntry
	var walk func(ol *html.Node, level int)
	walk = func(ol *html.Node, level int) {
		for li := ol.FirstChild; li != nil; li = li.NextSibling {
			if li.Type != html.ElementNode || li.Data != "li" {
				continue
			}
			for c := li.FirstChild; c != nil; c = c.NextSibling {
				if c.Type != html.ElementNode {
					continue
				}
				if c.Data == "a" {
					out = append(out, TocEntry{
						Title: strings.TrimSpace(textContent(c)),
						Href:  resolve(opfDir, attr(c, "href")),
						Level: level,
					})
					break
				}
				if c.Data == "span" {
					out = append(out, TocEntry{
						Title: strings.TrimSpace(textContent(c)),
						Level: level,
					})
					break
				}
			}
			for c := li.FirstChild; c != nil; c = c.NextSibling {
				if c.Type == html.ElementNode && c.Data == "ol" {
					walk(c, level+1)
				}
			}
		}
	}
	walk(ol, 0)
	return out
}

func findNav(n *html.Node) *html.Node {
	var nav *html.Node
	var walk func(*html.Node)
	walk = func(x *html.Node) {
		if nav != nil || x == nil {
			return
		}
		if x.Type == html.ElementNode && x.Data == "nav" {
			typ := attr(x, "epub:type")
			role := attr(x, "role")
			id := attr(x, "id")
			if strings.Contains(typ, "toc") || strings.Contains(role, "doc-toc") || id == "toc" || id == "toc-page" {
				nav = x
				return
			}
		}
		for c := x.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	if nav != nil {
		return nav
	}
	// Fall back to the first <nav> with an <ol>.
	walk = func(x *html.Node) {
		if nav != nil || x == nil {
			return
		}
		if x.Type == html.ElementNode && x.Data == "nav" && firstElement(x, "ol") != nil {
			nav = x
			return
		}
		for c := x.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return nav
}

func firstElement(n *html.Node, name string) *html.Node {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == name {
			return c
		}
	}
	return nil
}

func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

// textContent returns the concatenated text of a subtree, with <br> as newline.
func textContent(n *html.Node) string {
	var sb strings.Builder
	var walk func(*html.Node)
	walk = func(x *html.Node) {
		switch x.Type {
		case html.TextNode:
			sb.WriteString(x.Data)
		case html.ElementNode:
			switch x.Data {
			case "script", "style":
				return
			case "br":
				sb.WriteByte('\n')
				return
			}
		}
		for c := x.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return normalizeSpace(sb.String())
}

// normalizeSpace collapses whitespace runs, preserving explicit newlines.
func normalizeSpace(s string) string {
	lines := strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
	out := make([]string, 0, len(lines))
	for _, ln := range lines {
		ln = strings.Join(strings.Fields(ln), " ")
		out = append(out, strings.TrimSpace(ln))
	}
	return strings.Join(out, "\n")
}
