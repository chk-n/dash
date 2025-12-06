package parser

import (
	"strings"
	"testing"

	"dash-lang.io/src/ast"
	"dash-lang.io/src/lexer"
)

func TestType(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "simple array",
			input: "[]i64",
			want:  "[]i64",
		},
		{
			name:  "simple array - optional",
			input: "?[]i64",
			want:  "?[]i64",
		},
		{
			name:  "optional array pointer",
			input: "?*[]i64",
			want:  "?*[]i64",
		},
		{
			name:  "pointer to optional array",
			input: "*?[]i64",
			want:  "*?[]i64",
		},
		{
			name:  "array with size",
			input: "[5]i64",
			want:  "[5]i64",
		},
		{
			name:  "2d array",
			input: "[][]i64",
			want:  "[][]i64",
		},
		{
			name:  "array imported type",
			input: "[]ast.token",
			want:  "[]ast.unknown[token]",
		},
		{
			name:  "function empty",
			input: "fn()",
			want:  "fn()",
		},
		{
			name:  "function with return",
			input: "fn()string",
			want:  "fn()string",
		},
		{
			name:  "function with multi return",
			input: "fn()i64,i64",
			want:  "fn()i64,i64",
		},
		{
			name:  "function with singe param",
			input: "fn(string)",
			want:  "fn(string)",
		},
		{
			name:  "function with params",
			input: "fn(i64,string)",
			want:  "fn(i64,string)",
		},
		{
			name:  "simple pointer type",
			input: "*[]i64",
			want:  "*[]i64",
		},
		{
			name:  "function pointer type",
			input: "*fn(string)i64",
			want:  "*fn(string)i64",
		},
		{
			name:  "memory",
			input: "mut[string]",
			want:  "mut[string]",
		},
		{
			name:  "memory nested",
			input: "mut[[]i64]",
			want:  "mut[[]i64]",
		},
		{
			name:  "char",
			input: "char",
			want:  "char",
		},
		{
			name:  "error type",
			input: "error",
			want:  "error",
		},
		{
			name:  "any",
			input: "any",
			want:  "any",
		},
		{
			name:  "any array",
			input: "[]any",
			want:  "[]any",
		},
		{
			name:  "parameterized type single",
			input: "vec[i64]",
			want:  "unknown[vec[i64]]",
		},
		{
			name:  "parameterized type multiple",
			input: "map[string, i64]",
			want:  "unknown[map[string,i64]]",
		},
		{
			name:  "nested parameterized type",
			input: "option[vec[i64]]",
			want:  "unknown[option[unknown[vec[i64]]]]",
		},
		{
			name:  "optional parameterized type",
			input: "?vec[i64]",
			want:  "?unknown[vec[i64]]",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := getParser(tc.input)
			typ := p.parseType()

			if tc.want != typ.String() {
				t.Errorf("want %s but got %s\n%v", tc.want, typ.String(), typ)
			}
		})
	}
}

func TestPrefixExpression(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "test number prefix expression",
			input: "-2.2",
		},
		{
			name:  "test bool prefix expression",
			input: "!true",
		},
		{
			name:  "address of",
			input: "&a",
		},
		{
			name:  "value of",
			input: "*a",
		},
		{
			name:  "force unwrap",
			input: "?a",
		},
		{
			name:  "force unwrap value of",
			input: "?*a",
		},
		{
			name:  "bitwise NOT",
			input: "~a",
		},
		{
			name:  "bitwise NOT with number",
			input: "~5",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := getParser(tc.input)
			stmt := p.parsePrefixExpression()

			if tc.input != stmt.String() {
				t.Errorf("want %s but got %s\n%v", tc.input, stmt.String(), stmt)
			}
		})
	}
}

