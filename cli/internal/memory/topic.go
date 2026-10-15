package memory

import (
	"regexp"
	"strings"
	"unicode"
)

var topicKeyRe = regexp.MustCompile(`[^a-z0-9]+`)

type SuggestTopicKeyOptions struct {
	Title string
}

func SuggestTopicKey(opt SuggestTopicKeyOptions) string {
	title := strings.TrimSpace(opt.Title)
	if title == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range strings.ToLower(title) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else if r == ' ' || r == '-' || r == '_' {
			b.WriteByte('-')
		}
	}
	s := topicKeyRe.ReplaceAllString(b.String(), "-")
	s = strings.Trim(s, "-")
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	if len(s) > 128 {
		s = s[:128]
		s = strings.TrimRight(s, "-")
	}
	return s
}
