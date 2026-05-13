// Package pdf renders an ast.Song as a PDF document using gopdf.
//
// The renderer embeds Roboto Mono (SIL Open Font License) so output is
// self-contained — no system fonts required. The whole document is set in
// monospace, which suits chord-over-lyric alignment.
package pdf

import (
	"bytes"
	_ "embed"
	"fmt"
	"io"
	"strings"

	"github.com/signintech/gopdf"

	"github.com/chris-skud/go-chordpro/ast"
	"github.com/chris-skud/go-chordpro/chord"
	"github.com/chris-skud/go-chordpro/render"
)

//go:embed fonts/RobotoMono-Regular.ttf
var robotoMonoRegular []byte

//go:embed fonts/RobotoMono-Bold.ttf
var robotoMonoBold []byte

//go:embed fonts/RobotoMono-Italic.ttf
var robotoMonoItalic []byte

// Font family names registered with gopdf. Roboto Mono is monospaced so the
// "mono" face used for tabs is the same as the regular face.
const (
	fontRegular = "roboto-mono"
	fontBold    = "roboto-mono-bold"
	fontItalic  = "roboto-mono-italic"
	fontMono    = "roboto-mono"
)

// Layout constants (all units are PDF points: 72pt = 1 inch).
const (
	marginLeft   = 54.0 // 0.75"
	marginRight  = 54.0
	marginTop    = 54.0
	marginBottom = 54.0

	titleSize    = 18.0
	subtitleSize = 12.0
	metaSize     = 10.0
	sectionSize  = 11.0
	lyricSize    = 11.0
	chordSize    = 10.0
	commentSize  = 10.0
	tabSize      = 10.0

	lineGap      = 4.0 // extra gap below each rendered line
	chordPad     = 1.0 // vertical padding between chord row and lyric row
	chordSepMin  = 3.0 // minimum horizontal gap between adjacent chords
	sectionGap   = 8.0 // vertical gap before a section label
	stanzaGap    = 6.0 // vertical gap after a section
	commentGap   = 4.0
)

// Renderer renders to PDF.
type Renderer struct {
	render.Options
}

// New returns a PDF renderer with the given options.
func New(opts render.Options) *Renderer { return &Renderer{Options: opts} }

// Render writes the song to w as a single PDF document (A4 paper).
func (r *Renderer) Render(w io.Writer, song *ast.Song) error {
	pdf := &gopdf.GoPdf{}
	pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})
	if err := installFonts(pdf); err != nil {
		return err
	}
	pdf.AddPage()

	rc := &renderCtx{
		pdf:        pdf,
		opts:       r.Options,
		x:          marginLeft,
		y:          marginTop,
		pageHeight: gopdf.PageSizeA4.H,
		pageWidth:  gopdf.PageSizeA4.W,
	}

	rc.writeHeader(song)

	var lastChorus *ast.Section
	for _, it := range song.Items {
		switch v := it.(type) {
		case ast.Section:
			rc.writeSection(&v)
			if v.Kind == ast.SectionChorus {
				cp := v
				lastChorus = &cp
			}
		case ast.Line:
			rc.writeLine(v, false)
		case ast.Comment:
			rc.writeComment(v)
		case ast.ChorusRef:
			if lastChorus != nil {
				rc.writeSection(lastChorus)
			}
		case ast.Directive:
			// Skip silently for MVP.
		}
	}
	_, err := pdf.WriteTo(w)
	return err
}

// installFonts registers the embedded Roboto Mono faces with the PDF.
func installFonts(pdf *gopdf.GoPdf) error {
	fonts := []struct {
		name string
		data []byte
	}{
		{fontRegular, robotoMonoRegular},
		{fontBold, robotoMonoBold},
		{fontItalic, robotoMonoItalic},
	}
	for _, f := range fonts {
		if err := pdf.AddTTFFontByReader(f.name, bytes.NewReader(f.data)); err != nil {
			return fmt.Errorf("pdf: register %s: %w", f.name, err)
		}
	}
	return nil
}

// renderCtx holds the running render state — current cursor position and
// references to the document.
type renderCtx struct {
	pdf        *gopdf.GoPdf
	opts       render.Options
	x, y       float64
	pageHeight float64
	pageWidth  float64
}

// usableWidth returns the horizontal space available for content.
func (c *renderCtx) usableWidth() float64 {
	return c.pageWidth - marginLeft - marginRight
}

// ensureSpace advances to a new page if drawing a band of `needed` points
// vertically would overflow the bottom margin.
func (c *renderCtx) ensureSpace(needed float64) {
	if c.y+needed > c.pageHeight-marginBottom {
		c.pdf.AddPage()
		c.y = marginTop
	}
}

// setFont sets the active font family and size.
func (c *renderCtx) setFont(family string, size float64) {
	_ = c.pdf.SetFont(family, "", size)
}

// drawText draws a single line of text starting at (x, y), where y is the top
// of the line (gopdf uses a top-anchored cell baseline via Cell with SetXY).
func (c *renderCtx) drawText(x, y float64, text string) {
	c.pdf.SetXY(x, y)
	_ = c.pdf.Cell(nil, text)
}

// textWidth measures the width of text in the currently active font.
func (c *renderCtx) textWidth(text string) float64 {
	w, _ := c.pdf.MeasureTextWidth(text)
	return w
}

