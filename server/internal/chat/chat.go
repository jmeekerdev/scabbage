// Package chat tidies chat text and recognizes commands.
package chat

import (
	"strings"
	"unicode"
)

// MaxRunes is the longest chat line kept; the client stops typing at the same length.
const MaxRunes = 200

// Rate limits shared by chat and emotes.
const (
	PerSecond = 0.5
	Burst     = 5
)

const zeroWidthJoiner = '‍'

// Clean makes text one line of printable characters, or reports false if nothing is left.
func Clean(text string) (string, bool) {
	var b strings.Builder
	for _, r := range text {
		switch {
		case unicode.IsSpace(r):
			b.WriteRune(' ')
		case r == zeroWidthJoiner:
			b.WriteRune(r)
		case unicode.IsControl(r), unicode.Is(unicode.Cf, r):
		default:
			b.WriteRune(r)
		}
	}
	runes := []rune(strings.Join(strings.Fields(b.String()), " "))
	if len(runes) > MaxRunes {
		runes = runes[:MaxRunes]
	}
	return string(runes), len(runes) > 0
}

// Command is a chat line like "/approve Sam".
type Command struct {
	Name string // lower case, without the slash
	Arg  string
}

// ParseCommand splits a cleaned line that starts with "/".
func ParseCommand(text string) (Command, bool) {
	rest, ok := strings.CutPrefix(text, "/")
	if !ok {
		return Command{}, false
	}
	name, arg, _ := strings.Cut(rest, " ")
	return Command{Name: strings.ToLower(name), Arg: strings.TrimSpace(arg)}, true
}
