package epub

import (
	"encoding/xml"
	"fmt"
	"net/url"
	"path"
	"strings"
)

// containerXML mirrors META-INF/container.xml.
type containerXML struct {
	Rootfiles struct {
		Rootfile []struct {
			FullPath  string `xml:"full-path,attr"`
			MediaType string `xml:"media-type,attr"`
		} `xml:"rootfile"`
	} `xml:"rootfiles"`
}

// parseContainer returns the OPF path referenced by container.xml.
func parseContainer(data []byte) (string, error) {
	var c containerXML
	if err := xml.Unmarshal(data, &c); err != nil {
		return "", fmt.Errorf("epub: parse container.xml: %w", err)
	}
	for _, rf := range c.Rootfiles.Rootfile {
		if strings.TrimSpace(rf.FullPath) != "" {
			return rf.FullPath, nil
		}
	}
	return "", fmt.Errorf("epub: no rootfile in container.xml")
}

// cleanZipPath normalises a path for lookup inside the archive.
func cleanZipPath(p string) string {
	p = strings.TrimSpace(p)
	p = strings.ReplaceAll(p, "\\", "/")
	p = strings.TrimPrefix(p, "/")
	if p == "" {
		return ""
	}
	return path.Clean(p)
}

// resolve turns a manifest/nav href into a zip path relative to dir.
func resolve(dir, href string) string {
	href = strings.TrimSpace(href)
	if href == "" {
		return ""
	}
	if i := strings.IndexByte(href, '#'); i >= 0 {
		href = href[:i]
	}
	if href == "" {
		return ""
	}
	if u, err := url.PathUnescape(href); err == nil {
		href = u
	}
	// Skip external / non-local references.
	if strings.Contains(href, "://") {
		return ""
	}
	if strings.HasPrefix(href, "/") {
		return cleanZipPath(href)
	}
	return cleanZipPath(path.Join(dir, href))
}

func isHTMLType(mediaType string) bool {
	switch strings.ToLower(strings.TrimSpace(mediaType)) {
	case "application/xhtml+xml", "text/html", "application/x-dtbook+xml":
		return true
	}
	return false
}
