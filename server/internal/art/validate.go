package art

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	maxBlockNameRunes  = 20
	maxTextureNameLen  = 32
	opaqueAlphaAtLeast = 128
)

// size is the exact byte length of a kind's RGBA pixels.
func size(k Kind) int {
	if k == Skin {
		return SkinWidth * SkinHeight * 4
	}
	return BlockSize * BlockSize * 4
}

// cleanPixels copies RGBA pixels with each one made fully opaque or fully clear.
func cleanPixels(k Kind, px []byte) ([]byte, error) {
	if len(px) != size(k) {
		return nil, ErrBadSize
	}
	out := make([]byte, len(px))
	for i := 0; i < len(px); i += 4 {
		if px[i+3] >= opaqueAlphaAtLeast {
			copy(out[i:i+3], px[i:i+3])
			out[i+3] = 255
		}
	}
	return out, nil
}

// cleanBlockName trims a block name and checks it uses only friendly characters.
func cleanBlockName(s string) (string, error) {
	name := strings.Join(strings.Fields(s), " ")
	if name == "" || utf8.RuneCountInString(name) > maxBlockNameRunes {
		return "", ErrBadName
	}
	for _, r := range name {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && !strings.ContainsRune(" '_-", r) {
			return "", ErrBadName
		}
	}
	return name, nil
}

// validTextureName accepts built-in texture names such as "grass_side".
func validTextureName(s string) bool {
	if s == "" || len(s) > maxTextureNameLen {
		return false
	}
	for _, r := range s {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '_' {
			return false
		}
	}
	return true
}
