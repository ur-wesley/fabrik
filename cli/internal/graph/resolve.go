package graph

import (
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/mod/modfile"
)

type importMap struct {
	goModule   string
	byDir      map[string][]string
	byName     map[string][]string
	byPkg      map[string][]string
	byNS       map[string][]string
	symInFile  map[string]map[string]bool
}

func buildImportMap(repo string, indexed []string) importMap {
	m := importMap{
		byDir:     map[string][]string{},
		byName:    map[string][]string{},
		byPkg:     map[string][]string{},
		byNS:      map[string][]string{},
		symInFile: map[string]map[string]bool{},
	}
	data, err := os.ReadFile(filepath.Join(repo, "go.mod"))
	if err == nil {
		if mf, err := modfile.Parse("go.mod", data, nil); err == nil && mf.Module != nil {
			m.goModule = mf.Module.Mod.Path
		}
	}
	for _, rel := range indexed {
		lang := langForPath(rel)
		if lang == "go" {
			dir := filepath.ToSlash(filepath.Dir(rel))
			m.byDir[dir] = append(m.byDir[dir], rel)
			base := filepath.Base(rel)
			name := strings.TrimSuffix(base, ".go")
			m.byName[name] = append(m.byName[name], rel)
		}
		if lang == "java" || lang == "kotlin" || lang == "c_sharp" {
			abs := filepath.Join(repo, rel)
			src, err := os.ReadFile(abs)
			if err != nil {
				continue
			}
			fe := parseFile(rel, lang, src)
			if fe.Package != "" {
				if lang == "c_sharp" {
					m.byNS[fe.Package] = append(m.byNS[fe.Package], rel)
				} else {
					m.byPkg[fe.Package] = append(m.byPkg[fe.Package], rel)
				}
			}
			syms := map[string]bool{}
			for _, s := range fe.Symbols {
				syms[s.Name] = true
			}
			base := filepath.Base(rel)
			ext := filepath.Ext(base)
			typeName := strings.TrimSuffix(base, ext)
			syms[typeName] = true
			m.symInFile[rel] = syms
		}
	}
	return m
}

func resolveGoImport(imp string, m importMap) string {
	imp = strings.TrimSpace(imp)
	if imp == "" {
		return ""
	}
	var rel string
	if m.goModule != "" && strings.HasPrefix(imp, m.goModule) {
		rel = strings.TrimPrefix(imp, m.goModule)
		rel = strings.TrimPrefix(rel, "/")
	} else if strings.Contains(imp, "/") {
		for _, p := range m.byDir {
			for _, f := range p {
				if strings.HasSuffix(f, imp+".go") || strings.Contains(f, imp) {
					return f
				}
			}
		}
	}
	if rel != "" {
		if files := m.byDir[rel]; len(files) > 0 {
			return pickFile(files)
		}
		candidate := rel + ".go"
		for dir, files := range m.byDir {
			for _, f := range files {
				if f == candidate || strings.HasSuffix(f, "/"+candidate) {
					return f
				}
			}
			if dir == rel || strings.HasSuffix(dir, "/"+rel) {
				return pickFile(files)
			}
		}
	}
	return ""
}

func pickFile(files []string) string {
	if len(files) == 0 {
		return ""
	}
	best := files[0]
	for _, f := range files[1:] {
		if f < best {
			best = f
		}
	}
	return best
}

func resolveJvmImport(imp string, m importMap) string {
	imp = strings.TrimSpace(imp)
	if imp == "" || strings.HasSuffix(imp, ".*") {
		return ""
	}
	parts := strings.Split(imp, ".")
	if len(parts) < 1 {
		return ""
	}
	typeName := parts[len(parts)-1]
	pkg := strings.Join(parts[:len(parts)-1], ".")
	if pkg == "" {
		for _, rels := range m.byPkg {
			for _, rel := range rels {
				if m.symInFile[rel] != nil && m.symInFile[rel][typeName] {
					return rel
				}
			}
		}
		return ""
	}
	for _, rel := range m.byPkg[pkg] {
		if m.symInFile[rel] != nil && m.symInFile[rel][typeName] {
			return rel
		}
	}
	candidate := strings.ReplaceAll(pkg, ".", "/") + "/" + typeName
	for _, rel := range m.byPkg[pkg] {
		if strings.HasSuffix(rel, candidate+".java") || strings.HasSuffix(rel, candidate+".kt") {
			return rel
		}
	}
	if len(m.byPkg[pkg]) == 1 {
		return m.byPkg[pkg][0]
	}
	return ""
}

func resolveCSharpImport(imp string, m importMap) string {
	imp = strings.TrimSpace(imp)
	if imp == "" {
		return ""
	}
	if files := m.byNS[imp]; len(files) == 1 {
		return files[0]
	}
	return ""
}

func resolveCallee(name string, scopeFiles []string, byFile map[string][]string, byName map[string][]string) (dst string, confidence string) {
	var cands []string
	seen := map[string]bool{}
	if len(scopeFiles) > 0 {
		for _, rel := range scopeFiles {
			for _, id := range byFile[rel] {
				if !strings.HasSuffix(id, ":"+name) {
					continue
				}
				if seen[id] {
					continue
				}
				seen[id] = true
				cands = append(cands, id)
			}
		}
	}
	if len(cands) == 1 {
		return cands[0], "extracted"
	}
	if len(cands) > 1 {
		return "", "inferred"
	}
	if all := byName[name]; len(all) == 1 {
		return all[0], "extracted"
	}
	return "", "inferred"
}
