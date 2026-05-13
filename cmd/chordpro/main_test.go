package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sample = `{title: Swing Low}
{subtitle: Traditional spiritual}
{artist: Various}
{key: G}

{start_of_chorus}
Swing [D]low, sweet [G]chari-[D]ot,
Comin' for to [A7]carry me [D]home.
{end_of_chorus}

{start_of_verse}
I [D]looked over [G]Jordan and [D]what did I see,
Comin' for to [A7]carry me [D]home.
{end_of_verse}
`

func TestCLIText(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "song.cho")
	if err := os.WriteFile(in, []byte(sample), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	if err := run([]string{"-f", "text", in}, nil, &out, &errOut); err != nil {
		t.Fatalf("run: %v; stderr=%s", err, errOut.String())
	}
	for _, want := range []string{"Swing Low", "Traditional spiritual", "Artist: Various", "Key: G", "[Chorus]", "Swing low"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("missing %q in text output:\n%s", want, out.String())
		}
	}
}

func TestCLIHTML(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "song.cho")
	if err := os.WriteFile(in, []byte(sample), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	if err := run([]string{"-f", "html", in}, nil, &out, &errOut); err != nil {
		t.Fatalf("run: %v; stderr=%s", err, errOut.String())
	}
	for _, want := range []string{"<!DOCTYPE html>", "<title>Swing Low</title>", `<section class="chorus">`} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("missing %q in html output", want)
		}
	}
}

func TestCLITranspose(t *testing.T) {
	var out, errOut bytes.Buffer
	if err := run([]string{"-t", "2", "-f", "text"}, strings.NewReader("{key: C}\n[C]hi [G]there"), &out, &errOut); err != nil {
		t.Fatalf("run: %v; stderr=%s", err, errOut.String())
	}
	got := out.String()
	if !strings.Contains(got, "Key: D") {
		t.Errorf("key not transposed:\n%s", got)
	}
	// Original chord names should not appear after transposition.
	for _, gone := range []string{"[C]", "[G]"} {
		if strings.Contains(got, gone) {
			t.Errorf("untransposed %q present:\n%s", gone, got)
		}
	}
}

func TestCLIFlagsAfterInputFile(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "song.cho")
	if err := os.WriteFile(in, []byte("{title: Hi}\n[C]hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	// Flags appear after the positional input path — must still work.
	if err := run([]string{in, "-f", "text", "-t", "2"}, nil, &out, &errOut); err != nil {
		t.Fatalf("run: %v; stderr=%s", err, errOut.String())
	}
	if !strings.Contains(out.String(), "Hi") {
		t.Errorf("expected title in output:\n%s", out.String())
	}
	if strings.Contains(out.String(), "[C]") {
		t.Errorf("transpose flag not honored (chord untransposed):\n%s", out.String())
	}
}

func TestCLIBadFormat(t *testing.T) {
	var out, errOut bytes.Buffer
	if err := run([]string{"-f", "pdf"}, strings.NewReader(""), &out, &errOut); err == nil {
		t.Errorf("expected error for unknown format")
	}
}
