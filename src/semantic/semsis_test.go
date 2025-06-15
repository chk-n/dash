package semantic

import (
	"testing"

	"dash-lang.io/src/lexer"
	"dash-lang.io/src/parser"
)

type testCase struct {
	name   string
	input  string
	want   string
	errors []string
}

func TestAssignmentStatement(t *testing.T) {
	tests := []testCase{
		{
			name:  "single, int",
			input: "let a = 1 let b = a + 1",
			want:  "lib main pub fn main() { let a i64 = 1 let b i64 = (a + 1) }",
		},
		{
			name:  "single, string",
			input: `let a = "123"`,
			want:  `lib main pub fn main() { let a string = "123" }`,
		},
		{
			name:   "assignment of undefined variable",
			input:  "let a = b",
			errors: []string{"identifier 'b' not found"},
		},
		{
			name:   "assignment with expression - undefined variable",
			input:  "let a = 1 let b = 2 let c = x / b",
			errors: []string{"identifier 'x' not found"},
		},
		{
			name:  "assignment with expression",
			input: "let a = 1 let b = 2 let c = a / b",
			want:  "lib main pub fn main() { let a i64 = 1 let b i64 = 2 let c i64 = (a / b) }",
		},
		{
			name:  "assignment with function",
			input: "let add = fn(a i64, b i64) i64 { return a + b }",
			want:  "lib main pub fn main() { let add fn(i64,i64)i64 = fn(a i64,b i64) i64 { return (a + b) } }",
		},
		{
			name:  "multi, only let",
			input: "let a, let b = 1, 2",
			want:  "lib main pub fn main() { let a i64, let b i64 = 1, 2 }",
		},
		{
			name:  "multi, with reassign",
			input: "var a = 1 a, let b = 2, 3",
			want:  "lib main pub fn main() { var a i64 = 1 a i64, let b i64 = 2, 3 }",
		},
		{
			name:  "function with other vars",
			input: "let x, let y = test(), 1 fn test() i64 { return 1 }",
			want:  "lib main fn test() i64 { return 1 } pub fn main() { let x i64, let y i64 = test(), 1 }",
		},
		// NOTE: this might be made illegal in future as its not clear where which fn is from
		{
			name:  "multiple functions",
			input: "let x, let y = test(), test() fn test() i64 { return 1 }",
			want:  "lib main fn test() i64 { return 1 } pub fn main() { let x i64, let y i64 = test(), test() }",
		},
		{
			name:   "void function, illegal",
			input:  "let x = test() fn test() { return }",
			errors: []string{"cannot assign a void function to a variable"},
		},
		{
			name:   "multiple return assignments to 1 var",
			input:  "let x = match 1 { case _: get() } fn get() i64, i64 { return 0,0 }",
			errors: []string{"assignment mismmatch, assigned 2 values to 1 variables"},
		},
	}
	runAnalysisTests(t, tests)
}

func TestPointerOperations(t *testing.T) {
	tests := []testCase{
		{
			name:  "address of",
			input: "let a = 1 let b = &a",
			want:  "lib main pub fn main() { let a i64 = 1 let b *i64 = &a }",
		},
		{
			name:  "value of",
			input: "fn test(a *string) { let b = *a }",
			want:  "lib main fn test(a *string) { let b string = *a } pub fn main() { }",
		},
		{
			name:   "address of pointer",
			input:  "let a = 1 let b = &a let c = &b",
			errors: []string{"illegal 'address of' operation: unable to get address of pointer"},
		},
		{
			name:   "value of non-pointer",
			input:  "let a = 1 let b = *a",
			errors: []string{"illegal 'value of' operation: value not a pointer"},
		},
	}
	runAnalysisTests(t, tests)
}

func TestStringOperations(t *testing.T) {

	tests := []testCase{
		{
			name:  "concatination - vars",
			input: `let a = "1" let b = a + a`,
			want:  `lib main pub fn main() { let a string = "1" let b string = (a + a) }`,
		},
		{
			name:  "concatination - literals",
			input: `let a = "1" + "2"`,
			want:  `lib main pub fn main() { let a string = ("1" + "2") }`,
		},
		{
			name:  "concatenation - var and literal",
			input: `let a = "a" let b = a + "v"`,
			want:  `lib main pub fn main() { let a string = "a" let b string = (a + "v") }`,
		},
		{
			name:  "equality - vars",
			input: `let a = "1" let b = a == a`,
			want:  `lib main pub fn main() { let a string = "1" let b bool = (a == a) }`,
		},
		{
			name:  "equality - literals",
			input: `let a = "1" == "1"`,
			want:  `lib main pub fn main() { let a bool = ("1" == "1") }`,
		},
		{
			name:  "equality - var and literal",
			input: `let a = "1" let b = "2" == a`,
			want:  `lib main pub fn main() { let a string = "1" let b bool = ("2" == a) }`,
		},
		{
			name:  "inequality - vars",
			input: `let a = "1" let b = a != a`,
			want:  `lib main pub fn main() { let a string = "1" let b bool = (a != a) }`,
		},
		{
			name:  "inequality - literals",
			input: `let a = "1" != "2"`,
			want:  `lib main pub fn main() { let a bool = ("1" != "2") }`,
		},
		{
			name:  "inequality - var and literal",
			input: `let a = "1" let b = "2" == a`,
			want:  `lib main pub fn main() { let a string = "1" let b bool = ("2" == a) }`,
		},
		{
			name:  "index string",
			input: `let s = "123" let ch = s[0]`,
			want:  `lib main pub fn main() { let s string = "123" let ch byte = s[0] }`,
		},
	}
	runAnalysisTests(t, tests)
}

func TestByteComparison(t *testing.T) {
	tests := []testCase{
		{
			name:  "byte equality",
			input: "let b = byte(0) == byte(0)",
			want:  "lib main pub fn main() { let b bool = (byte(0) == byte(0)) }",
		},
		{
			name:  "byte inequality",
			input: "let b = byte(12) != byte(31)",
			want:  "lib main pub fn main() { let b bool = (byte(12) != byte(31)) }",
		},
		{
			name:  "equality int literal",
			input: "let b = 12 == byte(31)",
			want:  "lib main pub fn main() { let b bool = (12 == byte(31)) }",
		},
		{
			name:  "inequality int literal",
			input: "let b = byte(12) != 31",
			want:  "lib main pub fn main() { let b bool = (byte(12) != 31) }",
		},
		{
			name:  "greater than int literal",
			input: "let b = byte(12) > 31",
			want:  "lib main pub fn main() { let b bool = (byte(12) > 31) }",
		},
		{
			name:  "greater equal than int literal",
			input: "let b = 22 >= byte(31)",
			want:  "lib main pub fn main() { let b bool = (22 >= byte(31)) }",
		},
		{
			name:  "less than int literal",
			input: "let b = byte(12) < 31",
			want:  "lib main pub fn main() { let b bool = (byte(12) < 31) }",
		},
		{
			name:  "less equal than int literal",
			input: "let b = byte(12) <= 31",
			want:  "lib main pub fn main() { let b bool = (byte(12) <= 31) }",
		},
	}
	runAnalysisTests(t, tests)
}

