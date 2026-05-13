// Package parser parses ChordPro source into an ast.Song.
package parser

import (
	"bufio"
	"io"
	"strings"

	"github.com/chris-skud/go-chordpro/ast"
)

// Parse reads ChordPro source from r and returns the parsed song.
// Parsing is permissive: unknown directives are preserved as ast.Directive
// items, and malformed inline brackets are kept as literal text.
func Parse(r io.Reader) (*ast.Song, error) {
	song := &ast.Song{Meta: map[string][]string{}}
	p := &parser{song: song}
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		p.handleLine(sc.Text())
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	p.closeOpenSection()
	return song, nil
}

// ParseString is a convenience wrapper around Parse.
func ParseString(s string) (*ast.Song, error) {
	return Parse(strings.NewReader(s))
}

type parser struct {
	song    *ast.Song
	section *ast.Section // currently open section, if any
	inTab   bool         // true while between start_of_tab/end_of_tab
}

func (p *parser) appendItem(it ast.Item) {
	if p.section != nil {
		// Lines and tab lines live inside the section. Other items close it.
		if line, ok := it.(ast.Line); ok {
			p.section.Lines = append(p.section.Lines, line)
			return
		}
	}
	p.song.Items = append(p.song.Items, it)
}

func (p *parser) closeOpenSection() {
	if p.section != nil {
		p.song.Items = append(p.song.Items, *p.section)
		p.section = nil
	}
}

// handleLine routes a single source line.
func (p *parser) handleLine(raw string) {
	// Inside a tab section, lines are preserved verbatim until end_of_tab.
	if p.inTab {
		if name, _, ok := wholeLineDirective(raw); ok {
			canon := canonDirective(name)
			if canon == "end_of_tab" {
				p.inTab = false
				p.closeOpenSection()
				return
			}
		}
		if p.section != nil {
			p.section.Lines = append(p.section.Lines, ast.Line{
				Tokens: []ast.Token{ast.Lyric{Text: raw}},
			})
		}
		return
	}

	// Whole-line `#` comments are dropped.
	if strings.HasPrefix(strings.TrimLeft(raw, " \t"), "#") {
		return
	}

	// Whole-line directives (a single {...} with only whitespace around it).
	if name, value, ok := wholeLineDirective(raw); ok {
		p.handleDirective(name, value)
		return
	}

	// Otherwise: tokenize as a content line.
	line := tokenizeLine(raw)
	// A line that ended up empty (e.g. a blank source line) is preserved as an
	// empty line inside the current section so renderers can show stanza
	// breaks; outside a section, blank lines separate stanzas at the top level.
	if len(line.Tokens) == 0 {
		p.appendItem(ast.Line{})
		return
	}
	p.appendItem(line)
}

// wholeLineDirective reports whether raw, ignoring leading/trailing whitespace,
// consists of a single {name[: value]} directive. If so it returns name and value.
func wholeLineDirective(raw string) (name, value string, ok bool) {
	s := strings.TrimSpace(raw)
	if len(s) < 2 || s[0] != '{' || s[len(s)-1] != '}' {
		return "", "", false
	}
	// Must contain no other unbalanced braces.
	inner := s[1 : len(s)-1]
	if strings.ContainsAny(inner, "{}") {
		return "", "", false
	}
	if i := strings.IndexByte(inner, ':'); i >= 0 {
		return strings.TrimSpace(inner[:i]), strings.TrimSpace(inner[i+1:]), true
	}
	return strings.TrimSpace(inner), "", true
}

// directiveAlias maps short directive names to their canonical form.
var directiveAlias = map[string]string{
	"t":    "title",
	"st":   "subtitle",
	"c":    "comment",
	"ci":   "comment_italic",
	"cb":   "comment_box",
	"soc":  "start_of_chorus",
	"eoc":  "end_of_chorus",
	"sov":  "start_of_verse",
	"eov":  "end_of_verse",
	"sob":  "start_of_bridge",
	"eob":  "end_of_bridge",
	"sot":  "start_of_tab",
	"eot":  "end_of_tab",
	"sog":  "start_of_grid",
	"eog":  "end_of_grid",
	"np":   "new_page",
	"npp":  "new_physical_page",
	"colb": "column_break",
	"col":  "columns",
}

