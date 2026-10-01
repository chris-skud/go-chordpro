//go:build js && wasm

// Command chordpro-wasm exposes the ChordPro parser and renderers to
// JavaScript so the web UI can render songs with no server (e.g. as an
// offline PWA). Build with:
//
//	GOOS=js GOARCH=wasm go build -o chordpro.wasm ./cmd/chordpro-wasm
//
// It registers one global function:
//
//	chordproRender(source: string, transpose: number, format: string)
//	  -> { data: Uint8Array, contentType: string, ext: string } | { error: string }
package main

import (
	"bytes"
	"strings"
	"syscall/js"

	"github.com/chris-skud/go-chordpro/parser"
	"github.com/chris-skud/go-chordpro/render"
	"github.com/chris-skud/go-chordpro/render/formats"
)

func main() {
	js.Global().Set("chordproRender", js.FuncOf(renderJS))
	select {} // keep the Go runtime alive for callbacks
}

func renderJS(_ js.Value, args []js.Value) any {
	if len(args) != 3 {
		return errorResult("chordproRender(source, transpose, format): wrong number of arguments")
	}
	f, err := formats.Lookup(args[2].String())
	if err != nil {
		return errorResult(err.Error())
	}
	song, err := parser.Parse(strings.NewReader(args[0].String()))
	if err != nil {
		return errorResult("parse: " + err.Error())
	}
	var buf bytes.Buffer
	if err := f.New(render.Options{Transpose: args[1].Int()}).Render(&buf, song); err != nil {
		return errorResult("render: " + err.Error())
	}
	data := js.Global().Get("Uint8Array").New(buf.Len())
	js.CopyBytesToJS(data, buf.Bytes())
	return map[string]any{"data": data, "contentType": f.ContentType, "ext": f.Ext}
}

func errorResult(msg string) map[string]any {
	return map[string]any{"error": msg}
}
