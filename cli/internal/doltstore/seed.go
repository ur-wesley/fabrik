package doltstore

import (
	"strings"
)

func IssueExploreSeedText(iss Issue) string {
	var blob strings.Builder
	blob.WriteString(iss.Title)
	blob.WriteByte(' ')
	blob.WriteString(iss.Body)
	blob.WriteByte(' ')
	blob.WriteString(iss.Acceptance)
	for _, n := range iss.Notes {
		blob.WriteByte(' ')
		blob.WriteString(n.Body)
	}
	return blob.String()
}
