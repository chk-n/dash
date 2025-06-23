package evaluator

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
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
			e := NewEvaluator()
			got := e.Eval(n, NewContext(nil))
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
			name: "string inequality",
			prog: `"hel" != "lo"`,
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := parseExpression(tt.prog)
			e := NewEvaluator()
			got := e.Eval(n, NewContext(nil))
			if got != tt.want {
				t.Errorf("got: %s but want: %s", got, tt.want)
			}
		})
	}
}

func TestCharOperations(t *testing.T) {
	tests := []struct {
		name string
		prog string
		want any
	}{
		{
			name: "char literal",
			prog: "'a'",
			want: 'a',
		},
		{
			name: "char variable assignment",
			prog: "let c = 'x' c",
			want: 'x',
		},
		{
			name: "char in expression",
			prog: "let c = 'a' c == 'a'",
			want: true,
		},
		{
			name: "char comparison - equality",
			prog: "'a' == 'a'",
			want: true,
		},
		{
			name: "char comparison - inequality",
			prog: "'a' != 'b'",
			want: true,
		},
		{
			name: "char arithmetic",
			prog: "'a' + byte(1)",
			want: byte('b'),
		},
		// {
		// 	name: "char to string",
		// 	prog: "string('a')",
		// 	want: "a",
		// },
		{
			name: "char in struct",
			prog: `
			struct letter { val byte }
			let l = letter{val: 'a'}
			l.val`,
			want: byte('a'),
		},
		{
			name: "char comparison with byte",
			prog: "'a' == byte(97)",
			want: true,
		},
		{
			name: "byte to char",
			prog: "let b = byte(65) char(b)",
			want: 'A',
		},
		{
			name: "new line char",
			prog: `'\n'`,
			want: '\n',
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, err := parseExpressions(tt.prog)
			if err != nil {
				t.Error(err)
			}
			e := NewEvaluator()
			got := e.Eval(n, NewContext(nil))

			if !deepEqual(got, tt.want) {
				t.Errorf("got %v but want %v", got, tt.want)
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
			n, err := parseExpressions(tt.prog)
			if err != nil {
				t.Error(err)
			}
			e := NewEvaluator()
			got := e.Eval(n, NewContext(nil))
			if got != tt.want {
				t.Errorf("got: %s but want: %s", got, tt.want)
			}
		})
	}
}

func TestReassignment(t *testing.T) {
	tests := []struct {
		name string
		prog string
		want any
	}{
		{
			name: "reassign in match",
			prog: `
			var y = 0
			let x = 1
			match x {
			case 1: y = 1
			case _: y = 2
			}
			y`,
			want: int64(1),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, err := parseExpressions(tt.prog)
			if err != nil {
				t.Error(err)
			}
			e := NewEvaluator()
			got := e.Eval(n, NewContext(nil))
			if got != tt.want {
				t.Errorf("got: %d but want: %d", got, tt.want)
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
			n, err := parseExpressions(tt.prog)
			if err != nil {
				t.Error(err)
			}
			e := NewEvaluator()
			got := e.Eval(n, NewContext(nil))
			if got != tt.want {
				t.Errorf("got: %s but want: %s", got, tt.want)
			}
		})
	}
}