func TestCharOperations(t *testing.T) {
	tests := []testCase{
		{
			name:  "char equality",
			input: "let b = 'a' == 'b'",
			want:  "lib main pub fn main() { let b bool = ('a' == 'b') }",
		},
		{
			name:  "char inequality",
			input: "let b = 'a' != 'b'",
			want:  "lib main pub fn main() { let b bool = ('a' != 'b') }",
		},
		{
			name:  "equality, char lit with int",
			input: "let b = 'a' == 1",
			want:  "lib main pub fn main() { let b bool = ('a' == 1) }",
		},
		{
			name:  "inequality, char lit with int",
			input: "let b = 3 != '3'",
			want:  "lib main pub fn main() { let b bool = (3 != '3') }",
		},
		{
			name:  "equality char literal with byte",
			input: "let b = byte(3) == '3'",
			want:  "lib main pub fn main() { let b bool = (byte(3) == '3') }",
		},
		{
			name:  "inequality char literal with byte",
			input: "let b = 'a' != byte(1)",
			want:  "lib main pub fn main() { let b bool = ('a' != byte(1)) }",
		},
		{
			name:  "byte and char",
			input: `let d = byte(10) let v = d - '0'`,
			want:  "lib main pub fn main() { let d byte = byte(10) let v byte = (d - '0') }",
		},
		{
			name:  "string byte and char",
			input: `let s = "12" let n = s[0] - '0'`,
			want:  `lib main pub fn main() { let s string = "12" let n byte = (s[0] - '0') }`,
		},
	}
	runAnalysisTests(t, tests)
}
func TestOptionalType(t *testing.T) {
	tests := []testCase{
		{
			name:  "pass identifier to optional",
			input: "let s = 1 fn test(a ?i64) { } test(s)",
			want:  "lib main fn test(a ?i64) { } pub fn main() { let s i64 = 1 test(s) }",
		},
		{
			name:  "pass int literal to optional",
			input: `fn test(a ?i64) { } test(1)`,
			want:  `lib main fn test(a ?i64) { } pub fn main() { test(1) }`,
		},
		{
			name:  "pass string literal to optional",
			input: `fn test(a ?string) { } test("hello")`,
			want:  `lib main fn test(a ?string) { } pub fn main() { test("hello") }`,
		},
		{
			name:  "pass optional to optional",
			input: "fn test(t ?i64) { test2(t) } fn test2(t ?i64) {}",
			want:  "lib main fn test(t ?i64) { test2(t) } fn test2(t ?i64) { } pub fn main() { }",
		},
		{
			name:  "optional unwrap",
			input: "fn test(a ?i64) { let a' = a ?? 1 }",
			want:  "lib main fn test(a ?i64) { let a' i64 = (a ?? 1) } pub fn main() { }",
		},
		{
			name:  "optional null equality",
			input: "fn test(a ?i64) { let a' = a == null }",
			want:  "lib main fn test(a ?i64) { let a' bool = (a == null) } pub fn main() { }",
		},
		{
			name:  "optional null inequality",
			input: "fn test(a ?i64) { let a' = a != null }",
			want:  "lib main fn test(a ?i64) { let a' bool = (a != null) } pub fn main() { }",
		},
		{
			name:   "null coalesce on non optional literal",
			input:  "let a = 1 ?? 2",
			errors: []string{"illegal use of '??' operation on type 'i64'"},
		},
		{
			name:   "null coalesce on non optional ident",
			input:  "let a = 2 let b = a ?? 2",
			errors: []string{"illegal use of '??' operation on type 'i64'"},
		},
		{
			name:   "null coalsce with both optional",
			input:  "fn test(a ?i64) { let a' = a ?? a }",
			errors: []string{"invalid use of '??' with optional value 'a'"},
		},
		{
			name:   "null coalesce with null LHS",
			input:  "fn test(a ?i64) { let a' = null ?? a }",
			errors: []string{"invalid use of '??' with 'null'"},
		},
		{
			name:   "null coalesce with null RHS",
			input:  "fn test(a ?i64) { let a' = a ?? null }",
			errors: []string{"invalid use of '??' with 'null'"},
		},
		{
			name:  "return null",
			input: "let res = test() fn test() ?i64 { return null }",
			want:  "lib main fn test() ?i64 { return null } pub fn main() { let res ?i64 = test() }",
		},
		{
			name:  "compare against literal",
			input: "fn test(i ?i32) bool { return i == 1 }",
			want:  "lib main fn test(i ?i32) bool { return (i == 1) } pub fn main() { }",
		},
		{
			name:   "compare against literal, wrong type",
			input:  "fn test(i ?i32) bool { return i == 1.1 }",
			errors: []string{"type mistmatch, expected type '?i32' but got 'f64'"},
		},
		{
			name:  "compare against identifier",
			input: "fn test(i ?i32, v i32) bool { return i == v }",
			want:  "lib main fn test(i ?i32,v i32) bool { return (i == v) } pub fn main() { }",
		},
		{
			name:   "compare against identifier, wrong type",
			input:  "fn test(i ?i32, v u32) bool { return i == v }",
			errors: []string{"type mistmatch, expected type '?i32' but got 'u32'"},
		},
		{
			name:  "force unwrap",
			input: "fn test(i ?i32) { let v = ?i }",
			want:  "lib main fn test(i ?i32) { let v i32 = ?i } pub fn main() { }",
		},
		{
			name:   "force unwrap of non optional",
			input:  "let a = 1 let o = ?a",
			errors: []string{"illegal force unwrap of non optional type"},
		},
	}
	runAnalysisTests(t, tests)
}

func TestStructDefinition(t *testing.T) {
	tests := []testCase{
		{
			name:  "struct literal type - inference",
			input: `struct test {name string} let t = test{ name: "123" } let v = t.name`,
			want:  `lib main struct test {name string} pub fn main() { let t test = test{name string: "123"} let v string = t.name }`,
		},
		{
			name:   "struct literal - type mismatch",
			input:  "struct test {name string, age i64} let t = test{ name: 12, age: 20 }",
			errors: []string{"type mistmatch, expected type 'string' but got 'i64'"},
		},
		{
			name:   "struct literal - missing field",
			input:  "struct test {a i64, b i64} let t = test{a: 1}",
			errors: []string{"struct field 'b' not defined"},
		},
		{
			name:  "struct literal - unnamed fields",
			input: "struct point {i64, i64} let p = point{0,0}",
			want:  "lib main struct point {i64, i64} pub fn main() { let p point = point{i64: 0, i64: 0} }",
		},
		{
			name:   "struct literal unnamed, missing field",
			input:  "struct point {i64, i64} let p = point{0}",
			errors: []string{"struct 'point' has missing fields"},
		},
		{
			name:   "struct literal - unnamed fields, type mismatch",
			input:  "struct point {i64, i64} let p = point{0, 1.1}",
			errors: []string{"type mistmatch, expected type 'i64' but got 'f64'"},
		},
		{
			name:  "struct definition - recursive optional",
			input: "struct test{a i64, b ?test} let t = test{a: 1, b: test{a: 1}}",
			want:  "lib main struct test {a i64, b ?test} pub fn main() { let t test = test{a i64: 1, b ?test: test{a i64: 1}} }",
		},
		{
			name:  "struct definition - recursive optional field not set",
			input: "struct test{a i64, b ?test} let t = test{a: 1}",
			want:  "lib main struct test {a i64, b ?test} pub fn main() { let t test = test{a i64: 1} }",
		},
		{
			name:   "struct definition - recursive, illegal",
			input:  "struct test{a i64, b test} let t = test{a: 1, b: test{a: 1}}",
			errors: []string{"field 'b' in struct 'test' cannot reference itself", "struct field 'b' not defined"},
		},
		{
			name:  "struct definition - recursive pointer optional",
			input: "struct abc {a *?abc, b i64} let t = abc{a: &abc{b: 1}, b: 0}",
			want:  "lib main struct abc {a *?abc, b i64} pub fn main() { let t abc = abc{a *?abc: &abc{b i64: 1}, b i64: 0} }",
		},
		{
			name:  "assign null to pointer optional field",
			input: "struct abc {x *?abc} let s = abc{x: null}",
			want:  "lib main struct abc {x *?abc} pub fn main() { let s abc = abc{x *?abc: null} }",
		},
		{
			name:  "struct definition - out of order",
			input: "struct abc {a xyz} struct xyz {x i64} let a = abc{a: xyz{x : 1}} ",
			want:  "lib main struct abc {a xyz} struct xyz {x i64} pub fn main() { let a abc = abc{a xyz: xyz{x i64: 1}} }",
		},
		{
			name:  "struct definition - in order",
			input: "struct xyz {x i64} struct abc {a xyz} let a = abc{a: xyz{x : 1}} ",
			want:  "lib main struct xyz {x i64} struct abc {a xyz} pub fn main() { let a abc = abc{a xyz: xyz{x i64: 1}} }",
		},
		{
			name:   "struct definition - recursive out of order",
			input:  "struct abc {b xyz} struct xyz {a abc} ",
			errors: []string{"cyclical type declarations: abc -> xyz"},
		},
		{
			name:  "struct initialisation - using other struct field",
			input: "struct abc {x i64, y i64} let a = abc{x: 1, y: 1} let a' = abc{x: 0, y: a.y}",
			want:  "lib main struct abc {x i64, y i64} pub fn main() { let a abc = abc{x i64: 1, y i64: 1} let a' abc = abc{x i64: 0, y i64: a.y} }",
		},
		{
			name:  "struct initialisation - using variable",
			input: "struct abc {x i64, y i64} let v = 2 let a = abc{x: 0, y: v}",
			want:  "lib main struct abc {x i64, y i64} pub fn main() { let v i64 = 2 let a abc = abc{x i64: 0, y i64: v} }",
		},
		{
			name:  "cast struct to struct",
			input: `struct a { x i64, y string } struct b { y string, x i64 } let v1 = a{x: 12, y: ""} let v2 = b(v1)`,
			want:  `lib main struct a {x i64, y string} struct b {y string, x i64} pub fn main() { let v1 a = a{x i64: 12, y string: ""} let v2 b = b(v1) }`,
		},
		{
			name:   "cast struct to struct, wrong type",
			input:  `struct a { x i64, y string } struct b { x i64, y i64 } let v1 = a{x: 12, y: ""} let v2 = b(v1)`,
			errors: []string{"illegal type cast from 'a' to 'b'"},
		},
		{
			name:  "cast struct to struct, less fields",
			input: `struct a { x i64, y string } struct b { y string } let v1 = a{x: 12, y: ""} let v2 = b(v1)`,
			want:  `lib main struct a {x i64, y string} struct b {y string} pub fn main() { let v1 a = a{x i64: 12, y string: ""} let v2 b = b(v1) }`,
		},
		{
			name:   "cast struct to struct, optional in 'from'",
			input:  `struct a { x i64, y ?string } struct b { y string, x i64 } let v1 = a{x: 12 } let v2 = b(v1)`,
			errors: []string{"illegal type cast from 'a' to 'b'"},
		},
		{
			name:  "cast struct to struct, optional in 'to'",
			input: `struct a { x i64 } struct b { y ?string, x i64 } let v1 = a{ x: 12 } let v2 = b(v1)`,
			want:  `lib main struct a {x i64} struct b {y ?string, x i64} pub fn main() { let v1 a = a{x i64: 12} let v2 b = b(v1) }`,
		},
		{
			name:  "cast unnamed struct to unnamed struct",
			input: `struct a { i64, string } struct b { i64, string } let v1 = a{ 12, "" } let v2 = b(v1)`,
			want:  `lib main struct a {i64, string} struct b {i64, string} pub fn main() { let v1 a = a{i64: 12, string: ""} let v2 b = b(v1) }`,
		},
		{
			name:   "cast unnamed struct to unnamed struct, wrong order",
			input:  `struct a { i64, string } struct b { string, i64 } let v1 = a{ 12, "" } let v2 = b(v1)`,
			errors: []string{"illegal type cast from 'a' to 'b'"},
		},
		{
			name:  "cast unnamed struct to struct",
			input: `struct a { i64, f64 } struct b { x i64, y f64 } let v1 = a{ 12, 1.1 } let v2 = b(v1)`,
			want:  `lib main struct a {i64, f64} struct b {x i64, y f64} pub fn main() { let v1 a = a{i64: 12, f64: 1.1} let v2 b = b(v1) }`,
		},
		{
			name:  "cast struct to unnamed struct",
			input: `struct a { x i64, y string } struct b { i64, string } let v1 = a{x: 12, y: ""} let v2 = b(v1)`,
			want:  `lib main struct a {x i64, y string} struct b {i64, string} pub fn main() { let v1 a = a{x i64: 12, y string: ""} let v2 b = b(v1) }`,
		},
		{
			name:   "cast struct to unnamed struct, type mismatch",
			input:  `struct a { x i64, y string } struct b { string, i64 } let v1 = a{x: 12, y: ""} let v2 = b(v1)`,
			errors: []string{"illegal type cast from 'a' to 'b'"},
		},
		{
			name:  "cast named literal to struct",
			input: `struct a { y string, x i64 } let v1 = {x: 12, y: ""} let v2 = a(v1)`,
			want:  `lib main struct a {y string, x i64} pub fn main() { let v1 struct<x i64, y string> = {x i64: 12, y string: ""} let v2 a = a(v1) }`,
		},
		{
			name:   "cast unnamed literal to struct, invalid coalesce",
			input:  `struct a { x i32, y string } let v1 = { 12, "" } let v2 = a(v1)`,
			errors: []string{"illegal type cast from 'struct<i64, string>' to 'a'"},
		},
		{
			name:   "cast unnamed literal to struct, type mismatch",
			input:  `struct a { x i32, y string } let v1 = { "", ""} let v2 = a(v1)`,
			errors: []string{"illegal type cast from 'struct<string, string>' to 'a'"},
		},
		{
			name:  "nested struct, test infinite type equality avoided",
			input: "struct abc { child ?abc } fn test() abc { return abc{child: null} }",
			want:  "lib main struct abc {child ?abc} fn test() abc { return abc{child ?abc: null} } pub fn main() { }",
		},
	}
	runAnalysisTests(t, tests)
}

