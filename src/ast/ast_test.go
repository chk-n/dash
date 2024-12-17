package ast

import (
	"testing"

	"dash-lang.io/src/token"
)

func TestString(t *testing.T) {
	module := &Library{
		Token: token.Token{Type: token.LIBRARY, Literal: "module"},
		Name: &Identifier{
			Token: token.Token{Type: token.IDENT, Literal: "test"},
			Value: "test",
		},
		Imports: []*ImportStatement{
			{
				Token:   token.Token{Type: token.USE, Literal: "use"},
				Package: token.Token{Type: token.STRING, Literal: "github.io/code"},
			},
		},
	}
	want := "module test use \"github.io/code\""
	if module.String() != want {
		t.Errorf("program.String() wrong. got=%q", module.String())
	}
}
