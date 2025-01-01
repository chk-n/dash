package evaluator

import (
	"bytes"
	"io"
	"os"
	"testing"

	"dash-lang.io/src/ast"
	"dash-lang.io/src/lexer"
	"dash-lang.io/src/parser"
	"dash-lang.io/src/semantic"
)

func TestEvaluatePrefix(t *testing.T) {
	tests := []struct {
		name string
		prog string
		want any
	}{
		{
			name: "arithmetic negation",
			prog: "-5",
			want: int64(-5),
		},
		{
			name: "boolean negation",
			prog: "!true",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := parseExpression(tt.prog)
			e := New()
			got := e.Run(n)
			if got != tt.want {
				t.Errorf("got: %s but want: %s", got, tt.want)
			}
		})
	}
}

func TestEvaluateInfix(t *testing.T) {
	tests := []struct {
		name string
		prog string
		want any
	}{
		{
			name: "arithmetic operations",
			prog: "((5 - 9 + 5) * -10 / -5) % 2",
			want: int64(0),
		},
		{
			name: "logical operations",
			prog: "false && true || false",
			want: false,
		},
		{
			name: "relational operations",
			prog: "3 <= 4",
			want: true,
		},
		{
			name: "equality operation",
			prog: "3.4 == 3.4",
			want: true,
		},
		{
			name: "inequality operation",
			prog: "1.1 != 1.11",
			want: true,
		},
		{
			name: "boolean expression",
			prog: "(1 > 0) || false",
			want: true,
		},
		{
			name: "string concatenation",
			prog: `"hel" + "lo"`,
			want: "hello",
		},
		{
			name: "string equality",
			prog: `"hel" == "lo"`,
			want: false,
		},
		{
			name: "string equality",
			prog: `"hel" != "lo"`,
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := parseExpression(tt.prog)
			e := New()
			got := e.Run(n)
			if got != tt.want {
				t.Errorf("got: %s but want: %s", got, tt.want)
			}
		})
	}
}

func TestAssignmentStatement(t *testing.T) {
	tests := []struct {
		name string
		prog string
		want any
	}{
		{
			name: "single let int assignment",
			prog: "let v = 1 v",
			want: int64(1),
		},
		{
			name: "single let string assignment",
			prog: `let v = "hello" v`,
			want: "hello",
		},
		{
			name: "assign fn result",
			prog: `
				fn test() i64, i64 { return 11, 12}
				let a, let b = test()
				b`,
			want: int64(12),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := parseExpressions(tt.prog)
			e := New()
			got := e.Run(n)
			if got != tt.want {
				t.Errorf("got: %s but want: %s", got, tt.want)
			}
		})
	}
}

func TestIfElseExpression(t *testing.T) {
	tests := []struct {
		name string
		prog string
		want any
	}{
		{
			name: "if else expression",
			prog: "let x = if 5 > 2 { 0 } else { 1 } x",
			want: int64(0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := parseExpressions(tt.prog)
			e := New()
			got := e.Run(n)
			if got != tt.want {
				t.Errorf("got: %s but want: %s", got, tt.want)
			}
		})
	}
}

// func TestFunctionCallExpression(t *testing.T) {
// 	tests := []struct {
// 		name string
// 		prog string
// 		want []any
// 	}{
// 		{
// 			name: "call closure",
// 			prog: "let add = fn(a, b i64) i64 { return a + b } add(1,2)",
// 			want: []any{int64(3)},
// 		},
// 		{
// 			name: "call closure with variable captured",
// 			prog: "let x = 1 let add = fn(a i64) i64 { return a + x } add(1)",
// 			want: []any{int64(2)},
// 		},
// 		{
// 			name: "pass function to closure",
// 			prog: `let sub = fn(a, b i64) i64 { return a - b }
// 				   let do = fn(x, y i64, f fn(i64, i64) i64) i64 { return f(x, y) }
// 				   do(1,2, sub)`,
// 			want: []any{int64(2)},
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			n := parseExpressions(tt.prog)
// 			e := New()
// 			got_ := e.Run(n)
// 			got, ok := got_.([]any)
// 			if !ok {
// 				t.Error("got is not an array")
// 			}
// 			if len(got) != len(tt.want) {
// 				t.Errorf("got: %s but want: %s", got, tt.want)
// 			}
// 			for i := range got {
// 				if got[i] != tt.want[i] {
// 					t.Errorf("got: %s but want: %s", got[i], tt.want[i])
// 				}
// 			}
// 		})
// 	}
// }

