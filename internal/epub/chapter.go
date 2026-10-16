package epub

import (
	"bytes"
	"strings"

	"golang.org/x/net/html"
)

var blockTags = map[string]bool{
	"p": true, "div": true, "section": true, "article": true, "aside": true,
	"header": true, "footer": true, "main": true, "nav": true, "figure": true,
	"figcaption": true, "blockquote": true, "pre": true, "table": true,
	"tr": true, "td": true, "th": true, "li": true, "dd": true, "dt": true,
	"h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
	"hr": true, "body": true,
}

var skipTags = map[string]bool{
	"script": true, "style": true, "head": true, "title": true, "meta": true,
	"link": true, "svg": true, "img": true, "image": true,
}

// extractParas converts chapter XHTML into a flat list of paragraphs.
func extractParas(data []byte) []Para {
	doc, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return nil
	}
	body := findBody(doc)
	if body == nil {
		body = doc
	}

	var paras []Para
	var buf strings.Builder
	flush := func() {
		text := normalizeSpace(buf.String())
		buf.Reset()
		if text != "" {
			paras = append(paras, Para{Text: text})
		}
	}

	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			switch c.Type {
			case html.TextNode:
				buf.WriteString(c.Data)
			case html.ElementNode:
				switch {
				case skipTags[c.Data]:
					// ignore
				case c.Data == "br":
					buf.WriteByte('\n')
				case blockTags[c.Data]:
					if hasBlockChild(c) {
						flush()
						walk(c)
					} else {
						flush()
						text := normalizeSpace(textContent(c))
						if text != "" {
							paras = append(paras, Para{Text: text, Level: headingLevel(c.Data)})
						}
					}
				default:
					walk(c)
				}
			}
		}
	}
	walk(body)
	flush()

	paras = mergeHeadings(paras)
	return paras
}

func findBody(n *html.Node) *html.Node {
	var body *html.Node
	var walk func(*html.Node)
	walk = func(x *html.Node) {
		if body != nil {
			return
		}
		if x.Type == html.ElementNode && x.Data == "body" {
			body = x
			return
		}
		for c := x.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return body
}

func hasBlockChild(n *html.Node) bool {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && blockTags[c.Data] {
			return true
		}
	}
	return false
}

func headingLevel(tag string) int {
	switch tag {
	case "h1":
		return 1
	case "h2":
		return 2
	case "h3":
		return 3
	case "h4":
		return 4
	case "h5":
		return 5
	case "h6":
		return 6
	}
	return 0
}

// mergeHeadings joins consecutive normal paragraphs that contain no terminal
// punctuation, which is a common artefact of badly formatted books where a
// heading is split across several <p> tags. For MVP we keep blocks as-is but
// drop empty ones; the hook exists so render can rely on stable output.
func mergeHeadings(in []Para) []Para {
	out := in[:0]
	for _, p := range in {
		if strings.TrimSpace(p.Text) == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}