func TestFunctionClosures(t *testing.T) {
	tests := []struct {
		name string
		prog string
		want any
	}{
		{
			name: "call closure",
			prog: "let add = fn(a, b i64) i64 { return a + b } add(1,2)",
			want: &Return{[]any{int64(3)}},
		},
		{
			name: "call closure with variable captured",
			prog: "let x = 1 let add = fn(a i64) i64 { return a + x } add(1)",
			want: &Return{[]any{int64(2)}},
		},
		{
			name: "pass function to closure",
			prog: `
			let sub = fn(a, b i64) i64 { return a - b }
			let do = fn(x, y i64, f fn(i64, i64) i64) i64 { return f(x, y) }
			do(1,2, sub)`,
			want: &Return{[]any{int64(-1)}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, err := parseExpressions(tt.prog)
			if err != nil {
				t.Error(err)
			}
			e := NewEvaluator()
			got := e.Eval(n, NewContext(nil))
			if !deepEqual(got, tt.want) {
				t.Errorf("got: %s but want: %s", got, tt.want)
			}
		})
	}
}

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
		//	{
		//		name: "negative index",
		//		prog: "let arr = [1, 2, 3] arr[-1]",
		//		want: int64(3),
		//	},
		//
		//	{
		//		name: "index out of bounds",
		//		prog: "let arr = [1, 2, 3] arr[5]",
		//		want: nil,
		//	},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, err := parseExpressions(tt.prog)
			if err != nil {
				t.Error(err)
			}
			e := NewEvaluator()
			got := e.Eval(n, NewContext(nil))

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
			name: "function returning value used in struct field",
			prog: `
			fn get_tag() i64 { return 1 }
			struct node {
				tag i64
				data i64
			}
			let n = node{tag: get_tag(), data: 2}
			n.tag`,
			want: int64(1),
		},
		// NOTE: multiple dot expression not supported yet
		// {
		// 	name: "nested struct",
		// 	prog: `struct person {
		// 				name string
		// 				addr address
		// 			}
		// 			struct address {
		// 				city string
		// 			}
		// 			let p = person{name: "ada", addr: address{city: "zurich"}}
		// 			p.name.city`,
		// 	want: "zurich",
		// },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, err := parseExpressions(tt.prog)
			if err != nil {
				t.Error(err)
			}
			e := NewEvaluator()
			got := e.Eval(n, NewContext(nil))

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
			n, err := parseExpressions(tt.prog)
			if err != nil {
				t.Error(err)
			}
			e := NewEvaluator()
			got := e.Eval(n, NewContext(nil))
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
		{
			name: "next in classic for loop",
			prog: `
		    var sum = 0
		    for i = 0; i < 5; i++ {
		        if i == 2 {
		            next
		        }
		        sum = sum + 1
		    }
		    sum`,
			want: int64(4),
		},
		{
			name: "loop with custom increment",
			prog: `
		    var sum = 0
		    for i = 0; i < 5; i = i+2 {
		        sum = sum + 1
		    }
		    sum`,
			want: int64(3),
		},
		{
			name: "nested loops with break",
			prog: `
		    var sum = 0
		    for i = 0; i < 3; i++ {
		        for j = 0; j < 3; j++ {
		            if j == 2 {
		                break
		            }
		            sum = sum + 1
		        }
		    }
		    sum`,
			want: int64(6),
		},
		{
			name: "nested loops with next",
			prog: `
		    var sum = 0
		    for i = 0; i < 3; i++ {
		        for j = 0; j < 3; j++ {
		            if j == 1 {
		                next
		            }
		            sum = sum + 1
		        }
		    }
		    sum`,
			want: int64(6),
		},
		{
			name: "next inside conditional",
			prog: `
		    var sum = 0
		    for i = 0; i < 5; i++ {
		        if i > 0 {
		            if i % 2 == 0 {
		                next
		            }
		        }
		        sum = sum + 1
		    }
		    sum`,
			want: int64(3),
		},
		{
			name: "break inside conditional",
			prog: `
		    var sum = 0
		    for i = 0; i < 10; i++ {
		        if i > 0 {
		            if i == 5 {
		                break
		            }
		        }
		        sum = sum + 1
		    }
		    sum`,
			want: int64(5),
		},
		{
			name: "ensure proper iterations with try",
			prog: `let arr = [{1}, {2}]
			var cnt = 0
		    for i = 0; i < len(arr); i++ {
		    	cnt = cnt + arr[i].0
		    	try assert(arr[i].0 == arr[i].0, "")
		    }
		    cnt`,
			want: int64(3),
		},
		// NOTE: for in not supported yet by frontend
		//	{
		//		name: "for range over array",
		//		prog: `
		//	             var sum = 0
		//	             let arr = [1,2,3]
		//	             for i in arr {
		//	                 sum = sum + arr[i]
		//	             }
		//	             sum`,
		//		want: int64(6),
		//	},
		//
		//	{
		//		name: "for range with element",
		//		prog: `
		//	             var sum = 0
		//	             let arr = [1,2,3]
		//	             for i, e in arr {
		//	                 sum = sum + e
		//	             }
		//	             sum`,
		//		want: int64(6),
		//	},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, err := parseExpressions(tt.prog)
			if err != nil {
				t.Error(err)
			}
			e := NewEvaluator()
			got := e.Eval(n, NewContext(nil))

			if !deepEqual(got, tt.want) {
				t.Errorf("got %v but want %v", got, tt.want)
			}
		})
	}
}

