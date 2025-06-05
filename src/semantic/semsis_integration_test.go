package semantic

import (
	"fmt"
	"strings"
	"testing"

	"dash-lang.io/src/types"
)

func TestImportLibrary(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name: "import type definition",
			input: `
					lib one
					pub type a i64
					--
					lib two
					fn abc(v one.a) {}
				`,
			want: "lib two fn abc(v one.a) { }",
		},
		{
			name: "import function",
			input: `
					lib one
					pub fn test() u32 { return 0 }
					--
					lib two
					fn abc() {
						let x = one.test()
					}
				`,
			want: "lib two fn abc() { let x u32 = one.test() }",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			progs := strings.Split(tt.input, "--")

			// parse and analyse leaf library
			imports := make(map[string]map[string]types.TypeSpec)
			{
				p := GetParser(progs[0])
				ast := p.ParseLibrary()

				semsis := New("", nil)
				semsis.Analyse(ast)
				exports := ast.Exports()
				imports[ast.Name.Value] = exports
			}

			p := GetParser(progs[1])
			ast := p.ParseLibrary()

			fmt.Printf("%+v\n", imports)
			semsis := New("", imports)
			semsis.Analyse(ast)

			got := ast.String()

			if tt.want != got {
				t.Errorf("wanted:\n%s\nbut got:\n%s", tt.want, got)
			}
		})
	}
}
