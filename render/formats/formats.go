// Package formats maps output format names ("text", "html", "pdf") to their
// renderers, file extensions, and media types, so every front end (CLI, web
// server, WebAssembly) accepts the same names.
package formats

import (
	"fmt"

	"github.com/chris-skud/go-chordpro/render"
	htmlrender "github.com/chris-skud/go-chordpro/render/html"
	pdfrender "github.com/chris-skud/go-chordpro/render/pdf"
	textrender "github.com/chris-skud/go-chordpro/render/text"
)

// Format describes one output format.
type Format struct {
	Ext         string // file extension without the dot, e.g. "txt"
	ContentType string // HTTP media type
	New         func(render.Options) render.Renderer
}

var formats = map[string]Format{
	"text": {"txt", "text/plain; charset=utf-8", func(o render.Options) render.Renderer { return textrender.New(o) }},
	"html": {"html", "text/html; charset=utf-8", func(o render.Options) render.Renderer { return htmlrender.New(o) }},
	"pdf":  {"pdf", "application/pdf", func(o render.Options) render.Renderer { return pdfrender.New(o) }},
}

// Lookup returns the format with the given name. "txt" is accepted as an
// alias for "text".
func Lookup(name string) (Format, error) {
	if name == "txt" {
		name = "text"
	}
	f, ok := formats[name]
	if !ok {
		return Format{}, fmt.Errorf("unknown format %q (want text, html, or pdf)", name)
	}
	return f, nil
}
