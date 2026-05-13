package chord

import "testing"

func TestParseRoundTrip(t *testing.T) {
	cases := []string{
		"C", "Cm", "C7", "Cmaj7", "Csus4", "Cm7b5",
		"C#", "Db", "F#m7", "Bbmaj7",
		"G/B", "D/F#", "Am/G",
		"C#m7/G#",
	}
	for _, s := range cases {
		c, err := Parse(s)
		if err != nil {
			t.Fatalf("Parse(%q): unexpected error %v", s, err)
		}
		if got := c.String(); got != s {
			t.Errorf("round-trip(%q) = %q", s, got)
		}
	}
}

func TestParseErrors(t *testing.T) {
	cases := []string{"", "H", "/G", "C/"}
	for _, s := range cases {
		if _, err := Parse(s); err == nil {
			t.Errorf("Parse(%q): expected error", s)
		}
	}
}

func TestTransposeUp(t *testing.T) {
	cases := []struct {
		in   string
		semi int
		out  string
	}{
		{"C", 2, "D"},
		{"C", 1, "C#"},
		{"C", -1, "B"},
		{"C", 12, "C"},
		{"Am", 3, "Cm"},
		{"G7", 5, "C7"},
		{"F#m7", 1, "Gm7"},
		{"G/B", 2, "A/C#"},
		{"Bbmaj7", 1, "Bmaj7"},
		{"Db", 1, "D"},
		{"Db", -1, "C"}, // flat preference preserved
	}
	for _, tc := range cases {
		c, err := Parse(tc.in)
		if err != nil {
			t.Fatalf("Parse(%q): %v", tc.in, err)
		}
		got := c.Transpose(tc.semi, PreferAuto).String()
		if got != tc.out {
			t.Errorf("Transpose(%q, %d) = %q, want %q", tc.in, tc.semi, got, tc.out)
		}
	}
}

func TestTransposePreferenceOverride(t *testing.T) {
	c, _ := Parse("C")
	if got := c.Transpose(1, PreferFlat).String(); got != "Db" {
		t.Errorf("PreferFlat: got %q want Db", got)
	}
	if got := c.Transpose(1, PreferSharp).String(); got != "C#" {
		t.Errorf("PreferSharp: got %q want C#", got)
	}
}

func TestTransposeQualityPreserved(t *testing.T) {
	c, _ := Parse("Cm7b5")
	if got := c.Transpose(2, PreferAuto).String(); got != "Dm7b5" {
		t.Errorf("got %q want Dm7b5", got)
	}
}
