// Package html renders an ast.Song as a standalone HTML5 document.
package html

import (
	"bufio"
	"fmt"
	"html"
	"io"
	"strings"

	"github.com/chris-skud/go-chordpro/ast"
	"github.com/chris-skud/go-chordpro/chord"
	"github.com/chris-skud/go-chordpro/render"
)

// Renderer renders to HTML.
type Renderer struct {
	render.Options
}

// New returns an HTML renderer with the given options.
func New(opts render.Options) *Renderer { return &Renderer{Options: opts} }

const defaultCSS = `body { font-family: system-ui, sans-serif; max-width: 48rem; margin: 2rem auto; padding: 0 1rem; color: #222; }
header h1 { margin-bottom: 0.2em; }
header .meta { color: #555; font-size: 0.9em; }
section { margin: 1.2em 0; }
section.chorus { font-style: italic; border-left: 3px solid #ccc; padding-left: 0.8em; }
section.tab pre { background: #f6f6f6; padding: 0.6em; border-radius: 4px; overflow-x: auto; }
.section-label { font-weight: 600; color: #555; margin-bottom: 0.3em; }
p.line { margin: 0 0 0.6em 0; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; line-height: 1.1; white-space: pre-wrap; }
p.line .pair { display: inline-flex; flex-direction: column; align-items: flex-start; vertical-align: bottom; }
p.line .chord { color: #b00; font-weight: 600; }
p.line .annotation { color: #06c; font-style: italic; }
p.line .lyric { white-space: pre; }
p.comment { margin: 0.4em 0; color: #555; }
p.comment.italic { font-style: italic; }
p.comment.box { border: 1px solid #aaa; padding: 0.3em 0.6em; display: inline-block; }
`

// Render writes the song to w as an HTML5 document.
func (r *Renderer) Render(w io.Writer, song *ast.Song) error {
	bw := bufio.NewWriter(w)
	defer bw.Flush()

	title := song.MetaFirst("title")
	fmt.Fprintln(bw, `<!DOCTYPE html>`)
	fmt.Fprintln(bw, `<html lang="en">`)
	fmt.Fprintln(bw, `<head>`)
	fmt.Fprintln(bw, `<meta charset="utf-8">`)
	if title != "" {
		fmt.Fprintf(bw, "<title>%s</title>\n", html.EscapeString(title))
	}
	if !r.NoStyle {
		fmt.Fprintln(bw, `<style>`)
		fmt.Fprint(bw, defaultCSS)
		fmt.Fprintln(bw, `</style>`)
	}
	fmt.Fprintln(bw, `</head>`)
	fmt.Fprintln(bw, `<body>`)

	r.writeHeader(bw, song)

	fmt.Fprintln(bw, `<main>`)
	var lastChorus *ast.Section
	for _, it := range song.Items {
		switch v := it.(type) {
		case ast.Section:
			r.writeSection(bw, &v)
			if v.Kind == ast.SectionChorus {
				copy := v
				lastChorus = &copy
			}
		case ast.Line:
			r.writeFreeLine(bw, v)
		case ast.Comment:
			r.writeComment(bw, v)
		case ast.ChorusRef:
			if lastChorus != nil {
				repeat := *lastChorus
				r.writeSection(bw, &repeat)
			}
		case ast.Directive:
			// Skip for MVP.
		}
	}
	fmt.Fprintln(bw, `</main>`)
	fmt.Fprintln(bw, `</body></html>`)
	return nil
}

func (r *Renderer) writeHeader(w *bufio.Writer, song *ast.Song) {
	if !hasAny(song, "title", "subtitle", "artist", "key", "capo") {
		return
	}
	fmt.Fprintln(w, `<header>`)
	if t := song.MetaFirst("title"); t != "" {
		fmt.Fprintf(w, "  <h1>%s</h1>\n", html.EscapeString(t))
	}
	if st := song.MetaFirst("subtitle"); st != "" {
		fmt.Fprintf(w, "  <p class=\"subtitle\">%s</p>\n", html.EscapeString(st))
	}
	var meta []string
	if a := song.MetaFirst("artist"); a != "" {
		meta = append(meta, "Artist: "+html.EscapeString(a))
	}
	if k := song.MetaFirst("key"); k != "" {
		shown := k
		if r.Transpose != 0 {
			if c, err := chord.Parse(k); err == nil {
				shown = c.Transpose(r.Transpose, chord.PreferAuto).String()
			}
		}
		meta = append(meta, "Key: "+html.EscapeString(shown))
	}
	if c := song.MetaFirst("capo"); c != "" {
		meta = append(meta, "Capo: "+html.EscapeString(c))
	}
	if len(meta) > 0 {
		fmt.Fprintf(w, "  <p class=\"meta\">%s</p>\n", strings.Join(meta, " &middot; "))
	}
	fmt.Fprintln(w, `</header>`)
}