func TestTryExpression(t *testing.T) {
	tests := []struct {
		name string
		prog string
		want any
	}{
		{
			name: "try function that succeeds",
			prog: `
				  error divide_by_zero
		          fn divide(a, b i64)! i64 {
		              if b == 0 {
		                  raise divide_by_zero
		              }
		              return a / b
		          }
		          try divide(10, 2)
		          `,
			want: &Return{Values: []any{int64(5)}},
		},
		{
			name: "try propagates error",
			prog: `
			error divide_by_zero
	        fn divide(a, b i64)! i64 {
	            if b == 0 {
	                raise divide_by_zero
	            }
	            return a / b
	        }
	        fn safe_divide(a, b i64)! i64 {
	            try divide(a, b)
	            return 0
	        }
	        safe_divide(10, 0)`,
			want: &Return{Values: []any{&Error{Err: "divide_by_zero"}}},
		},
		{
			name: "try with multiple return values",
			prog: `
			error divide_by_zero	
	        fn div_mod(a, b i64)! i64, i64 {
		       if b == 0 {
		           raise divide_by_zero
		        }
		        return a / b, a % b
		    }
		    try div_mod(10, 3)
		    `,
			want: &Return{Values: []any{int64(3), int64(1)}},
		},
		{
			name: "try function in function returning value",
			prog: `
			error divide_by_zero
	        fn no_err()! { return }
	        fn test2()! i64 {
	        	try no_err()
	        	return 1
	        }
		    let res = try test2()
		    res`,
			want: int64(1),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, err := parseExpressions(tt.prog)
			if err != nil {
				t.Error(err)
			}
			e := NewEvaluator()
			got := e.Eval(n, NewContext(nil))
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
			want: &Return{Values: []any{int64(3)}},
		},
		// BUG: panic: type is nil in semsis
		//
		//	{
		//		name: "empty array",
		//		prog: "len([])",
		//		want: int64(0),
		//	},
		{
			name: "variable array",
			prog: `
			let arr = [1, 2, 3, 4]
			len(arr)`,
			want: &Return{Values: []any{int64(4)}},
		},
		{
			name: "nested arrays",
			prog: `
			let arr = [[1, 2], [3, 4], [5, 6]]
			len(arr)`,
			want: &Return{Values: []any{int64(3)}},
		},
		{
			name: "len in expression",
			prog: `
			let arr = [1, 2, 3]
			len(arr) + 1`,
			want: int64(4),
		},
		{
			name: "len with function return",
			prog: `
			fn get_arr() []i64 { 
			    return [1, 2, 3, 4, 5]
			}
			len(get_arr())`,
			want: &Return{Values: []any{int64(5)}},
		},
		{
			name: "len with array argument",
			prog: `
			fn check_len(arr []i64) i64 {
			    return len(arr)
			}
			let l = check_len([1, 2, 3])
			l`,
			want: int64(3),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, err := parseExpressions(tt.prog)
			if err != nil {
				t.Error(err)
			}
			e := NewEvaluator()
			got := e.Eval(n, NewContext(nil))

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
		// NOTE: can't be deterministically tested
		// with current set up
		// {
		// 	name: "print struct",
		// 	prog: `
		// 	struct point { x i64, y i64 }
		// 	let p = point{1, 2}
		// 	println(p)
		// 	`,
		// 	wantText: "{0: 1, 1: 2}\n",
		// },
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
			n, err := parseExpressions(tt.prog)
			if err != nil {
				t.Error(err)
			}
			e := NewEvaluator()
			e.Eval(n, NewContext(nil))

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
			want: &Return{[]any{[]any{nil, nil, nil}}},
		},
		{
			name: "make float array",
			prog: "make([]f64, 2)",
			want: &Return{[]any{[]any{nil, nil}}},
		},
		{
			name: "make string array",
			prog: `make([]string, 1)`,
			want: &Return{[]any{[]any{nil}}},
		},
		{
			name: "make empty array",
			prog: "make([]i64, 0)",
			want: &Return{[]any{[]any{}}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, err := parseExpressions(tt.prog)
			if err != nil {
				t.Error(err)
			}
			e := NewEvaluator()
			got := e.Eval(n, NewContext(nil))

			if !deepEqual(got, tt.want) {
				t.Errorf("got %v but want %v", got, tt.want)
			}
		})
	}
}

