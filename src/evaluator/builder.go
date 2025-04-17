package evaluator

import (
	"dash-lang.io/src/ast"
)

// Build builds a dash program from a source directory
func Execute(libs map[string]*ast.Library) error {
	mainLib, ok := libs["main"]
	if mainLib == nil || !ok {
		panic("no 'main' defined")
	}
	// we need to remove main
	delete(libs, "main")

	e := New(libs)
	ctx := NewContext(nil)
	e.Eval(mainLib, ctx)

	return nil
}