func TestDotExpression(t *testing.T) {
	tests := []testCase{
		{
			name:  "nested struct access",
			input: "struct a { i b } struct b { j i64 } let s = a{i: b{j: 1}} let f = s.i.j",
			want:  "lib main struct a {i b} struct b {j i64} pub fn main() { let s a = a{i b: b{j i64: 1}} let f i64 = s.i.j }",
		},
	}
	runAnalysisTests(t, tests)
}

func TestSliceExpression(t *testing.T) {
	tests := []testCase{
		{
			name:  "slice of index expression",
			input: "struct abc { s string } fn test(a abc) { let v = a.s[0:2]}",
			want:  "lib main struct abc {s string} fn test(a abc) { let v string = a.s[(0 : 2)] } pub fn main() { }",
		},
	}
	runAnalysisTests(t, tests)
}

func TestArray(t *testing.T) {
	tests := []testCase{
		{
			name:  "struct initialisation",
			input: "struct test {x i64} let a = [test{x:1}, test{x:2}] let res = a[0].x - a[1].x",
			want:  "lib main struct test {x i64} pub fn main() { let a []test = [test{x i64: 1},test{x i64: 2}] let res i64 = (a[0].x - a[1].x) }",
		},
		{
			name:  "nested initialisation",
			input: "let a = [[1,2], [3,4]",
			want:  "lib main pub fn main() { let a [][]i64 = [[1,2],[3,4]] }",
		},
		{
			name:  "direct nested access",
			input: "let a = [[1,2], [3,4]] let i = a[0][0]",
			want:  "lib main pub fn main() { let a [][]i64 = [[1,2],[3,4]] let i i64 = a[0][0] }",
		},
		{
			name:  "leveled nested access",
			input: "let a = [[1,2], [3,4]] let a' = a[0] let v = a'[0]",
			want:  "lib main pub fn main() { let a [][]i64 = [[1,2],[3,4]] let a' []i64 = a[0] let v i64 = a'[0] }",
		},
		{
			name:  "fix size",
			input: "fn test(a [2]i64) { let c = a }",
			want:  "lib main fn test(a [2]i64) { let c [2]i64 = a } pub fn main() { }",
		},
		{
			name:   "pass array to sized array type",
			input:  "let arr = [1,2,3] test(arr) fn test(a [2]i64) { }",
			errors: []string{"type mistmatch, expected type '[2]i64' but got '[]i64'"},
		},
		{
			name:  "array slicing",
			input: "let arr = [1,2,3] let arr' = arr[0:1]",
			want:  "lib main pub fn main() { let arr []i64 = [1,2,3] let arr' []i64 = arr[(0 : 1)] }",
		},
		{
			name:  "array slicing",
			input: "let arr = [[1,2,3], [4,5,6], [2,3]] let arr' = arr[1:3][0:1]",
			want:  "lib main pub fn main() { let arr [][]i64 = [[1,2,3],[4,5,6],[2,3]] let arr' [][]i64 = arr[(1 : 3)][(0 : 1)] }",
		},
		{
			name:  "assign casted char to byte array",
			input: "let arr = make([]byte, 1) arr[0] = byte(0 + '0')",
			want:  "lib main pub fn main() { let arr mut<[]byte> = make([]byte,1) arr[0] byte = byte((0 + '0')) }",
		},
		{
			name:  "assign char literal to byte array",
			input: "let arr = make([]byte, 1) arr[0] = '0'",
			want:  "lib main pub fn main() { let arr mut<[]byte> = make([]byte,1) arr[0] byte = '0' }",
		},
		{
			name:   "assign char to byte array",
			input:  "let arr = make([]byte, 1) arr[0] = 0 + '0' }",
			errors: []string{"type mistmatch, expected type 'byte' but got 'char'"},
		},
	}
	runAnalysisTests(t, tests)
}

