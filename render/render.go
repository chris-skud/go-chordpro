// Package render defines the common Renderer interface.
package render

import (
	"io"

	"github.com/chris-skud/go-chordpro/ast"
)

// Options configures rendering behavior shared across formats.
type Options struct {
	// Transpose shifts every chord by this many semitones (may be negative).
	Transpose int
	// NoStyle is honored by formats that embed styling (e.g. HTML).
	NoStyle bool
}

// Renderer is the interface implemented by all output formats.
type Renderer interface {
	Render(w io.Writer, song *ast.Song) error
}
