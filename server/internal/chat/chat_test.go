package chat

import (
	"strings"
	"testing"
)

func TestCleanTidiesText(t *testing.T) {
	cases := map[string]string{
		"  hi there  ":          "hi there",
		"line\none\tand\x00two": "line one andtwo",
		"a\u202eevil":           "aevil",           // bidi override
		"👨\u200d👩\u200d👧":       "👨\u200d👩\u200d👧", // family emoji keeps its joiners
	}
	for in, want := range cases {
		if got, ok := Clean(in); !ok || got != want {
			t.Errorf("Clean(%q) = %q, %v; want %q", in, got, ok, want)
		}
	}
}

func TestCleanRejectsBlankText(t *testing.T) {
	for _, in := range []string{"", "   ", "\n\t\x01"} {
		if _, ok := Clean(in); ok {
			t.Errorf("Clean(%q) accepted blank text", in)
		}
	}
}

func TestCleanCapsLength(t *testing.T) {
	got, _ := Clean(strings.Repeat("é", MaxRunes+50))
	if n := len([]rune(got)); n != MaxRunes {
		t.Fatalf("kept %d runes, want %d", n, MaxRunes)
	}
}

func TestParseCommand(t *testing.T) {
	if c, ok := ParseCommand("/Approve  Sam Jones "); !ok || c.Name != "approve" || c.Arg != "Sam Jones" {
		t.Fatalf("got %+v, %v", c, ok)
	}
	if _, ok := ParseCommand("hello /approve"); ok {
		t.Fatal("plain text is not a command")
	}
}
