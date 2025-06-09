package semantic

import (
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
					let x = one.test()
				`,
			want: "lib two let x u32 = one.test()",
		},
		{
			name: "struct literal from imported type",
			input: `
					lib one
					pub struct abc { x i64 }
					--
					lib two
					let x = one.abc{x: 1}
				`,
			want: "lib two let x abc = one.abc{x i64: 1}",
		},
		{
			name: "use imported type in expression",
			input: `
					lib one
					pub type a i64
					--
					lib two
					fn test() one.a { return one.a(1) }
					let x = test() + 1
				`,
			want: "lib two fn test() one.a { return one.a(1) } let x one.a = (test() + 1)",
		},
		{
			name: "import type in struct field",
			input: `
					lib one
					pub type a i64
					--
					lib two
					struct abc {
						f one.a
					}
					let x = abc{f: 1}
				`,
			want: "lib two struct abc {f one.a} let x abc = abc{f one.a: 1}",
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
			if len(p.Errors()) > 0 {
				t.Errorf("%s", strings.Join(p.Errors(), "\n"))
			}

			semsis := New("", imports)
			semsis.Analyse(ast)

			if len(semsis.Errors()) > 0 {
				t.Errorf("%s", strings.Join(semsis.Errors(), "\n"))
			}

			got := ast.String()

			if tt.want != got {
				t.Errorf("wanted:\n%s\nbut got:\n%s", tt.want, got)
			}
		})
	}
}
