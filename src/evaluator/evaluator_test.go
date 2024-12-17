package evaluator

import (
	"testing"

	"dash-lang.io/src/ast"
	"dash-lang.io/src/lexer"
	"dash-lang.io/src/parser"
)

func TestIArith(t *testing.T) {
	input := "(5 - 9 + 5) * -10 / -5"
	want := 2

	exp := parse(input)

	eval := New()

	got, ok := eval.IArith(exp)
	if !ok {
		t.Errorf("eval.IArith return 'not ok' status")
	}

	if want != got {
		t.Errorf("want %d but got %d", want, got)
	}
}

// Helper

func parse(input string) ast.Node {
	lcfg := &lexer.Config{SkipComments: true}
	l := lexer.New("", input, lcfg)
	p := parser.New(l)

	return p.ParseExpression()
}