func TestEnum(t *testing.T) {
	tests := []testCase{
		{
			name:  "enum definition",
			input: "enum status{offline, online} let s = status.offline",
			want:  "lib main enum status {offline, online} pub fn main() { let s status = status.offline }",
		},
		{
			name:  "field equality",
			input: "enum status{offline, online} let b = status.offline == status.online",
			want:  "lib main enum status {offline, online} pub fn main() { let b bool = (status.offline == status.online) }",
		},
		{
			name:  "field inequality",
			input: "enum status{offline, online} let b = status.offline != status.online",
			want:  "lib main enum status {offline, online} pub fn main() { let b bool = (status.offline != status.online) }",
		},
		{
			name:  "greater comparison",
			input: "enum level{low, medium, high} let b = level.high > level.medium",
			want:  "lib main enum level {low, medium, high} pub fn main() { let b bool = (level.high > level.medium) }",
		},
		{
			name:  "less comparison",
			input: "enum level{low, medium, high} let b = level.low < level.medium",
			want:  "lib main enum level {low, medium, high} pub fn main() { let b bool = (level.low < level.medium) }",
		},
		{
			name:  "greater than or equal comparison",
			input: "enum level{low, medium, high} let b = level.high >= level.low",
			want:  "lib main enum level {low, medium, high} pub fn main() { let b bool = (level.high >= level.low) }",
		},
		{
			name:  "less than or equal comparison",
			input: "enum level{low, medium, high} let b = level.low <= level.high",
			want:  "lib main enum level {low, medium, high} pub fn main() { let b bool = (level.low <= level.high) }",
		},
		{
			name:   "equality with other enum",
			input:  "enum abc{x} enum xyz{x} let b = abc.x == xyz.x",
			errors: []string{"type mistmatch, expected type 'abc' but got 'xyz'"},
		},
		{
			name:   "equality with integer",
			input:  "enum status{offline, online} let b = status.offline == 1",
			errors: []string{"type mistmatch, expected type 'status' but got 'i64'"},
		},
		// {
		// 	name:   "enum definition - fiel",
		// 	input:  "enum status{offline, online} let s = status.on",
		// 	errors: []string{""},
		// },
	}
	runAnalysisTests(t, tests)
}

func TestEnumPatternMatching(t *testing.T) {
	tests := []testCase{
		{
			name:  "simple",
			input: "enum abc{x, y} let v = abc.x match v { case abc.x: 1 case abc.y: 2 }",
			want:  "lib main enum abc {x, y} pub fn main() { let v abc = abc.x match v { case abc.x: 1 case abc.y: 2 } }",
		},
		{
			name:   "different enum types",
			input:  "enum abc{x} enum xyz{a} let v = xyz.a match v { case abc.x: false case xyz.a: true }",
			errors: []string{"type mistmatch, expected type 'xyz' but got 'abc'"},
		},
		// TODO: add not all cases matched
	}
	runAnalysisTests(t, tests)
}

func TestUnion(t *testing.T) {
	tests := []testCase{
		{
			name:  "normal",
			input: "union abc { a, b, c, f64 } struct a {} enum b {} type c i64",
			want:  "lib main union abc {a, b, c, f64} struct a {} enum b {} type c i64 pub fn main() { }",
		},
		{
			name:   "cycle between union and union field",
			input:  "union abc { abc }",
			errors: []string{"type in union 'abc' cannot reference itself"},
		},
		{
			name:   "cycle between unions",
			input:  "union abc { xyz } union xyz { abc }",
			errors: []string{"cyclical type declarations: abc -> xyz"},
		},
		{
			name:  "with type def and underlying type",
			input: "union abc { a, i64 } type a i64",
			want:  "lib main union abc {a, i64} type a i64 pub fn main() { }",
		},
		{
			name:   "with duplicate types",
			input:  "union abc { a, a } type a i64",
			errors: []string{"duplicate field 'a' in union 'abc'"},
		},
		{
			name:  "cast using valid type",
			input: "union abc { f64, i64 } let x = abc(1)",
			want:  "lib main union abc {f64, i64} pub fn main() { let x abc = abc(1) }",
		},
		{
			name:   "cast using invalid type",
			input:  "union abc { string } let x = abc(1)",
			errors: []string{"illegal type cast from 'i64' to 'abc'"},
		},
		{
			name:  "match using scalar types",
			input: "union abc { i64, f64 } let a = abc(1.1) let res = match a { case i64: true case f64: false }",
			want:  "lib main union abc {i64, f64} pub fn main() { let a abc = abc(1.1) let res bool = match a { case i64: true case f64: false } }",
		},
		{
			name:   "match using scalar types, wrong type",
			input:  "union abc { i64 } let a = abc(1) match a { case f64: 2 }",
			errors: []string{"type mistmatch, expected type 'abc' but got 'f64'"},
		},
		{
			name:  "match using scalar types",
			input: "union abc { i64 } let a = abc(1) match a { case i64: let x = a } let a' = a",
			want:  "lib main union abc {i64} pub fn main() { let a abc = abc(1) match a { case i64: let x i64 = a } let a' abc = a }",
		},
	}
	runAnalysisTests(t, tests)
}

func TestMatchUnion(t *testing.T) {
	tests := []testCase{
		{
			name:  "simple",
			input: "union abc{i64, f64} let v = abc(1) match v' = v { case f64: let i = v let j = v' }",
			want:  "lib main union abc {i64, f64} pub fn main() { let v abc = abc(1) match (v' = v) { case f64: let i abc = v let j f64 = v' } }",
		},
		{
			name:  "union with structs",
			input: "struct a { x i64 } struct b { y string } union ab { a, b } let s = ab(a{x: 1}) match s' = s { case a: let i = s'.x }",
			want:  "lib main struct a {x i64} struct b {y string} union ab {a, b} pub fn main() { let s ab = ab(a{x i64: 1}) match (s' = s) { case a: let i i64 = s'.x } }",
		},
		{
			name: "ensure type infered if dot expression",
			input: `
			union abc { a }
			struct a { x ?abc }
			fn test(n abc) i64 {
				match n' = n {
				case a:
					let x = n'.x
				}
			}`,
			want: "lib main union abc {a} struct a {x ?abc} fn test(n abc) i64 { match (n' = n) { case a: let x ?abc = n'.x } } pub fn main() { }",
		},
		{
			name:  "ensure default case infers type",
			input: "let x = 1 match x { case _: let v = x }",
			want:  "lib main pub fn main() { let x i64 = 1 match x { case _: let v i64 = x } }",
		},

		// TODO: add not all cases matched
	}
	runAnalysisTests(t, tests)
}

func TestAnonymousStruct(t *testing.T) {
	tests := []testCase{
		{
			name:  "anonymous struct - named fields",
			input: "let a = {x: 1, y: 0.2}",
			want:  "lib main pub fn main() { let a struct<x i64, y f64> = {x i64: 1, y f64: 0.2} }",
		},
		{
			name:  "anonymous struct - unnamed fields",
			input: `let a = {1, "hi"}`,
			want:  `lib main pub fn main() { let a struct<i64, string> = {i64: 1, string: "hi"} }`,
		},
		{
			name:   "anonymous struct - mixed field naming",
			input:  `let a = {x: 1, "hi"}`,
			errors: []string{"mixed named and unnamed fields in 'anonymous' struct"},
		},
		{
			name:  "anonymous struct - access unnamed fields",
			input: `let a = {1, 2} let x, let y = a.0, a.1`,
			want:  "lib main pub fn main() { let a struct<i64, i64> = {i64: 1, i64: 2} let x i64, let y i64 = a.0, a.1 }",
		},
		{
			name:  "anonymous struct - access unnamed fields",
			input: `enum token_type { t } let n = {token_type.t} let typ = n.0`,
			want:  "lib main enum token_type {t} pub fn main() { let n struct<token_type> = {token_type: token_type.t} let typ token_type = n.0 }",
		},
	}
	runAnalysisTests(t, tests)
}

