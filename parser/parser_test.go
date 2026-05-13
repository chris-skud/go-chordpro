package parser

import (
	"reflect"
	"testing"

	"github.com/chris-skud/go-chordpro/ast"
)

func TestMetadata(t *testing.T) {
	src := `{title: Swing Low}
{subtitle: Traditional}
{artist: Various}
{key: G}
{capo: 2}`
	song, err := ParseString(src)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string][]string{
		"title":    {"Swing Low"},
		"subtitle": {"Traditional"},
		"artist":   {"Various"},
		"key":      {"G"},
		"capo":     {"2"},
	}
	if !reflect.DeepEqual(song.Meta, want) {
		t.Errorf("Meta = %v, want %v", song.Meta, want)
	}
}

func TestShortAliases(t *testing.T) {
	src := `{t: T}
{st: Sub}
{c: hey}
{soc}
hello
{eoc}`
	song, err := ParseString(src)
	if err != nil {
		t.Fatal(err)
	}
	if song.MetaFirst("title") != "T" || song.MetaFirst("subtitle") != "Sub" {
		t.Errorf("aliases didn't expand: %+v", song.Meta)
	}
	if len(song.Items) < 2 {
		t.Fatalf("expected comment + section, got %d items", len(song.Items))
	}
	if c, ok := song.Items[0].(ast.Comment); !ok || c.Text != "hey" {
		t.Errorf("expected plain comment, got %#v", song.Items[0])
	}
	sec, ok := song.Items[1].(ast.Section)
	if !ok || sec.Kind != ast.SectionChorus {
		t.Fatalf("expected chorus section, got %#v", song.Items[1])
	}
	if len(sec.Lines) != 1 {
		t.Fatalf("expected 1 line in chorus, got %d", len(sec.Lines))
	}
}

func TestInlineChordsAndLyrics(t *testing.T) {
	song, err := ParseString("Swing [D]low, sweet [G]chari-[D]ot")
	if err != nil {
		t.Fatal(err)
	}
	if len(song.Items) != 1 {
		t.Fatalf("items: %d", len(song.Items))
	}
	line := song.Items[0].(ast.Line)
	want := []ast.Token{
		ast.Lyric{Text: "Swing "},
		ast.Chord{Raw: "D"},
		ast.Lyric{Text: "low, sweet "},
		ast.Chord{Raw: "G"},
		ast.Lyric{Text: "chari-"},
		ast.Chord{Raw: "D"},
		ast.Lyric{Text: "ot"},
	}
	if !reflect.DeepEqual(line.Tokens, want) {
		t.Errorf("tokens = %#v\n  want %#v", line.Tokens, want)
	}
}

func TestAnnotation(t *testing.T) {
	song, err := ParseString("Take it [*softly] now")
	if err != nil {
		t.Fatal(err)
	}
	line := song.Items[0].(ast.Line)
	if len(line.Tokens) != 3 {
		t.Fatalf("tokens: %#v", line.Tokens)
	}
	if a, ok := line.Tokens[1].(ast.Annotation); !ok || a.Text != "softly" {
		t.Errorf("got %#v", line.Tokens[1])
	}
}

func TestSections(t *testing.T) {
	src := `{start_of_verse}
[C]Verse line
{end_of_verse}
{soc}
[G]Chorus line
{eoc}`
	song, err := ParseString(src)
	if err != nil {
		t.Fatal(err)
	}
	if len(song.Items) != 2 {
		t.Fatalf("expected 2 sections, got %d items: %#v", len(song.Items), song.Items)
	}
	v, ok := song.Items[0].(ast.Section)
	if !ok || v.Kind != ast.SectionVerse || len(v.Lines) != 1 {
		t.Errorf("verse parsed wrong: %#v", song.Items[0])
	}
	c, ok := song.Items[1].(ast.Section)
	if !ok || c.Kind != ast.SectionChorus || len(c.Lines) != 1 {
		t.Errorf("chorus parsed wrong: %#v", song.Items[1])
	}
}

func TestTabPassthrough(t *testing.T) {
	src := `{start_of_tab}
E|--0--2--3--|
B|--1--3--0--|
{end_of_tab}`
	song, err := ParseString(src)
	if err != nil {
		t.Fatal(err)
	}
	if len(song.Items) != 1 {
		t.Fatalf("items: %d", len(song.Items))
	}
	sec := song.Items[0].(ast.Section)
	if sec.Kind != ast.SectionTab || len(sec.Lines) != 2 {
		t.Errorf("got %#v", sec)
	}
	if l, ok := sec.Lines[0].Tokens[0].(ast.Lyric); !ok || l.Text != "E|--0--2--3--|" {
		t.Errorf("tab line 0 = %#v", sec.Lines[0])
	}
}

func TestCommentVariants(t *testing.T) {
	src := `{comment: plain}
{comment_italic: italic}
{cb: boxed}`
	song, err := ParseString(src)
	if err != nil {
		t.Fatal(err)
	}
	if len(song.Items) != 3 {
		t.Fatalf("items: %d", len(song.Items))
	}
	want := []ast.Comment{
		{Text: "plain", Style: ast.CommentPlain},
		{Text: "italic", Style: ast.CommentItalic},
		{Text: "boxed", Style: ast.CommentBox},
	}
	for i, w := range want {
		c, ok := song.Items[i].(ast.Comment)
		if !ok || c != w {
			t.Errorf("item %d = %#v, want %#v", i, song.Items[i], w)
		}
	}
}

func TestUnknownDirectivePreserved(t *testing.T) {
	song, err := ParseString("{x_custom_thing: hello}")
	if err != nil {
		t.Fatal(err)
	}
	if len(song.Items) != 1 {
		t.Fatalf("items: %d", len(song.Items))
	}
	d, ok := song.Items[0].(ast.Directive)
	if !ok || d.Name != "x_custom_thing" || d.Value != "hello" {
		t.Errorf("got %#v", song.Items[0])
	}
}

func TestHashCommentSkipped(t *testing.T) {
	song, err := ParseString("# this is a maintainer note\n[C]hello")
	if err != nil {
		t.Fatal(err)
	}
	if len(song.Items) != 1 {
		t.Fatalf("items: %d", len(song.Items))
	}
}

func TestUnmatchedBracketIsLiteral(t *testing.T) {
	song, err := ParseString("foo [bar baz")
	if err != nil {
		t.Fatal(err)
	}
	line := song.Items[0].(ast.Line)
	if len(line.Tokens) != 1 {
		t.Fatalf("tokens: %#v", line.Tokens)
	}
	if l, ok := line.Tokens[0].(ast.Lyric); !ok || l.Text != "foo [bar baz" {
		t.Errorf("got %#v", line.Tokens[0])
	}
}

func TestBareChorusRef(t *testing.T) {
	song, err := ParseString("{chorus}")
	if err != nil {
		t.Fatal(err)
	}
	if len(song.Items) != 1 {
		t.Fatalf("items: %d", len(song.Items))
	}
	if _, ok := song.Items[0].(ast.ChorusRef); !ok {
		t.Errorf("got %#v", song.Items[0])
	}
}

func TestBlankLinePreserved(t *testing.T) {
	song, err := ParseString("line one\n\nline two")
	if err != nil {
		t.Fatal(err)
	}
	if len(song.Items) != 3 {
		t.Fatalf("items: %d", len(song.Items))
	}
	if l, ok := song.Items[1].(ast.Line); !ok || len(l.Tokens) != 0 {
		t.Errorf("expected empty line, got %#v", song.Items[1])
	}
}