func TestArrayOperations(t *testing.T) {
	tests := []struct {
		name string
		prog string
		want any
	}{
		{
			name: "array initialization",
			prog: "let arr = [1, 2, 3] arr",
			want: []any{int64(1), int64(2), int64(3)},
		},
		{
			name: "nested array initialization",
			prog: "let arr = [[1, 2], [3, 4]] arr",
			want: []any{
				[]any{int64(1), int64(2)},
				[]any{int64(3), int64(4)},
			},
		},
		{
			name: "single index access",
			prog: "let arr = [1, 2, 3] arr[1]",
			want: int64(2),
		},
		{
			name: "multi-dimensional index access",
			prog: "let arr = [[1, 2], [3, 4]] arr[1][0]",
			want: int64(3),
		},
		// {
		// 	name: "negative index",
		// 	prog: "let arr = [1, 2, 3] arr[-1]",
		// 	want: int64(3),
		// },
		// {
		// 	name: "index out of bounds",
		// 	prog: "let arr = [1, 2, 3] arr[5]",
		// 	want: nil,
		// },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := parseExpressions(tt.prog)
			e := New()
			got := e.Run(n)

			if !deepEqual(got, tt.want) {
				t.Errorf("got %v but want %v", got, tt.want)
			}
		})
	}
}

func TestStructLiteral(t *testing.T) {
	tests := []struct {
		name string
		prog string
		want any
	}{
		{
			name: "named struct initialization",
			prog: `struct user {
					name string
					age i64
				}
				let p = user{name: "ada", age: 24}
				p`,
			want: map[string]any{
				"name": "ada",
				"age":  int64(24),
			},
		},
		{
			name: "unnamed struct initialization",
			prog: `struct point { i64, i64 }
				let p = point{1, 2} 
				p`,
			want: map[string]any{
				"0": int64(1),
				"1": int64(2),
			},
		},
		{
			name: "nested struct",
			prog: `struct person {
					name string
					addr address
				}
				struct address {
					city string
				}
				let p = person{name: "ada", addr: address{city: "zurich"}} p`,
			want: map[string]any{
				"name": "ada",
				"addr": map[string]any{
					"city": "zurich",
				},
			},
		},
		{
			name: "field access",
			prog: `struct user {
					name string
					age i64
				}
				let p = user{name: "ada", age: 24} 
				p.name`,
			want: "ada",
		},
		{
			name: "unnamed field access",
			prog: "struct point {i64,i64} let p = point{1, 2} p.0",
			want: int64(1),
		},
		// {
		// 	name: "nested struct",
		// 	prog: `struct person {
		// 			name string
		// 			addr address
		// 		}
		// 		struct address {
		// 			city string
		// 		}
		// 		let p = person{name: "ada", addr: address{city: "zurich"}}
		// 		p.name.city`,
		// 	want: "zurich",
		// },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := parseExpressions(tt.prog)
			e := New()
			got := e.Run(n)

			if !deepEqual(got, tt.want) {
				t.Errorf("got %v but want %v", got, tt.want)
			}
		})
	}
}

func TestEnumDotExpression(t *testing.T) {
	tests := []struct {
		name string
		prog string
		want any
	}{
		{
			name: "field access",
			prog: `
                enum status { on off }
                let x = status.off
                x`,
			want: int64(1),
		},
		{
			name: "enum equality",
			prog: `
                enum status { on off }
                let x = status.on == status.off
                x`,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := parseExpressions(tt.prog)
			e := New()
			got := e.Run(n)
			if !deepEqual(got, tt.want) {
				t.Errorf("got %v but want %v", got, tt.want)
			}
		})
	}
}

