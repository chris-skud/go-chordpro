package html

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

func TestBasicDocument(t *testing.T) {
	got := renderString(t, "{title: Test}\n[C]Hello", render.Options{})
	for _, want := range []string{
		"<!DOCTYPE html>",
		"<title>Test</title>",
		"<style>",
		"<h1>Test</h1>",
		`<span class="chord">C</span>`,
		`<span class="lyric">Hello</span>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

func TestNoStyle(t *testing.T) {
	got := renderString(t, "{title: Test}", render.Options{NoStyle: true})
	if strings.Contains(got, "<style>") {
		t.Errorf("expected no <style>, got:\n%s", got)
	}
}

func TestHTMLEscaping(t *testing.T) {
	got := renderString(t, "{title: A & B <c>}\nfoo & bar", render.Options{})
	if strings.Contains(got, "A & B <c>") {
		t.Errorf("title not escaped:\n%s", got)
	}
	if !strings.Contains(got, "A &amp; B &lt;c&gt;") {
		t.Errorf("expected escaped title:\n%s", got)
	}
	if !strings.Contains(got, "foo &amp; bar") {
		t.Errorf("expected escaped lyric:\n%s", got)
	}
}

func TestChorusSemanticClass(t *testing.T) {
	got := renderString(t, "{soc}\n[G]ho\n{eoc}", render.Options{})
	if !strings.Contains(got, `<section class="chorus">`) {
		t.Errorf("missing chorus class:\n%s", got)
	}
}

func TestTabAsPre(t *testing.T) {
	got := renderString(t, "{sot}\nE|--0--|\n{eot}", render.Options{})
	if !strings.Contains(got, "<pre>") || !strings.Contains(got, "E|--0--|") {
		t.Errorf("tab not rendered as pre:\n%s", got)
	}
}

func TestTransposeKey(t *testing.T) {
	got := renderString(t, "{key: C}\n[C]hi", render.Options{Transpose: 2})
	if !strings.Contains(got, "Key: D") {
		t.Errorf("key not transposed in html meta:\n%s", got)
	}
	if !strings.Contains(got, `<span class="chord">D</span>`) {
		t.Errorf("chord not transposed:\n%s", got)
	}
}

func TestChorusRefRepeats(t *testing.T) {
	src := `{soc}
[G]Sing it
{eoc}
[C]Verse
{chorus}`
	got := renderString(t, src, render.Options{})
	if strings.Count(got, "Sing it") != 2 {
		t.Errorf("expected chorus to repeat:\n%s", got)
	}
}
