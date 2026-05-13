// Command chordpro renders a ChordPro source file to text, HTML, or PDF.
//
// Output is always written to a file. If -o is not given, the output filename
// is derived from the source file's basename plus the format's extension, in
// the current working directory.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/chris-skud/go-chordpro/parser"
	"github.com/chris-skud/go-chordpro/render"
	htmlrender "github.com/chris-skud/go-chordpro/render/html"
	pdfrender "github.com/chris-skud/go-chordpro/render/pdf"
	textrender "github.com/chris-skud/go-chordpro/render/text"
)

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "chordpro:", err)
		os.Exit(1)
	}
	if err := run(os.Args[1:], cwd, os.Stdin, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "chordpro:", err)
		os.Exit(1)
	}
}

// extForFormat returns the canonical file extension for a format name.
func extForFormat(format string) (string, error) {
	switch format {
	case "text", "txt":
		return ".txt", nil
	case "html":
		return ".html", nil
	case "pdf":
		return ".pdf", nil
	default:
		return "", fmt.Errorf("unknown format %q (want text, html, or pdf)", format)
	}
}

func run(args []string, cwd string, stdin io.Reader, stderr io.Writer) error {
	fs := flag.NewFlagSet("chordpro", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var (
		format    string
		output    string
		noStyle   bool
		transpose int
	)
	fs.StringVar(&format, "format", "text", "output format: text, html, or pdf")
	fs.StringVar(&format, "f", "text", "output format (shorthand)")
	fs.StringVar(&output, "output", "", "output file (default: derived from input name)")
	fs.StringVar(&output, "o", "", "output file (shorthand)")
	fs.BoolVar(&noStyle, "no-style", false, "omit default CSS (html only)")
	fs.IntVar(&transpose, "transpose", 0, "transpose chords by N semitones (may be negative)")
	fs.IntVar(&transpose, "t", 0, "transpose (shorthand)")

	fs.Usage = func() {
		fmt.Fprintln(stderr, "usage: chordpro [flags] [input.cho]")
		fmt.Fprintln(stderr, "  Without -o, writes <input-basename>.<ext> in the current directory.")
		fmt.Fprintln(stderr, "  When reading from stdin, -o is required.")
		fs.PrintDefaults()
	}
	// Parse in a loop so flags may appear before or after the positional
	// input path. flag.Parse stops at the first non-flag; we collect it and
	// continue with the remainder.
	var positionals []string
	remaining := args
	for len(remaining) > 0 {
		if err := fs.Parse(remaining); err != nil {
			return err
		}
		rest := fs.Args()
		if len(rest) == 0 {
			break
		}
		positionals = append(positionals, rest[0])
		remaining = rest[1:]
	}

	// Validate format up front so a typo doesn't waste a parse.
	ext, err := extForFormat(format)
	if err != nil {
		return err
	}

	if len(positionals) > 1 {
		return fmt.Errorf("expected at most one input file, got %d", len(positionals))
	}

	// Resolve input.
	var src io.Reader = stdin
	var inputPath string
	if len(positionals) == 1 {
		inputPath = positionals[0]
		f, err := os.Open(inputPath)
		if err != nil {
			return err
		}
		defer f.Close()
		src = f
	}

	// Resolve output path. With no -o, derive from the input basename.
	if output == "" {
		if inputPath == "" {
			return fmt.Errorf("-o is required when reading from stdin")
		}
		base := filepath.Base(inputPath)
		base = strings.TrimSuffix(base, filepath.Ext(base))
		output = filepath.Join(cwd, base+ext)
	}

	song, err := parser.Parse(src)
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}

	outFile, err := os.Create(output)
	if err != nil {
		return err
	}
	defer outFile.Close()

	opts := render.Options{Transpose: transpose, NoStyle: noStyle}
	var r render.Renderer
	switch format {
	case "text", "txt":
		r = textrender.New(opts)
	case "html":
		r = htmlrender.New(opts)
	case "pdf":
		r = pdfrender.New(opts)
	}
	if err := r.Render(outFile, song); err != nil {
		return err
	}
	fmt.Fprintln(stderr, "wrote", output)
	return nil
}
