package fuzzer

import (
	"encoding/xml"
	"html"
	"io"
	"net/http"
	"regexp"
	"strings"
)

// plainSnippetSize is how much of a body without markup goes into a message.
// Markup gets the whole snippet: the words can come well after a page's <head>.
const plainSnippetSize = 1 << 10

// responseMessage condenses a response into one line, as message does, after
// taking the text out of XML and HTML bodies: the start of a page is mostly
// markup, which would use up the message. If there is no text to find, or the
// format is any other, the message is the start of the body as it came.
func responseMessage(contentType string, body []byte, status int) string {
	text := ""
	switch markupOf(contentType, body) {
	case markupHTML:
		text = htmlText(string(body))
	case markupXML:
		text = xmlText(string(body))
	}
	if strings.TrimSpace(text) != "" {
		return message([]byte(text), status)
	}
	return message(body[:min(len(body), plainSnippetSize)], status)
}

type markup int

const (
	markupNone markup = iota
	markupHTML
	markupXML
)

// markupOf goes by the declared Content-Type, and sniffs the body only when the
// server didn't declare one.
func markupOf(contentType string, body []byte) markup {
	if strings.TrimSpace(contentType) == "" {
		contentType = http.DetectContentType(body)
	}
	media, _, _ := strings.Cut(strings.ToLower(contentType), ";")
	media = strings.TrimSpace(media)
	switch {
	case media == "text/html" || media == "application/xhtml+xml":
		return markupHTML
	case strings.HasSuffix(media, "/xml") || strings.HasSuffix(media, "+xml"):
		return markupXML
	}
	return markupNone
}

// xmlText renders a document as "name: text" pairs, attributes included, which
// also makes a SOAP fault readable: "faultcode: Server; faultstring: boom". It
// reads raw tokens because a body may be cut short or malformed, and whatever
// came before the problem is still useful. Entities are never expanded.
func xmlText(body string) string {
	dec := xml.NewDecoder(strings.NewReader(strings.ToValidUTF8(body, "")))
	dec.Strict = false
	// The text is already UTF-8 by now, whatever the document declares.
	dec.CharsetReader = func(_ string, input io.Reader) (io.Reader, error) { return input, nil }

	var parts []string
	var text strings.Builder
	name := ""
	flush := func() {
		content := strings.TrimSpace(text.String())
		text.Reset()
		switch {
		case content == "":
		case name == "":
			parts = append(parts, content)
		default:
			parts = append(parts, name+": "+content)
		}
	}
	for {
		token, err := dec.RawToken()
		if err != nil {
			flush()
			return strings.Join(parts, "; ")
		}
		switch t := token.(type) {
		case xml.StartElement:
			flush()
			name = t.Name.Local
			for _, attr := range t.Attr {
				// Namespace declarations and xml:lang say nothing about the result.
				if attr.Name.Space != "xmlns" && attr.Name.Space != "xml" && attr.Name.Local != "xmlns" {
					parts = append(parts, attr.Name.Local+": "+attr.Value)
				}
			}
		case xml.EndElement:
			flush()
			name = ""
		case xml.CharData:
			text.Write(t)
		}
	}
}

// Raw text elements: what they hold isn't content, and may contain any "<".
var htmlRawEnd = map[string]*regexp.Regexp{
	"script": regexp.MustCompile(`(?i)</script[\s/>]`),
	"style":  regexp.MustCompile(`(?i)</style[\s/>]`),
}

// htmlText returns the words of a page, so its title first. encoding/xml can't be
// used here: its non-strict mode still stops at the "a<b" of an inline script.
func htmlText(body string) string {
	s := strings.ToValidUTF8(body, "")
	var b strings.Builder
	for s != "" {
		open := strings.IndexByte(s, '<')
		if open < 0 {
			b.WriteString(html.UnescapeString(s))
			break
		}
		b.WriteString(html.UnescapeString(s[:open]))
		s = s[open:]

		if strings.HasPrefix(s, "<!--") {
			_, rest, found := strings.Cut(s[4:], "-->")
			if !found {
				break
			}
			s = rest
			continue
		}
		if len(s) < 2 || !startsTag(s[1]) {
			b.WriteByte('<') // a literal "<", as in "1 < 2"
			s = s[1:]
			continue
		}

		end := tagEnd(s)
		if end < 0 {
			break // cut off inside a tag
		}
		name := tagName(s[1:end])
		selfClosed := s[end-1] == '/'
		s = s[end+1:]
		b.WriteByte(' ') // tags separate words: <td>a</td><td>b</td> isn't "ab"

		if rawEnd := htmlRawEnd[name]; rawEnd != nil && !selfClosed {
			closing := rawEnd.FindStringIndex(s)
			if closing == nil {
				break
			}
			s = s[closing[0]:] // the closing tag itself goes through the loop
		}
	}
	return b.String()
}

func startsTag(c byte) bool {
	return c == '/' || c == '!' || c == '?' || ('a' <= c && c <= 'z') || ('A' <= c && c <= 'Z')
}

// tagEnd returns the index of the ">" closing the tag s starts with, or -1. A ">"
// inside a quoted attribute doesn't close it.
func tagEnd(s string) int {
	var quote byte
	for i := 1; i < len(s); i++ {
		switch c := s[i]; {
		case quote != 0:
			if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
		case c == '>':
			return i
		}
	}
	return -1
}

// tagName returns the lowercase name of a tag, given what is between its brackets.
func tagName(tag string) string {
	end := strings.IndexAny(tag, " \t\r\n/")
	if end == 0 { // a closing tag has no raw content to skip
		return ""
	}
	if end > 0 {
		tag = tag[:end]
	}
	return strings.ToLower(tag)
}
