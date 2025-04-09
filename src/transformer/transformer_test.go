package transformer_test

import (
	// "testing"

	"dash-lang.io/src/lexer"
	"dash-lang.io/src/parser"
	// "dash-lang.io/src/semantic"
	// "dash-lang.io/src/transformer"
)

// func TestTranform(t *testing.T) {
// 	tests := []struct {
// 		name  string
// 		input string
// 		want  string
// 	}{
// 		{
// 			name:  "defer function",
// 			input: "lib main fn main() { defer log() } fn log() { return }",
// 			want:  "lib main pub fn main() { log() } fn log() { return }",
// 		},
// 		{
// 			name:  "defer block",
// 			input: "lib main fn main() { defer { let a = log() } } fn log() i64 { return 1 }",
// 			want:  "lib main pub fn main() { let a i64 = log() } fn log() i64 { return 1 }",
// 		},
// 		{
// 			name:  "defer with void return",
// 			input: "lib main fn test() { defer log() return } fn log() { return }",
// 			want:  "lib main fn test() { log() return } fn log() { return }",
// 		},
// 		{
// 			name:  "defer with single return",
// 			input: "lib main fn test() []i64 { defer log() return [1,2,3] } fn log() { return }",
// 			want:  "lib main fn test() []i64 { let $0 []i64 = [1,2,3] log() return $0 } fn log() { return }",
// 		},
// 		{
// 			name:  "defer with multi return",
// 			input: "lib main fn test() i64, []i64 { defer log() return 2, [1,2,3] } fn log() { return }",
// 			want:  "lib main fn test() i64, []i64 { let $0 i64 = 2 let $1 []i64 = [1,2,3] log() return $0, $1 } fn log() { return }",
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			p := GetParser(tt.input)
// 			ast := p.ParseLibrary()
// 			tsm := transformer.New()
// 			tsm.Tranform(ast)
// 			s := semantic.New()
// 			s.Analyse(ast)
// 			if len(s.Errors()) != 0 {
// 				t.Errorf("semantic analysis: %s", s.Errors())
// 			}

// 			if tt.want != ast.String() {
// 				t.Errorf("want: %s but got: %s", tt.want, ast)
// 			}
// 		})
// 	}
// }

// ------ //
// Helper //
// ------ //

func GetParser(input string) *parser.Parser {
	lcfg := &lexer.Config{SkipComments: true}
	l := lexer.New("", input, lcfg)
	return parser.New(l)
}