func TestInfixExpression(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "basic infix",
			input: "3 + 10 * 2 / 2 - 1",
			want:  "((3 + ((10 * 2) / 2)) - 1)",
		},
		{
			name:  "infix with paren",
			input: "(3 + 10) * 2",
			want:  "((3 + 10) * 2)",
		},
		{
			name:  "infix with multi paren",
			input: "(3 + 10 * 2 / 4)",
			want:  "(3 + ((10 * 2) / 4))",
		},
		{
			name:  "complex integer infix with paren",
			input: "5 / (2 + 10) * 2 - 5 / 5 % 2",
			want:  "(((5 / (2 + 10)) * 2) - ((5 / 5) % 2))",
		},
		{
			name:  "float infix expression",
			input: "(2.2 + 10.1) / (5 % 3.6 - 5.4) % (5.5 * 2.6)",
			want:  "(((2.2 + 10.1) / ((5 % 3.6) - 5.4)) % (5.5 * 2.6))",
		},
		{
			name:  "bool infix expression: true",
			input: "true",
			want:  "true",
		},
		{
			name:  "bool infix expression: false",
			input: "false",
			want:  "false",
		},
		{
			name:  "bool infix expression: basic",
			input: "true == !false",
			want:  "(true == !false)",
		},
		{
			name:  "bool infix expression: complex",
			input: "5 > 3 != false || (5 < 6) <= 3 == (true && 5 >= 10) != true",
			want:  "(((5 > 3) != false) || ((((5 < 6) <= 3) == (true && (5 >= 10))) != true))",
		},
		{
			name:  "compound boolean epxression <",
			input: "0 < b < 10",
			want:  "((0 < b) < 10)",
		},
		{
			name:  "compound boolean epxression <=",
			input: "0 < b <= 10",
			want:  "(0 < (b <= 10))",
		},
		{
			name:  "pipe expression",
			input: "a + 1 * 5 |> to_map |> to_array",
			want:  "(((a + (1 * 5)) |> to_map) |> to_array)",
		},
		{
			name:  "dot access",
			input: "a.b.c",
			want:  "a.b.c",
		},
		{
			name:  "dot access with function call",
			input: "a.b().c",
			want:  "a.b().c",
		},
		{
			name:  "dot access with arithmetic and function call",
			input: "a.b + a.c()",
			want:  "(a.b + a.c())",
		},
		{
			name:  "bitwise left shift",
			input: "1 << 2",
			want:  "(1 << 2)",
		},
		{
			name:  "bitwise right shift",
			input: "8 >> 2",
			want:  "(8 >> 2)",
		},
		{
			name:  "bitwise AND",
			input: "5 & 3",
			want:  "(5 & 3)",
		},
		{
			name:  "bitwise OR",
			input: "5 | 3",
			want:  "(5 | 3)",
		},
		{
			name:  "bitwise XOR",
			input: "5 ^ 3",
			want:  "(5 ^ 3)",
		},
		{
			name:  "bitwise XOR with variables",
			input: "a ^ b",
			want:  "(a ^ b)",
		},
		{
			name:  "combined bitwise operations",
			input: "a << 2 | b & 255",
			want:  "((a << 2) | (b & 255))",
		},
		{
			name:  "bitwise with XOR",
			input: "a ^ b & c",
			want:  "(a ^ (b & c))",
		},
		{
			name:  "XOR with OR",
			input: "a | b ^ c",
			want:  "(a | (b ^ c))",
		},
		{
			name:  "XOR precedence",
			input: "1 & 2 ^ 3 | 4",
			want:  "(((1 & 2) ^ 3) | 4)",
		},
		{
			name:  "bitwise with arithmetic",
			input: "a + b & 255",
			want:  "((a + b) & 255)",
		},
		{
			name:  "shift with arithmetic",
			input: "1 << 2 + 3",
			want:  "((1 << 2) + 3)",
		},
		{
			name:  "complex bitwise expression",
			input: "(a & 255) << 8 | (b & 255)",
			want:  "(((a & 255) << 8) | (b & 255))",
		},
		{
			name:  "bitwise NOT with infix",
			input: "~a & 255",
			want:  "(~a & 255)",
		},
		{
			name:  "precedence: OR vs AND",
			input: "a | b & c",
			want:  "(a | (b & c))",
		},
		{
			name:  "precedence: AND vs shift",
			input: "a & b << 2",
			want:  "(a & (b << 2))",
		},
		{
			name:  "precedence: shift vs addition",
			input: "a << b + c",
			want:  "((a << b) + c)",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := getParser(tc.input)
			stmt := p.parseExpression(LOWEST)

			if tc.want != stmt.String() {
				t.Errorf("want %s but got %s\n%v", tc.want, stmt.String(), stmt)
			}
		})
	}
}

func TestPostfixExpression(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "increment",
			input: "i++",
			want:  "i++",
		},
		{
			name:  "decrement",
			input: "i--",
			want:  "i--",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := getParser(tt.input)
			exp := p.parseExpression(LOWEST)

			if tt.want != exp.String() {
				t.Errorf("want %s but got %s", tt.want, exp.String())
			}
		})
	}
}

func TestForStatement(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "for statement",
			input: "for i = 0; i < N; i++ { }",
			want:  "for i = 0; (i < N); i++ { }",
		},
		{
			name:  "for statement - with break",
			input: "for i = 0; i < N; i++ { break }",
			want:  "for i = 0; (i < N); i++ { break }",
		},
		{
			name:  "for statement - with next",
			input: "for i = 0; i < N; i++ { next }",
			want:  "for i = 0; (i < N); i++ { next }",
		},
		{
			name:  "for statement - with exit condition",
			input: "for i = 0; i < N; i++ { if i == 2 { next } else { break } }",
			want:  "for i = 0; (i < N); i++ { if (i == 2) { next } else { break } }",
		},
		{
			name:  "infinite",
			input: "for { }",
			want:  "for { }",
		},
		{
			name:  "boolean",
			input: "for x < 0 { }",
			want:  "for (x < 0) { }",
		},
		{
			name:  "function condition",
			input: "for has_more() { }",
			want:  "for has_more() { }",
		},
		{
			name:  "for statement - with custom increment",
			input: "for i = 0; i < N; i = i + 2 { }",
			want:  "for i = 0; (i < N); (i = (i + 2)) { }",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := getParser(tc.input)
			stmt := p.parseForStatement()

			if tc.want != stmt.String() {
				t.Errorf("want %s but got %s\n%v", tc.want, stmt.String(), stmt)
			}
		})
	}
}

// TestForRangeStatement(t *testing.T) {}
//
// {
// 		name:  "for range",
// 		input: "for i in 0..10 { s = i + 1 }",
// 		want:  "for i in 0..10 { s = (i + 1) }",
// 	},
// 	{
// 		name:  "for range - custom step",
// 		input: "for i in 0..10; i + 2 { }",
// 		want:  "for i in 0..10; (i + 2) { }",
// 	},

