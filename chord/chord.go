// Package chord parses musical chord symbols (e.g. "C", "Am7", "F#m7/A")
// and supports semitone transposition.
package chord

import (
	"errors"
	"strings"
)

// Accidental represents a chromatic alteration of a natural pitch.
type Accidental int8

const (
	Natural Accidental = 0
	Sharp   Accidental = 1
	Flat    Accidental = -1
)

func (a Accidental) String() string {
	switch a {
	case Sharp:
		return "#"
	case Flat:
		return "b"
	}
	return ""
}

// Chord is a parsed chord symbol.
type Chord struct {
	Root     byte       // 'A'..'G'
	Acc      Accidental // sharp / flat / natural
	Quality  string     // everything after the root+accidental and before any '/' (e.g. "m7", "maj7", "sus4")
	Bass     byte       // 0 if no bass; otherwise 'A'..'G'
	BassAcc  Accidental
}

// String renders the chord back to text.
func (c Chord) String() string {
	var b strings.Builder
	b.WriteByte(c.Root)
	b.WriteString(c.Acc.String())
	b.WriteString(c.Quality)
	if c.Bass != 0 {
		b.WriteByte('/')
		b.WriteByte(c.Bass)
		b.WriteString(c.BassAcc.String())
	}
	return b.String()
}

// ErrEmpty is returned when parsing an empty string.
var ErrEmpty = errors.New("chord: empty")

// ErrBadRoot is returned when the first character is not a valid note letter.
var ErrBadRoot = errors.New("chord: bad root note")

// Parse parses a chord symbol. It is permissive about quality strings and only
// validates the root (and bass) notes.
func Parse(s string) (Chord, error) {
	if s == "" {
		return Chord{}, ErrEmpty
	}
	var c Chord
	rest, err := parseNote(s, &c.Root, &c.Acc)
	if err != nil {
		return Chord{}, err
	}
	if i := strings.IndexByte(rest, '/'); i >= 0 {
		c.Quality = rest[:i]
		bassPart := rest[i+1:]
		if bassPart == "" {
			return Chord{}, errors.New("chord: empty bass after '/'")
		}
		remainder, err := parseNote(bassPart, &c.Bass, &c.BassAcc)
		if err != nil {
			return Chord{}, err
		}
		if remainder != "" {
			// Trailing quality on the bass (rare; e.g. "C/G7"). Append it to
			// the bass note text by treating it as part of the chord's tail
			// quality — but to keep round-tripping simple we just reject it.
			return Chord{}, errors.New("chord: unexpected text after bass note")
		}
	} else {
		c.Quality = rest
	}
	return c, nil
}

// parseNote consumes a note letter and optional accidental from the start of s.
// On success it sets root and acc and returns the remainder of s.
func parseNote(s string, root *byte, acc *Accidental) (string, error) {
	if s == "" {
		return "", ErrEmpty
	}
	r := s[0]
	if r < 'A' || r > 'G' {
		return "", ErrBadRoot
	}
	*root = r
	s = s[1:]
	if s == "" {
		*acc = Natural
		return s, nil
	}
	switch s[0] {
	case '#':
		*acc = Sharp
		s = s[1:]
	case 'b':
		// Heuristic: a leading 'b' is a flat only when the chord root is a
		// note that can be flatted (i.e. not 'C' or 'F' — though Cb and Fb
		// are valid enharmonic spellings, so we accept them) AND the next
		// character isn't a quality starter that would be ambiguous.
		// In practice ChordPro chords use 'b' for flats consistently, so we
		// treat any 'b' following a root letter as a flat.
		*acc = Flat
		s = s[1:]
	default:
		*acc = Natural
	}
	return s, nil
}

// semitoneSharp gives the semitone index (0..11, C=0) for each note using
// sharps for chromatic pitches.
var semitoneSharp = [...]int{
	'A': 9, 'B': 11, 'C': 0, 'D': 2, 'E': 4, 'F': 5, 'G': 7,
}

// nameSharp is the chromatic name table using sharps.
var nameSharp = [12]struct {
	Root byte
	Acc  Accidental
}{
	{'C', Natural}, {'C', Sharp}, {'D', Natural}, {'D', Sharp},
	{'E', Natural}, {'F', Natural}, {'F', Sharp}, {'G', Natural},
	{'G', Sharp}, {'A', Natural}, {'A', Sharp}, {'B', Natural},
}

// nameFlat is the chromatic name table using flats.
var nameFlat = [12]struct {
	Root byte
	Acc  Accidental
}{
	{'C', Natural}, {'D', Flat}, {'D', Natural}, {'E', Flat},
	{'E', Natural}, {'F', Natural}, {'G', Flat}, {'G', Natural},
	{'A', Flat}, {'A', Natural}, {'B', Flat}, {'B', Natural},
}

// Preference selects how transposed notes spell their accidentals.
type Preference int

const (
	// PreferAuto keeps flats if the source note was flat, otherwise uses sharps.
	PreferAuto Preference = iota
	PreferSharp
	PreferFlat
)

// semitone returns the chromatic position 0..11 of a (root, acc) pair.
func semitone(root byte, acc Accidental) int {
	n := semitoneSharp[root] + int(acc)
	n %= 12
	if n < 0 {
		n += 12
	}
	return n
}

// spell returns the (root, acc) pair for chromatic position p, choosing
// accidentals according to pref.
func spell(p int, pref Preference) (byte, Accidental) {
	p %= 12
	if p < 0 {
		p += 12
	}
	var t *[12]struct {
		Root byte
		Acc  Accidental
	}
	switch pref {
	case PreferFlat:
		t = &nameFlat
	default:
		t = &nameSharp
	}
	return t[p].Root, t[p].Acc
}

// Transpose returns a new chord shifted by n semitones (may be negative).
// The quality string is preserved verbatim.
func (c Chord) Transpose(n int, pref Preference) Chord {
	resolved := pref
	if pref == PreferAuto {
		if c.Acc == Flat || c.BassAcc == Flat {
			resolved = PreferFlat
		} else {
			resolved = PreferSharp
		}
	}
	out := c
	p := (semitone(c.Root, c.Acc) + n) % 12
	if p < 0 {
		p += 12
	}
	out.Root, out.Acc = spell(p, resolved)
	if c.Bass != 0 {
		bp := (semitone(c.Bass, c.BassAcc) + n) % 12
		if bp < 0 {
			bp += 12
		}
		out.Bass, out.BassAcc = spell(bp, resolved)
	}
	return out
}
