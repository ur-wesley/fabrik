package graph

import (
	"strings"

	gts "github.com/odvcencio/gotreesitter"
	"github.com/odvcencio/gotreesitter/grammars"
)

type extractedSymbol struct {
	Name string
	Line int
}

type extractedCall struct {
	Name string
	Line int
}

type fileExtract struct {
	Package string
	Symbols []extractedSymbol
	Imports []string
	Calls   []extractedCall
	OK      bool
}

func parseFile(relPath string, lang string, source []byte) fileExtract {
	bt, err := grammars.ParseFilePooled(relPath, source)
	if err != nil {
		return fileExtract{OK: false}
	}
	defer bt.Release()
	root := bt.RootNode()
	if root == nil {
		return fileExtract{OK: false}
	}
	fe := fileExtract{OK: !bt.ParseStoppedEarly()}
	walkExtract(bt, root, lang, &fe)
	return fe
}

func walkExtract(bt *gts.BoundTree, n *gts.Node, lang string, fe *fileExtract) {
	if n == nil {
		return
	}
	typ := bt.NodeType(n)
	switch lang {
	case "go":
		extractGo(bt, n, typ, fe)
	case "rust":
		extractRust(bt, n, typ, fe)
	case "typescript", "tsx", "javascript", "jsx":
		extractTS(bt, n, typ, fe)
	case "python":
		extractPy(bt, n, typ, fe)
	case "java":
		extractJava(bt, n, typ, fe)
	case "kotlin":
		extractKotlin(bt, n, typ, fe)
	case "c_sharp":
		extractCSharp(bt, n, typ, fe)
	}
	for i := 0; i < n.ChildCount(); i++ {
		walkExtract(bt, n.Child(i), lang, fe)
	}
}

func line1(n *gts.Node) int {
	if n == nil {
		return 0
	}
	return int(n.StartPoint().Row) + 1
}

func extractGo(bt *gts.BoundTree, n *gts.Node, typ string, fe *fileExtract) {
	switch typ {
	case "function_declaration", "method_declaration":
		name := bt.NodeText(bt.ChildByField(n, "name"))
		if name != "" {
			fe.Symbols = append(fe.Symbols, extractedSymbol{Name: name, Line: line1(n)})
		}
	case "type_declaration":
		for i := 0; i < n.ChildCount(); i++ {
			c := n.Child(i)
			if bt.NodeType(c) == "type_spec" {
				name := bt.NodeText(bt.ChildByField(c, "name"))
				if name != "" {
					fe.Symbols = append(fe.Symbols, extractedSymbol{Name: name, Line: line1(c)})
				}
			}
		}
	case "import_spec":
		path := strings.Trim(bt.NodeText(bt.ChildByField(n, "path")), `"`)
		if path != "" {
			fe.Imports = append(fe.Imports, path)
		}
	case "import_declaration":
		for i := 0; i < n.ChildCount(); i++ {
			c := n.Child(i)
			if bt.NodeType(c) == "import_spec" {
				path := strings.Trim(bt.NodeText(bt.ChildByField(c, "path")), `"`)
				if path != "" {
					fe.Imports = append(fe.Imports, path)
				}
			}
		}
	case "call_expression":
		callee := callNameGo(bt, bt.ChildByField(n, "function"))
		if callee != "" {
			fe.Calls = append(fe.Calls, extractedCall{Name: callee, Line: line1(n)})
		}
	}
}

func callNameGo(bt *gts.BoundTree, fn *gts.Node) string {
	if fn == nil {
		return ""
	}
	switch bt.NodeType(fn) {
	case "identifier":
		return bt.NodeText(fn)
	case "selector_expression":
		return bt.NodeText(bt.ChildByField(fn, "field"))
	case "parenthesized_expression":
		return callNameGo(bt, fn.Child(1))
	default:
		return ""
	}
}

func extractRust(bt *gts.BoundTree, n *gts.Node, typ string, fe *fileExtract) {
	switch typ {
	case "function_item", "function_signature_item":
		name := bt.NodeText(bt.ChildByField(n, "name"))
		if name != "" {
			fe.Symbols = append(fe.Symbols, extractedSymbol{Name: name, Line: line1(n)})
		}
	case "struct_item", "enum_item", "trait_item", "type_item":
		name := bt.NodeText(bt.ChildByField(n, "name"))
		if name != "" {
			fe.Symbols = append(fe.Symbols, extractedSymbol{Name: name, Line: line1(n)})
		}
	case "use_declaration", "extern_crate_declaration":
		txt := strings.TrimSpace(bt.NodeText(n))
		if txt != "" {
			fe.Imports = append(fe.Imports, txt)
		}
	case "call_expression":
		fn := bt.ChildByField(n, "function")
		if fn != nil && bt.NodeType(fn) == "identifier" {
			fe.Calls = append(fe.Calls, extractedCall{Name: bt.NodeText(fn), Line: line1(n)})
		}
	}
}

