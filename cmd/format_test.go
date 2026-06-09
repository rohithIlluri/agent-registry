package cmd

import "testing"

func TestTruncate(t *testing.T) {
	cases := []struct {
		in   string
		max  int
		want string
	}{
		{"short", 10, "short"},
		{"exactly-ten", 11, "exactly-ten"},
		{"this is far too long", 10, "this is f…"},
		// Multibyte runes must not be split mid-sequence.
		{"em—dash—heavy—description—here", 10, "em—dash—h…"},
		{"  padded  ", 20, "padded"},
	}
	for _, c := range cases {
		got := truncate(c.in, c.max)
		if got != c.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", c.in, c.max, got, c.want)
		}
	}
}