func TestTypeDefinition(t *testing.T) {
	tests := []testCase{
		{
			name:  "scalar type - i64",
			input: "type custom i64 let x = custom(1) let y = x + x",
			want:  "lib main type custom i64 pub fn main() { let x custom = custom(1) let y custom = (x + x) }",
		},
		{
			name:  "aggregate type - array",
			input: "type custom []u64 let x = custom([1, 2]) let y = x[0]",
			want:  "lib main type custom []u64 pub fn main() { let x custom = custom([1,2]) let y u64 = x[0] }",
		},
		{
			name:   "aggregate type - array, invalid type",
			input:  "type custom []u64 let x = custom([1, 2.1]) let y = x[0]",
			errors: []string{"type mistmatch, expected type 'u64' but got 'f64'"},
		},
		{
			name:  "aggregate type - passed and return",
			input: "type custom []byte fn test(c custom) custom { return c }",
			want:  "lib main type custom []byte fn test(c custom) custom { return c } pub fn main() { }",
		},
		{
			name:  "aggregate type - used in memory",
			input: "type custom []byte fn test(mem mut<custom>) { }",
			want:  "lib main type custom []byte fn test(mem mut<custom>) { } pub fn main() { }",
		},
		{
			name:  "aggregate type - struct",
			input: "struct abc { a i64 } type custom abc let x = custom{a: 1}",
			want:  "lib main struct abc {a i64} type custom abc pub fn main() { let x custom = custom{a i64: 1} }",
		},
		{
			name:  "aggregate type - struct, out of order",
			input: "type custom abc struct abc { a i64 } let x = custom{a: 1}",
			want:  "lib main type custom abc struct abc {a i64} pub fn main() { let x custom = custom{a i64: 1} }",
		},
		{
			name: "function type",
			input: `alias reduce fn(i64) i64
				fn get(f reduce) { }
				let r = fn(a i64) i64 { return a }
				get(r)`,
			want: "lib main alias reduce fn(i64)i64 fn get(f fn(i64)i64) { } pub fn main() { let r fn(i64)i64 = fn(a i64) i64 { return a } get(r) }",
		},
		{
			name:  "comparison type def and literal",
			input: "type custom i64 let x = custom(1) == 0",
			want:  "lib main type custom i64 pub fn main() { let x bool = (custom(1) == 0) }",
		},
		{
			name:  "comparison type def and literal expression",
			input: "type custom i64 let x = custom(1) == -1",
			want:  "lib main type custom i64 pub fn main() { let x bool = (custom(1) == -1) }",
		},
		{
			name:  "comparison type def and literal, flipped",
			input: "type custom i64 let x = 0 == custom(1)",
			want:  "lib main type custom i64 pub fn main() { let x bool = (0 == custom(1)) }",
		},
		{
			name:  "algebraic operation with literal",
			input: "type custom i64 let x = custom(1) - 1",
			want:  "lib main type custom i64 pub fn main() { let x custom = (custom(1) - 1) }",
		},
		{
			name:  "cast to guarded type, marks value dirty",
			input: "type custom i64 | custom > 0 let x = custom(1)",
			want:  "lib main type custom i64 | (custom > 0) pub fn main() { let x dirty<custom> = custom(1) }",
		},
		{
			name:  "algebraic operation with literal before validation",
			input: "type custom i64 | custom > 0 let x = custom(1) + 1",
			want:  "lib main type custom i64 | (custom > 0) pub fn main() { let x dirty<custom> = (custom(1) + 1) }",
		},
		{
			name:  "algrebaic operation with literal after validation",
			input: "type custom i64 | custom > 0 let x = custom(1) let valid = validate(x) let y = x let z = x + 1",
			want:  "lib main type custom i64 | (custom > 0) pub fn main() { let x dirty<custom> = custom(1) let valid bool = validate(x) let y custom = x let z dirty<custom> = (x + 1) }",
		},
		{
			name:  "type guarded by function",
			input: "type abc []u8 | is_abc(abc) fn is_abc(a dirty<abc>) { }",
			want:  "lib main type abc []u8 | is_abc(abc) fn is_abc(a dirty<abc>) { } pub fn main() { }",
		},
		{
			name:  "access struct field of type definition",
			input: "struct abc { a i64 } type custom abc let c = custom{a: 1} let val = c.a",
			want:  "lib main struct abc {a i64} type custom abc pub fn main() { let c custom = custom{a i64: 1} let val i64 = c.a }",
		},
		{
			name:  "out of order type def reference in struct",
			input: "type a u32 struct b { f a }",
			want:  "lib main type a u32 struct b {f a} pub fn main() { }",
		},
	}
	runAnalysisTests(t, tests)
}

func TestIfElseExpression(t *testing.T) {
	tests := []testCase{
		{
			name:  "if else expression",
			input: "fn test(a i64) i64 { let b = if a > 1 { a } else { 1 } return b }",
			want:  "lib main fn test(a i64) i64 { let b i64 = if (a > 1) { a } else { 1 } return b } pub fn main() { }",
		},
		{
			name:   "if else expression, return",
			input:  "fn test(a i64) i64 { let b = if a > 1 { a } else { return a } return b }",
			errors: []string{"last value in if else expression not an expression"},
		},
		{
			name:   "if else expression, type mismatch",
			input:  "fn test(a i64) i64 { let b = if a > 1 { a } else { 1.0 } return b }",
			errors: []string{"type mismatch in if else expression got (i64, f64)"},
		},
	}
	runAnalysisTests(t, tests)
}

func TestTypeCast(t *testing.T) {
	tests := []testCase{
		{
			name:  "compare cast u8 with literal",
			input: "let b = u8(255) != 255",
			want:  "lib main pub fn main() { let b bool = (u8(255) != 255) }",
		},
		{
			name:  "compare cast u8 with literal, flipped",
			input: "let b = 255 != u8(255)",
			want:  "lib main pub fn main() { let b bool = (255 != u8(255)) }",
		},
		{
			name:   "compare i8 with u8 cast",
			input:  "let b = i8(127) != u8(127)",
			errors: []string{"type mistmatch, expected type 'i8' but got 'u8'"},
		},
		{
			name:  "cast to type def",
			input: `type a string let x = a("123")`,
			want:  `lib main type a string pub fn main() { let x a = a("123") }`,
		},
		{
			name:  "cast guarded to unguarded",
			input: "type a i8 | a > 0 type b i8 let x = a(1) let y = b(x)",
			want:  "lib main type a i8 | (a > 0) type b i8 pub fn main() { let x dirty<a> = a(1) let y b = b(x) }",
		},
		{
			name:  "cast unguarded to guarded",
			input: "type a f32 type b f32 | b < 0.0 let x = a(1.0) let y = b(x)",
			want:  "lib main type a f32 type b f32 | (b < 0.0) pub fn main() { let x a = a(1.0) let y dirty<b> = b(x) }",
		},
		{
			name:   "cast int literal overflow",
			input:  "let v = u8(256)",
			errors: []string{"integer literal '256' overflows 'u8'"},
		},
		{
			name:  "cast literal to f32",
			input: "let x = f32(1.5)",
			want:  "lib main pub fn main() { let x f32 = f32(1.5) }",
		},
		{
			name:  "cast literal to f64",
			input: "let x = f64(3.4)",
			want:  "lib main pub fn main() { let x f64 = f64(3.4) }",
		},
		{
			name:  "compare casted f32 to literal",
			input: "let b = f32(1.5) == 1.5",
			want:  "lib main pub fn main() { let b bool = (f32(1.5) == 1.5) }",
		},
		{
			name:  "int literal array to []byte",
			input: "let b = []byte([1,2,3,4])",
			want:  "lib main pub fn main() { let b []byte = []byte([1,2,3,4]) }",
		},
		{
			name:   "int literal array to []byte, overflow",
			input:  "let b = []byte([1,0,256])",
			errors: []string{"integer literal '256' overflows 'byte'"},
		},
		{
			name:  "string to []byte",
			input: `let b = []byte("hello, world!")`,
			want:  `lib main pub fn main() { let b []byte = []byte("hello, world!") }`,
		},
		{
			name:  "[]byte to string",
			input: "let b = []byte([1,2,3,4]) let s = string(b)",
			want:  "lib main pub fn main() { let b []byte = []byte([1,2,3,4]) let s string = string(b) }",
		},
		{
			name:  "",
			input: `let s = "12" let n = i64(s[0] - '0')`,
			want:  `lib main pub fn main() { let s string = "12" let n i64 = i64((s[0] - '0')) }`,
		},
		{
			name:  "error type cast",
			input: `let e = error("test message")`,
			want:  `lib main pub fn main() { let e error = error("test message") }`,
		},
		// {
		// 	name:   "int literal array to string, overflow",
		// 	input:  "let s = string([1,0,256])",
		// 	errors: []string{"integer literal '256' overflows 'string'"},
		// },
	}
	runAnalysisTests(t, tests)
}

func TestForLoop(t *testing.T) {
	tests := []testCase{
		{
			name:  "simple for loop",
			input: "for i = 0; i < 10; i++ { }",
			want:  "lib main pub fn main() { for i i64 = 0; (i < 10); i++ { } }",
		},
		{
			name:  "infinite",
			input: "for { }",
			want:  "lib main pub fn main() { for { } }",
		},
		{
			name:  "boolean",
			input: "let x = 2 for x < 0 { }",
			want:  "lib main pub fn main() { let x i64 = 2 for (x < 0) { } }",
		},
		{
			name:  "multiple boolean conditions",
			input: "for true || false && true { }",
			want:  "lib main pub fn main() { for (true || (false && true)) { } }",
		},
		{
			name:   "for loop - invalid condition",
			input:  "for i = 0; i + 10; i++ {}",
			errors: []string{"invalid boolean condition used '+'"},
		},
		{
			name:   "for loop - iden in loop condition",
			input:  "for i = 0; j < 10; i++ {}",
			errors: []string{"identifier 'j' not found"},
		},
		{
			name:   "for loop - iden in increment",
			input:  "for i = 0; i < 10; j++ {}",
			errors: []string{"identifier 'j' not found"},
		},
		// TODO: add test break or next last instruction in block
		{
			name:   "for loop - break not last",
			input:  "for i = 0; i < 10; i++ { break let v = 1 }",
			errors: []string{"'break' not last instruction in block"},
		},
		{
			name:  "for loop - break in if stmt",
			input: "for i = 0; i < 10; i++ { if i == 2 { break } }",
			want:  "lib main pub fn main() { for i i64 = 0; (i < 10); i++ { if (i == 2) { break } } }",
		},
		{
			name:   "for loop - break not last in if stmt",
			input:  "for i = 0; i < 10; i++ { if i == 2 { break let v = 1 } }",
			errors: []string{"'break' not last instruction in block"},
		},
		{
			name:   "function - break not in for loop",
			input:  "fn test() { break }",
			errors: []string{"'break' used outside of 'use' or 'for' block"},
		},
		{
			name:   "for loop - next not last",
			input:  "for i = 0; i < 10; i++ { next let v = 1 }",
			errors: []string{"'next' not last instruction in block"},
		},
		{
			name:   "for loop - next not last in if stmt",
			input:  "for i = 0; i < 10; i++ { if i == 2 { next let v = 1 } }",
			errors: []string{"'next' not last instruction in block"},
		},
		{
			name:   "function - next not in for loop",
			input:  "fn test() { next }",
			errors: []string{"'next' used outside of for loop"},
		},
	}
	runAnalysisTests(t, tests)
}