func extractTS(bt *gts.BoundTree, n *gts.Node, typ string, fe *fileExtract) {
	switch typ {
	case "function_declaration", "method_definition", "class_declaration":
		name := bt.NodeText(bt.ChildByField(n, "name"))
		if name != "" {
			fe.Symbols = append(fe.Symbols, extractedSymbol{Name: name, Line: line1(n)})
		}
	case "interface_declaration", "type_alias_declaration", "enum_declaration":
		name := bt.NodeText(bt.ChildByField(n, "name"))
		if name != "" {
			fe.Symbols = append(fe.Symbols, extractedSymbol{Name: name, Line: line1(n)})
		}
	case "import_statement":
		src := bt.ChildByField(n, "source")
		if src != nil {
			fe.Imports = append(fe.Imports, strings.Trim(bt.NodeText(src), `"'`))
		}
	case "call_expression":
		fn := bt.ChildByField(n, "function")
		if fn != nil {
			name := callNameTS(bt, fn)
			if name != "" {
				fe.Calls = append(fe.Calls, extractedCall{Name: name, Line: line1(n)})
			}
		}
	}
}

func callNameTS(bt *gts.BoundTree, fn *gts.Node) string {
	switch bt.NodeType(fn) {
	case "identifier":
		return bt.NodeText(fn)
	case "member_expression":
		return bt.NodeText(bt.ChildByField(fn, "property"))
	default:
		return ""
	}
}

func extractPy(bt *gts.BoundTree, n *gts.Node, typ string, fe *fileExtract) {
	switch typ {
	case "function_definition":
		name := bt.NodeText(bt.ChildByField(n, "name"))
		if name != "" {
			fe.Symbols = append(fe.Symbols, extractedSymbol{Name: name, Line: line1(n)})
		}
	case "class_definition":
		name := bt.NodeText(bt.ChildByField(n, "name"))
		if name != "" {
			fe.Symbols = append(fe.Symbols, extractedSymbol{Name: name, Line: line1(n)})
		}
	case "import_statement", "import_from_statement":
		fe.Imports = append(fe.Imports, strings.TrimSpace(bt.NodeText(n)))
	case "call":
		fn := bt.ChildByField(n, "function")
		if fn != nil {
			name := callNamePy(bt, fn)
			if name != "" {
				fe.Calls = append(fe.Calls, extractedCall{Name: name, Line: line1(n)})
			}
		}
	}
}

func callNamePy(bt *gts.BoundTree, fn *gts.Node) string {
	switch bt.NodeType(fn) {
	case "identifier":
		return bt.NodeText(fn)
	case "attribute":
		return bt.NodeText(bt.ChildByField(fn, "attribute"))
	default:
		return ""
	}
}

func extractJava(bt *gts.BoundTree, n *gts.Node, typ string, fe *fileExtract) {
	switch typ {
	case "package_declaration":
		name := bt.NodeText(bt.ChildByField(n, "name"))
		if name != "" {
			fe.Package = strings.TrimSpace(name)
		}
	case "class_declaration", "interface_declaration", "enum_declaration", "record_declaration":
		name := bt.NodeText(bt.ChildByField(n, "name"))
		if name != "" {
			fe.Symbols = append(fe.Symbols, extractedSymbol{Name: name, Line: line1(n)})
		}
	case "method_declaration", "constructor_declaration":
		name := bt.NodeText(bt.ChildByField(n, "name"))
		if name != "" {
			fe.Symbols = append(fe.Symbols, extractedSymbol{Name: name, Line: line1(n)})
		}
	case "import_declaration":
		imp := strings.TrimSpace(bt.NodeText(n))
		imp = strings.TrimPrefix(imp, "import")
		imp = strings.TrimPrefix(imp, "static")
		imp = strings.TrimSpace(imp)
		imp = strings.TrimSuffix(imp, ";")
		imp = strings.TrimSpace(imp)
		if imp != "" {
			fe.Imports = append(fe.Imports, imp)
		}
	case "method_invocation":
		name := bt.NodeText(bt.ChildByField(n, "name"))
		if name != "" {
			fe.Calls = append(fe.Calls, extractedCall{Name: name, Line: line1(n)})
		}
	}
}

