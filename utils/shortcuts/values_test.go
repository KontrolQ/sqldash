package shortcuts

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	"sqldash/utils/collections"
)

var reserved = map[string]string{
	"Problem": "the flash message set by the chrome middleware",
	"Done":    "the flash message set by the chrome middleware",
	"Account": "the signed in account set by the account middleware",
	"Theme":   "the chosen theme set by the chrome middleware",
}

func TestPageDataNeverShadowsMiddlewareValues(t *testing.T) {
	root := filepath.Join("..", "..")

	walkError := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkError error) error {
		if walkError != nil {
			return walkError
		}

		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		if !strings.Contains(path, string(filepath.Separator)+"services"+string(filepath.Separator)) {
			return nil
		}

		parsed, parseError := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if parseError != nil {
			return parseError
		}

		ast.Inspect(parsed, func(node ast.Node) bool {
			declared, isStruct := node.(*ast.TypeSpec)
			if !isStruct {
				return true
			}

			fields, hasFields := declared.Type.(*ast.StructType)
			if !hasFields || !strings.HasSuffix(declared.Name.Name, "Context") {
				return true
			}

			for _, field := range fields.Fields.List {
				for _, name := range field.Names {
					if why, taken := reserved[name.Name]; taken {
						t.Errorf(
							"%s.%s shadows %s; the render merge writes page data last, so this silently blanks it",
							declared.Name.Name, name.Name, why,
						)
					}
				}
			}

			return true
		})

		return nil
	})

	if walkError != nil {
		t.Fatalf("could not read the services tree: %v", walkError)
	}
}

func TestPageDataIsWrittenOverContextValues(t *testing.T) {
	held := collections.Record[string, any]{"Done": "Saved.", "Title": "from the middleware"}

	if mergeError := mergeBindData(held, struct{ Title string }{Title: "from the page"}); mergeError != nil {
		t.Fatalf("merge failed: %v", mergeError)
	}

	if held["Title"] != "from the page" {
		t.Errorf("page data did not win, got %v", held["Title"])
	}

	if held["Done"] != "Saved." {
		t.Errorf("a value the page does not carry was lost, got %v", held["Done"])
	}
}
