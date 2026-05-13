// Command chordpro renders a ChordPro source file to text or HTML.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/chris-skud/go-chordpro/parser"
	"github.com/chris-skud/go-chordpro/render"
	htmlrender "github.com/chris-skud/go-chordpro/render/html"
	textrender "github.com/chris-skud/go-chordpro/render/text"
)

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "chordpro:", err)
		os.Exit(1)
	}
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("chordpro", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var (
		format    string
		output    string
		noStyle   bool
		transpose int
	)
	fs.StringVar(&format, "format", "text", "output format: text or html")
	fs.StringVar(&format, "f", "text", "output format (shorthand)")
	fs.StringVar(&output, "output", "", "output file (default: stdout)")
	fs.StringVar(&output, "o", "", "output file (shorthand)")
	fs.BoolVar(&noStyle, "no-style", false, "omit default CSS (html only)")
	fs.IntVar(&transpose, "transpose", 0, "transpose chords by N semitones (may be negative)")
	fs.IntVar(&transpose, "t", 0, "transpose (shorthand)")

	fs.Usage = func() {
		fmt.Fprintln(stderr, "usage: chordpro [flags] [input.cho]")
		fmt.Fprintln(stderr, "  reads stdin if no input file is given.")
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

	// Resolve input.
	var src io.Reader = stdin
	if len(positionals) > 0 {
		if len(positionals) > 1 {
			return fmt.Errorf("expected at most one input file, got %d", len(positionals))
		}
		f, err := os.Open(positionals[0])
		if err != nil {
			return err
		}
		defer f.Close()
		src = f
	}

	song, err := parser.Parse(src)
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}

	// Resolve output.
	var out io.Writer = stdout
	if output != "" {
		f, err := os.Create(output)
		if err != nil {
			return err
		}
		defer f.Close()
		out = f
	}

	opts := render.Options{Transpose: transpose, NoStyle: noStyle}
	var r render.Renderer
	switch format {
	case "text", "txt":
		r = textrender.New(opts)
	case "html":
		r = htmlrender.New(opts)
	default:
		return fmt.Errorf("unknown format %q (want text or html)", format)
	}
	return r.Render(out, song)
}
