package graph

import (
	"fmt"
	"strings"
)

func fileID(relPath string) string {
	return "file:" + relPath
}

func symID(relPath, name string) string {
	return fmt.Sprintf("sym:%s:%s", relPath, name)
}

func parseSymID(id string) (path, name string, ok bool) {
	if !strings.HasPrefix(id, "sym:") {
		return "", "", false
	}
	rest := id[4:]
	i := strings.LastIndex(rest, ":")
	if i <= 0 {
		return "", "", false
	}
	return rest[:i], rest[i+1:], true
}