// TODO: add tests for next
func TestForStatement(t *testing.T) {
	tests := []struct {
		name string
		prog string
		want any
	}{
		{
			name: "classic for loop",
			prog: `
		        var sum = 0
		        for i = 0; i < 5; i++ {
		        	sum = sum + 1
		        }
		        sum`,
			want: int64(5),
		},
		{
			name: "boolean condition loop",
			prog: `
		             var x = 0
		             for x < 3 {
		                 x = x + 1
		             }
		             x`,
			want: int64(3),
		},
		{
			name: "infinite loop with break",
			prog: `
		             var x = 0
		             for {
		                 x = x + 1
		                 if x == 5 {
		                     break
		                 }
		             }
		             x`,
			want: int64(5),
		},
		// {
		// 	name: "for range over array",
		// 	prog: `
		//              var sum = 0
		//              let arr = [1,2,3]
		//              for i in arr {
		//                  sum = sum + arr[i]
		//              }
		//              sum`,
		// 	want: int64(6),
		// },
		// {
		// 	name: "for range with element",
		// 	prog: `
		//              var sum = 0
		//              let arr = [1,2,3]
		//              for i, e in arr {
		//                  sum = sum + e
		//              }
		//              sum`,
		// 	want: int64(6),
		// },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := parseExpressions(tt.prog)
			e := New()
			got := e.Run(n)

			if !deepEqual(got, tt.want) {
				t.Errorf("got %v but want %v", got, tt.want)
			}
		})
	}
}

func TestLenFunction(t *testing.T) {
	tests := []struct {
		name string
		prog string
		want any
	}{
		{
			name: "literal array",
			prog: "len([1, 2, 3])",
			want: int64(3),
		},
		// BUG: panic: type is nil in semsis
		// {
		// 	name: "empty array",
		// 	prog: "len([])",
		// 	want: int64(0),
		// },
		{
			name: "variable array",
			prog: `
                let arr = [1, 2, 3, 4]
                len(arr)
            `,
			want: int64(4),
		},
		{
			name: "nested arrays",
			prog: `
                let arr = [[1, 2], [3, 4], [5, 6]]
                len(arr)
            `,
			want: int64(3),
		},
		{
			name: "len in expression",
			prog: `
                let arr = [1, 2, 3]
                len(arr) + 1
            `,
			want: int64(4),
		},
		{
			name: "len with function return",
			prog: `
                fn get_arr() []i64 { 
                    return [1, 2, 3, 4, 5]
                }
                len(get_arr())
            `,
			want: int64(5),
		},
		{
			name: "len with array argument",
			prog: `
                fn check_len(arr []i64) i64 {
                    return len(arr)
                }
                let l = check_len([1, 2, 3])
                l
            `,
			want: int64(3),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := parseExpressions(tt.prog)
			e := New()
			got := e.Run(n)

			if !deepEqual(got, tt.want) {
				t.Errorf("got %v but want %v", got, tt.want)
			}
		})
	}
}

func TestPrintln(t *testing.T) {
	tests := []struct {
		name     string
		prog     string
		wantText string
	}{
		{
			name:     "print single integer",
			prog:     "println(42)",
			wantText: "42\n",
		},
		{
			name:     "print array",
			prog:     "println([1, 2, 3])",
			wantText: "[1, 2, 3]\n",
		},
		{
			name: "print struct",
			prog: `
                struct point { x i64, y i64 }
                let p = point{1, 2}
                println(p)
            `,
			wantText: "{0: 1, 1: 2}\n",
		},
		{
			name: "print variable",
			prog: `
                let x = 42
                println(x)
            `,
			wantText: "42\n",
		},
		{
			name:     "print expression",
			prog:     "println(1 + 2 * 3)",
			wantText: "7\n",
		},
		{
			name: "print function result",
			prog: `
                fn add(a, b i64) i64 { return a + b }
                println(add(2, 3))
            `,
			wantText: "5\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// set up capture stdout
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Run the program
			n := parseExpressions(tt.prog)
			e := New()
			e.Run(n)

			w.Close()
			os.Stdout = old

			var buf bytes.Buffer
			io.Copy(&buf, r)
			got := buf.String()

			if got != tt.wantText {
				t.Errorf("\ngot :\n%s\nwant:\n%s", got, tt.wantText)
			}
		})
	}
}

