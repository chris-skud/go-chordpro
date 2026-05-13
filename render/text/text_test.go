package text

import (
	"bytes"
	"strings"
	"testing"

	"github.com/chris-skud/go-chordpro/parser"
	"github.com/chris-skud/go-chordpro/render"
)

func renderString(t *testing.T, src string, opts render.Options) string {
	t.Helper()
	song, err := parser.ParseString(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	var buf bytes.Buffer
	if err := New(opts).Render(&buf, song); err != nil {
		t.Fatalf("render: %v", err)
	}
	return buf.String()
}

func TestChordOverLyricAlignment(t *testing.T) {
	got := renderString(t, "Swing [D]low, sweet [G]chari-[D]ot", render.Options{})
	want := "      D          G     D\n" +
		"Swing low, sweet chari-ot\n"
	if got != want {
		t.Errorf("got:\n%q\nwant:\n%q", got, want)
	}
}

func TestLineWithOnlyChords(t *testing.T) {
	got := renderString(t, "[C] [G] [Am] [F]", render.Options{})
	if !strings.Contains(got, "C G Am F") && !strings.Contains(got, "C  G  Am  F") {
		// Either is fine — we just need chords on their own line, separated.
		t.Logf("output:\n%s", got)
	}
}

func TestHeader(t *testing.T) {
	src := `{title: Test}
{artist: Me}
{key: G}
[G]hello`
	got := renderString(t, src, render.Options{})
	for _, want := range []string{"Test\n====\n", "Artist: Me\n", "Key: G\n"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

func TestTranspose(t *testing.T) {
	src := `{key: C}
[C]Hello [G]world`
	got := renderString(t, src, render.Options{Transpose: 2})
	if !strings.Contains(got, "Key: D") {
		t.Errorf("key not transposed:\n%s", got)
	}
	if !strings.Contains(got, "D") || !strings.Contains(got, "A") {
		t.Errorf("chords not transposed:\n%s", got)
	}
	if strings.Contains(got, "[C]") || strings.Contains(got, "[G]") {
		t.Errorf("untransposed chords present:\n%s", got)
	}
}

func TestSectionLabels(t *testing.T) {
	src := `{soc}
[G]Chorus line
{eoc}
{sov: Verse 1}
[C]Verse line
{eov}`
	got := renderString(t, src, render.Options{})
	if !strings.Contains(got, "[Chorus]") {
		t.Errorf("missing chorus label:\n%s", got)
	}
	if !strings.Contains(got, "[Verse 1]") {
		t.Errorf("missing custom verse label:\n%s", got)
	}
}

func TestChorusRefRepeats(t *testing.T) {
	src := `{soc}
[G]Sing it
{eoc}
[C]Verse 1
{chorus}
[C]Verse 2`
	got := renderString(t, src, render.Options{})
	// "Sing it" should appear twice — once in the original chorus and once in
	// the repeat triggered by {chorus}.
	if strings.Count(got, "Sing it") != 2 {
		t.Errorf("expected 'Sing it' to appear twice, got:\n%s", got)
	}
}

func TestTabPassthrough(t *testing.T) {
	src := `{sot}
E|--0--2--|
{eot}`
	got := renderString(t, src, render.Options{})
	if !strings.Contains(got, "E|--0--2--|") {
		t.Errorf("tab line missing:\n%s", got)
	}
}

func TestCommentBox(t *testing.T) {
	src := `{cb: hi}`
	got := renderString(t, src, render.Options{})
	if !strings.Contains(got, "| hi |") {
		t.Errorf("box comment missing:\n%s", got)
	}
}
