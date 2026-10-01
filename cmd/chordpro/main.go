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

	"github.com/chris-skud/go-chordpro/internal/server"
	"github.com/chris-skud/go-chordpro/parser"
	"github.com/chris-skud/go-chordpro/render"
	"github.com/chris-skud/go-chordpro/render/formats"
)

func main() {
	// Subcommand dispatch: `chordpro serve` launches the web UI; everything
	// else is the render flow (preserving the original CLI contract).
	if len(os.Args) > 1 && os.Args[1] == "serve" {
		if err := serveCmd(os.Args[2:], os.Stderr); err != nil {
			fmt.Fprintln(os.Stderr, "chordpro:", err)
			os.Exit(1)
		}
		return
	}
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

// serveCmd parses serve-subcommand flags and starts the embedded web server.
func serveCmd(args []string, stderr io.Writer) error {
	fs := flag.NewFlagSet("chordpro serve", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var addr string
	fs.StringVar(&addr, "addr", "127.0.0.1:8080", "address to listen on (host:port)")
	fs.Usage = func() {
		fmt.Fprintln(stderr, "usage: chordpro serve [--addr host:port]")
		fmt.Fprintln(stderr, "  Starts the local web editor. Defaults to 127.0.0.1:8080.")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	return server.Run(server.Config{Addr: addr})
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
	fmtInfo, err := formats.Lookup(format)
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
		output = filepath.Join(cwd, base+"."+fmtInfo.Ext)
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
	if err := fmtInfo.New(opts).Render(outFile, song); err != nil {
		return err
	}
	fmt.Fprintln(stderr, "wrote", output)
	return nil
}