func TestAssert(t *testing.T) {
	tests := []struct {
		name string
		prog string
		want any
	}{
		{
			name: "assert literal",
			prog: `assert(true, "")`,
			want: nil,
		},
		{
			name: "assert function call",
			prog: `fn call() bool {return false} assert(call(), "want")`,
			want: &Return{[]any{&Error{Err: "want"}}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, err := parseExpressions(tt.prog)
			if err != nil {
				t.Error(err)
			}
			e := NewEvaluator()
			got := e.Eval(n, NewContext(nil))

			if !deepEqual(got, tt.want) {
				t.Errorf("got %v but want %v", got, tt.want)
			}
		})
	}
}

func TestTypeCasting(t *testing.T) {
	tests := []struct {
		name string
		prog string
		want any
	}{
		// {
		// 	name: "scalar type cast",
		// 	prog: "let x = 2 u8(x)",
		// 	want: uint8(2),
		// },
		{
			name: "byte to string",
			prog: "let x = byte(0) string(x)",
			want: string(byte(0)),
		},
		{
			name: "aggregate type cast",
			prog: `
			let x = "hello"
			let y = []byte(x)
			y`,
			want: []byte{104, 101, 108, 108, 111},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, err := parseExpressions(tt.prog)
			if err != nil {
				t.Error(err)
			}
			e := NewEvaluator()
			got := e.Eval(n, NewContext(nil))

			if !deepEqual(got, tt.want) {
				t.Errorf("got %v but want %v", got, tt.want)
			}
		})
	}
}

func TestCustomTypeCasting(t *testing.T) {
	tests := []struct {
		name string
		prog string
		want any
	}{
		{
			name: "type definition type cast",
			prog: `type age i64
				let x = 1
				let a = age(x)
				a`,
			want: int64(1),
		},
		{
			name: "union type cast",
			prog: `union num { i64, u64 } 
				let x = 1
				let n = num(x)
				n`,
			want: &Union{descriptor: uint32(3332055654), value: int64(1)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, err := parseExpressions(tt.prog)
			if err != nil {
				t.Error(err)
			}
			e := NewEvaluator()
			got := e.Eval(n, NewContext(nil))

			if !deepEqual(got, tt.want) {
				t.Errorf("got %v but want %v", got, tt.want)
			}
		})
	}
}

func TestTypeDefinition(t *testing.T) {
	tests := []struct {
		name string
		prog string
		want any
	}{
		{
			name: "accept and return scalar type def",
			prog: `type user string
				fn get(u user) user {
					return u
				}
				let u = get(user("peter"))
				u`,
			want: &Return{Values: []any{"peter"}},
		},
		// BUG: leads to semsis nil panic
		// {
		// 	name: "accept and return aggregate type def",
		// 	prog: `struct person { name string }
		// 		type user person
		// 		fn get(u user) user {
		// 			return u
		// 		}
		// 		let u = get(user(person{name: "peter"}))
		// 		u`,
		// 	want: &Return{vals: []any{"peter"}},
		// },
		// {
		// 	name: "accept and return aggregate type def",
		// 	prog: `type reduce fn(i64, i64) i64
		// 		fn get(f reduce) reduce {
		// 			return f
		// 		}
		// 		let r = fn(a, b i64) i64 { return a + b }
		// 		let func = get(reduce(r))
		// 		func(1, 2)`,
		// 	want: &Return{vals: []any{3}},
		// },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, err := parseExpressions(tt.prog)
			if err != nil {
				t.Error(err)
			}
			e := NewEvaluator()
			got := e.Eval(n, NewContext(nil))

			if !deepEqual(got, tt.want) {
				t.Errorf("got %v but want %v", got, tt.want)
			}
		})
	}
}

