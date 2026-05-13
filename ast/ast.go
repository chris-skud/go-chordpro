// Package ast defines the parsed representation of a ChordPro song.
package ast

// Song is the top-level node produced by the parser.
type Song struct {
	// Meta holds metadata directives keyed by canonical (long) name.
	// Values are kept as a slice to support repeatable keys (e.g. multiple artists).
	Meta map[string][]string
	// Items is the ordered body content: sections, free lines outside sections,
	// stray directives, and comments.
	Items []Item
}

// MetaFirst returns the first value for key, or "" if absent.
func (s *Song) MetaFirst(key string) string {
	if v, ok := s.Meta[key]; ok && len(v) > 0 {
		return v[0]
	}
	return ""
}

// Item is anything that can appear at the top level of a song body.
type Item interface{ isItem() }

// SectionKind identifies the type of a paired section.
type SectionKind string

const (
	SectionVerse  SectionKind = "verse"
	SectionChorus SectionKind = "chorus"
	SectionBridge SectionKind = "bridge"
	SectionTab    SectionKind = "tab"
	SectionGrid   SectionKind = "grid"
)

// Section is a paired environment such as verse/chorus/bridge/tab/grid.
type Section struct {
	Kind  SectionKind
	Label string // optional label from {start_of_x: label}
	Lines []Line
}

func (Section) isItem() {}

// Line is a single rendered line within a section, or a free line outside one.
type Line struct {
	Tokens []Token
}

func (Line) isItem() {}

// Token is a part of a Line.
type Token interface{ isToken() }

// Chord is an inline chord like [G] or [Am7/E].
type Chord struct {
	Raw string // exact text between brackets
}

func (Chord) isToken() {}

// Annotation is an inline annotation like [*hold].
type Annotation struct {
	Text string // text after the leading "*"
}

func (Annotation) isToken() {}

// Lyric is plain text between chords/annotations.
type Lyric struct {
	Text string
}

func (Lyric) isToken() {}

// Directive is a directive that isn't otherwise transformed into structured
// state (sections, metadata, comments). Preserved so renderers and downstream
// tools can pass them through.
type Directive struct {
	Name  string // canonical (long) name
	Value string // may be empty
}

func (Directive) isItem() {}

// CommentStyle identifies how a {comment*} directive should be rendered.
type CommentStyle string

const (
	CommentPlain  CommentStyle = "plain"
	CommentItalic CommentStyle = "italic"
	CommentBox    CommentStyle = "box"
)

// Comment is the result of {comment}, {comment_italic}, or {comment_box}.
type Comment struct {
	Text  string
	Style CommentStyle
}

func (Comment) isItem() {}

// TabLine is a verbatim line inside a start_of_tab / end_of_tab section.
type TabLine struct {
	Raw string
}

// ChorusRef marks the location of a bare {chorus} directive, asking the
// renderer to repeat the most recent chorus inline.
type ChorusRef struct{}

func (ChorusRef) isItem() {}