func (c *renderCtx) writeHeader(song *ast.Song) {
	if !hasAny(song, "title", "subtitle", "artist", "key", "capo") {
		return
	}
	if t := song.MetaFirst("title"); t != "" {
		c.setFont(fontBold, titleSize)
		c.ensureSpace(titleSize + 4)
		c.drawText(marginLeft, c.y, t)
		c.y += titleSize + 4
	}
	if st := song.MetaFirst("subtitle"); st != "" {
		c.setFont(fontItalic, subtitleSize)
		c.ensureSpace(subtitleSize + 2)
		c.drawText(marginLeft, c.y, st)
		c.y += subtitleSize + 2
	}
	var meta []string
	if a := song.MetaFirst("artist"); a != "" {
		meta = append(meta, "Artist: "+a)
	}
	if k := song.MetaFirst("key"); k != "" {
		shown := k
		if c.opts.Transpose != 0 {
			if ch, err := chord.Parse(k); err == nil {
				shown = ch.Transpose(c.opts.Transpose, chord.PreferAuto).String()
			}
		}
		meta = append(meta, "Key: "+shown)
	}
	if cap := song.MetaFirst("capo"); cap != "" {
		meta = append(meta, "Capo: "+cap)
	}
	if len(meta) > 0 {
		c.setFont(fontRegular, metaSize)
		c.ensureSpace(metaSize + 2)
		c.drawText(marginLeft, c.y, strings.Join(meta, "  ·  "))
		c.y += metaSize + 2
	}
	c.y += sectionGap
}

func hasAny(song *ast.Song, keys ...string) bool {
	for _, k := range keys {
		if song.MetaFirst(k) != "" {
			return true
		}
	}
	return false
}

func (c *renderCtx) writeSection(sec *ast.Section) {
	c.y += sectionGap
	if label := sectionLabel(sec); label != "" {
		c.setFont(fontBold, sectionSize)
		c.ensureSpace(sectionSize + 2)
		c.drawText(marginLeft, c.y, label)
		c.y += sectionSize + 2
	}
	if sec.Kind == ast.SectionTab {
		c.setFont(fontMono, tabSize)
		for _, ln := range sec.Lines {
			text := ""
			for _, t := range ln.Tokens {
				if l, ok := t.(ast.Lyric); ok {
					text = l.Text
					break
				}
			}
			c.ensureSpace(tabSize + lineGap)
			c.drawText(marginLeft, c.y, text)
			c.y += tabSize + lineGap
		}
	} else {
		for _, ln := range sec.Lines {
			c.writeLine(ln, false)
		}
	}
	c.y += stanzaGap
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

func (c *renderCtx) writeComment(cm ast.Comment) {
	c.y += commentGap
	switch cm.Style {
	case ast.CommentItalic:
		c.setFont(fontItalic, commentSize)
	default:
		c.setFont(fontRegular, commentSize)
	}
	c.ensureSpace(commentSize + lineGap)
	if cm.Style == ast.CommentBox {
		w := c.textWidth(cm.Text)
		// Light border around the comment text.
		c.pdf.SetLineWidth(0.5)
		c.pdf.Rectangle(marginLeft, c.y-1, marginLeft+w+8, c.y+commentSize+2, "D", 0, 0)
		c.drawText(marginLeft+4, c.y, cm.Text)
	} else {
		c.drawText(marginLeft, c.y, cm.Text)
	}
	c.y += commentSize + lineGap
}

// writeLine renders a single content line with chords positioned above the
// lyrics. Both rows are drawn at appropriate Y offsets.
func (c *renderCtx) writeLine(line ast.Line, _ bool) {
	if len(line.Tokens) == 0 {
		// Blank line — small vertical advance.
		c.ensureSpace(lyricSize)
		c.y += lyricSize / 2
		return
	}
	// Reserve vertical space for chord row + lyric row.
	rowHeight := chordSize + chordPad + lyricSize + lineGap
	c.ensureSpace(rowHeight)

	chordY := c.y
	lyricY := c.y + chordSize + chordPad

	x := marginLeft
	chordRowMaxX := marginLeft // tracks rightmost chord extent for separation

	for _, t := range line.Tokens {
		switch v := t.(type) {
		case ast.Chord:
			text := c.transposeText(v.Raw)
			c.setFont(fontBold, chordSize)
			placeX := x
			if chordRowMaxX > placeX {
				placeX = chordRowMaxX
			}
			c.drawText(placeX, chordY, text)
			chordRowMaxX = placeX + c.textWidth(text) + chordSepMin
		case ast.Annotation:
			c.setFont(fontItalic, chordSize)
			placeX := x
			if chordRowMaxX > placeX {
				placeX = chordRowMaxX
			}
			c.drawText(placeX, chordY, v.Text)
			chordRowMaxX = placeX + c.textWidth(v.Text) + chordSepMin
		case ast.Lyric:
			c.setFont(fontRegular, lyricSize)
			c.drawText(x, lyricY, v.Text)
			x += c.textWidth(v.Text)
		}
	}
	c.y = lyricY + lyricSize + lineGap
}

func (c *renderCtx) transposeText(raw string) string {
	if c.opts.Transpose == 0 {
		return raw
	}
	ch, err := chord.Parse(raw)
	if err != nil {
		return raw
	}
	return ch.Transpose(c.opts.Transpose, chord.PreferAuto).String()
}