func TestMatchExpressionStatement(t *testing.T) {
	tests := []testCase{
		{
			name:  "statement",
			input: "let x = 1 match x { case 1: 0 }",
			want:  "lib main pub fn main() { let x i64 = 1 match x { case 1: 0 } }",
		},
		{
			name:  "statement, return",
			input: "fn test(x i64) i64 { match x { case 1: return 0 } }",
			want:  "lib main fn test(x i64) i64 { match x { case 1: return 0 } } pub fn main() { }",
		},
		{
			name:   "scruinee",
			input:  "match x { case 1: 0 }",
			errors: []string{"identifier 'x' not found"},
		},
		{
			name:  "different case types",
			input: `let x = 1 let y = match x { case 1: 0 case 2.0: 1 case "3": 2 } }`,
			errors: []string{
				"type mistmatch, expected type 'i64' but got 'f64'",
				"type mistmatch, expected type 'i64' but got 'string'",
			},
		},
		// // TODO: this should be tested in return statement
		{
			name:   "statement, return wrong type",
			input:  "fn test(x f64) i64 { match x { case 1: return 0 } }",
			errors: []string{"type mistmatch, expected type 'f64' but got 'i64'"},
		},
		{
			name:  "expression",
			input: "let x = 1.1 let y = match x { case 1.1: 0 }",
			want:  "lib main pub fn main() { let x f64 = 1.1 let y i64 = match x { case 1.1: 0 } }",
		},
		{
			name: "statement, different return types",
			input: `
			fn test(x i64) i64 {
				match x {
					case 0: return false
					case 1: return 1
					case 2: return 1.1
				}
			}`,
			errors: []string{
				"type mistmatch, expected type 'i64' but got 'bool'",
				"type mistmatch, expected type 'i64' but got 'f64'",
			},
		},
		// NOTE: maybe we change semsis to issue a warning that case -1 is impossible
		// due to 'y' being unsigned even if u8 can be coalesced to i64.
		{
			name:  "int, cases out of bounds for type",
			input: `let x = u8(1) let y = match x { case 256: 0 case -1: 1 }`,
			want:  "lib main pub fn main() { let x u8 = u8(1) let y i64 = match x { case 256: 0 case -1: 1 } }",
		},
		{
			name:  "byte with char literals",
			input: "let b = byte(0) match b { case '0': let c = b }",
			want:  "lib main pub fn main() { let b byte = byte(0) match b { case '0': let c char = b } }",
		},
	}
	runAnalysisTests(t, tests)
}

func TestCopyExpression(t *testing.T) {
	tests := []testCase{
		{
			name:  "copy",
			input: "let x = 1 let y = x^",
			want:  "lib main pub fn main() { let x i64 = 1 let y i64 = x^ }",
		},
		{
			name:  "copy and update - struct",
			input: "let a = {x: 1,y: 2} let b = a^ { b.x = 2 }",
			want:  "lib main pub fn main() { let a struct<x i64, y i64> = {x i64: 1, y i64: 2} let b struct<x i64, y i64> = a^ { b.x i64 = 2 } }",
		},
		{
			name:  "copy and update - array",
			input: "let a = [0,1,2,3] let b = a^ { b[2] = 1 }",
			want:  "lib main pub fn main() { let a []i64 = [0,1,2,3] let b []i64 = a^ { b[2] i64 = 1 } }",
		},
		{
			name:   "update outside of CopyUpdateExpression",
			input:  "let a = {x: 1,y: 2} a.x = 2",
			errors: []string{"illegal update of 'a'"},
		},
		{
			name:  "copy update guarded type",
			input: `type abc []string | len(abc) < 10 let a = abc(["h", "w"]) let b = a^ { b[0] = "1" }`,
			want:  `lib main type abc []string | (len(abc) < 10) pub fn main() { let a dirty<abc> = abc(["h","w"]) let b dirty<abc> = a^ { b[0] string = "1" } }`,
		},
		// TODO: improve semantic analysis
		// {
		// 	name:   "copy and update - copy passed to function",
		// 	input:  "let a = [0,1,2,3] let b = a^ { test(b) } fn test(a []i64) { }",
		// 	errors: []string{""},
		// },
		// {
		// 	name:   "copy and update - return in update block",
		// 	input:  "a = [0,1,2,3] b = a^ { return b }",
		// 	errors: []string{""},
		// },
		{
			name:  "copy and update - anonymous fn in update block",
			input: "let a = [0,1,2,3] let b = a^ { let test = fn(x i64) { } }",
			want:  "lib main pub fn main() { let a []i64 = [0,1,2,3] let b []i64 = a^ { let test fn(i64) = fn(x i64) { } } }",
		},
		{
			name:  "copy update entire struct",
			input: "struct abc {x i64} let a = abc{x: 1} let b = a^ { b = abc{x: 2}",
			want:  "lib main struct abc {x i64} pub fn main() { let a abc = abc{x i64: 1} let b abc = a^ { b abc = abc{x i64: 2} } }",
		},
	}
	runAnalysisTests(t, tests)
}

func TestMutable(t *testing.T) {
	tests := []testCase{
		{
			name:  "assign value to element from make",
			input: "let arr = make([]i64,10) arr[0] = 1",
			want:  "lib main pub fn main() { let arr mut<[]i64> = make([]i64,10) arr[0] i64 = 1 }",
		},
		{
			name:  "assign value from argument",
			input: `fn test(arr mut<[]i64>) []i64 { arr[0] = 1 return arr }`,
			want:  `lib main fn test(arr mut<[]i64>) []i64 { arr[0] i64 = 1 return arr } pub fn main() { }`,
		},
		{
			name:  "assign value for mutable guarded type",
			input: "type abc []i64 | len(abc) == 10 fn test(arr mut<abc>) { arr[0] = 1}",
			want:  "lib main type abc []i64 | (len(abc) == 10) fn test(arr mut<dirty<abc>>) { arr[0] i64 = 1 } pub fn main() { }",
		},
		{
			name:  "get element from mutable",
			input: "union abc { i64 } fn test(m mut<[]abc>) { let v = m[0] }",
			want:  "lib main union abc {i64} fn test(m mut<[]abc>) { let v abc = m[0] } pub fn main() { }",
		},
		{
			name:  "assign slice to mutable from make",
			input: "union abc { i64 } let arr = make([]abc,10) arr[0:3] = [1,2,3]",
			want:  "lib main union abc {i64} pub fn main() { let arr mut<[]abc> = make([]abc,10) arr[(0 : 3)] mut<[]abc> = [1,2,3] }",
		},
	}
	runAnalysisTests(t, tests)
}

func TestMemorySemantics(t *testing.T) {
	tests := []testCase{
		{
			name:  "passing memory to fn makes it unusable",
			input: "let a = make([]u8, 1) test(&a) let b = a fn test(m *mut<[]u8>) {}",
			want:  "lib main fn test(m *mut<[]u8>) { } pub fn main() { let a mut<[]u8> = make([]u8,1) test(&a) let b []u8 = a }",
		},
	}
	runAnalysisTests(t, tests)
}