func TestMake(t *testing.T) {
	tests := []struct {
		name string
		prog string
		want any
	}{
		{
			name: "make integer array",
			prog: "make([]i64, 3)",
			want: []any{[]any{int64(0), int64(0), int64(0)}},
		},
		{
			name: "make float array",
			prog: "make([]f64, 2)",
			want: []any{[]any{float64(0), float64(0)}},
		},
		{
			name: "make string array",
			prog: `make([]string, 2)`,
			want: []any{[]any{"", ""}},
		},
		{
			name: "make bool array",
			prog: "make([]bool, 2)",
			want: []any{[]any{false, false}},
		},
		{
			name: "make empty array",
			prog: "make([]i64, 0)",
			want: []any{[]any{}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := parseExpressions(tt.prog)
			e := New()
			got := e.Run(n)

			if !deepEqual(got, tt.want) {
				t.Errorf("got %v but want %v", got, tt.want)
			}
		})
	}
}

func TestUseExpression(t *testing.T) {
	tests := []struct {
		name string
		prog string
		want any
	}{
		{
			name: "modify array using index expression",
			prog: `
              let buf = make([]i64, 3)
              let buf' = use buf {
                  buf[0] = 1
                  buf[1] = 2
              }
              buf'`,
			want: []any{int64(1), int64(2), int64(0)},
		},
		{
			name: "modify array using slice expression",
			prog: `
	            let buf = make([]i64, 3)
	            let buf' = use buf {
	                buf[0:3] = [3,2,1]
	            }
	            buf'`,
			want: []any{int64(3), int64(2), int64(1)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := parseExpressions(tt.prog)
			e := New()
			got := e.Run(n)

			if !deepEqual(got, tt.want) {
				t.Errorf("got %v but want %v", got, tt.want)
			}
		})
	}
}

func TestCopyUpdateExpression(t *testing.T) {
	tests := []struct {
		name string
		prog string
		want any
	}{
		{
			name: "array update",
			prog: `
               let arr = [1, 2, 3]
               let arr' = arr^{
                   arr'[0] = 10
                   arr'[2] = 30
               }
               arr'`,
			want: []any{int64(10), int64(2), int64(30)},
		},
		{
			name: "struct update",
			prog: `
               struct point { x i64, y i64 }
               let p = point{x: 0, y: 1}
               let p' = p^{
                   p'.x = 5 
               }
               p'`,
			want: map[string]any{
				"x": int64(5),
				"y": int64(1),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := parseExpressions(tt.prog)
			e := New()
			got := e.Run(n)

			if !deepEqual(got, tt.want) {
				t.Errorf("got %v but want %v", got, tt.want)
			}
		})
	}
}

func TestIArith(t *testing.T) {
	input := "(5 - 9 + 5) * -10 / -5"
	want := 2

	exp := parseExpression(input)

	eval := New()

	got, ok := eval.IArith(exp)
	if !ok {
		t.Errorf("eval.IArith return 'not ok' status")
	}

	if want != got {
		t.Errorf("want %d but got %d", want, got)
	}
}

// ------ //
// Helper //
// ------ //

func parseExpression(input string) ast.Node {
	lcfg := &lexer.Config{SkipComments: true}
	l := lexer.New("", input, lcfg)
	p := parser.New(l)

	return p.ParseExpression()
}

func parseExpressions(input string) ast.Node {
	lcfg := &lexer.Config{SkipComments: true}
	l := lexer.New("", input, lcfg)

	p := parser.New(l)
	ast := p.ParseExpressions()

	s := semantic.New()
	s.AnalyseNode(ast)

	return ast
}

func deepEqual(a, b any) bool {
	switch a := a.(type) {
	case []any:
		b, ok := b.([]any)
		if !ok || len(a) != len(b) {
			return false
		}
		for i := range a {
			if !deepEqual(a[i], b[i]) {
				return false
			}
		}
		return true
	case map[string]any:
		b, ok := b.(map[string]any)
		if !ok || len(a) != len(b) {
			return false
		}
		for k, v := range a {
			bv, exists := b[k]
			if !exists || !deepEqual(v, bv) {
				return false
			}
		}
		return true
	default:
		return a == b
	}
}