func extractKotlin(bt *gts.BoundTree, n *gts.Node, typ string, fe *fileExtract) {
	switch typ {
	case "package_header":
		for i := 0; i < n.ChildCount(); i++ {
			c := n.Child(i)
			if bt.NodeType(c) == "package_identifier" {
				fe.Package = strings.TrimSpace(bt.NodeText(c))
				break
			}
		}
	case "class_declaration", "object_declaration", "interface_declaration", "enum_class_body":
		name := bt.NodeText(bt.ChildByField(n, "name"))
		if name != "" {
			fe.Symbols = append(fe.Symbols, extractedSymbol{Name: name, Line: line1(n)})
		}
	case "function_declaration":
		name := bt.NodeText(bt.ChildByField(n, "name"))
		if name != "" {
			fe.Symbols = append(fe.Symbols, extractedSymbol{Name: name, Line: line1(n)})
		}
	case "import_header":
		for i := 0; i < n.ChildCount(); i++ {
			c := n.Child(i)
			if bt.NodeType(c) == "import_list" {
				for j := 0; j < c.ChildCount(); j++ {
					item := c.Child(j)
					if bt.NodeType(item) == "import_alias" || bt.NodeType(item) == "identifier" {
						txt := strings.TrimSpace(bt.NodeText(item))
						txt = strings.TrimPrefix(txt, "import")
						txt = strings.TrimSpace(txt)
						if txt != "" {
							fe.Imports = append(fe.Imports, txt)
						}
					}
				}
			}
		}
	case "call_expression":
		fn := bt.ChildByField(n, "function")
		if fn != nil {
			name := callNameKotlin(bt, fn)
			if name != "" {
				fe.Calls = append(fe.Calls, extractedCall{Name: name, Line: line1(n)})
			}
		}
	}
}

func callNameKotlin(bt *gts.BoundTree, fn *gts.Node) string {
	switch bt.NodeType(fn) {
	case "simple_identifier", "identifier":
		return bt.NodeText(fn)
	case "navigation_expression":
		return bt.NodeText(bt.ChildByField(fn, "name"))
	case "call_expression":
		inner := bt.ChildByField(fn, "function")
		return callNameKotlin(bt, inner)
	default:
		return ""
	}
}

func extractCSharp(bt *gts.BoundTree, n *gts.Node, typ string, fe *fileExtract) {
	switch typ {
	case "file_scoped_namespace_declaration", "namespace_declaration":
		name := bt.NodeText(bt.ChildByField(n, "name"))
		if name != "" {
			fe.Package = strings.TrimSpace(name)
		}
	case "class_declaration", "interface_declaration", "struct_declaration", "record_declaration", "enum_declaration":
		name := bt.NodeText(bt.ChildByField(n, "name"))
		if name != "" {
			fe.Symbols = append(fe.Symbols, extractedSymbol{Name: name, Line: line1(n)})
		}
	case "method_declaration", "constructor_declaration":
		name := bt.NodeText(bt.ChildByField(n, "name"))
		if name != "" {
			fe.Symbols = append(fe.Symbols, extractedSymbol{Name: name, Line: line1(n)})
		}
	case "using_directive":
		txt := strings.TrimSpace(bt.NodeText(n))
		txt = strings.TrimPrefix(txt, "using")
		txt = strings.TrimPrefix(txt, "static")
		txt = strings.TrimSpace(txt)
		txt = strings.TrimSuffix(txt, ";")
		txt = strings.TrimSpace(txt)
		if txt != "" && !strings.HasPrefix(txt, "global") {
			fe.Imports = append(fe.Imports, txt)
		}
	case "invocation_expression":
		fn := bt.ChildByField(n, "function")
		if fn != nil {
			name := callNameCSharp(bt, fn)
			if name != "" {
				fe.Calls = append(fe.Calls, extractedCall{Name: name, Line: line1(n)})
			}
		}
	}
}

func callNameCSharp(bt *gts.BoundTree, fn *gts.Node) string {
	switch bt.NodeType(fn) {
	case "identifier":
		return bt.NodeText(fn)
	case "member_access_expression":
		return bt.NodeText(bt.ChildByField(fn, "name"))
	default:
		return ""
	}
}