func TestMutable(t *testing.T) {
	tests := []struct {
		name string
		prog string
		want any
	}{
		{
			name: "modify array using index expression",
			prog: `
			let buf = make([]i64, 3)
			buf[0] = 1
			buf[1] = 2
			buf`,
			want: []any{int64(1), int64(2), nil},
		},
		{
			name: "modify array using slice expression",
			prog: `
			let buf = make([]i64, 3)
		    buf[0:3] = [3,2,1]
			buf`,
			want: []any{int64(3), int64(2), int64(1)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, err := parseExpressions(tt.prog)
			if err != nil {
				t.Error(err)
			}
			e := NewEvaluator()
			got := e.Eval(n, NewContext(nil))

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
			n, err := parseExpressions(tt.prog)
			if err != nil {
				t.Error(err)
			}
			e := NewEvaluator()
			got := e.Eval(n, NewContext(nil))

			if !deepEqual(got, tt.want) {
				t.Errorf("got %v but want %v", got, tt.want)
			}
		})
	}
}

func TestMatchStatement(t *testing.T) {
	tests := []struct {
		name string
		prog string
		want any
	}{
		{
			name: "single value match",
			prog: `
			let x = 1
			let res = match x {
			case 1: 10
			case _: 0
			}
			res`,
			want: int64(10),
		},
		{
			name: "default case",
			prog: `
			let x = 5.2
			let res = match x {
			case 5.0: 10
			case _: 0
			}
			res`,
			want: int64(0),
		},
		{
			name: "expression as scrutinee",
			prog: `
			let res = match 1 + 1 {
			case 2: 10
			case _: 0
			}
			res`,
			want: int64(10),
		},
		{
			name: "match enum field",
			prog: `
			enum status {
			   online
			   offline
			}
			let s = status.offline
			match s {
			case status.online: 1
			case status.offline: 2
			}`,
			want: int64(2),
		},
		{
			name: "match enum from function",
			prog: `
			enum status { ok err }
			fn get_status() status { return status.err }
			let res = match get_status() {
			case status.ok: 0
			case status.err: 1
			}
			res`,
			want: int64(1),
		},
		{
			name: "match union of scalar",
			prog: `union num { i64, f64 }
			let n = num(1.1)
			match n {
			case f64: 1
			case i64: -1
			}`,
			want: int64(1),
		},
		{
			name: "match union of struct",
			prog: `struct a {}
			struct b {}
			union ab { a, b }
			let n = ab(b{})
			match n {
			case a: 1
			case b: -1
			}`,
			want: int64(-1),
		},
		{
			name: "return match expression",
			prog: `
				fn multi_ret(x i64) i64, string {
					return x, "test"
				}
				fn test() i64, string {
					return match 1 {
						case 1: multi_ret(42)
						case _: multi_ret(0)
					}
				}
				test()
			`,
			want: &Return{Values: []any{int64(42), "test"}},
		},
		{
			name: "raise in match case",
			prog: `
				error some_err
				
				fn test()! i64 {
					 let x = match 0 {
						case 1: 1
						case _:
							raise some_err
							2
					}
					return x
				}
				test()
			`,
			want: &Return{Values: []any{&Error{Err: "some_err"}}},
		},
		{
			name: "match error",
			prog: `
				error one
				error two
				fn test(err error) i64 {
					return match err {
					case one: 1
					case two: 2
					}
				}
				test(one)
			`,
			want: &Return{Values: []any{int64(1)}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, err := parseExpressions(tt.prog)
			if err != nil {
				t.Error(err)
			}
			e := NewEvaluator()
			got := e.Eval(n, NewContext(nil))
			if !deepEqual(got, tt.want) {
				t.Errorf("got %v but want %v", got, tt.want)
			}
		})
	}
}

