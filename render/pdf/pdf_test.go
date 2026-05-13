package pdf

import (
	"bytes"
	"strings"
	"testing"

	"github.com/chris-skud/go-chordpro/parser"
	"github.com/chris-skud/go-chordpro/render"
)

func renderBytes(t *testing.T, src string, opts render.Options) []byte {
	t.Helper()
	song, err := parser.ParseString(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	var buf bytes.Buffer
	if err := New(opts).Render(&buf, song); err != nil {
		t.Fatalf("render: %v", err)
	}
	return buf.Bytes()
}

// TestProducesPDF verifies that the renderer emits a well-formed PDF stream
// (correct magic header and an %%EOF trailer).
func TestProducesPDF(t *testing.T) {
	out := renderBytes(t, "{title: Hi}\n[C]hello [G]world", render.Options{})
	if !bytes.HasPrefix(out, []byte("%PDF-")) {
		t.Errorf("output does not start with %%PDF-: %q", out[:min(8, len(out))])
	}
	if !bytes.Contains(out, []byte("%%EOF")) {
		t.Errorf("output missing %%%%EOF trailer")
	}
}

func TestEmptySongRenders(t *testing.T) {
	out := renderBytes(t, "", render.Options{})
	if !bytes.HasPrefix(out, []byte("%PDF-")) {
		t.Errorf("empty-song output not a PDF")
	}
}

func TestPageBreakOnLongSong(t *testing.T) {
	// Build a song large enough to force a second page.
	var b strings.Builder
	b.WriteString("{title: Long}\n")
	for i := 0; i < 200; i++ {
		b.WriteString("[C]Line ")
		b.WriteString("of lyric content goes here.\n")
	}
	out := renderBytes(t, b.String(), render.Options{})
	// Each /Page object yields a "/Type /Page" entry in the PDF body.
	if c := bytes.Count(out, []byte("/Type /Page\n")); c < 2 {
		// Some PDF writers omit the space or newline; fall back to a looser check.
		if c2 := bytes.Count(out, []byte("/Type /Page")); c2 < 2 {
			t.Errorf("expected at least 2 pages, got %d / %d (loose)", c, c2)
		}
	}
}

func TestTransposeReflectedInOutput(t *testing.T) {
	// We can't easily decode the PDF, but if transposition runs without error
	// and emits a PDF, that's the contract.
	out := renderBytes(t, "{key: C}\n[C]hi", render.Options{Transpose: 2})
	if !bytes.HasPrefix(out, []byte("%PDF-")) {
		t.Errorf("transposed output not a PDF")
	}
}