// func TestWhileLoop(t *testing.T) {
// 	tests := []struct {
// 		name  string
// 		input string
// 		want  string
// 	}{
// 		{
// 			name:  "simple loop",
// 			input: "while true { }",
// 			want:  "while true { }",
// 		},
// 		{
// 			name:  "loop with complex expression",
// 			input: "while a / 2 > 5 || 6 - 2 * 2 >= 10 { }",
// 			want:  "while (((a / 2) > 5) || ((6 - (2 * 2)) >= 10)) { }",
// 		},
// 	}
// 	for _, tc := range tests {
// 		t.Run(tc.name, func(t *testing.T) {
// 			p := getParser(tc.input)
// 			stmt := p.parseWhileStatement()

// 			if tc.want != stmt.String() {
// 				t.Errorf("want %s but got %s\n%v", tc.want, stmt.String(), stmt)
// 			}
// 		})
// 	}
// }

func TestReturnStatement(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "return identifier and literal",
			input: `return a, 1`,
		},
		{
			name:  "return expressions",
			input: "return (a < b), (v - 2)",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := getParser(tc.input)
			stmt := p.parseReturnStatement()

			if tc.input != stmt.String() {
				t.Errorf("want %s but got %s\n%+v", tc.input, stmt.String(), stmt)
			}
		})
	}
}

func TestBlock(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "empty block",
			input: "{ }",
			want:  "{ }",
		},
		{
			name:  "block with return",
			input: "{ return a }",
			want:  "{ return a }",
		},
		{
			name: "block with vars and return",
			input: `
				{
					v = "12"
					b = 1 * 2 -3
					return v, b
				}`,
			want: `{ v = "12" b = ((1 * 2) - 3) return v, b }`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := getParser(tc.input)
			stmt := p.parseBlockStatement()

			if tc.want != stmt.String() {
				t.Errorf("want %s but got %s\n%+v", tc.want, stmt.String(), stmt)
			}
		})
	}
}

func TestIfElseExpression(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "if expression no else",
			input: `if a != b { 1 }`,
			want:  "if (a != b) { 1 }",
		},
		{
			name:  "if expression with else",
			input: `if 1 > 3 { 1 } else { 2 }`,
			want:  "if (1 > 3) { 1 } else { 2 }",
		},
		{
			name:  "if expression with else if",
			input: `if a > 10 { 1 } else if a < 10 { 2 } else { 3 }`,
			want:  "if (a > 10) { 1 } else if (a < 10) { 2 } else { 3 }",
		},
		{
			name:  "if range",
			input: `if 0 < b <= 10 { return b }`,
			want:  `if (0 < (b <= 10)) { return b }`,
		},
		{
			name:  "if, else if, else",
			input: `if a { return 1 * 4 } else if b < a || c == "1" { return b } else { return "123" }`,
			want:  `if a { return (1 * 4) } else if ((b < a) || (c == "1")) { return b } else { return "123" }`,
		},
		{
			name:  "parse single statement",
			input: "if x { } else { } if z { }",
			want:  "if x { } else { }",
		},
		{
			name:  "dot expression in if else",
			input: "if x.a { } else { }",
			want:  "if x.a { } else { }",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := getParser(tc.input)
			stmt := p.parseIfElseExpression()

			if tc.want != stmt.String() {
				t.Errorf("want %s but got %s\n%v", tc.want, stmt.String(), stmt)
			}
		})
	}
}

func TestCharLiteral(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "simple",
			input: `'1'`,
		},
		{
			name:  "newline",
			input: `'\n'`,
		},
		{
			name:  "escape '",
			input: `'\''`,
		},
		{
			name:  "escape \\",
			input: `'\\'`,
		},
		// 2, 4 and 8 hexadecimal code points e.g. \uE4
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := getParser(tc.input)
			stmt := p.parseCharacterLiteral()

			if tc.input != stmt.String() {
				t.Errorf("want %s but got %s\n%v", tc.input, stmt.String(), stmt)
			}
		})
	}
}

func TestHexLiteral(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  uint64
	}{
		{
			name:  "simple hex",
			input: `0xFF`,
			want:  255,
		},
		{
			name:  "hex with underscore",
			input: `0xFF_FF`,
			want:  65535,
		},
		{
			name:  "lowercase hex",
			input: `0xdeadbeef`,
			want:  3735928559,
		},
		{
			name:  "uppercase hex",
			input: `0xDEADBEEF`,
			want:  3735928559,
		},
		{
			name:  "mixed case hex",
			input: `0xDeAdBeEf`,
			want:  3735928559,
		},
		{
			name:  "zero",
			input: `0x0`,
			want:  0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := getParser(tc.input)
			stmt := p.parseHexLiteral()

			if tc.input != stmt.String() {
				t.Errorf("want %s but got %s\n%v", tc.input, stmt.String(), stmt)
			}

			hexLit := stmt.(*ast.HexLiteral)
			if hexLit.Value != tc.want {
				t.Errorf("want value %d but got %d", tc.want, hexLit.Value)
			}
		})
	}
}