func TestOptionalTypes(t *testing.T) {
	tests := []struct {
		name      string
		prog      string
		want      any
		wantPanic bool
	}{
		{
			name: "coalesce null",
			prog: `
			fn get_value() ?i64 { return null }
			let x = get_value() ?? 42
			x`,
			want: int64(42),
		},
		{
			name: "coalesce non-null",
			prog: `
			fn get_value() ?i64 { return 7 }
			let x = get_value() ?? 42
			x`,
			want: int64(7),
		},
		{
			name: "force unwrap non-null function call",
			prog: `
			fn get_value() ?i64 { return 42 }
			let x = ?get_value()
			x`,
			want: int64(42),
		},
		{
			name: "force unwrap in expression",
			prog: `
			fn get_value() ?i64 { return 7 }
			let x = ?get_value() + 3
			x`,
			want: int64(10),
		},
		// NOTE: this panics for now but in future it will be error handled in Dash
		{
			name: "force unwrap null panics",
			prog: `
			fn get_value() ?i64 { return null }
			?get_value()`,
			wantPanic: true,
		},
		{
			name: "null equality",
			prog: `
	       fn get_value() ?i64 { return null }
	       let x = get_value() == 10
	       x`,
			want: false,
		},

		{
			name: "null inequality",
			prog: `
	       fn get_value() ?i64 { return 42 }
	       let x = get_value() != null
	       x`,
			want: true,
		},

		{
			name: "null equality between optionals",
			prog: `
	       fn a() ?i64 { return null }
	       fn b() ?i64 { return null }
	       let x = a() == b()
	       x`,
			want: true,
		},

		{
			name: "value equality between optionals",
			prog: `
	       fn a() ?i64 { return 42 }
	       fn b() ?i64 { return 42 }
	       let x = a() == b()
	       x`,
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Error("expected panic but got none")
					}
				}()
			}

			n, err := parseExpressions(tt.prog)
			if err != nil {
				t.Error(err)
			}
			e := NewEvaluator()
			got := e.Eval(n, NewContext(nil))
			if !deepEqual(got, tt.want) {
				t.Errorf("got %v but want %v", got, tt.want)
			}
		})
	}
}

func TestFirstClassFunctions(t *testing.T) {
	tests := []struct {
		name string
		prog string
		want any
	}{
		{
			name: "accept function as argument",
			prog: `
			fn apply(x i64, f fn(i64) i64) i64 {
			  return f(x)
			}
			fn double(x i64) i64 { return x * 2 }
			apply(5, double)`,
			want: &Return{[]any{int64(10)}},
		},
		{
			name: "return function",
			prog: `
			fn makeAdder(x i64) fn(i64) i64 {
			  return fn(y i64) i64 { return x + y }
			}
			let add5 = makeAdder(5)
			add5(3)`,
			want: &Return{[]any{int64(8)}},
		},
		// TODO: requires type definitions and type alias
		// {
		// 	name: "function type definition",
		// 	prog: `
		// 	type unary_fn fn(i64) i64
		// 	fn apply(x i64, f unary_fn) i64 {
		// 	  return f(x)
		// 	}
		// 	fn double(x i64) i64 { return x * 2 }
		// 	apply(5, double)`,
		// 	want: []any{int64(10)},
		// },
		// {
		// 	name: "return defined function type",
		// 	prog: `
		//               type binaryFn fn(i64, i64) i64
		//               fn getOp(op string) binaryFn {
		//                   if op == "add" {
		//                       return fn(x, y i64) i64 { return x + y }
		//                   }
		//                   return fn(x, y i64) i64 { return x * y }
		//               }
		//               let add = getOp("add")
		//               add(2, 3)
		//           `,
		// 	want: []any{int64(5)},
		// },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, err := parseExpressions(tt.prog)
			if err != nil {
				t.Error(err)
			}
			e := NewEvaluator()
			got := e.Eval(n, NewContext(nil))
			if !deepEqual(got, tt.want) {
				t.Errorf("got %v but want %v", got, tt.want)
			}
		})
	}
}