func TestTryExpression(t *testing.T) {
	tests := []testCase{
		{
			name:  "simple try with error-prone function",
			input: "error some_err fn may_error()! { raise some_err } fn test()! { try may_error() }",
			want:  "lib main error some_err fn may_error()! { raise some_err } fn test()! { try may_error() } pub fn main() { }",
		},
		{
			name: "try with multiple return values",
			input: `error some_err 
				fn may_error()! i64, bool {
					if true {
						raise some_err
					}  
					return 1, true 
				} 
				fn test()! { 
					let x, let y = try may_error() 
				}`,
			want: "lib main error some_err fn may_error()! i64, bool { if true { raise some_err } return 1, true } fn test()! { let x i64, let y bool = try may_error() } pub fn main() { }",
		},
		// NOTE: semsis rule not implemented yet
		// {
		// 	name:   "try with non-error-prone function",
		// 	input:  "fn no_error() i64 { return 1 } fn test()! { let x = try no_error() }",
		// 	errors: []string{"'try' used with non-error-prone function"},
		// },
		// NOTE: semsis rule not implemented yet
		// {
		// 	name:   "try with non-error-prone function",
		// 	input:  "fn no_error()! fn test() { let x = try no_error() }",
		// 	errors: []string{"'try' found but test' not marked as error-prone"},
		// },
		// NOTE: error-prone divide not implemented
		// {
		// 	name:  "try with force unwrap in error-prone function",
		// 	input: "fn test(x ?i64)! { let y = ?x }",
		// 	want:  "lib main fn test(x ?i64)! { let y i64 = ?x } pub fn main() { }",
		// },
		// NOTE: error-prone divide not implemented
		// {
		// 	name:  "try with division operation",
		// 	input: "fn test(y i64)! { let x = 10 / y }",
		// 	want:  "lib main fn test(y i64)! { let x i64 = (10 / y) } pub fn main() { }",
		// },
		// NOTE: error-prone index op not implemented yet
		// {
		// 	name:  "try with array indexing operation",
		// 	input: "fn test() { let arr = [1, 2, 3] let x = arr[5] }",
		// 	want:  "lib main fn test() { let arr []i64 = [1,2,3] let x i64 = arr[5] } pub fn main() { }",
		// },
	}
	runAnalysisTests(t, tests)
}

func TestFunction(t *testing.T) {
	tests := []testCase{
		{
			name:  "function definition",
			input: "fn test(a i64, b i64) i64 { let c = a + 1 return a / b }",
			want:  "lib main fn test(a i64,b i64) i64 { let c i64 = (a + 1) return (a / b) } pub fn main() { }",
		},
		{
			name:  "function with error parameter",
			input: "fn test(err error) { }",
			want:  "lib main fn test(err error) { } pub fn main() { }",
		},
		{
			name:  "function definition with infered field",
			input: "fn add(x, y i64, a, b string) i64 { return x + y }",
			want:  "lib main fn add(x i64,y i64,a string,b string) i64 { return (x + y) } pub fn main() { }",
		},
		{
			name:  "function call",
			input: "fn test(a i64) i64 { return a / 2 } let c = test(1)",
			want:  "lib main fn test(a i64) i64 { return (a / 2) } pub fn main() { let c i64 = test(1) }",
		},
		{
			name:  "function call - out of order",
			input: "let c = test(1) fn test(a i64) i64 { return 1 / 2 }",
			want:  "lib main fn test(a i64) i64 { return (1 / 2) } pub fn main() { let c i64 = test(1) }",
		},
		{
			name:  "function call multiple return",
			input: "fn test() i64, i64 { return 1 + 2, 2 } let a, let b = test() let c = a * 10",
			want:  "lib main fn test() i64, i64 { return (1 + 2), 2 } pub fn main() { let a i64, let b i64 = test() let c i64 = (a * 10) }",
		},
		{
			name:  "function - struct argument",
			input: "struct abc { a i64 } fn test(a abc) { let b = a.a }",
			want:  "lib main struct abc {a i64} fn test(a abc) { let b i64 = a.a } pub fn main() { }",
		},
		{
			name:  "struct as parameter",
			input: "struct Test{a i64} let t = Test{a: 1} let i = test(t) fn test(t Test) i64 { return t.a }",
			want:  "lib main struct Test {a i64} fn test(t Test) i64 { return t.a } pub fn main() { let t Test = Test{a i64: 1} let i i64 = test(t) }",
		},
		{
			name:  "function - array argument",
			input: "let arr = [1,2,3] let el = test(arr) fn test(a []i64) i64 { return a[-1] }",
			want:  "lib main fn test(a []i64) i64 { return a[-1] } pub fn main() { let arr []i64 = [1,2,3] let el i64 = test(arr) }",
		},
		{
			name:  "function - return struct",
			input: "struct abc { a i64 } fn test() abc { return abc{a: 1} }",
			want:  "lib main struct abc {a i64} fn test() abc { return abc{a i64: 1} } pub fn main() { }",
		},
		{
			name:  "function - return pointer struct",
			input: "struct abc { a i64 } fn test() *abc { return &abc{a: 1} } let v = test()",
			want:  "lib main struct abc {a i64} fn test() *abc { return &abc{a i64: 1} } pub fn main() { let v *abc = test() }",
		},
		{
			name:  "function - return enum",
			input: "enum abc { a } fn test() abc { return abc.a }",
			want:  "lib main enum abc {a} fn test() abc { return abc.a } pub fn main() { }",
		},
		{
			name:  "function - pass enum",
			input: "enum abc { a } fn test(e abc) { let a = e }",
			want:  "lib main enum abc {a} fn test(e abc) { let a abc = e } pub fn main() { }",
		},
		{
			name:   "too many return values",
			input:  "fn test() i64 { return 1, 2 }",
			errors: []string{"too many return values, got (i64) want (i64, i64)"},
		},
		{
			name:   "too little return values",
			input:  "fn test() i64, f64 { return 1 }",
			errors: []string{"too little return values, got (i64) want (i64, f64)"},
		},
		{
			name:  "returning fn with multiple return values",
			input: "fn test() i64, i64 {return 0, 1} fn test2() i64, i64 { return test() }",
			want:  "lib main fn test() i64, i64 { return 0, 1 } fn test2() i64, i64 { return test() } pub fn main() { }",
		},
		{
			name:  "regression prevention, x.b not infered properly",
			input: "struct xyz { i i64 } struct abc { a xyz b ?abc } fn test(x *abc) ?abc { return x.b }",
			want:  "lib main struct xyz {i i64} struct abc {a xyz, b ?abc} fn test(x *abc) ?abc { return x.b } pub fn main() { }",
		},
		{
			name:  "regression prevention, ",
			input: "struct abc { i i64 } fn test(a *abc, b abc) { } let x = &abc{i:1} test(x, abc{i: 2})",
			want:  "lib main struct abc {i i64} fn test(a *abc,b abc) { } pub fn main() { let x *abc = &abc{i i64: 1} test(x,abc{i i64: 2}) }",
		},
		{
			name:  "return function value",
			input: "fn test1() i64, f64 { return 0, 0.0 } fn test2() fn()i64,f64 { return test1 }",
			want:  "lib main fn test1() i64, f64 { return 0, 0.0 } fn test2() fn()i64,f64 { return test1 } pub fn main() { }",
		},
		{
			name:  "return function value, type def",
			input: "type xyz fn()i64 fn test1() i64 { return 0 } fn test2() xyz { return test1 }",
			want:  "lib main type xyz fn()i64 fn test1() i64 { return 0 } fn test2() xyz { return test1 } pub fn main() { }",
		},
		{
			name:  "return function value, optional type",
			input: "type xyz fn()i64 fn test1() i64 { return 0 } fn test2() ?xyz { return test1 }",
			want:  "lib main type xyz fn()i64 fn test1() i64 { return 0 } fn test2() ?xyz { return test1 } pub fn main() { }",
		},
		{
			name:  "call public function",
			input: "pub fn test() {} test()",
			want:  "lib main pub fn test() { } pub fn main() { test() }",
		},
		{
			name:  "call function value",
			input: "type xyz fn()i64 fn test1() i64 { return 0 } fn test2() xyz { return test1 } let func = test2() let v = func()",
			want:  "lib main type xyz fn()i64 fn test1() i64 { return 0 } fn test2() xyz { return test1 } pub fn main() { let func xyz = test2() let v i64 = func() }",
		},
		{
			name: "call function value, optional type",
			input: `struct abc {x f64}
				type xyz fn(string, bool)i64,abc
				fn test1(s string, b bool) i64, abc {
				    return 0, abc{x: 1.1}
				}
				fn test2() ?xyz {
				    return test1
				}
				let func = ?test2()
				let a, let b = func("h", false)`,
			want: `lib main struct abc {x f64} type xyz fn(string,bool)i64,abc fn test1(s string,b bool) i64, abc { return 0, abc{x f64: 1.1} } fn test2() ?xyz { return test1 } pub fn main() { let func xyz = ?test2() let a i64, let b abc = func("h",false) }`,
		},
	}
	runAnalysisTests(t, tests)
}