func TestArrayLiteral(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "simple",
			input: "[1,2,3,4]",
			want:  "[1,2,3,4]",
		},
		{
			name:  "nested",
			input: "[[1,2,3],[1,2,3]]",
			want:  "[[1,2,3],[1,2,3]]",
		},
		{
			name:  "eat training comma",
			input: "[1,]",
			want:  "[1]",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := getParser(tc.input)
			stmt := p.parseArrayLiteralTypeOrCast()

			if tc.want != stmt.String() {
				t.Errorf("want %s but got %s\n%v", tc.input, stmt.String(), stmt)
			}
		})
	}
}

// ---------------------- //
// Function related tests //
// ---------------------- //

func TestFunctionLiteral(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		want   string
		errors []string
	}{
		{
			name: "public normal function",
			input: `pub fn add(a, b i64) i64 {
					return a + b
				}`,
			want: "pub fn add(a,b i64) i64 { return (a + b) }",
		},
		{
			name: "multiple return values",
			input: `pub fn add(a, b i64) i64, i64 {
					return a, b
				}`,
			want: "pub fn add(a,b i64) i64, i64 { return a, b }",
		},
		{
			name: "anonymous function",
			input: `fn(v string) string {
				return v
				}`,
			want: "fn(v string) string { return v }",
		},
		{
			name:  "main function",
			input: "fn main() {}",
			want:  "fn main() { }",
		},
		{
			name:  "function with array return",
			input: "pub fn test() []i64 { return [1,2,3,4] }",
			want:  "pub fn test() []i64 { return [1,2,3,4] }",
		},
		{
			name:  "return null",
			input: "pub fn test() ?i64 { return null }",
			want:  "pub fn test() ?i64 { return null }",
		},
		{
			name:  "function with variable args",
			input: "pub fn test(args ...i64) {}",
			want:  "pub fn test(args []i64) { }",
		},
		{
			name:   "missing argument type",
			input:  "fn test(x) {} ",
			errors: []string{"argument 'x' missing type"},
		},
		// {
		// 	name:  "ensure no infinite recursion",
		// 	input: "fn some_func(m, mut<i64>)",
		// 	error: "",
		// },
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := getParser(tc.input)
			stmt := p.parseFunctionExpression()

			// Check for expected errors
			if tc.errors != nil {
				if len(p.Errors()) != len(tc.errors) {
					t.Errorf("expected %d errors but got %d", len(tc.errors), len(p.Errors()))
				}
				for i, err := range p.Errors() {
					if i < len(tc.errors) && !strings.Contains(err, tc.errors[i]) {
						t.Errorf("want error %s but got %s", tc.errors[i], err)
					}
				}
			} else {
				// Check that the function was parsed correctly when no errors expected
				if tc.want != stmt.String() {
					t.Errorf("want %s but got %s\n%v", tc.want, stmt.String(), stmt)
				}
			}
		})
	}
}

func TestErrorStatement(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "static error",
			input: "error divide_by_zero",
			want:  "error divide_by_zero",
		},
		{
			name:  "dynamic error with single param",
			input: "error invalid_value{val string}",
			want:  "error invalid_value{val string}",
		},
		{
			name:  "dynamic error with multiple params",
			input: "error out_of_bounds{index i64 size i64}",
			want:  "error out_of_bounds{index i64, size i64}",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := getParser(tc.input)
			stmt := p.parseErrorStatement()

			if stmt == nil {
				t.Fatal("parseErrorStatement() returned nil")
			}

			if tc.want != stmt.String() {
				t.Errorf("want %s but got %s\n%v", tc.want, stmt.String(), stmt)
			}
		})
	}
}

func TestTryCatchExpression(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "try statement",
			input: "try some_fn()",
			want:  "try some_fn()",
		},
		{
			name:  "catch with default",
			input: "some_fn() catch err { 0 }",
			want:  "some_fn() catch err { 0 }",
		},
		{
			name:  "catch with reraise",
			input: "some_fn() catch err { raise err }",
			want:  "some_fn() catch err { raise err }",
		},
		{
			name:  "try within function call",
			input: "other_fn(try risky_fn())",
			want:  "other_fn(try risky_fn())",
		},
		{
			name:  "try with dot access",
			input: "try get(arr, 0).x",
			want:  "try get(arr,0).x",
		},
		{
			name:  "try with comparison expression",
			input: "try true == get()",
			want:  "try (true == get())",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := getParser(tc.input)
			stmt := p.parseExpression(LOWEST)

			if tc.want != stmt.String() {
				t.Errorf("want %s but got %s\n%v", tc.want, stmt.String(), stmt)
			}
		})
	}
}

func TestErrorProneFunction(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "error prone function",
			input: "fn test()! { }",
			want:  "fn test()! { }",
		},
		{
			name:  "error prone function with return",
			input: "fn test()! i64 { return 1 }",
			want:  "fn test()! i64 { return 1 }",
		},
		{
			name:  "error prone with try",
			input: "fn test()! { try risky() }",
			want:  "fn test()! { try risky() }",
		},
		{
			name:  "function with keyword parameter names",
			input: "fn test(int i64, float f64, string string) {}",
			want:  "fn test(int i64,float f64,string string) { }",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := getParser(tc.input)
			stmt := p.parseFunctionExpression()

			if tc.want != stmt.String() {
				t.Errorf("want %s but got %s\n%v", tc.want, stmt.String(), stmt)
			}
		})
	}
}

