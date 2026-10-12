package graph

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const maxFileSize = 1 << 20

var skipDirs = map[string]bool{
	".git": true, ".fabrik": true, "vendor": true, "node_modules": true,
	"dist": true, "build": true, "target": true, "graphify-out": true,
	"bin": true, "obj": true, ".gradle": true,
}

var extLang = map[string]string{
	".go":   "go",
	".rs":   "rust",
	".ts":   "typescript",
	".tsx":  "tsx",
	".js":   "javascript",
	".jsx":  "jsx",
	".py":   "python",
	".java": "java",
	".kt":   "kotlin",
	".kts":  "kotlin",
	".cs":   "c_sharp",
}

func langForPath(rel string) string {
	ext := strings.ToLower(filepath.Ext(rel))
	return extLang[ext]
}

func walkSourceFiles(repo string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(repo, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if skipDirs[d.Name()] {
				return fs.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(repo, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if langForPath(rel) == "" {
			return nil
		}
		out = append(out, rel)
		return nil
	})
	return out, err
}

func fileMtimeRFC3339(path string) (string, error) {
	st, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	return st.ModTime().UTC().Format("2006-01-02T15:04:05.000000000Z"), nil
}
