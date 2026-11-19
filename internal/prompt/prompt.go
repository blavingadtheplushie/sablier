package prompt

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"unicode"
)

func Normalize(text string) string {
	return strings.Join(strings.Fields(strings.ToLower(text)), " ")
}

// salted so short prompts like "yes" can't be guessed
func Hash(salt []byte, text string) string {
	m := hmac.New(sha256.New, salt)
	m.Write([]byte(Normalize(text)))
	return hex.EncodeToString(m.Sum(nil)[:16])
}

// whole prompt matches only
var approvals = map[string]bool{}

func init() {
	for _, s := range []string{
		"y", "yes", "yep", "yeah", "yup", "sure", "ok", "okay", "k", "kk",
		"go", "go ahead", "go for it", "do it", "continue", "proceed", "keep going",
		"yes please", "please continue", "lgtm", "looks good", "sounds good", "ship it",
	} {
		approvals[stripPunct(s)] = true
	}
}

func ApprovalLike(text string) bool {
	return approvals[stripPunct(Normalize(text))]
}

func stripPunct(s string) string {
	s = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsNumber(r) || unicode.IsSpace(r) {
			return r
		}
		if r == '-' || r == '\'' {
			return ' '
		}
		return -1
	}, s)
	return strings.Join(strings.Fields(s), " ")
}