func TestCallExpression(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "simple",
			input: `get(1, [1,2,4,5], 4.23, "123")`,
			want:  `get(1,[1,2,4,5],4.23,"123")`,
		},
		{
			name:  "with expressions",
			input: "get(1 < 2, (1 + 2) * 3)",
			want:  "get((1 < 2),((1 + 2) * 3))",
		},
		{
			name:  "generic function call single type param",
			input: "identity[i32](42)",
			want:  "identity[i32](42)",
		},
		{
			name:  "generic function call multiple type params",
			input: "make_pair[i32, string](10, \"hello\")",
			want:  `make_pair[i32, string](10,"hello")`,
		},
		{
			name:  "generic function call with generic type param",
			input: "make_box[T](value)",
			want:  "make_box[unknown[T]](value)",
		},
		{
			name:  "generic function using dot expression",
			input: "abc.d[i32](42)",
			want:  "abc.d[i32](42)",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := getParser(tc.input)
			stmt := p.parseExpression(LOWEST)

			if tc.want != stmt.String() {
				t.Errorf("want %s but got %s\n%v", tc.want, stmt.String(), stmt)
			}
		})
	}
}

func TestTypeCast(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "scalar",
			input: "u8(1)",
			want:  "u8(1)",
		},
		{
			name:  "aggregate, array",
			input: "[]byte([1,2,3])",
			want:  "[]byte([1,2,3])",
		},
		{
			name:  "aggregate, sized array",
			input: "[3]byte([1,2,3])",
			want:  "[3]byte([1,2,3])",
		},
		{
			name:  "nested array",
			input: "[][]i8([[3],[4,6]])",
			want:  "[][]i8([[3],[4,6]])",
		},
		{
			name:  "aggregate, struct",
			input: "abc(v)",
			want:  "abc(v)",
		},
		{
			name:  "error type cast",
			input: `error("test message")`,
			want:  `error("test message")`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := getParser(tc.input)
			stmt := p.parseExpression(LOWEST)

			if tc.want != stmt.String() {
				t.Errorf("want %s but got %s\n%v", tc.want, stmt.String(), stmt)
			}
		})
	}
}

func TestDefer(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "defer function",
			input: "defer a()",
			want:  "defer a()",
		},
		{
			name:  "defer dot expression",
			input: "defer a.b()",
			want:  "defer a.b()",
		},
		{
			name:  "defer block",
			input: "defer { 1 + 2 }",
			want:  "defer { (1 + 2) }",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := getParser(tc.input)
			stmt := p.parseDeferExpression()

			if tc.want != stmt.String() {
				t.Errorf("want %s but got %s\n%v", tc.want, stmt.String(), stmt)
			}
		})

	}
}

// func TestCatch(t *testing.T) {
// 	tests := []struct {
// 		name  string
// 		input string
// 	}{
// 		{
// 			name:  "return error",
// 			input: "catch { return err }",
// 		},
// 		{
// 			name:  "handle error",
// 			input: "catch { handle(err) }",
// 		},
// 	}

// 	for _, tc := range tests {
// 		t.Run(tc.name, func(t *testing.T) {
// 			p := getParser(tc.input)
// 			stmt := p.parseCatchExpression()

// 			if tc.input != stmt.String() {
// 				t.Errorf("want %s but got %s\n%v", tc.input, stmt.String(), stmt)
// 			}
// 		})
// 	}
// }

func TestEnumStatement(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name: "enum",
			input: `enum status {
					running
					stopped
					unknown
				}`,
			want: "enum status {running, stopped, unknown}",
		},
		{
			name: "enum with type keywords",
			input: `enum tag {
						int
						float
						string
						bool
					}`,
			want: "enum tag {int, float, string, bool}",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := getParser(tc.input)
			stmt := p.parseEnumStatement()

			if tc.want != stmt.String() {
				t.Errorf("want %s but got %s\n%v", tc.want, stmt.String(), stmt)
			}
		})
	}
}

// NOTE: public structs need to be tested in TestModule
func TestTypeDefinitionStatement(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "int type",
			input: "type custom_int i64",
			want:  "type custom_int i64",
		},
		{
			name:  "float type",
			input: "type custom_float f64",
			want:  "type custom_float f64",
		},
		{
			name:  "bool type",
			input: "type custom_bool bool",
			want:  "type custom_bool bool",
		},
		{
			name:  "array type",
			input: "type custom_array []i64",
			want:  "type custom_array []i64",
		},
		{
			name:  "function type",
			input: "type custom_fn fn(string)!",
			want:  "type custom_fn fn(string)!",
		},
		{
			name:  "nested type def",
			input: "type user abc",
			want:  "type user unknown[abc]",
		},
		{
			name:  "with predicate",
			input: "type age i64 | 18 < age && age <= 65",
			want:  "type age i64 | ((18 < age) && (age <= 65))",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := getParser(tc.input)
			stmt := p.parseTypeDefinitionStatement()

			if tc.want != stmt.String() {
				t.Errorf("want %s but got %s\n%v", tc.want, stmt.String(), stmt)
			}
		})
	}
}