func hasAny(song *ast.Song, keys ...string) bool {
	for _, k := range keys {
		if song.MetaFirst(k) != "" {
			return true
		}
	}
	return false
}

func (r *Renderer) writeSection(w *bufio.Writer, sec *ast.Section) {
	class := string(sec.Kind)
	fmt.Fprintf(w, "<section class=%q>\n", class)
	if label := sectionLabel(sec); label != "" {
		fmt.Fprintf(w, "  <p class=\"section-label\">%s</p>\n", html.EscapeString(label))
	}
	if sec.Kind == ast.SectionTab {
		fmt.Fprintln(w, "  <pre>")
		for _, ln := range sec.Lines {
			for _, t := range ln.Tokens {
				if l, ok := t.(ast.Lyric); ok {
					fmt.Fprintln(w, html.EscapeString(l.Text))
				}
			}
		}
		fmt.Fprintln(w, "  </pre>")
	} else {
		for _, ln := range sec.Lines {
			r.writeLine(w, ln)
		}
	}
	fmt.Fprintln(w, "</section>")
}

func sectionLabel(sec *ast.Section) string {
	if sec.Label != "" {
		return sec.Label
	}
	switch sec.Kind {
	case ast.SectionVerse:
		return "Verse"
	case ast.SectionChorus:
		return "Chorus"
	case ast.SectionBridge:
		return "Bridge"
	case ast.SectionTab:
		return "Tab"
	case ast.SectionGrid:
		return "Grid"
	}
	return ""
}

func (r *Renderer) writeFreeLine(w *bufio.Writer, line ast.Line) {
	r.writeLine(w, line)
}

// writeLine emits a paragraph for a single line, pairing each chord/annotation
// with the lyric run that follows it so they stack visually via CSS.
func (r *Renderer) writeLine(w *bufio.Writer, line ast.Line) {
	if len(line.Tokens) == 0 {
		fmt.Fprintln(w, `  <p class="line">&nbsp;</p>`)
		return
	}
	fmt.Fprint(w, `  <p class="line">`)
	// Walk tokens, emitting a "pair" wrapper for each (chord-or-annotation, optional-lyric-run).
	i := 0
	for i < len(line.Tokens) {
		t := line.Tokens[i]
		switch v := t.(type) {
		case ast.Chord:
			lyric := nextLyric(line.Tokens, i+1)
			fmt.Fprintf(w, `<span class="pair"><span class="chord">%s</span><span class="lyric">%s</span></span>`,
				html.EscapeString(r.transposeText(v.Raw)),
				escapeLyric(lyric))
			i = consumeLyric(line.Tokens, i+1)
		case ast.Annotation:
			lyric := nextLyric(line.Tokens, i+1)
			fmt.Fprintf(w, `<span class="pair"><span class="annotation">%s</span><span class="lyric">%s</span></span>`,
				html.EscapeString(v.Text),
				escapeLyric(lyric))
			i = consumeLyric(line.Tokens, i+1)
		case ast.Lyric:
			// A lyric run with no preceding chord: emit as bare lyric.
			fmt.Fprintf(w, `<span class="lyric">%s</span>`, escapeLyric(v.Text))
			i++
		default:
			i++
		}
	}
	fmt.Fprintln(w, `</p>`)
}

// nextLyric returns the text of the lyric token at position i, or "" if the
// next token is not a lyric.
func nextLyric(tokens []ast.Token, i int) string {
	if i >= len(tokens) {
		return ""
	}
	if l, ok := tokens[i].(ast.Lyric); ok {
		return l.Text
	}
	return ""
}

// consumeLyric returns the index after the lyric at position i, or i if none.
func consumeLyric(tokens []ast.Token, i int) int {
	if i < len(tokens) {
		if _, ok := tokens[i].(ast.Lyric); ok {
			return i + 1
		}
	}
	return i
}

// escapeLyric escapes HTML and ensures a non-breaking placeholder when empty
// so the stacked layout reserves space.
func escapeLyric(s string) string {
	if s == "" {
		return "&nbsp;"
	}
	return html.EscapeString(s)
}

func (r *Renderer) writeComment(w *bufio.Writer, c ast.Comment) {
	class := "comment"
	switch c.Style {
	case ast.CommentItalic:
		class += " italic"
	case ast.CommentBox:
		class += " box"
	}
	fmt.Fprintf(w, "<p class=%q>%s</p>\n", class, html.EscapeString(c.Text))
}

func (r *Renderer) transposeText(raw string) string {
	if r.Transpose == 0 {
		return raw
	}
	c, err := chord.Parse(raw)
	if err != nil {
		return raw
	}
	return c.Transpose(r.Transpose, chord.PreferAuto).String()
}
