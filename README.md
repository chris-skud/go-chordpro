# go-chordpro

A Go port of the [ChordPro](https://www.chordpro.org/) lead-sheet processor.
Parses ChordPro source into an AST and renders to plain text (chords above
lyrics), HTML, or PDF.

This is an MVP: it covers the format's metadata, sections, inline chords and
annotations, comments, tabs, and transposition. Chord diagrams, custom chord
definitions, and multi-column layout are not implemented.

## Install

```sh
go install github.com/chris-skud/go-chordpro/cmd/chordpro@latest
```

Or, from a checkout:

```sh
go build -o chordpro ./cmd/chordpro
```

## CLI

```
chordpro [flags] [input.cho]

  -f, --format    text|html|pdf  output format (default: text)
  -o, --output    FILE           output file (default: derived from input name)
  -t, --transpose N              shift chords by N semitones (may be negative)
      --no-style                 omit the default CSS in HTML output
```

Output is always written to a file. Without `-o`, the output filename is
derived from the input file's basename plus the format extension (`.txt`,
`.html`, or `.pdf`), in the current working directory. When reading from stdin,
`-o` is required.

### Examples

Render `song.cho` to `song.txt` in the current directory:

```sh
$ chordpro song.cho
wrote /current/dir/song.txt
```

Transpose up two semitones:

```sh
$ chordpro -t 2 song.cho
```

Render to HTML (lands at `song.html` in cwd):

```sh
$ chordpro -f html song.cho
```

Render to PDF (A4, embedded Roboto Mono — no system-font dependency):

```sh
$ chordpro -f pdf song.cho
```

Override the output path:

```sh
$ chordpro -f pdf -o /tmp/out.pdf song.cho
```

### Web UI

`chordpro serve` starts a local web editor with a textarea and a live HTML
preview. The HTML, CSS, and JS are embedded in the binary, so no separate
assets need to be deployed.

```sh
$ chordpro serve
chordpro web UI: http://127.0.0.1:8080/
```

Defaults to `127.0.0.1` (loopback only). Use `--addr 0.0.0.0:8080` to expose
on the network — only do that on a trusted network, since the render endpoint
accepts arbitrary ChordPro source.

Features:

- Open a `.cho` file or paste source into the editor
- Live HTML preview, debounced
- Transpose ± buttons (semitones)
- Save the edited `.cho` source back to the file it was opened from (⌘S / Ctrl+S); a `•` after the filename marks unsaved changes
- Download as text, HTML, or PDF using the format dropdown

There is no server-side filesystem coupling: all file access happens in the
browser. In Chromium-based browsers (Chrome, Edge, Arc) Open and Save use the
[File System Access API](https://developer.mozilla.org/en-US/docs/Web/API/File_System_API),
so Save overwrites the opened file in place (the browser asks for write
permission on the first save). If no file is open, Save prompts for a
location. Browsers without the API (Firefox, Safari) fall back to a regular
file input for Open and a `.cho` download for Save.

## Library

```go
import (
    "os"

    "github.com/chris-skud/go-chordpro/parser"
    "github.com/chris-skud/go-chordpro/render"
    textrender "github.com/chris-skud/go-chordpro/render/text"
)

func main() {
    song, _ := parser.ParseString(`{title: Hello}
[C]Hello [G]world`)
    textrender.New(render.Options{Transpose: 0}).Render(os.Stdout, song)
}
```

## Supported directives

**Metadata** (populates `Song.Meta`): `title`/`t`, `sorttitle`, `subtitle`/`st`,
`artist`, `sortartist`, `composer`, `lyricist`, `copyright`, `album`, `year`,
`key`, `time`, `tempo`, `duration`, `capo`, `tag`, `meta`.

**Sections**: `start_of_verse`/`sov`, `end_of_verse`/`eov`,
`start_of_chorus`/`soc`, `end_of_chorus`/`eoc`,
`start_of_bridge`/`sob`, `end_of_bridge`/`eob`,
`start_of_tab`/`sot`, `end_of_tab`/`eot`,
`start_of_grid`/`sog`, `end_of_grid`/`eog`. A bare `{chorus}` repeats the most
recent chorus.

**Comments**: `comment`/`c`, `comment_italic`/`ci`, `comment_box`/`cb`.

**Inline**: `[chord]` for chord markers, `[*text]` for annotations.

Unknown directives are preserved in the AST as `ast.Directive` items so callers
and future renderers can handle them.

## Packages

- `ast` — node types for the parsed song
- `parser` — `Parse(io.Reader)` and `ParseString(string)`
- `chord` — chord parsing and semitone transposition
- `render` — common `Renderer` interface and `Options`
- `render/text` — chord-over-lyric plain-text renderer
- `render/html` — standalone HTML5 renderer with optional embedded CSS
- `render/pdf` — PDF renderer (A4) with embedded Roboto Mono
- `internal/server` — embedded web editor (HTML/CSS/JS + `/api/render`)
- `cmd/chordpro` — CLI and `serve` subcommand

## Status / out of scope

Not implemented (yet):

- LaTeX output
- Chord diagram rendering and `{define}`
- Multi-column PDF layout, configurable paper size, custom fonts
- Configuration files / templates
- Pango-style inline markup
- Capo math beyond preserving the `capo:` value in metadata

## License

MIT (or as configured by the repo).

The PDF renderer embeds [Roboto Mono](https://fonts.google.com/specimen/Roboto+Mono)
by Christian Robertson under the SIL Open Font License 1.1
(`render/pdf/fonts/OFL.txt`).
