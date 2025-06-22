package semantic

import (
	"strings"
	"testing"

	"dash-lang.io/src/types"
)

func TestImportLibrary(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		want   string
		errors []string
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
		{
			name: "import enum",
			input: `
					lib one
					pub enum abc { a }
					--
					lib two
					let x = one.abc.a
				`,
			want: "lib two let x one.abc = one.abc.a",
		},
		{
			name: "import enum with invalid field",
			input: `
					lib one
					pub enum status { online, offline }
					--
					lib two
					let x = one.status.invalid
				`,
			errors: []string{"enum 'status' has no field named 'invalid'"},
		},
		{
			name: "import enum with invalid field in expression",
			input: `
					lib one
					pub enum level { low, medium, high }
					--
					lib two
					let result = one.level.invalid == one.level.low
				`,
			errors: []string{"enum 'level' has no field named 'invalid'"},
		},
		{
			name: "import global var of type struct",
			input: `
					lib one
					pub struct abc {
						a i64
					}
					pub let c = abc{a: 1}
					--
					lib two
					let x = one.c.a
				`,
			want: "lib two let x i64 = one.c.a",
		},
		{
			name: "imported function, wrong argument count",
			input: `
					lib one
					pub fn test(a, b i64) i64 { return a }
					--
					lib two
					let x = one.test(1)
				`,
			errors: []string{"too little arguments passed to function 'test'"},
		},
		{
			name: "imported function, not found",
			input: `
					lib one
					--
					lib two
					let x = one.next(1)
				`,
			errors: []string{"identifier 'one.next' not found"},
		},
		{
			name: "error field with imported type",
			input: `
					lib one
					pub type a u32
					--
					lib two
					error abc {
						x one.a
					}
					fn test()! {
						raise abc{x: 1}
					}
				`,
			want: "lib two error abc{x one.a} fn test()! { raise abc{x one.a: 1} }",
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

			// if we want errors compare error output
			// otherwise compare AST
			if len(tt.errors) > 0 {
				gotErrs := semsis.Errors()
				if len(gotErrs) != len(tt.errors) {
					t.Errorf("want %d errors but got %d errors. %s", len(gotErrs), len(tt.errors), gotErrs)
				}
				for i, err := range semsis.Errors() {
					if err != tt.errors[i] {
						t.Errorf("want error %s but got %s", tt.errors[i], err)
					}
				}
			} else {
				if len(semsis.Errors()) > 0 {
					t.Errorf("%s", strings.Join(semsis.Errors(), "\n"))
				}

				got := ast.String()

				if tt.want != got {
					t.Errorf("wanted:\n%s\nbut got:\n%s", tt.want, got)
				}
			}
		})
	}
}