func canonDirective(name string) string {
	n := strings.ToLower(strings.TrimSpace(name))
	if c, ok := directiveAlias[n]; ok {
		return c
	}
	return n
}

// metaKeys is the set of directives that populate Song.Meta.
var metaKeys = map[string]bool{
	"title": true, "sorttitle": true, "subtitle": true,
	"artist": true, "sortartist": true, "composer": true,
	"lyricist": true, "copyright": true, "album": true,
	"year": true, "key": true, "time": true, "tempo": true,
	"duration": true, "capo": true, "tag": true, "meta": true,
}

// sectionStart maps start_of_* directives to their SectionKind.
var sectionStart = map[string]ast.SectionKind{
	"start_of_chorus": ast.SectionChorus,
	"start_of_verse":  ast.SectionVerse,
	"start_of_bridge": ast.SectionBridge,
	"start_of_tab":    ast.SectionTab,
	"start_of_grid":   ast.SectionGrid,
}

// sectionEnd is the set of end_of_* directives.
var sectionEnd = map[string]bool{
	"end_of_chorus": true,
	"end_of_verse":  true,
	"end_of_bridge": true,
	"end_of_tab":    true,
	"end_of_grid":   true,
}

func (p *parser) handleDirective(name, value string) {
	canon := canonDirective(name)

	switch {
	case metaKeys[canon]:
		p.song.Meta[canon] = append(p.song.Meta[canon], value)
		return
	case canon == "comment":
		p.appendItem(ast.Comment{Text: value, Style: ast.CommentPlain})
		return
	case canon == "comment_italic":
		p.appendItem(ast.Comment{Text: value, Style: ast.CommentItalic})
		return
	case canon == "comment_box":
		p.appendItem(ast.Comment{Text: value, Style: ast.CommentBox})
		return
	case canon == "chorus":
		// Bare {chorus} asks the renderer to repeat the most recent chorus.
		p.appendItem(ast.ChorusRef{})
		return
	}

	if kind, ok := sectionStart[canon]; ok {
		p.closeOpenSection()
		p.section = &ast.Section{Kind: kind, Label: value}
		if kind == ast.SectionTab {
			p.inTab = true
		}
		return
	}
	if sectionEnd[canon] {
		p.closeOpenSection()
		return
	}

	// Unknown / passthrough directive.
	p.appendItem(ast.Directive{Name: canon, Value: value})
}

// tokenizeLine breaks a content line into chord, annotation, and lyric tokens.
// Unmatched brackets are treated as literal text.
func tokenizeLine(s string) ast.Line {
	var tokens []ast.Token
	var buf strings.Builder
	flush := func() {
		if buf.Len() > 0 {
			tokens = append(tokens, ast.Lyric{Text: buf.String()})
			buf.Reset()
		}
	}
	for i := 0; i < len(s); {
		switch s[i] {
		case '[':
			end := strings.IndexByte(s[i+1:], ']')
			if end < 0 {
				// Unmatched: treat the rest as literal text.
				buf.WriteString(s[i:])
				i = len(s)
				continue
			}
			inner := s[i+1 : i+1+end]
			flush()
			if strings.HasPrefix(inner, "*") {
				tokens = append(tokens, ast.Annotation{Text: inner[1:]})
			} else {
				tokens = append(tokens, ast.Chord{Raw: inner})
			}
			i += end + 2
		case '{':
			// Inline directives are uncommon but legal. Split them out.
			end := strings.IndexByte(s[i+1:], '}')
			if end < 0 {
				buf.WriteString(s[i:])
				i = len(s)
				continue
			}
			// We don't currently emit inline directives as part of a line;
			// keep them literal for now to preserve the lyric flow.
			buf.WriteString(s[i : i+end+2])
			i += end + 2
		default:
			buf.WriteByte(s[i])
			i++
		}
	}
	flush()
	return ast.Line{Tokens: tokens}
}
