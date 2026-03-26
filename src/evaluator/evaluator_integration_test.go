package evaluator

import (
	"strings"
	"testing"

	"dash-lang.io/src/ast"
	"dash-lang.io/src/semantic"
	"dash-lang.io/src/types"
)

func TestEvalLibrary(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  any
	}{
		{
			name: "struct update with imported type",
			input: `
			lib one
			pub type abc i64
			--
			use "one"
			

			struct point { x one.abc }

			var p = point{x: 1}
			p = point{..p, x: p.x + 4}
			p`,
			want: map[string]any{
				"x": int64(5),
			},
		},
		{
			name: "match error",
			input: `
				lib one
				pub error a {
					x i64
				}
				pub error b
				--
				use "one"

				fn test(err error) i64 {
					return match err {
					case one.a: 1
					case one.b: 2
					}
				}
				test(one.a)
			`,
			want: &Return{Values: []any{int64(1)}},
		},
		{
			name: "negated try in if condition with imported function",
			input: `
				lib one
				pub fn is_false() !bool {
					return false
				}
				--
				use "one"

				fn test() i64 {
					let b = true
					if !try one.is_false() && b {
						return 1
					}
					return 0
				}
				test()`,
			want: &Return{Values: []any{int64(1)}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			progs := strings.Split(tt.input, "--")

			libs := make(map[string]*ast.Library)
			// parse and analyse leaf library
			imports := make(map[string]map[string]types.Type)
			{
				p := getParser(progs[0])
				lib := p.ParseLibrary()
				if len(p.Errors()) > 0 {
					t.Errorf("parser error:\n%s", strings.Join(p.Errors(), "\n"))
				}

				s := semantic.New("", nil)
				s.Analyse(lib)
				if len(s.Errors()) > 0 {
					t.Errorf("semsis error:\n%s", strings.Join(s.Errors(), "\n"))
				}

				libs[lib.Name.Value] = lib
				exports := lib.Exports()
				imports[lib.Name.Value] = exports
			}
			p := getParser(progs[1])
			lib := p.ParseREPL()
			if len(p.Errors()) > 0 {
				t.Errorf("parser error:\n%s", strings.Join(p.Errors(), "\n"))
			}

			s := semantic.New("", imports)
			s.Analyse(lib)
			if len(s.Errors()) > 0 {
				t.Errorf("semsis error:\n%s", strings.Join(s.Errors(), "\n"))
			}

			lib = removeMainFn(lib)

			e := New(libs)
			ctx := NewContext(nil)
			e.InitialiseLib(lib, ctx)

			got := e.eval(lib, ctx)

			if !deepEqual(tt.want, got) {
				t.Errorf("wanted:\n%s\nbut got:\n%s", tt.want, got)
			}
		})
	}

}