func TestReturn(t *testing.T) {
	tests := []struct {
		name string
		prog string
		want any
	}{
		{
			name: "return in loop conditional",
			prog: `
		          fn find_number(target i64) i64 {
		              for i = 0; i < 10; i++ {
		                  if i == target {
		                      return i
		                  }
		              }
		              return -1
		          }
		          find_number(5)`,
			want: &Return{[]any{int64(5)}},
		},
		{
			name: "early return in nested conditionals",
			prog: `
		          fn find_in_range(start, end i64) i64 {
		              for i = start; i < end; i++ {
		                  if i > 0 {
		                      if i % 3 == 0 {
		                          return i
		                      }
		                  }
		              }
		              return -1
		          }
		          find_in_range(1, 10)`,
			want: &Return{[]any{int64(3)}},
		},
		{
			name: "no match returns default",
			prog: `
            fn find_number(target i64) i64 {
            	var i = 0
                for i < 5 {
                    if i == target {
                        return i
                    }
                    i = i + 1
                }
                return -1
            }
            find_number(10)`,
			want: &Return{[]any{int64(-1)}},
		},
		{
			name: "return function call",
			prog: `
				fn num(a, b i64) i64, i64 {
					return a, b
				}
				fn test() i64, i64 {
					return num(1,2)
				}
				test()`,
			want: &Return{[]any{int64(1), int64(2)}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, err := parseExpressions(tt.prog)
			if err != nil {
				t.Error(err)
			}
			e := NewEvaluator()
			got := e.Eval(n, NewContext(nil))
			if !deepEqual(got, tt.want) {
				t.Errorf("got %v but want %v", got, tt.want)
			}
		})
	}
}

func TestVariableScoping(t *testing.T) {
	tests := []struct {
		name string
		prog string
		want any
	}{
		{
			name: "function scope isolation",
			prog: `
			let x = 5
			fn add(x i64) i64 { return x + 1 }
			let y = add(10)
			x`,
			want: int64(5),
		},
		{
			name: "if block scope",
			prog: `
			let x = 5
			if true {
				let x = 10
			}
			x`,
			want: int64(5),
		},
		{
			name: "closure variable capture",
			prog: `
			fn make_adder(x i64) fn(i64) i64 {
				let y = 2
				return fn(z i64) i64 { return x + y + z }
			}
			let add = make_adder(1)
			add(3)`,
			want: &Return{[]any{int64(6)}},
		},
		{
			name: "variable shadowing in nested functions",
			prog: `
			let x = 1
			fn outer() i64 {
				let x = 2
				fn inner() i64 {
					let x = 3
					return x
				}
				return inner()
			}
			outer()`,
			want: &Return{[]any{int64(3)}},
		},
		{
			name: "match expression scope",
			prog: `
			let x = 1
			let y = match x {
				case 1: 
					let x = 10
					x
				case _: x
			}
			x`,
			want: int64(1),
		},
		// NOTE: blocks not supported yet
		// {
		// 	name: "nested block scope",
		// 	prog: `
		// 		let x = 1
		// 		{
		// 			let x = 2
		// 			let y = x + 1
		// 		}
		// 		x`,
		// 	want: int64(1),
		// },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, err := parseExpressions(tt.prog)
			if err != nil {
				t.Error(err)
			}
			e := NewEvaluator()
			got := e.Eval(n, NewContext(nil))
			if !deepEqual(got, tt.want) {
				t.Errorf("got %v but want %v", got, tt.want)
			}
		})
	}
}

func TestAppend(t *testing.T) {
	tests := []struct {
		name string
		prog string
		want any
	}{
		{
			name: "append single element to array literal",
			prog: `
			let arr = [1, 2]
			append(arr, 3)`,
			want: &Return{[]any{[]any{int64(1), int64(2), int64(3)}}},
		},
		{
			name: "append function result to array - this triggers the bug",
			prog: `
			fn get_arr() []i64 { return [3, 4] }
			let arr = [1, 2]
			append(arr, get_arr())`,
			want: &Return{[]any{[]any{int64(1), int64(2), int64(3), int64(4)}}},
		},
		{
			name: "append function result single element",
			prog: `
			fn get_num() i64 { return 42 }
			let arr = [1, 2]
			append(arr, get_num())`,
			want: &Return{[]any{[]any{int64(1), int64(2), int64(42)}}},
		},
		{
			name: "append single element to array wihout len",
			prog: `
			let arr = make([]byte, 2)
			append(arr, 3)`,
			want: &Return{[]any{[]any{nil, nil, byte(3)}}},
		},
		{
			name: "append single element to array",
			prog: `
			let arr = make([]byte, 0, 2)
			append(arr, 3)`,
			want: &Return{[]any{[]any{byte(3)}}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, err := parseExpressions(tt.prog)
			if err != nil {
				t.Error(err)
			}
			e := NewEvaluator()
			got := e.Eval(n, NewContext(nil))

			if !deepEqual(got, tt.want) {
				t.Errorf("got %v but want %v", got, tt.want)
			}
		})
	}
}

