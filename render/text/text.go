// Package text renders an ast.Song as plain text with chords above lyrics.
package text

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/chris-skud/go-chordpro/ast"
	"github.com/chris-skud/go-chordpro/chord"
	"github.com/chris-skud/go-chordpro/render"
)

// Renderer renders to plain text.
type Renderer struct {
	render.Options
}

// New returns a text renderer with the given options.
func New(opts render.Options) *Renderer { return &Renderer{Options: opts} }

// Render writes the song to w.
func (r *Renderer) Render(w io.Writer, song *ast.Song) error {
	bw := bufio.NewWriter(w)
	defer bw.Flush()
	r.writeHeader(bw, song)

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
			r.writeLine(bw, v, false)
		case ast.Comment:
			r.writeComment(bw, v)
		case ast.ChorusRef:
			if lastChorus != nil {
				fmt.Fprintln(bw, "[Chorus]")
				for _, ln := range lastChorus.Lines {
					r.writeLine(bw, ln, lastChorus.Kind == ast.SectionTab)
				}
				fmt.Fprintln(bw)
			}
		case ast.Directive:
			// Skip silently for MVP.
		}
	}
	return nil
}

func (r *Renderer) writeHeader(w *bufio.Writer, song *ast.Song) {
	if t := song.MetaFirst("title"); t != "" {
		fmt.Fprintln(w, t)
		fmt.Fprintln(w, strings.Repeat("=", len(t)))
	}
	if st := song.MetaFirst("subtitle"); st != "" {
		fmt.Fprintln(w, st)
	}
	if a := song.MetaFirst("artist"); a != "" {
		fmt.Fprintln(w, "Artist: "+a)
	}
	if k := song.MetaFirst("key"); k != "" {
		shown := k
		if r.Transpose != 0 {
			if c, err := chord.Parse(k); err == nil {
				shown = c.Transpose(r.Transpose, chord.PreferAuto).String()
			}
		}
		fmt.Fprintln(w, "Key: "+shown)
	}
	if c := song.MetaFirst("capo"); c != "" {
		fmt.Fprintln(w, "Capo: "+c)
	}
	// Trailing blank line after header if anything was printed.
	if hasAny(song, "title", "subtitle", "artist", "key", "capo") {
		fmt.Fprintln(w)
	}
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
	label := sectionLabel(sec)
	if label != "" {
		fmt.Fprintln(w, "["+label+"]")
	}
	tab := sec.Kind == ast.SectionTab
	for _, ln := range sec.Lines {
		r.writeLine(w, ln, tab)
	}
	fmt.Fprintln(w)
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

func (r *Renderer) writeComment(w *bufio.Writer, c ast.Comment) {
	switch c.Style {
	case ast.CommentBox:
		bar := strings.Repeat("-", len(c.Text)+4)
		fmt.Fprintln(w, bar)
		fmt.Fprintln(w, "| "+c.Text+" |")
		fmt.Fprintln(w, bar)
	case ast.CommentItalic:
		fmt.Fprintln(w, "/"+c.Text+"/")
	default:
		fmt.Fprintln(w, c.Text)
	}
}

// writeLine renders a single line as a chord row above a lyric row. If verbatim
// is true the line is printed as-is (used inside tab sections).
func (r *Renderer) writeLine(w *bufio.Writer, line ast.Line, verbatim bool) {
	if verbatim {
		// Tab lines: lyrics-only tokens preserved verbatim.
		for _, t := range line.Tokens {
			if l, ok := t.(ast.Lyric); ok {
				fmt.Fprintln(w, l.Text)
				return
			}
		}
		fmt.Fprintln(w)
		return
	}
	if len(line.Tokens) == 0 {
		fmt.Fprintln(w)
		return
	}
	var chordRow, lyricRow strings.Builder
	for _, t := range line.Tokens {
		switch v := t.(type) {
		case ast.Chord:
			r.placeAbove(&chordRow, &lyricRow, r.transposeText(v.Raw))
		case ast.Annotation:
			r.placeAbove(&chordRow, &lyricRow, v.Text)
		case ast.Lyric:
			lyricRow.WriteString(v.Text)
		}
	}
	cr := strings.TrimRight(chordRow.String(), " ")
	lr := lyricRow.String()
	if cr != "" {
		fmt.Fprintln(w, cr)
	}
	if lr != "" || cr == "" {
		fmt.Fprintln(w, lr)
	}
}

// placeAbove positions text in chordRow at the current column of lyricRow,
// padding chordRow so adjacent chords keep at least one space between them.
// The lyric row is intentionally left alone — the chord row may extend past
// the lyric row, with subsequent lyric tokens flowing naturally.
func (r *Renderer) placeAbove(chordRow, lyricRow *strings.Builder, text string) {
	target := lyricRow.Len()
	if chordRow.Len() > 0 && chordRow.Len() >= target {
		target = chordRow.Len() + 1
	}
	if pad := target - chordRow.Len(); pad > 0 {
		chordRow.WriteString(strings.Repeat(" ", pad))
	}
	chordRow.WriteString(text)
}

// transposeText returns the chord text shifted by the renderer's Transpose
// option. If the text isn't a valid chord, it is returned unchanged.
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
