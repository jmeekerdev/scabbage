package access

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

const maxNameRunes = 16

// cleanName trims a display name and checks it uses only friendly characters.
func cleanName(s string) (string, error) {
	name := strings.TrimSpace(s)
	if name == "" || utf8.RuneCountInString(name) > maxNameRunes {
		return "", ErrBadName
	}
	for _, r := range name {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != ' ' && r != '_' && r != '-' {
			return "", ErrBadName
		}
	}
	return name, nil
}