// func TestPipeOperator(t *testing.T) {
// 	tests := []testCase{
// 		{
// 			name:  "",
// 			input: "let res = byte(0) |> test fn test(b byte) i64 { return i64(b) }",
// 			want:  "",
// 		},
// 	}
// 	runAnalysisTests(t, tests)
// }

func TestHigherOrderFunctions(t *testing.T) {
	tests := []testCase{
		{
			name:  "infer fn type",
			input: "fn test(f fn()) { let t = f }",
			want:  "lib main fn test(f fn()) { let t fn() = f } pub fn main() { }",
		},
		{
			name:  "arg visible as function",
			input: "fn test(f fn()) { f() }",
			want:  "lib main fn test(f fn()) { f() } pub fn main() { }",
		},
		{
			name:  "return fn",
			input: "fn test() fn(i64)i64,i64 { return abc } fn abc(x i64) i64, i64 { return x, x }",
			want:  "lib main fn test() fn(i64)i64,i64 { return abc } fn abc(x i64) i64, i64 { return x, x } pub fn main() { }",
		},
	}
	runAnalysisTests(t, tests)
}

func TestBuiltInFunction(t *testing.T) {
	tests := []testCase{
		{
			name:  "built-in fns - literal argument",
			input: `let l = len([1,2,3]) let c = cap([1,2]) let s = size([1,2,3])`,
			want:  `lib main pub fn main() { let l i64 = len([1,2,3]) let c i64 = cap([1,2]) let s i64 = size([1,2,3]) }`,
		},
		{
			name:  "built-in fns - variable argument",
			input: `let arr = [1,2,3] let l = len(arr) let c = cap(arr) let s = size(arr)`,
			want:  `lib main pub fn main() { let arr []i64 = [1,2,3] let l i64 = len(arr) let c i64 = cap(arr) let s i64 = size(arr) }`,
		},
		{
			name:   "built-in fns - wrong variable argument type",
			input:  `let a = 1 let l = len(a) let c = cap(a)`,
			errors: []string{"type mistmatch, expected type 'T | struct<len i64>, []T, string, mut<[]T>' but got 'i64'", "type mistmatch, expected type 'T | struct<cap i64>, []T, mut<[]T>' but got 'i64'"},
		},
		{
			name:  "make",
			input: "let arr = make([]i64, 10)",
			want:  "lib main pub fn main() { let arr mut<[]i64> = make([]i64,10) }",
		},
		{
			name:  "make with initial value",
			input: "let arr = make([]i64, 10, 0)",
			want:  "lib main pub fn main() { let arr mut<[]i64> = make([]i64,10,0) }",
		},
		{
			name:  "validate",
			input: "type abc i64 | abc == 10 let a = abc(10) let valid = validate(a)",
			want:  "lib main type abc i64 | (abc == 10) pub fn main() { let a dirty<abc> = abc(10) let valid bool = validate(a) }",
		},
		{
			name:  "length of memory",
			input: "let arr = make([]byte, 256) len(arr)",
			want:  "lib main pub fn main() { let arr mut<[]byte> = make([]byte,256) len(arr) }",
		},
		{
			name:  "capacity of memory",
			input: "let arr = make([]byte, 256) cap(arr)",
			want:  "lib main pub fn main() { let arr mut<[]byte> = make([]byte,256) cap(arr) }",
		},
		{
			name:  "assert",
			input: `fn err()! { try assert(true, "") } try err()`,
			want:  `lib main fn err()! { try assert(true,"") } pub fn main() { try err() }`,
		},
	}
	runAnalysisTests(t, tests)
}

func TestExternCTypeConversion(t *testing.T) {
	tests := []testCase{
		{
			name:  "string to []byte",
			input: `@extern(c) fn test(buf []byte) let s = "1" test([]byte(s))`,
			want:  `lib main @extern(c) fn test(buf []byte) pub fn main() { let s string = "1" test([]byte(s)) }`,
		},
		{
			name:  "*string tp *[]byte",
			input: `@extern(c) fn test(buf *[]byte) let s = []byte("1") test(&s)`,
			want:  `lib main @extern(c) fn test(buf *[]byte) pub fn main() { let s []byte = []byte("1") test(&s) }`,
		},
	}
	runAnalysisTests(t, tests)
}

func TestCallArgumentTypeCoercion(t *testing.T) {
	tests := []testCase{
		{
			name:  "int literal, no overflow",
			input: "fn test(x u8) {} test(255)",
			want:  "lib main fn test(x u8) { } pub fn main() { test(255) }",
		},
		{
			name:   "int literal, overflow",
			input:  "fn test(x u8) {} test(257)",
			errors: []string{"integer literal '257' overflows 'u8'"},
		},
		{
			name:  "append error to error array",
			input: "struct abc { errs []error } fn test(a abc, errs []error) { let a = a^{ a.errs = errs } }",
			want:  "lib main struct abc {errs []error} fn test(a abc,errs []error) { let a abc = a^ { a.errs []error = errs } } pub fn main() { }",
		},
	}
	runAnalysisTests(t, tests)
}

func TestAnonymousFunction(t *testing.T) {
	tests := []testCase{
		{
			name:  "no arg, no return",
			input: "let test = fn() { } test()",
			want:  "lib main pub fn main() { let test fn() = fn() { } test() }",
		},
		{
			name:  "multiple arguments",
			input: "let add = fn(a i64, b i64) i64 { let res = a + b return res } let sum = add(1,-1)",
			want:  "lib main pub fn main() { let add fn(i64,i64)i64 = fn(a i64,b i64) i64 { let res i64 = (a + b) return res } let sum i64 = add(1,-1) }",
		},
		{
			name:  "multiple return",
			input: "let add = fn() i64, i64 { return 1,2 } let a, let b = add()",
			want:  "lib main pub fn main() { let add fn()i64,i64 = fn() i64, i64 { return 1, 2 } let a i64, let b i64 = add() }",
		},
		{
			name:  "infer type when left out",
			input: "let do = fn(l string, r string) string { let t = l }",
			want:  "lib main pub fn main() { let do fn(string,string)string = fn(l string,r string) string { let t string = l } }",
		},
		{
			name:  "return pointer struct",
			input: "struct abc { a i64 } let test = fn() *abc { return &abc{a: 1} } let v = test() let r = v.a",
			want:  "lib main struct abc {a i64} pub fn main() { let test fn()*abc = fn() *abc { return &abc{a i64: 1} } let v *abc = test() let r i64 = v.a }",
		},
	}
	runAnalysisTests(t, tests)
}

func TestErrorStatement(t *testing.T) {
	tests := []testCase{
		{
			name:  "error without parameters",
			input: "error divide_by_zero",
			want:  "lib main error divide_by_zero pub fn main() { }",
		},
		{
			name:  "error with single parameter",
			input: "error invalid_value(val string)",
			want:  "lib main error invalid_value(val string) pub fn main() { }",
		},
		{
			name:  "error with multiple parameters",
			input: "error out_of_bounds(index i64, size i64)",
			want:  "lib main error out_of_bounds(index i64, size i64) pub fn main() { }",
		},
		{
			name:  "custom error constructor call with assignment to generic error",
			input: "error custom_error(val i64) fn handle_error(err error) {} let e = custom_error(42) handle_error(e)",
			want:  "lib main error custom_error(val i64) fn handle_error(err error) { } pub fn main() { let e custom_error = custom_error(42) handle_error(e) }",
		},
		{
			name:  "custom error constructor call directly as function argument",
			input: "error my_error(msg string) fn process_error(err error) {} process_error(my_error(\"test\"))",
			want:  "lib main error my_error(msg string) fn process_error(err error) { } pub fn main() { process_error(my_error(\"test\")) }",
		},
	}
	runAnalysisTests(t, tests)
}

func runAnalysisTests(t *testing.T, tests []testCase) {
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			p := GetParser(tt.input)
			ast := p.ParseREPL()

			semsis := New("", nil)
			semsis.Analyse(ast)

			if len(tt.errors) != len(semsis.Errors()) {
				t.Errorf("want %d error(s) but got %d error(s). Errors: %v", len(tt.errors), len(semsis.Errors()), semsis.Errors())
				return
			}

			if len(tt.errors) == 0 && tt.want != ast.String() {
				t.Errorf("want %s but got %s", tt.want, ast.String())
				return
			}

			for i, err := range semsis.Errors() {
				if err != tt.errors[i] {
					t.Errorf("want error %s but got %s", tt.errors[i], err)
				}
			}
		})
	}
}

// ------- //
// Helpers //
// ------- //

func GetParser(input string) *parser.Parser {
	lcfg := &lexer.Config{SkipComments: true}
	l := lexer.New("", input, lcfg)
	return parser.New(l)
}