func TestTypeAliasStatement(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "i64",
			input: "alias custom_int i64",
			want:  "alias custom_int i64",
		},
		{
			name:  "f64",
			input: "alias custom_float f64",
			want:  "alias custom_float f64",
		},
		{
			name:  "bool",
			input: "alias custom_bool bool",
			want:  "alias custom_bool bool",
		},
		{
			name:  "array",
			input: "alias custom_array []i64",
			want:  "alias custom_array []i64",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := getParser(tc.input)
			stmt := p.parseTypeAliasStatement()

			if tc.want != stmt.String() {
				t.Errorf("want %s but got %s\n%v", tc.want, stmt.String(), stmt)
			}
		})
	}
}

func TestUnionDefinition(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "no type",
			input: "union abc { }",
			want:  "union abc {}",
		},
		{
			name:  "one type",
			input: "union abc { type_a }",
			want:  "union abc {unknown[type_a]}",
		},
		{
			name:  "multiple types",
			input: "union abc { type_a, type_b }",
			want:  "union abc {unknown[type_a], unknown[type_b]}",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := getParser(tc.input)
			stmt := p.parseUnionStatement()

			if tc.want != stmt.String() {
				t.Errorf("want %s but got %s\n%v", tc.want, stmt.String(), stmt)
			}
		})
	}
}

func TestStructDefinition(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "basic",
			input: "struct user {name string age i64}",
			want:  "struct user {name string, age i64}",
		},
		{
			name:  "empty",
			input: "struct user {}",
			want:  "struct user {}",
		},
		{
			name:  "nested struct",
			input: "struct user {x f64, friend user}",
			want:  "struct user {x f64, friend unknown[user]}",
		},
		{
			name:  "struct - fields can use type keywords as names",
			input: "struct point {int i64, float f64}",
			want:  "struct point {int i64, float f64}",
		},
		{
			name:  "with validation",
			input: "struct abc { x i64 | x > 1 y f64 | y < 0 }",
			want:  "struct abc {x i64 | (x > 1), y f64 | (y < 0)}",
		},
		{
			name: "comment after field",
			input: `struct point {
						x i64 // some really informative comment
						y i64
					}`,
			want: "struct point {x i64, y i64}",
		},
		{
			name: "struct with keyword field names",
			input: `struct data {
						int    i64
						float  f64
						string string
						bool   bool
					}`,
			want: "struct data {int i64, float f64, string string, bool bool}",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := getParser(tc.input)
			stmt := p.parseStructStatement()

			if tc.want != stmt.String() {
				t.Errorf("want %s but got %s\n%v", tc.want, stmt.String(), stmt)
			}
		})
	}
}

func TestGenericStructDefinition(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "single generic parameter",
			input: "struct foo[T any] {a T, b i64}",
			want:  "struct foo[T any] {a unknown[T], b i64}",
		},
		{
			name:  "multiple generic parameters with same constraint",
			input: "struct bar[K, V any] {x K, y V}",
			want:  "struct bar[K any, V any] {x unknown[K], y unknown[V]}",
		},
		{
			name:  "multiple generic parameters with different constraints",
			input: "struct baz[T any, E error] {a T, b E}",
			want:  "struct baz[T any, E error] {a unknown[T], b unknown[E]}",
		},
		{
			name:  "generic struct with no fields",
			input: "struct empty[T any] {}",
			want:  "struct empty[T any] {}",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := getParser(tc.input)
			stmt := p.parseStructStatement()

			if tc.want != stmt.String() {
				t.Errorf("want %s but got %s\n%v", tc.want, stmt.String(), stmt)
			}
		})
	}
}