func TestPointers(t *testing.T) {
	tests := []struct {
		name string
		prog string
		want any
	}{
		{
			name: "take address of integer",
			prog: `
            let x = 42
            let ptr = &x
            ptr`,
			want: int64(42),
		},
		{
			name: "dereference pointer",
			prog: `
            let x = 42
            let ptr = &x
            let val = *ptr
            val`,
			want: int64(42),
		},
		{
			name: "reference struct field",
			prog: `
			struct point { x i64 }
			let p = point{x: 42}
			let ptr = &p
			ptr.x`,
			want: int64(42),
		},
		{
			name: "pass reference to function",
			prog: `
			fn use_ptr(p *i64) i64 {
			  return *p + 1
			}
			let x = 42
			use_ptr(&x)`,
			want: &Return{[]any{int64(43)}},
		},
		// BUG: both of these cause semsis errors
		// {
		// 	name: "reference array element",
		// 	prog: `
		// 	let arr = [1, 2, 3]
		// 	let ptr = &arr
		// 	(*ptr)[1]`,
		// 	want: int64(2),
		// },
		// {
		// 	name: "return reference from function",
		// 	prog: `
		// 	fn get_ptr(x i64) *i64 {
		// 	  return &x
		// 	}
		// 	let ptr = get_ptr(42)
		// 	*ptr`,
		// 	want: int64(42),
		// },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, err := parseExpressions(tt.prog)
			if err != nil {
				t.Error(err)
			}
			e := NewEvaluator()
			got := e.Eval(n, NewContext(nil))

			if !deepEqual(got, tt.want) {
				t.Errorf("got %v but want %v", got, tt.want)
			}
		})
	}
}

// ------ //
// Helper //
// ------ //

func getParser(input string) *parser.Parser {
	lcfg := &lexer.Config{SkipComments: true}
	l := lexer.New("", input, lcfg)
	return parser.New(l)
}

func parseExpression(input string) ast.Node {
	p := getParser(input)

	return p.ParseExpression()
}

func parseExpressions(input string) (*ast.Library, error) {
	p := getParser(input)
	lib := p.ParseREPL()

	// sanity check
	s := semantic.New("", nil)
	s.Analyse(lib)
	if len(s.ErrorsFmt()) != 0 {
		return nil, errors.New(strings.Join(s.ErrorsFmt(), "\n"))
	}

	// remove main fn
	lib = removeMainFn(lib)
	return lib, nil
}

func removeMainFn(old *ast.Library) *ast.Library {
	new := &ast.Library{Token: old.Token, Name: old.Name}
	for _, n := range old.Nodes {
		switch n := n.(type) {
		case *ast.FunctionExpression:
			if n.Name.Value != "main" {
				new.Nodes = append(new.Nodes, n)
				continue
			}
			for _, stmt := range n.Body.Statements {
				new.Nodes = append(new.Nodes, stmt)
			}
		default:
			new.Nodes = append(new.Nodes, n)
		}
	}
	return new
}

func NewEvaluator() *Evaluator {
	return &Evaluator{
		libs: make(map[string]*ast.Library),
		ctxs: make(map[string]*Context),
	}
}

func deepEqual(a, b any) bool {
	switch a := a.(type) {
	case []uint8:
		b, ok := b.([]uint8)
		if !ok || len(a) != len(b) {
			return false
		}
		for i := range a {
			if !deepEqual(a[i], b[i]) {
				return false
			}
		}
		return true

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
	case *Union:
		b, ok := b.(*Union)
		if !ok {
			return false
		}
		return a.descriptor == b.descriptor && deepEqual(a.value, b.value)
	case *Return:
		b, ok := b.(*Return)
		if !ok {
			return false
		}
		return deepEqual(a.Values, b.Values)
	case *Error:
		b, ok := b.(*Error)
		if !ok {
			return false
		}
		return a.Err == b.Err
	case error:
		b, ok := b.(error)
		if !ok {
			return false
		}
		return a.Error() == b.Error()
	default:
		return a == b
	}
}
