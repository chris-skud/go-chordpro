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
	cwd := t.TempDir()
	in := filepath.Join(cwd, "song.cho")
	if err := os.WriteFile(in, []byte(sample), 0o644); err != nil {
		t.Fatal(err)
	}
	var errOut bytes.Buffer
	if err := run([]string{"-f", "text", in}, cwd, nil, &errOut); err != nil {
		t.Fatalf("run: %v; stderr=%s", err, errOut.String())
	}
	out, err := os.ReadFile(filepath.Join(cwd, "song.txt"))
	if err != nil {
		t.Fatalf("output file missing: %v", err)
	}
	for _, want := range []string{"Swing Low", "Traditional spiritual", "Artist: Various", "Key: G", "[Chorus]", "Swing low"} {
		if !strings.Contains(string(out), want) {
			t.Errorf("missing %q in output:\n%s", want, out)
		}
	}
}

func TestCLIHTML(t *testing.T) {
	cwd := t.TempDir()
	in := filepath.Join(cwd, "song.cho")
	if err := os.WriteFile(in, []byte(sample), 0o644); err != nil {
		t.Fatal(err)
	}
	var errOut bytes.Buffer
	if err := run([]string{"-f", "html", in}, cwd, nil, &errOut); err != nil {
		t.Fatalf("run: %v; stderr=%s", err, errOut.String())
	}
	out, err := os.ReadFile(filepath.Join(cwd, "song.html"))
	if err != nil {
		t.Fatalf("output file missing: %v", err)
	}
	for _, want := range []string{"<!DOCTYPE html>", "<title>Swing Low</title>", `<section class="chorus">`} {
		if !strings.Contains(string(out), want) {
			t.Errorf("missing %q in html output", want)
		}
	}
}

func TestCLIDerivedNameInCWD(t *testing.T) {
	// Source is outside cwd; output must land in cwd using the source basename.
	srcDir := t.TempDir()
	cwd := t.TempDir()
	in := filepath.Join(srcDir, "my-song.cho")
	if err := os.WriteFile(in, []byte("{title: t}\nhi"), 0o644); err != nil {
		t.Fatal(err)
	}
	var errOut bytes.Buffer
	if err := run([]string{"-f", "text", in}, cwd, nil, &errOut); err != nil {
		t.Fatalf("run: %v; stderr=%s", err, errOut.String())
	}
	want := filepath.Join(cwd, "my-song.txt")
	if _, err := os.Stat(want); err != nil {
		t.Fatalf("expected output at %s: %v", want, err)
	}
	// And nothing in srcDir.
	if _, err := os.Stat(filepath.Join(srcDir, "my-song.txt")); err == nil {
		t.Errorf("unexpected output file in source dir")
	}
}

func TestCLIExplicitOutputOverrides(t *testing.T) {
	cwd := t.TempDir()
	in := filepath.Join(cwd, "a.cho")
	if err := os.WriteFile(in, []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	custom := filepath.Join(cwd, "custom-name.txt")
	var errOut bytes.Buffer
	if err := run([]string{"-o", custom, in}, cwd, nil, &errOut); err != nil {
		t.Fatalf("run: %v; stderr=%s", err, errOut.String())
	}
	if _, err := os.Stat(custom); err != nil {
		t.Fatalf("custom output missing: %v", err)
	}
	// Derived name should NOT exist.
	if _, err := os.Stat(filepath.Join(cwd, "a.txt")); err == nil {
		t.Errorf("derived output should not be created when -o is given")
	}
}

func TestCLIStdinRequiresOutput(t *testing.T) {
	cwd := t.TempDir()
	var errOut bytes.Buffer
	err := run([]string{"-f", "text"}, cwd, strings.NewReader("hi"), &errOut)
	if err == nil {
		t.Errorf("expected error when piping with no -o")
	}
	if !strings.Contains(err.Error(), "-o") {
		t.Errorf("error should mention -o, got: %v", err)
	}
}

func TestCLIStdinWithOutput(t *testing.T) {
	cwd := t.TempDir()
	out := filepath.Join(cwd, "piped.txt")
	var errOut bytes.Buffer
	if err := run([]string{"-o", out}, cwd, strings.NewReader("hello"), &errOut); err != nil {
		t.Fatalf("run: %v; stderr=%s", err, errOut.String())
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatalf("stdin output missing: %v", err)
	}
}

func TestCLITranspose(t *testing.T) {
	cwd := t.TempDir()
	in := filepath.Join(cwd, "t.cho")
	if err := os.WriteFile(in, []byte("{key: C}\n[C]hi [G]there"), 0o644); err != nil {
		t.Fatal(err)
	}
	var errOut bytes.Buffer
	if err := run([]string{"-t", "2", "-f", "text", in}, cwd, nil, &errOut); err != nil {
		t.Fatalf("run: %v; stderr=%s", err, errOut.String())
	}
	out, err := os.ReadFile(filepath.Join(cwd, "t.txt"))
	if err != nil {
		t.Fatal(err)
	}
	got := string(out)
	if !strings.Contains(got, "Key: D") {
		t.Errorf("key not transposed:\n%s", got)
	}
	for _, gone := range []string{"[C]", "[G]"} {
		if strings.Contains(got, gone) {
			t.Errorf("untransposed %q present:\n%s", gone, got)
		}
	}
}

func TestCLIFlagsAfterInputFile(t *testing.T) {
	cwd := t.TempDir()
	in := filepath.Join(cwd, "song.cho")
	if err := os.WriteFile(in, []byte("{title: Hi}\n[C]hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	var errOut bytes.Buffer
	if err := run([]string{in, "-f", "text", "-t", "2"}, cwd, nil, &errOut); err != nil {
		t.Fatalf("run: %v; stderr=%s", err, errOut.String())
	}
	out, err := os.ReadFile(filepath.Join(cwd, "song.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "Hi") {
		t.Errorf("expected title in output:\n%s", out)
	}
	if strings.Contains(string(out), "[C]") {
		t.Errorf("transpose flag not honored:\n%s", out)
	}
}

func TestCLIPDF(t *testing.T) {
	cwd := t.TempDir()
	in := filepath.Join(cwd, "song.cho")
	if err := os.WriteFile(in, []byte(sample), 0o644); err != nil {
		t.Fatal(err)
	}
	var errOut bytes.Buffer
	if err := run([]string{"-f", "pdf", in}, cwd, nil, &errOut); err != nil {
		t.Fatalf("run: %v; stderr=%s", err, errOut.String())
	}
	data, err := os.ReadFile(filepath.Join(cwd, "song.pdf"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		t.Errorf("output is not a PDF (first bytes: %q)", data[:8])
	}
}

func TestCLIBadFormat(t *testing.T) {
	cwd := t.TempDir()
	var errOut bytes.Buffer
	err := run([]string{"-f", "latex"}, cwd, strings.NewReader(""), &errOut)
	if err == nil {
		t.Errorf("expected error for unknown format")
	}
}