func TestStructLiteral(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "imported struct literal",
			input: `some_lib.strct{a: 1}`,
			want:  `some_lib.strct{a: 1}`,
		},
		{
			name:  "parameterized struct literal with single type parameter",
			input: `abc[i32]{a: 1}`,
			want:  `abc[i32]{a: 1}`,
		},
		{
			name:  "parameterized struct literal with multiple type parameters",
			input: `pair[i32, string]{first: 1, second: "hello"}`,
			want:  `pair[i32, string]{first: 1, second: "hello"}`,
		},
		{
			name:  "parameterized struct literal with complex types",
			input: `container[[]i32]{data: [1, 2, 3]}`,
			want:  `container[[]i32]{data: [1,2,3]}`,
		},
		{
			name:  "parameterized struct literal with complex types",
			input: `abc.d[[]i32]{}`,
			want:  `abc.d[[]i32]{}`,
		},
		{
			name:  "literal with copy",
			input: `strct{..s, a: 1}`,
			want:  `strct{..s, a: 1}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := getParser(tc.input)
			stmt := p.parseExpression(LOWEST)

			if tc.want != stmt.String() {
				t.Errorf("want %s but got %s\n%v", tc.want, stmt.String(), stmt)
			}
		})
	}
}

func TestAnonymousStruct(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "anonymous struct",
			input: `{"+", token_type.PLUS}`,
			want:  `{"+", token_type.PLUS}`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := getParser(tc.input)
			stmt := p.parseAnonymousStructLiteral()

			if tc.want != stmt.String() {
				t.Errorf("want %s but got %s\n%v", tc.want, stmt.String(), stmt)
			}
		})
	}
}

func TestAssignmentStatement(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "let",
			input: "let x = 1",
			want:  "let x = 1",
		},
		{
			name:  "var",
			input: "let x = {1, 2}",
			want:  "let x = {1, 2}",
		},
		{
			name:  "multi, let",
			input: "let x, let y = 1, 2",
			want:  "let x, let y = 1, 2",
		},
		{
			name:  "multi, var",
			input: "var x, var y = test()",
			want:  "var x, var y = test()",
		},
		{
			name:  "multi, reassign",
			input: "x, y = a(), b()",
			want:  "x, y = a(), b()",
		},
		{
			name:  "multi, mixed - keyword first",
			input: "let x, var y, z = test()",
			want:  "let x, var y, z = test()",
		},
		{
			name:  "multi, mixed - ident first",
			input: "x, var y, let z = test()",
			want:  "x, var y, let z = test()",
		},
		{
			name:  "assignment to if else",
			input: "let b = if a > 1 { a } else { 1 }",
			want:  "let b = if (a > 1) { a } else { 1 }",
		},
		{
			name:  "match with expression",
			input: `let res = match v { case a: v.i + v.j case _: 0 }`,
			want:  "let res = match v { case a: (v.i + v.j) case _: 0 }",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := getParser(tc.input)
			stmt := p.parseAssignmentStatement()

			if tc.want != stmt.String() {
				t.Errorf("want %s but got %s\n%+v", tc.want, stmt.String(), stmt)
			}
		})
	}
}

func TestReassignmentStatement(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "int assignment",
			input: "v = 1",
			want:  "(v = 1)",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := getParser(tc.input)
			stmt := p.parseExpression(LOWEST)

			if tc.want != stmt.String() {
				t.Errorf("want %s but got %s\n%+v", tc.want, stmt.String(), stmt)
			}
		})
	}
}

func TestMatchStatement(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "match, single line",
			input: `match x { case 1: return "1" }`,
			want:  `match x { case 1: return "1" }`,
		},
		{
			name:  "match, default",
			input: "match x { case _: return 0 }",
			want:  "match x { case _: return 0 }",
		},
		{
			name:  "match, expression",
			input: "match x { case 1 + 2 * 3: return 0}",
			want:  "match x { case (1 + (2 * 3)): return 0 }",
		},
		{
			name:  "match, multiple cases",
			input: `match x { case "1": 1 case "2": 2 }`,
			want:  `match x { case "1": 1 case "2": 2 }`,
		},
		{
			name: "match, empty blocks",
			input: `
				match x {
				case 1:
				case 2:
				}`,
			want: "match x { case 1:  case 2:  }",
		},
		{
			name: "match, block",
			input: `
				match x {
				case 1:
					return 1
				case 2:
					return 2
				case _:
					return 0
				}`,
			want: "match x { case 1: return 1 case 2: return 2 case _: return 0 }",
		},
		{
			name: "match, with dot expression",
			input: `match v {
			    case abc: v.i
			    case _: ""
				}`,
			want: `match v { case abc: v.i case _: "" }`,
		},
		{
			name:  "match with assign",
			input: "match x' = x {}",
			want:  "match (x' = x) { }",
		},
		{
			name:  "match with library identifier in case",
			input: "match x { case l.tag.field: l.tag.other case _: l.tag.default }",
			want:  "match x { case l.tag.field: l.tag.other case _: l.tag.default }",
		},
		{
			name:  "match with multiple predicates in single case",
			input: `match x { case 1, 2, 3: return "small" }`,
			want:  `match x { case 1, 2, 3: return "small" }`,
		},
		{
			name:  "match with multiple predicates in multiple cases",
			input: `match x { case 1, 2: return "low" case 3, 4: return "high" }`,
			want:  `match x { case 1, 2: return "low" case 3, 4: return "high" }`,
		},
		{
			name:  "match with complex expressions as multiple predicates",
			input: `match x { case a + b, c * d: return "complex" }`,
			want:  `match x { case (a + b), (c * d): return "complex" }`,
		},
		{
			name:  "match with mixed single and multiple predicates",
			input: `match x { case 1: return "one" case 2, 3, 4: return "multiple" case _: return "default" }`,
			want:  `match x { case 1: return "one" case 2, 3, 4: return "multiple" case _: return "default" }`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := getParser(tc.input)
			stmt := p.parseMatchExpression()

			if tc.want != stmt.String() {
				t.Errorf("want %s but got %s\n%+v", tc.want, stmt.String(), stmt)
			}
		})
	}
}

// NOTE: pointless test as semsis validates if they are built in or not
func TestBuiltinFunction(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "len",
			input: "len(a)",
			want:  "len(a)",
		},
		{
			name:  "len",
			input: "cap(a)",
			want:  "cap(a)",
		},
		{
			name:  "printf",
			input: `printf("%s",a)`,
			want:  `printf("%s",a)`,
		},
		{
			name:  "size",
			input: "size(a)",
			want:  "size(a)",
		},
		{
			name:  "make",
			input: "make([]i64, 10)",
			want:  "make([]i64,10)",
		},
		{
			name:  "validate",
			input: "validate(abc)",
			want:  "validate(abc)",
		},
		{
			name:  "assert",
			input: `assert(1 == 1, "uh no")`,
			want:  `assert((1 == 1),"uh no")`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := getParser(tt.input)
			stmt := p.parseExpression(LOWEST)

			if tt.want != stmt.String() {
				t.Errorf("want %s but got %s\n%+v", tt.want, stmt.String(), stmt)
			}
		})
	}
}

func TestFunctionAttributes(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "extern - c",
			input: "lib main @extern(c) fn test()",
			want:  "lib main @extern(c) fn test()",
		},
		{
			name:  "inline - never",
			input: "lib main @inline(never) fn test() {}",
			want:  "lib main @inline(never) fn test() { }",
		},
		{
			name:  "inline - hint",
			input: "lib main @inline(hint) fn test() {}",
			want:  "lib main @inline(hint) fn test() { }",
		},
		{
			name:  "inline - always",
			input: "lib main @inline(always) fn test() {}",
			want:  "lib main @inline(always) fn test() { }",
		},
		{
			name:  "test",
			input: "lib main @test fn abc() {}",
			want:  "lib main @test fn abc() { }",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := getParser(tt.input)
			stmt := p.ParseLibrary()

			if tt.want != stmt.String() {
				t.Errorf("want %s but got %s", tt.want, stmt.String())
			}
		})
	}
}

func TestLibrary(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "test empty library",
			input: `lib empty`,
			want:  `lib empty`,
		},
		{
			name: "test import",
			input: `lib test
					use "abc"
					fn main() { 1 + 1 }`,
			want: `lib test use "abc" fn main() { (1 + 1) }`,
		},
		{
			name:  "public struct",
			input: "lib test pub struct user {x f64}",
			want:  "lib test pub struct user {x f64}",
		},
		{
			name: "public enum",
			input: `lib test 
					pub enum status {
						running
						stopped
						unknown
				}`,
			want: "lib test pub enum status {running, stopped, unknown}",
		},
		{
			name: "issue 62",
			input: `lib main

		@extern(c)
		pub fn exit(c i64)

		fn test() {}
		`,
			want: "lib main @extern(c) pub fn exit(c i64) fn test() { }",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := getParser(tc.input)
			mod := p.ParseLibrary()

			if tc.want != mod.String() {
				t.Errorf("want %s but got %s", tc.want, mod.String())
			}
		})
	}
}

func TestGenericFunctionDefinition(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
		error []string
	}{
		{
			name:  "single generic type with constraint",
			input: "fn func[T any](x T) T {}",
			want:  "fn func[T any](x unknown[T]) unknown[T] { }",
		},
		{
			name:  "multiple generic types with same constraint",
			input: "fn func[T, E any](x T, y E) {}",
			want:  "fn func[T any, E any](x unknown[T],y unknown[E]) { }",
		},
		{
			name:  "multiple generic types with different constraints",
			input: "fn func[T any, E abc](x T, y E) {}",
			want:  "fn func[T any, E unknown[abc]](x unknown[T],y unknown[E]) { }",
		},
		// {
		// 	name:  "missing generic constraint",
		// 	input: "lib test fn func[T](x T) T {}",
		// 	error: []string{},
		// },
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := getParser(tc.input)
			mod := p.parseFunctionExpression()

			if len(p.Errors()) != 0 {
				t.Errorf("unexpected parsing errors: %v", p.Errors())
				return
			}

			if tc.want != mod.String() {
				t.Errorf("want %s but got %s", tc.want, mod.String())
			}
		})
	}
}

func TestStringEscapeSequences(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "newline escape",
			input: `"hello\nworld"`,
			want:  "hello\nworld",
		},
		{
			name:  "tab escape",
			input: `"hello\tworld"`,
			want:  "hello\tworld",
		},
		{
			name:  "carriage return escape",
			input: `"hello\rworld"`,
			want:  "hello\rworld",
		},
		{
			name:  "bell escape",
			input: `"\a"`,
			want:  "\a",
		},
		{
			name:  "backspace escape",
			input: `"\b"`,
			want:  "\b",
		},
		{
			name:  "quote escape",
			input: `"say \"hello\""`,
			want:  `say "hello"`,
		},
		{
			name:  "backslash escape",
			input: `"path\\to\\file"`,
			want:  `path\to\file`,
		},
		{
			name:  "all escapes",
			input: `"\a\b\n\r\t\\\""`,
			want:  "\a\b\n\r\t\\\"",
		},
		{
			name:  "adjacent escapes - backslash before n",
			input: `"\\n"`,
			want:  `\n`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := getParser(tc.input)
			stmt := p.parseExpression(LOWEST)

			if tc.want != stmt.TokenLiteral() {
				t.Errorf("want %s but got %s\n%v", tc.want, stmt.String(), stmt)
			}
		})
	}
}

func TestStringEscapeErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "invalid escape q",
			input: `"hello\qworld"`,
		},
		{
			name:  "invalid escape x",
			input: `"hello\xworld"`,
		},
		{
			name:  "invalid escape u",
			input: `"\u"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := getParser(tt.input)
			_ = p.parseExpression(LOWEST)

			if len(p.Errors()) == 0 {
				t.Error("expected parser error for invalid escape sequence")
			}
		})
	}
}

//------------------//
// Helper functions //
//------------------//

func getParser(input string) *Parser {
	lcfg := &lexer.Config{
		SkipComments: true,
	}
	l := lexer.New("", input, lcfg)
	return New(l)
}
