# go-chordpro

A Go port of the [ChordPro](https://www.chordpro.org/) lead-sheet processor.
Parses ChordPro source into an AST and renders to plain text (chords above
lyrics) or HTML.

This is an MVP: it covers the format's metadata, sections, inline chords and
annotations, comments, tabs, and transposition. PDF, chord diagrams, custom
chord definitions, and multi-column layout are not implemented.

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

  -f, --format    text|html      output format (default: text)
  -o, --output    FILE           write to FILE instead of stdout
  -t, --transpose N              shift chords by N semitones (may be negative)
      --no-style                 omit the default CSS in HTML output
```

If no input file is given, ChordPro source is read from stdin.

### Example

```sh
$ chordpro song.cho
```

```
Swing Low
=========
Key: G

[Chorus]
      D          G     D
Swing low, sweet chari-ot
              A7       D
Comin' for to carry me home.
```

Transpose up two semitones:

```sh
$ chordpro -t 2 song.cho
```

```
Swing Low
=========
Key: A

[Chorus]
      E          A     E
Swing low, sweet chari-ot
              B7       E
Comin' for to carry me home.
```

Render to HTML:

```sh
$ chordpro -f html song.cho > song.html
```

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
- `cmd/chordpro` — CLI

## Status / out of scope

Not implemented (yet):

- PDF / LaTeX output
- Chord diagram rendering and `{define}`
- Multi-column layout, page breaks
- Configuration files / templates
- Pango-style inline markup
- Capo math beyond preserving the `capo:` value in metadata

## License

MIT (or as configured by the repo).
