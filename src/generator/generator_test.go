package generator

import (
	"testing"

	"dash-lang.io/src/lexer"
	"dash-lang.io/src/parser"
	"dash-lang.io/src/semantic"
	"dash-lang.io/src/transformer"
)

func TestReassignment(t *testing.T) {
	tests := []testCase{
		// 		{
		// 			name:  "scalar",
		// 			input: "lib main fn main() { var x = 0 x = test() for x != 1 { x = test() } } fn test() i64 { return 1 }",
		// 			want:  ``,
		// 		},
		// 		{
		// 			name:  "array",
		// 			input: "lib main fn main() { var x = [1,2,3,4] x = [5]}",
		// 			want: `; ModuleID = 'main-ir'
		// source_filename = "main-ir"
		// target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
		// target triple = "aarch64-linux-unknown"

		// %array = type { i64, ptr }

		// define fastcc void @main() {
		// main_entry:
		//   %0 = alloca %array, align 8
		//   %1 = alloca [4 x i64], align 8
		//   %2 = getelementptr inbounds %array, ptr %0, i32 0, i32 0
		//   store i64 4, ptr %2, align 8
		//   %3 = getelementptr inbounds [4 x i64], ptr %1, i64 0, i64 0
		//   store i64 1, ptr %3, align 8
		//   %4 = getelementptr inbounds [4 x i64], ptr %1, i64 0, i64 1
		//   store i64 2, ptr %4, align 8
		//   %5 = getelementptr inbounds [4 x i64], ptr %1, i64 0, i64 2
		//   store i64 3, ptr %5, align 8
		//   %6 = getelementptr inbounds [4 x i64], ptr %1, i64 0, i64 3
		//   store i64 4, ptr %6, align 8
		//   %7 = getelementptr inbounds %array, ptr %0, i32 0, i32 1
		//   store ptr %1, ptr %7, align 8
		//   %8 = alloca %array, align 8
		//   %9 = alloca [1 x i64], align 8
		//   %10 = getelementptr inbounds %array, ptr %8, i32 0, i32 0
		//   store i64 1, ptr %10, align 8
		//   %11 = getelementptr inbounds [1 x i64], ptr %9, i64 0, i64 0
		//   store i64 5, ptr %11, align 8
		//   %12 = getelementptr inbounds %array, ptr %8, i32 0, i32 1
		//   store ptr %9, ptr %12, align 8
		//   ret void
		// }
		// `,
		// 		},
		// 		{
		// 			name:  "struct",
		// 			input: "lib main struct abc {x i64} fn main() { var x = abc{x: 1} x = abc{x: 2} }",
		// 			want: `; ModuleID = 'main-ir'
		// source_filename = "main-ir"
		// target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
		// target triple = "aarch64-linux-unknown"

		// %abc = type { i64 }

		// define fastcc void @main() {
		// main_entry:
		//   %0 = alloca %abc, align 8
		//   %1 = getelementptr inbounds %abc, ptr %0, i32 0, i32 0
		//   store i64 1, ptr %1, align 8
		//   %2 = alloca %abc, align 8
		//   %3 = getelementptr inbounds %abc, ptr %2, i32 0, i32 0
		//   store i64 2, ptr %3, align 8
		//   ret void
		// }
		// `,
		// 		},
		// 		{
		// 			name:  "anonymous struct",
		// 			input: "lib main fn main() { var x = {x: 1} x = {x: 2} }",
		// 			want: `; ModuleID = 'main-ir'
		// source_filename = "main-ir"
		// target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
		// target triple = "aarch64-linux-unknown"

		// %0 = type { i64 }

		// define fastcc void @main() {
		// main_entry:
		//   %0 = alloca %0, align 8
		//   %1 = getelementptr inbounds %0, ptr %0, i32 0, i32 0
		//   store i64 1, ptr %1, align 8
		//   %2 = alloca %0, align 8
		//   %3 = getelementptr inbounds %0, ptr %2, i32 0, i32 0
		//   store i64 2, ptr %3, align 8
		//   ret void
		// }
		// `,
		// 		},
		// 		{
		// 			name: "reassign within loop, scalar",
		// 			input: `lib main
		// 			fn add(i i64) i64 { return i + 1 }
		// 			fn test(i i64) {
		// 				var i = i
		// 				for {
		// 					if i > 10 {
		// 						break
		// 					}
		// 					i = add(i)
		// 				}
		// 			}
		// 			fn main() { test(0) }
		// 			`,
		// 			want: `; ModuleID = 'main-ir'
		// source_filename = "main-ir"
		// target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
		// target triple = "aarch64-linux-unknown"

		// define internal fastcc i64 @main.add(i64 %0) {
		// add_entry:
		//   %add = add i64 %0, 1
		//   ret i64 %add
		// }

		// define internal fastcc void @main.test(i64 %0) {
		// test_entry:
		//   %1 = alloca i64, align 8
		//   store i64 %0, ptr %1, align 8
		//   br label %2

		// 2:                                                ; preds = %6, %test_entry
		//   %3 = load i64, ptr %1, align 8
		//   %gt = icmp sgt i64 %3, 10
		//   br i1 %gt, label %5, label %6

		// 4:                                                ; preds = %5
		//   ret void

		// 5:                                                ; preds = %2
		//   br label %4

		// 6:                                                ; preds = %2
		//   %7 = load i64, ptr %1, align 8
		//   %8 = call i64 @main.add(i64 %7)
		//   store i64 %8, ptr %1, align 8
		//   br label %2
		// }

		// define fastcc void @main() {
		// main_entry:
		//   call void @main.test(i64 0)
		//   ret void
		// }
		// `,
		// 		},
		// 		{
		// 			name:  "reassign within loop without fn call, struct",
		// 			input: "lib main struct abc{x i64} fn main() { var a = abc{x: 1} for { if a.x > 5 { break } a = abc{x: a.x + 1}}}",
		// 			want: `; ModuleID = 'main-ir'
		// source_filename = "main-ir"
		// target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
		// target triple = "aarch64-linux-unknown"

		// %abc = type { i64 }

		// define fastcc void @main() {
		// main_entry:
		//   %0 = alloca %abc, align 8
		//   %1 = getelementptr inbounds %abc, ptr %0, i32 0, i32 0
		//   store i64 1, ptr %1, align 8
		//   %2 = alloca %abc, align 8
		//   br label %3

		// 3:                                                ; preds = %8, %main_entry
		//   %4 = getelementptr inbounds %abc, ptr %0, i32 0, i32 0
		//   %5 = load i64, ptr %4, align 8
		//   %gt = icmp sgt i64 %5, 5
		//   br i1 %gt, label %7, label %8

		// 6:                                                ; preds = %7
		//   ret void

		// 7:                                                ; preds = %3
		//   br label %6

		// 8:                                                ; preds = %3
		//   %9 = getelementptr inbounds %abc, ptr %0, i32 0, i32 0
		//   %10 = load i64, ptr %9, align 8
		//   %add = add i64 %10, 1
		//   %11 = getelementptr inbounds %abc, ptr %2, i32 0, i32 0
		//   store i64 %add, ptr %11, align 8
		//   call void @llvm.memcpy.p0.p0.i64(ptr %0, ptr %2, i64 ptrtoint (ptr getelementptr (%abc, ptr null, i32 1) to i64), i1 false)
		//   br label %3
		// }

		// ; Function Attrs: nocallback nofree nounwind willreturn memory(argmem: readwrite)
		// declare void @llvm.memcpy.p0.p0.i64(ptr noalias nocapture writeonly, ptr noalias nocapture readonly, i64, i1 immarg) #0

		// attributes #0 = { nocallback nofree nounwind willreturn memory(argmem: readwrite) }
		// `,
		// 		},
		// 		{
		// 			name: "reassign within loop, struct",
		// 			input: `lib main
		// 				struct abc {x i64}
		// 				fn main() { var a = abc{x: 1} for { if a.x == 5 { break } a = get(a) } }
		// 				fn get(s abc) abc { return abc{x: s.x + 1} }`,
		// 			want: `; ModuleID = 'main-ir'
		// source_filename = "main-ir"
		// target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
		// target triple = "aarch64-linux-unknown"

		// %abc = type { i64 }

		// define fastcc void @main() {
		// main_entry:
		//   %0 = alloca %abc, align 8
		//   %1 = getelementptr inbounds %abc, ptr %0, i32 0, i32 0
		//   store i64 1, ptr %1, align 8
		//   %2 = alloca %abc, align 8
		//   br label %3

		// 3:                                                ; preds = %8, %main_entry
		//   %4 = getelementptr inbounds %abc, ptr %0, i32 0, i32 0
		//   %5 = load i64, ptr %4, align 8
		//   %eq = icmp eq i64 %5, 5
		//   br i1 %eq, label %7, label %8

		// 6:                                                ; preds = %7
		//   ret void

		// 7:                                                ; preds = %3
		//   br label %6

		// 8:                                                ; preds = %3
		//   %9 = call ptr @main.get(ptr %0, ptr %2)
		//   call void @llvm.memcpy.p0.p0.i64(ptr %0, ptr %9, i64 ptrtoint (ptr getelementptr (%abc, ptr null, i32 1) to i64), i1 false)
		//   br label %3
		// }

		// define internal fastcc ptr @main.get(ptr %0, ptr %1) {
		// get_entry:
		//   %2 = alloca %abc, align 8
		//   %3 = getelementptr inbounds %abc, ptr %0, i32 0, i32 0
		//   %4 = load i64, ptr %3, align 8
		//   %add = add i64 %4, 1
		//   %5 = getelementptr inbounds %abc, ptr %2, i32 0, i32 0
		//   store i64 %add, ptr %5, align 8
		//   call void @llvm.memcpy.p0.p0.i64(ptr %1, ptr %2, i64 ptrtoint (ptr getelementptr (%abc, ptr null, i32 1) to i64), i1 false)
		//   ret ptr %1
		// }

		// ; Function Attrs: nocallback nofree nounwind willreturn memory(argmem: readwrite)
		// declare void @llvm.memcpy.p0.p0.i64(ptr noalias nocapture writeonly, ptr noalias nocapture readonly, i64, i1 immarg) #0

		// attributes #0 = { nocallback nofree nounwind willreturn memory(argmem: readwrite) }
		// `,
		// 		},
		// 		{
		// 			name: "reassign union",
		// 			input: `lib main
		// 				union xyz{ abc } struct abc{x i64}
		// 				fn main() { var x = xyz(abc{x: 1}) match x' = x { case abc: x = xyz(abc{x: x'.x + 1 }) } }
		// 				`,
		// 			want: `; ModuleID = 'main-ir'
		// source_filename = "main-ir"
		// target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
		// target triple = "aarch64-linux-unknown"

		// %abc = type { i64 }
		// %main.xyz = type { i64, [8 x i8] }

		// define fastcc void @main() {
		// main_entry:
		//   %0 = alloca %abc, align 8
		//   %1 = getelementptr inbounds %abc, ptr %0, i32 0, i32 0
		//   store i64 1, ptr %1, align 8
		//   %2 = alloca %main.xyz, align 8
		//   %3 = getelementptr inbounds %main.xyz, ptr %2, i32 0, i32 0
		//   store i64 6581891533131236247, ptr %3, align 8
		//   %4 = getelementptr inbounds %main.xyz, ptr %2, i32 0, i32 1
		//   call void @llvm.memcpy.p0.p0.i64(ptr %4, ptr %0, i64 ptrtoint (ptr getelementptr (%abc, ptr null, i32 1) to i64), i1 false)
		//   %5 = getelementptr inbounds %main.xyz, ptr %2, i32 0, i32 0
		//   %6 = load i64, ptr %5, align 8
		//   %7 = alloca %abc, align 8
		//   %8 = alloca %main.xyz, align 8
		//   switch i64 %6, label %16 [
		//     i64 6581891533131236247, label %9
		//   ]

		// 9:                                                ; preds = %main_entry
		//   %10 = getelementptr inbounds %main.xyz, ptr %2, i32 0, i32 1
		//   %11 = getelementptr inbounds %abc, ptr %10, i32 0, i32 0
		//   %12 = load i64, ptr %11, align 8
		//   %add = add i64 %12, 1
		//   %13 = getelementptr inbounds %abc, ptr %7, i32 0, i32 0
		//   store i64 %add, ptr %13, align 8
		//   %14 = getelementptr inbounds %main.xyz, ptr %8, i32 0, i32 0
		//   store i64 6581891533131236247, ptr %14, align 8
		//   %15 = getelementptr inbounds %main.xyz, ptr %8, i32 0, i32 1
		//   call void @llvm.memcpy.p0.p0.i64(ptr %15, ptr %7, i64 ptrtoint (ptr getelementptr (%abc, ptr null, i32 1) to i64), i1 false)
		//   call void @llvm.memcpy.p0.p0.i64(ptr %2, ptr %8, i64 ptrtoint (ptr getelementptr (%main.xyz, ptr null, i32 1) to i64), i1 false)
		//   br label %17

		// 16:                                               ; preds = %main_entry
		//   br label %17

		// 17:                                               ; preds = %16, %9
		//   ret void
		// }

		// ; Function Attrs: nocallback nofree nounwind willreturn memory(argmem: readwrite)
		// declare void @llvm.memcpy.p0.p0.i64(ptr noalias nocapture writeonly, ptr noalias nocapture readonly, i64, i1 immarg) #0

		// attributes #0 = { nocallback nofree nounwind willreturn memory(argmem: readwrite) }
		// `,
		// 		},
		// {
		// 	name: "reassign within loop, union",
		// 	input: `lib main
		// 	union xyz{ abc } struct abc{x i64}
		// 	fn main() {
		// 	    var x = xyz(abc{x: 1})
		// 		for {
		// 			match x {
		// 			case abc:
		// 				if x.x > 10 {
		// 					break
		// 				}
		// 			    x = xyz(abc{x: x.x + 1})
		// 			}
		// 		}

		// 	}`,
		// },
	}
	runTests(t, tests)
}

func TestArithmeticI64(t *testing.T) {
	tests := []testCase{
		{
			name:  "i64 multiplication",
			input: "lib main pub fn main() { let m = mul(8, 8) } fn mul(x i64, y i64) i64 { return x * y }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

define fastcc void @main() {
main_entry:
  %0 = call i64 @main.mul(i64 8, i64 8)
  ret void
}

define internal fastcc i64 @main.mul(i64 %0, i64 %1) {
mul_entry:
  %mul = mul i64 %0, %1
  ret i64 %mul
}
`,
		},
		{
			name:  "i64 division",
			input: "lib main pub fn main() { let d = div(64, 8) } fn div(x i64, y i64) i64 { return x / y }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

define fastcc void @main() {
main_entry:
  %0 = call i64 @main.div(i64 64, i64 8)
  ret void
}

define internal fastcc i64 @main.div(i64 %0, i64 %1) {
div_entry:
  %sdiv = sdiv i64 %0, %1
  ret i64 %sdiv
}
`,
		},
	}
	runTests(t, tests)
}

func TestArithmeticF64(t *testing.T) {
	tests := []testCase{
		{
			name:  "f64 addition",
			input: "lib main pub fn main() { let a = add(2.124, -124.43) } fn add(x f64, y f64) f64 { return x + y }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

define fastcc void @main() {
main_entry:
  %0 = call double @main.add(double 2.124000e+00, double -1.244300e+02)
  ret void
}

define internal fastcc double @main.add(double %0, double %1) {
add_entry:
  %fadd = fadd double %0, %1
  ret double %fadd
}
`,
		},
		{
			name:  "f64 subtraction",
			input: "lib main pub fn main() { let s = sub(-1.323, 2.452) } fn sub(x f64, y f64) f64 { return x - y }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

define fastcc void @main() {
main_entry:
  %0 = call double @main.sub(double -1.323000e+00, double 2.452000e+00)
  ret void
}

define internal fastcc double @main.sub(double %0, double %1) {
sub_entry:
  %fsub = fsub double %0, %1
  ret double %fsub
}
`,
		},
		{
			name:  "f64 multiplication",
			input: "lib main pub fn main() { let m = mul(-13.3, 0.0) } fn mul(x f64, y f64) f64 { return x * y }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

define fastcc void @main() {
main_entry:
  %0 = call double @main.mul(double -1.330000e+01, double 0.000000e+00)
  ret void
}

define internal fastcc double @main.mul(double %0, double %1) {
mul_entry:
  %fmul = fmul double %0, %1
  ret double %fmul
}
`,
		},
		{
			name:  "f64 division",
			input: "lib main pub fn main() { let d = div(12.32, -1.1111) } fn div(x f64, y f64) f64 { return x / y }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

define fastcc void @main() {
main_entry:
  %0 = call double @main.div(double 1.232000e+01, double -1.111100e+00)
  ret void
}

define internal fastcc double @main.div(double %0, double %1) {
div_entry:
  %fdiv = fdiv double %0, %1
  ret double %fdiv
}
`,
		},
	}
	runTests(t, tests)
}

func TestComparisonI64(t *testing.T) {
	tests := []testCase{
		{
			name:  "i64 gte",
			input: "lib main pub fn main() { let g = gte(2, 3) } fn gte(x i64, y i64) bool { return x >= y }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

define fastcc void @main() {
main_entry:
  %0 = call i1 @main.gte(i64 2, i64 3)
  ret void
}

define internal fastcc i1 @main.gte(i64 %0, i64 %1) {
gte_entry:
  %ge = icmp sge i64 %0, %1
  ret i1 %ge
}
`,
		},
		{
			name:  "i64 greater than",
			input: "lib main pub fn main() { let g = gt(0, 10) } fn gt(x i64, y i64) bool { return x > y }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

define fastcc void @main() {
main_entry:
  %0 = call i1 @main.gt(i64 0, i64 10)
  ret void
}

define internal fastcc i1 @main.gt(i64 %0, i64 %1) {
gt_entry:
  %gt = icmp sgt i64 %0, %1
  ret i1 %gt
}
`,
		},
		{
			name:  "i64 equal",
			input: "lib main pub fn main() { let e = eq(1,1) } fn eq(x i64, y i64) bool { return x == y }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

define fastcc void @main() {
main_entry:
  %0 = call i1 @main.eq(i64 1, i64 1)
  ret void
}

define internal fastcc i1 @main.eq(i64 %0, i64 %1) {
eq_entry:
  %eq = icmp eq i64 %0, %1
  ret i1 %eq
}
`,
		},
		{
			name:  "i64 not equal",
			input: "lib main pub fn main() { let n = neq(5, 5) } fn neq(x i64, y i64) bool { return x != y }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

define fastcc void @main() {
main_entry:
  %0 = call i1 @main.neq(i64 5, i64 5)
  ret void
}

define internal fastcc i1 @main.neq(i64 %0, i64 %1) {
neq_entry:
  %ne = icmp ne i64 %0, %1
  ret i1 %ne
}
`,
		},
		{
			name:  "i64 less than",
			input: "lib main pub fn main() { let l = lt(100, 100) } fn lt(x i64, y i64) bool { return x < y }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

define fastcc void @main() {
main_entry:
  %0 = call i1 @main.lt(i64 100, i64 100)
  ret void
}

define internal fastcc i1 @main.lt(i64 %0, i64 %1) {
lt_entry:
  %lt = icmp slt i64 %0, %1
  ret i1 %lt
}
`,
		},
		{
			name:  "i64 less than or equal",
			input: "lib main pub fn main() { let l = lte(2,2) } fn lte(x i64, y i64) bool { return x <= y }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

define fastcc void @main() {
main_entry:
  %0 = call i1 @main.lte(i64 2, i64 2)
  ret void
}

define internal fastcc i1 @main.lte(i64 %0, i64 %1) {
lte_entry:
  %le = icmp sle i64 %0, %1
  ret i1 %le
}
`,
		},
	}
	runTests(t, tests)
}

func TestComparisonF64(t *testing.T) {
	tests := []testCase{
		{
			name:  "f64 greater than or equal",
			input: "lib main pub fn main() { let g = gte(0.1, 2.0) } fn gte(x f64, y f64) bool { return x >= y }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

define fastcc void @main() {
main_entry:
  %0 = call i1 @main.gte(double 1.000000e-01, double 2.000000e+00)
  ret void
}

define internal fastcc i1 @main.gte(double %0, double %1) {
gte_entry:
  %fge = fcmp oge double %0, %1
  ret i1 %fge
}
`,
		},
		{
			name:  "f64 greater than",
			input: "lib main pub fn main() { let g = gt(20.0, 10.0) } fn gt(x f64, y f64) bool { return x > y }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

define fastcc void @main() {
main_entry:
  %0 = call i1 @main.gt(double 2.000000e+01, double 1.000000e+01)
  ret void
}

define internal fastcc i1 @main.gt(double %0, double %1) {
gt_entry:
  %fgt = fcmp ogt double %0, %1
  ret i1 %fgt
}
`,
		},
		{
			name:  "f64 equal",
			input: "lib main pub fn main() { let e = eq(1.11, 1.11) } fn eq(x f64, y f64) bool { return x == y }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

define fastcc void @main() {
main_entry:
  %0 = call i1 @main.eq(double 1.110000e+00, double 1.110000e+00)
  ret void
}

define internal fastcc i1 @main.eq(double %0, double %1) {
eq_entry:
  %feq = fcmp oeq double %0, %1
  ret i1 %feq
}
`,
		},
		{
			name:  "f64 not equal",
			input: "lib main pub fn main() { let n = neq(0.0001, 2.12) } fn neq(x f64, y f64) bool { return x != y }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

define fastcc void @main() {
main_entry:
  %0 = call i1 @main.neq(double 1.000000e-04, double 2.120000e+00)
  ret void
}

define internal fastcc i1 @main.neq(double %0, double %1) {
neq_entry:
  %fne = fcmp one double %0, %1
  ret i1 %fne
}
`,
		},
		{
			name:  "f64 less than",
			input: "lib main pub fn main() { let l = lt(0.1, 0.2) } fn lt(x f64, y f64) bool { return x < y }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

define fastcc void @main() {
main_entry:
  %0 = call i1 @main.lt(double 1.000000e-01, double 2.000000e-01)
  ret void
}

define internal fastcc i1 @main.lt(double %0, double %1) {
lt_entry:
  %flt = fcmp olt double %0, %1
  ret i1 %flt
}
`,
		},
		{
			name:  "f64 less than or equal",
			input: "lib main pub fn main() { let l = lte(0.002, 0.1) } fn lte(x f64, y f64) bool { return x <= y }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

define fastcc void @main() {
main_entry:
  %0 = call i1 @main.lte(double 2.000000e-03, double 1.000000e-01)
  ret void
}

define internal fastcc i1 @main.lte(double %0, double %1) {
lte_entry:
  %fle = fcmp ole double %0, %1
  ret i1 %fle
}
`,
		},
	}
	runTests(t, tests)
}

func TestComparisonBool(t *testing.T) {
	tests := []testCase{
		{
			name:  "boolean equal",
			input: "lib main pub fn main() { let e = eq(true, true) } fn eq(x bool, y bool) bool { return x == y }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

define fastcc void @main() {
main_entry:
  %0 = call i1 @main.eq(i1 true, i1 true)
  ret void
}

define internal fastcc i1 @main.eq(i1 %0, i1 %1) {
eq_entry:
  %eq = icmp eq i1 %0, %1
  ret i1 %eq
}
`,
		},
		{
			name:  "boolean not equal",
			input: "lib main pub fn main() { let b = neq(true, false) } fn neq(x bool, y bool) bool { return x != y }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

define fastcc void @main() {
main_entry:
  %0 = call i1 @main.neq(i1 true, i1 false)
  ret void
}

define internal fastcc i1 @main.neq(i1 %0, i1 %1) {
neq_entry:
  %ne = icmp ne i1 %0, %1
  ret i1 %ne
}
`,
		},
	}
	runTests(t, tests)
}

func TestPointerOperation(t *testing.T) {
	tests := []testCase{
		{
			name:  "equality - i64 pointer",
			input: "lib main pub fn main() { let x = 1 let eq = &x == &x }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

define fastcc void @main() {
main_entry:
  %0 = alloca i64, align 8
  store i64 1, ptr %0, align 8
  %1 = alloca i64, align 8
  store i64 1, ptr %1, align 8
  %eq = icmp eq ptr %0, %1
  ret void
}
`,
		},
		{
			name:  "equality - struct pointer",
			input: "lib main pub struct abc { a i64 } fn main() { let x = abc{a: 1} let eq = &x == &x }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%abc = type { i64 }

define fastcc void @main() {
main_entry:
  %0 = alloca %abc, align 8
  %1 = getelementptr inbounds %abc, ptr %0, i32 0, i32 0
  store i64 1, ptr %1, align 8
  %eq = icmp eq ptr %0, %0
  ret void
}
`,
		},
		{
			name:  "equality - array pointer",
			input: "lib main pub fn main() { let x = [1,2,3,4] let eq = &x == &x }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%array = type { i64, ptr }

define fastcc void @main() {
main_entry:
  %0 = alloca %array, align 8
  %1 = alloca [4 x i64], align 8
  %2 = getelementptr inbounds %array, ptr %0, i32 0, i32 0
  store i64 4, ptr %2, align 8
  %3 = getelementptr inbounds [4 x i64], ptr %1, i64 0, i64 0
  store i64 1, ptr %3, align 8
  %4 = getelementptr inbounds [4 x i64], ptr %1, i64 0, i64 1
  store i64 2, ptr %4, align 8
  %5 = getelementptr inbounds [4 x i64], ptr %1, i64 0, i64 2
  store i64 3, ptr %5, align 8
  %6 = getelementptr inbounds [4 x i64], ptr %1, i64 0, i64 3
  store i64 4, ptr %6, align 8
  %7 = getelementptr inbounds %array, ptr %0, i32 0, i32 1
  store ptr %1, ptr %7, align 8
  %eq = icmp eq ptr %0, %0
  ret void
}
`,
		},
		{
			name:  "address of index expression",
			input: "lib main fn main() { let a = [1] let b = &a[0] }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%array = type { i64, ptr }

define fastcc void @main() {
main_entry:
  %0 = alloca %array, align 8
  %1 = alloca [1 x i64], align 8
  %2 = getelementptr inbounds %array, ptr %0, i32 0, i32 0
  store i64 1, ptr %2, align 8
  %3 = getelementptr inbounds [1 x i64], ptr %1, i64 0, i64 0
  store i64 1, ptr %3, align 8
  %4 = getelementptr inbounds %array, ptr %0, i32 0, i32 1
  store ptr %1, ptr %4, align 8
  %5 = getelementptr inbounds %array, ptr %0, i32 0, i32 1
  %6 = load ptr, ptr %5, align 8
  %7 = getelementptr inbounds [1 x i64], ptr %6, i64 0, i64 0
  %8 = load i64, ptr %7, align 8
  %9 = alloca i64, align 8
  store i64 %8, ptr %9, align 8
  ret void
}
`,
		},
		{
			name:    "value of dot expression",
			skipRun: true, // to avoid infinite recursion
			input:   "lib main union xyz { abc } struct abc { a *xyz } fn test(x xyz) { match x { case abc: test(*x.a) } }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%main.xyz = type { i64, [8 x i8] }
%abc = type { ptr }

define internal fastcc void @main.test(ptr %0) {
test_entry:
  %1 = getelementptr inbounds %main.xyz, ptr %0, i32 0, i32 0
  %2 = load i64, ptr %1, align 8
  switch i64 %2, label %7 [
    i64 6581891533131236247, label %3
  ]

3:                                                ; preds = %test_entry
  %4 = getelementptr inbounds %main.xyz, ptr %0, i32 0, i32 1
  %5 = getelementptr inbounds %abc, ptr %4, i32 0, i32 0
  %6 = load ptr, ptr %5, align 8
  call void @main.test(ptr %6)
  br label %8

7:                                                ; preds = %test_entry
  br label %8

8:                                                ; preds = %7, %3
  ret void
}
`,
		},
	}
	runTests(t, tests)
}

func TestCompareChar(t *testing.T) {
	tests := []testCase{
		{
			name:  "char equality",
			input: "lib main fn main() { test() } fn test() bool { return 'a' == 'a' }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

define fastcc void @main() {
main_entry:
  %0 = call i1 @main.test()
  ret void
}

define internal fastcc i1 @main.test() {
test_entry:
  ret i1 true
}
`,
		},
		{
			name:  "char inequality",
			input: "lib main fn main() { test() } fn test() bool { return 'a' != 'b' }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

define fastcc void @main() {
main_entry:
  %0 = call i1 @main.test()
  ret void
}

define internal fastcc i1 @main.test() {
test_entry:
  ret i1 true
}
`,
		},
		{
			name:  "equality, char lit with int",
			input: "lib main fn main() { test() } fn test() bool { return 'a' == 1 }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

define fastcc void @main() {
main_entry:
  %0 = call i1 @main.test()
  ret void
}

define internal fastcc i1 @main.test() {
test_entry:
  ret i1 false
}
`,
		},
		{
			name:  "inequality char literal with byte",
			input: "lib main fn main() { test() } fn test() bool { return 'a' != byte(1) }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

define fastcc void @main() {
main_entry:
  %0 = call i1 @main.test()
  ret void
}

define internal fastcc i1 @main.test() {
test_entry:
  ret i1 true
}
`,
		},
	}
	runTests(t, tests)
}

func TestString(t *testing.T) {
	tests := []testCase{
		// {
		// 	name:  "len of literal",
		// 	input: `lib main fn main() { let l = len("123") }`,
		// },
		{
			name:  "empty string",
			input: `lib main fn main() { let empty = "" }`,
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%string = type { i64, ptr }

@0 = internal constant [0 x i8] null
@1 = internal constant %string { i64 0, ptr @0 }

define fastcc void @main() {
main_entry:
  ret void
}
`,
		},
		{
			name:  "string definition",
			input: `lib main fn main() { let str = get_str() } fn get_str() string { return "12" }`,
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%string = type { i64, ptr }

@0 = internal constant [2 x i8] c"12"
@1 = internal constant %string { i64 2, ptr @0 }

define fastcc void @main() {
main_entry:
  %0 = call ptr @main.get_str()
  ret void
}

define internal fastcc ptr @main.get_str() {
get_str_entry:
  ret ptr @1
}
`,
		},
		{
			name:    "concatenation - 2",
			skipRun: true,
			input:   `lib main fn main() { let s = "1" + "2" }`,
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%string = type { i64, ptr }

@0 = internal constant [1 x i8] c"1"
@1 = internal constant %string { i64 1, ptr @0 }
@2 = internal constant [1 x i8] c"2"
@3 = internal constant %string { i64 1, ptr @2 }

define fastcc void @main() {
main_entry:
  %0 = load i64, ptr @1, align 8
  %1 = load i64, ptr @3, align 8
  %2 = add i64 %0, %1
  %3 = alloca i8, i64 %2, align 4
  %4 = alloca %string, align 8
  %5 = getelementptr inbounds %string, ptr %4, i32 0, i32 0
  store i64 %2, ptr %5, align 8
  %6 = getelementptr inbounds %string, ptr %4, i32 0, i32 1
  store ptr %3, ptr %6, align 8
  %7 = call fastcc ptr @runtime.str_concat2(ptr %4, ptr @1, ptr @3)
  ret void
}

declare fastcc ptr @runtime.str_concat2(ptr, ptr, ptr)
`,
		},
		{
			name:    "equality",
			skipRun: true,
			input:   `lib main fn main() { let s = "1" == "2" }`,
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%string = type { i64, ptr }

@0 = internal constant [1 x i8] c"1"
@1 = internal constant %string { i64 1, ptr @0 }
@2 = internal constant [1 x i8] c"2"
@3 = internal constant %string { i64 1, ptr @2 }

define fastcc void @main() {
main_entry:
  %0 = call fastcc i1 @runtime.str_cmp(ptr @1, ptr @3)
  ret void
}

declare fastcc i1 @runtime.str_cmp(ptr, ptr)
`,
		},
		{
			name:    "inequality",
			skipRun: true,
			input:   `lib main fn main() { let s = "1" != "2" }`,
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%string = type { i64, ptr }

@0 = internal constant [1 x i8] c"1"
@1 = internal constant %string { i64 1, ptr @0 }
@2 = internal constant [1 x i8] c"2"
@3 = internal constant %string { i64 1, ptr @2 }

define fastcc void @main() {
main_entry:
  %0 = call fastcc i1 @runtime.str_cmp(ptr @1, ptr @3)
  %1 = xor i1 %0, true
  ret void
}

declare fastcc i1 @runtime.str_cmp(ptr, ptr)
`,
		},
		{
			name: "len of array after string type cast",
			input: `
lib main

fn main() {
	test("123")
}

fn test(rsn string) i64 {
    let buf = []byte(rsn)
    return len(buf)
}`,
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%string = type { i64, ptr }
%array = type { i64, ptr }

@0 = internal constant [3 x i8] c"123"
@1 = internal constant %string { i64 3, ptr @0 }

define fastcc void @main() {
main_entry:
  %0 = call i64 @main.test(ptr @1)
  ret void
}

define internal fastcc i64 @main.test(ptr %0) {
test_entry:
  %1 = getelementptr inbounds %array, ptr %0, i32 0, i32 0
  %2 = load i64, ptr %1, align 8
  ret i64 %2
}
`,
		},
		{
			name:    "equality check with string struct field",
			skipRun: true,
			input:   `lib main struct abc{x string} fn main() { let a = abc{x: "h"} let b = abc{x:"e"} let b = a.x == b.x }`,
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%string = type { i64, ptr }
%abc = type { %string }

@0 = internal constant [1 x i8] c"h"
@1 = internal constant %string { i64 1, ptr @0 }
@2 = internal constant [1 x i8] c"e"
@3 = internal constant %string { i64 1, ptr @2 }

define fastcc void @main() {
main_entry:
  %0 = alloca %abc, align 8
  %1 = getelementptr inbounds %abc, ptr %0, i32 0, i32 0
  store ptr @1, ptr %1, align 8
  %2 = alloca %abc, align 8
  %3 = getelementptr inbounds %abc, ptr %2, i32 0, i32 0
  store ptr @3, ptr %3, align 8
  %4 = getelementptr inbounds %abc, ptr %0, i32 0, i32 0
  %5 = load ptr, ptr %4, align 8
  %6 = getelementptr inbounds %abc, ptr %2, i32 0, i32 0
  %7 = load ptr, ptr %6, align 8
  %8 = call fastcc i1 @runtime.str_cmp(ptr %5, ptr %7)
  ret void
}

declare fastcc i1 @runtime.str_cmp(ptr, ptr)
`,
		},
	}
	runTests(t, tests)
}
func TestIfElse(t *testing.T) {
	tests := []testCase{
		{
			name:  "if statement",
			input: "lib main pub fn main() { let t = test(-10) } fn test(a i64) i64 { if a >= 0 { return a } return -a }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

define fastcc void @main() {
main_entry:
  %0 = call i64 @main.test(i64 -10)
  ret void
}

define internal fastcc i64 @main.test(i64 %0) {
test_entry:
  %ge = icmp sge i64 %0, 0
  br i1 %ge, label %1, label %2

1:                                                ; preds = %test_entry
  ret i64 %0

2:                                                ; preds = %test_entry
  %neg = sub i64 0, %0
  ret i64 %neg
}
`,
		},
		{
			name:  "multiple if statements",
			input: "lib main pub fn main() { test(-10) } fn test(i i64) i64 { if i > 0 { return i } if i > 0 { return -i } return 0 }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

define fastcc void @main() {
main_entry:
  %0 = call i64 @main.test(i64 -10)
  ret void
}

define internal fastcc i64 @main.test(i64 %0) {
test_entry:
  %gt = icmp sgt i64 %0, 0
  br i1 %gt, label %1, label %2

1:                                                ; preds = %test_entry
  ret i64 %0

2:                                                ; preds = %test_entry
  %gt1 = icmp sgt i64 %0, 0
  br i1 %gt1, label %3, label %4

3:                                                ; preds = %2
  %neg = sub i64 0, %0
  ret i64 %neg

4:                                                ; preds = %2
  ret i64 0
}
`,
		},
		{
			name:  "multiple else if statements",
			input: "lib main pub fn main() { test(-10) } fn test(i i64) i64 { if i > 0 { return i } else if i > 0 { return -i } return 0 }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

define fastcc void @main() {
main_entry:
  %0 = call i64 @main.test(i64 -10)
  ret void
}

define internal fastcc i64 @main.test(i64 %0) {
test_entry:
  %gt = icmp sgt i64 %0, 0
  br i1 %gt, label %1, label %2

1:                                                ; preds = %test_entry
  ret i64 %0

2:                                                ; preds = %test_entry
  %gt1 = icmp sgt i64 %0, 0
  br i1 %gt1, label %3, label %4

3:                                                ; preds = %2
  %neg = sub i64 0, %0
  ret i64 %neg

4:                                                ; preds = %2
  ret i64 0
}
`,
		},
		{
			name:  "if else expression",
			input: "lib main pub fn main() { let t = test(1) } fn test(a i64) i64 { let b = if a > 1 { a } else { 1 } return b }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

define fastcc void @main() {
main_entry:
  %0 = call i64 @main.test(i64 1)
  ret void
}

define internal fastcc i64 @main.test(i64 %0) {
test_entry:
  %gt = icmp sgt i64 %0, 1
  br i1 %gt, label %1, label %2

1:                                                ; preds = %test_entry
  br label %3

2:                                                ; preds = %test_entry
  br label %3

3:                                                ; preds = %2, %1
  %phi = phi i64 [ %0, %1 ], [ 1, %2 ]
  ret i64 %phi
}
`,
		},
		// 		// TODO: WE NEED TO ADD UNREACHABILITY (exhaustiveness check) for this test to pass without return 0 at the end
		{
			name:  "if else statement",
			input: `lib main pub fn main() { let a = abs(-10) } fn abs(a i64) i64 { if a < 0 { return -a } else { return a } return 0 }`,
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

define fastcc void @main() {
main_entry:
  %0 = call i64 @main.abs(i64 -10)
  ret void
}

define internal fastcc i64 @main.abs(i64 %0) {
abs_entry:
  %lt = icmp slt i64 %0, 0
  br i1 %lt, label %1, label %2

1:                                                ; preds = %abs_entry
  %neg = sub i64 0, %0
  ret i64 %neg

2:                                                ; preds = %abs_entry
  ret i64 %0

3:                                                ; No predecessors!
  ret i64 0
}
`,
		},
		// TODO: ADD UNREACHABILITY and remove "return 0" for if else that have return in all blocks
		{
			name:  "if, else if, else statement",
			input: `lib main pub fn main() { let t = test(1) } fn test(i i64) i64 { if i > 1 { return i } else if i == 0 { return 0 } else { return -i } return 0 }`,
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

define fastcc void @main() {
main_entry:
  %0 = call i64 @main.test(i64 1)
  ret void
}

define internal fastcc i64 @main.test(i64 %0) {
test_entry:
  %gt = icmp sgt i64 %0, 1
  br i1 %gt, label %1, label %2

1:                                                ; preds = %test_entry
  ret i64 %0

2:                                                ; preds = %test_entry
  %eq = icmp eq i64 %0, 0
  br i1 %eq, label %3, label %4

3:                                                ; preds = %2
  ret i64 0

4:                                                ; preds = %2
  %neg = sub i64 0, %0
  ret i64 %neg

5:                                                ; No predecessors!
  ret i64 0
}
`,
		},
		{
			name:  "if with multiple else if statement",
			input: `lib main pub fn main() { let t = test(1) } fn test(i i64) i64 { if i > 1 { return i } else if i == 1 { return 0 } else if i < 1 { return -i } return 0 }`,
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

define fastcc void @main() {
main_entry:
  %0 = call i64 @main.test(i64 1)
  ret void
}

define internal fastcc i64 @main.test(i64 %0) {
test_entry:
  %gt = icmp sgt i64 %0, 1
  br i1 %gt, label %1, label %2

1:                                                ; preds = %test_entry
  ret i64 %0

2:                                                ; preds = %test_entry
  %eq = icmp eq i64 %0, 1
  br i1 %eq, label %3, label %4

3:                                                ; preds = %2
  ret i64 0

4:                                                ; preds = %2
  %lt = icmp slt i64 %0, 1
  br i1 %lt, label %5, label %6

5:                                                ; preds = %4
  %neg = sub i64 0, %0
  ret i64 %neg

6:                                                ; preds = %4
  ret i64 0
}
`,
		},
		{
			name:  "if, else if, else expression",
			input: `lib main pub fn main() { let i = 10 let a = if i > 1 { i } else if i == 1 { 0 } else { -i } }`,
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

define fastcc void @main() {
main_entry:
  br i1 true, label %0, label %1

0:                                                ; preds = %main_entry
  br label %4

1:                                                ; preds = %main_entry
  br i1 false, label %2, label %3

2:                                                ; preds = %1
  br label %4

3:                                                ; preds = %1
  br label %4

4:                                                ; preds = %3, %2, %0
  %phi = phi i64 [ 10, %0 ], [ 0, %2 ], [ -10, %3 ]
  ret void
}
`,
		},
		// TODO: but not possible yet, as optional required
		{
			name:  "if expression",
			input: "lib main pub fn main() { let i = 10 let a = if i > 1 { i }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

define fastcc void @main() {
main_entry:
  br i1 true, label %0, label %1

0:                                                ; preds = %main_entry
  br label %1

1:                                                ; preds = %0, %main_entry
  %phi = phi i64 [ 10, %main_entry ], [ 10, %0 ]
  ret void
}
`,
		},
		// TODO: we need optional type here
		{
			name:  "if elif elif expression",
			input: `lib main pub fn main() { let i = 10 let a = if i > 1 { i } else if i == 1 { 0 } else if i < 1 { -i } }`,
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

define fastcc void @main() {
main_entry:
  br i1 true, label %0, label %1

0:                                                ; preds = %main_entry
  br label %5

1:                                                ; preds = %main_entry
  br i1 false, label %2, label %3

2:                                                ; preds = %1
  br label %5

3:                                                ; preds = %1
  br i1 false, label %4, label %5

4:                                                ; preds = %3
  br label %5

5:                                                ; preds = %4, %3, %2, %0
  %phi = phi i64 [ 10, %0 ], [ 0, %2 ], [ -10, %3 ], [ -10, %4 ]
  ret void
}
`,
		},
	}
	runTests(t, tests)
}

func TestStruct(t *testing.T) {
	tests := []testCase{
		{
			name:  "struct - definition and initialisation",
			input: `lib main struct test { a i64, b f64, c bool} pub fn main() { let t = test{a: 1, b: 1.0, c: false} }`,
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%test = type { i64, double, i1 }

define fastcc void @main() {
main_entry:
  %0 = alloca %test, align 8
  %1 = getelementptr inbounds %test, ptr %0, i32 0, i32 0
  store i64 1, ptr %1, align 8
  %2 = getelementptr inbounds %test, ptr %0, i32 0, i32 1
  store double 1.000000e+00, ptr %2, align 8
  %3 = getelementptr inbounds %test, ptr %0, i32 0, i32 2
  store i1 false, ptr %3, align 1
  ret void
}
`,
		},
		{
			name:  "nested struct - definition and initialisation",
			input: `lib main struct B {b i64} struct A {a i64, b B} pub fn main() { let t = A{a: 1, b: B{b: 2}} }`,
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%A = type { i64, %B }
%B = type { i64 }

define fastcc void @main() {
main_entry:
  %0 = alloca %A, align 8
  %1 = getelementptr inbounds %A, ptr %0, i32 0, i32 0
  store i64 1, ptr %1, align 8
  %2 = alloca %B, align 8
  %3 = getelementptr inbounds %B, ptr %2, i32 0, i32 0
  store i64 2, ptr %3, align 8
  %4 = getelementptr inbounds %A, ptr %0, i32 0, i32 1
  store ptr %2, ptr %4, align 8
  ret void
}
`,
		},
		{
			name:  "struct access",
			input: `lib main struct test {a i64} pub fn main() { let t = test{a: 1} let a = t.a }`,
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%test = type { i64 }

define fastcc void @main() {
main_entry:
  %0 = alloca %test, align 8
  %1 = getelementptr inbounds %test, ptr %0, i32 0, i32 0
  store i64 1, ptr %1, align 8
  %2 = getelementptr inbounds %test, ptr %0, i32 0, i32 0
  %3 = load i64, ptr %2, align 8
  ret void
}
`,
		},
		{
			name:  "struct - definition out of order and initialisation",
			input: `lib main struct A {a B, b i64} struct B {a i64} pub fn main() { let t = A{a: B{a: 12}, b: 1} }`,
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%A = type { %B, i64 }
%B = type { i64 }

define fastcc void @main() {
main_entry:
  %0 = alloca %A, align 8
  %1 = alloca %B, align 8
  %2 = getelementptr inbounds %B, ptr %1, i32 0, i32 0
  store i64 12, ptr %2, align 8
  %3 = getelementptr inbounds %A, ptr %0, i32 0, i32 0
  store ptr %1, ptr %3, align 8
  %4 = getelementptr inbounds %A, ptr %0, i32 0, i32 1
  store i64 1, ptr %4, align 8
  ret void
}
`,
		},
		{
			name:  "struct initialisation - using other struct",
			input: "lib main struct abc {x i64, y i64} pub fn main() { let a = abc{x: 1, y: 1} let a' = abc{x: 1, y: a.y} }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%abc = type { i64, i64 }

define fastcc void @main() {
main_entry:
  %0 = alloca %abc, align 8
  %1 = getelementptr inbounds %abc, ptr %0, i32 0, i32 0
  store i64 1, ptr %1, align 8
  %2 = getelementptr inbounds %abc, ptr %0, i32 0, i32 1
  store i64 1, ptr %2, align 8
  %3 = alloca %abc, align 8
  %4 = getelementptr inbounds %abc, ptr %3, i32 0, i32 0
  store i64 1, ptr %4, align 8
  %5 = getelementptr inbounds %abc, ptr %0, i32 0, i32 1
  %6 = load i64, ptr %5, align 8
  %7 = getelementptr inbounds %abc, ptr %3, i32 0, i32 1
  store i64 %6, ptr %7, align 8
  ret void
}
`,
		},
		{
			name:  "struct initialisation - using variable",
			input: "lib main struct abc {x i64, y i64} pub fn main() { let v = 1 let a = abc{x: 1, y: v} }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%abc = type { i64, i64 }

define fastcc void @main() {
main_entry:
  %0 = alloca %abc, align 8
  %1 = getelementptr inbounds %abc, ptr %0, i32 0, i32 0
  store i64 1, ptr %1, align 8
  %2 = getelementptr inbounds %abc, ptr %0, i32 0, i32 1
  store i64 1, ptr %2, align 8
  ret void
}
`,
		},
		{
			name:  "struct - definition, initialisation and access unnamed fields",
			input: `lib main struct point {i64, i64} pub fn main() { let p = point{1, 2} let x, let y = p.0, p.1}`,
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%point = type { i64, i64 }

define fastcc void @main() {
main_entry:
  %0 = alloca %point, align 8
  %1 = getelementptr inbounds %point, ptr %0, i32 0, i32 0
  store i64 1, ptr %1, align 8
  %2 = getelementptr inbounds %point, ptr %0, i32 0, i32 1
  store i64 2, ptr %2, align 8
  %3 = getelementptr inbounds %point, ptr %0, i32 0, i32 0
  %4 = load i64, ptr %3, align 8
  %5 = getelementptr inbounds %point, ptr %0, i32 0, i32 1
  %6 = load i64, ptr %5, align 8
  ret void
}
`,
		},
		{
			name:  "anonymous struct - initialisation and access named fields",
			input: `lib main pub fn main() { let p = {x: 1, y: 2} let x, let y = p.x, p.y }`,
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%0 = type { i64, i64 }

define fastcc void @main() {
main_entry:
  %0 = alloca %0, align 8
  %1 = getelementptr inbounds %0, ptr %0, i32 0, i32 0
  store i64 1, ptr %1, align 8
  %2 = getelementptr inbounds %0, ptr %0, i32 0, i32 1
  store i64 2, ptr %2, align 8
  %3 = getelementptr inbounds %0, ptr %0, i32 0, i32 0
  %4 = load i64, ptr %3, align 8
  %5 = getelementptr inbounds %0, ptr %0, i32 0, i32 1
  %6 = load i64, ptr %5, align 8
  ret void
}
`,
		},
		{
			name:  "anonymous struct - initialisation and access unamed fields",
			input: `lib main pub fn main() { let p = {1, 2} let x, let y = p.0, p.1 }`,
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%0 = type { i64, i64 }

define fastcc void @main() {
main_entry:
  %0 = alloca %0, align 8
  %1 = getelementptr inbounds %0, ptr %0, i32 0, i32 0
  store i64 1, ptr %1, align 8
  %2 = getelementptr inbounds %0, ptr %0, i32 0, i32 1
  store i64 2, ptr %2, align 8
  %3 = getelementptr inbounds %0, ptr %0, i32 0, i32 0
  %4 = load i64, ptr %3, align 8
  %5 = getelementptr inbounds %0, ptr %0, i32 0, i32 1
  %6 = load i64, ptr %5, align 8
  ret void
}
`,
		},
	}
	runTests(t, tests)
}

func TestGlobalStructLiteral(t *testing.T) {
	tests := []testCase{
		{
			name: "define global struct literal",
			// skipRun: true,
			input: "lib main let x = abc{i: 1} struct abc { i i64 } fn main() {}",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%abc = type { i64 }

@x = internal constant %abc { i64 1 }

define fastcc void @main() {
main_entry:
  ret void
}
`,
		},
		{
			name:  "use global struct literal",
			input: "lib main let x = abc{i: 1} struct abc { i i64 } fn main() { let v = x.i }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%abc = type { i64 }

@x = internal constant %abc { i64 1 }

define fastcc void @main() {
main_entry:
  %0 = load i64, ptr @x, align 8
  ret void
}
`,
		},
	}
	runTests(t, tests)
}

func TestGenericStruct(t *testing.T) {
	tests := []testCase{
		{
			name:  "generic struct - definition",
			input: "lib main gen struct xyz { b i64 } fn main() { return }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

define fastcc void @main() {
main_entry:
  ret void
}
`,
		},
	}
	runTests(t, tests)

}

func TestOptional(t *testing.T) {
	tests := []testCase{
		{
			name:  "pass int literal to optional",
			input: "lib main pub fn main() { test(1) } fn test(t ?i64) {}",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%option_i64 = type { i1, i64 }

define fastcc void @main() {
main_entry:
  call void @main.test(%option_i64 { i1 true, i64 1 })
  ret void
}

define internal fastcc void @main.test(%option_i64 %0) {
test_entry:
  ret void
}
`,
		},
		{
			name:  "pass array literal return optional",
			input: "lib main pub fn main() { let a = [1,2,3] test(a) } fn test(t ?[]i64) ?[]i64 { return t }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%array = type { i64, ptr }
%option_ptr = type { i1, ptr }

define fastcc void @main() {
main_entry:
  %0 = alloca %array, align 8
  %1 = alloca [3 x i64], align 8
  %2 = getelementptr inbounds %array, ptr %0, i32 0, i32 0
  store i64 3, ptr %2, align 8
  %3 = getelementptr inbounds [3 x i64], ptr %1, i64 0, i64 0
  store i64 1, ptr %3, align 8
  %4 = getelementptr inbounds [3 x i64], ptr %1, i64 0, i64 1
  store i64 2, ptr %4, align 8
  %5 = getelementptr inbounds [3 x i64], ptr %1, i64 0, i64 2
  store i64 3, ptr %5, align 8
  %6 = getelementptr inbounds %array, ptr %0, i32 0, i32 1
  store ptr %1, ptr %6, align 8
  %7 = insertvalue %option_ptr { i1 true, ptr null }, ptr %0, 1
  %8 = call %option_ptr @main.test(%option_ptr %7)
  ret void
}

define internal fastcc %option_ptr @main.test(%option_ptr %0) {
test_entry:
  ret %option_ptr %0
}
`,
		},
		{
			name:  "pass string literal return optional",
			input: `lib main pub fn main() { let s = "hello" test(s) } fn test(t ?string) ?string { return t}`,
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%string = type { i64, ptr }
%option_ptr = type { i1, ptr }

@0 = internal constant [5 x i8] c"hello"
@1 = internal constant %string { i64 5, ptr @0 }

define fastcc void @main() {
main_entry:
  %0 = call %option_ptr @main.test(%option_ptr { i1 true, ptr @1 })
  ret void
}

define internal fastcc %option_ptr @main.test(%option_ptr %0) {
test_entry:
  ret %option_ptr %0
}
`,
		},
		// TODO: fix optional structs in semsis
		// {
		// 	name:  "pass struct literal to optional",
		// 	input: "lib main struct abc { x i64 } pub fn main() { let s = abc{x: 2} test(s) } fn test(t ?abc) {}",
		// },
		{
			name:  "pass optional to optional",
			input: "lib main pub fn main() { test(1) } fn test(x ?i64) { test2(x) return } fn test2(x ?i64) { return  }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%option_i64 = type { i1, i64 }

define fastcc void @main() {
main_entry:
  call void @main.test(%option_i64 { i1 true, i64 1 })
  ret void
}

define internal fastcc void @main.test(%option_i64 %0) {
test_entry:
  call void @main.test2(%option_i64 %0)
  ret void
}

define internal fastcc void @main.test2(%option_i64 %0) {
test2_entry:
  ret void
}
`,
		},
		{
			name:  "null coalesce, scalar",
			input: "lib main pub fn main() { let res = test(1) } fn test(x ?i64) i64 { return x ?? 1 }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%option_i64 = type { i1, i64 }

define fastcc void @main() {
main_entry:
  %0 = call i64 @main.test(%option_i64 { i1 true, i64 1 })
  ret void
}

define internal fastcc i64 @main.test(%option_i64 %0) {
test_entry:
  %1 = extractvalue %option_i64 %0, 0
  %2 = extractvalue %option_i64 %0, 1
  %3 = icmp eq i1 true, %1
  %4 = select i1 %3, i64 %2, i64 1
  ret i64 %4
}
`,
		},
		{
			name:  "null coalesce, aggregate",
			input: `lib main pub fn main() { let res = test("1") } fn test(x ?string) string { return x ?? "2" }`,
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%string = type { i64, ptr }
%option_ptr = type { i1, ptr }

@0 = internal constant [1 x i8] c"1"
@1 = internal constant %string { i64 1, ptr @0 }
@2 = internal constant [1 x i8] c"2"
@3 = internal constant %string { i64 1, ptr @2 }

define fastcc void @main() {
main_entry:
  %0 = call ptr @main.test(%option_ptr { i1 true, ptr @1 })
  ret void
}

define internal fastcc ptr @main.test(%option_ptr %0) {
test_entry:
  %1 = extractvalue %option_ptr %0, 0
  %2 = extractvalue %option_ptr %0, 1
  %3 = icmp eq i1 true, %1
  %4 = select i1 %3, ptr %2, ptr @3
  ret ptr %4
}
`,
		},
		{
			name:  "return scalar literal",
			input: "lib main pub fn main() { let res = test() } fn test() ?i64 { return 1 }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%option_i64 = type { i1, i64 }

define fastcc void @main() {
main_entry:
  %0 = call %option_i64 @main.test()
  ret void
}

define internal fastcc %option_i64 @main.test() {
test_entry:
  ret %option_i64 { i1 true, i64 1 }
}
`,
		},
		{
			name:  "return string literal",
			input: `lib main pub fn main() { let res = test() } fn test() ?string { return "hello" }`,
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%string = type { i64, ptr }
%option_ptr = type { i1, ptr }

@0 = internal constant [5 x i8] c"hello"
@1 = internal constant %string { i64 5, ptr @0 }

define fastcc void @main() {
main_entry:
  %0 = call %option_ptr @main.test()
  ret void
}

define internal fastcc %option_ptr @main.test() {
test_entry:
  ret %option_ptr { i1 true, ptr @1 }
}
`,
		},
		// TODO: returning arrays requires heap memory
		// TODO: returning structs requires heap memory
		// TODO: requires returned optional types to be allocated by caller
		{
			name:  "return null",
			input: "lib main pub fn main() { let res = test() } fn test() ?i64 { return null }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%option_i64 = type { i1, i64 }

define fastcc void @main() {
main_entry:
  %0 = call %option_i64 @main.test()
  ret void
}

define internal fastcc %option_i64 @main.test() {
test_entry:
  ret %option_i64 zeroinitializer
}
`,
		},
		{
			name:  "force unwrap",
			input: "lib main fn test(i ?i32) i32 { return ?i } fn main() { test(1) }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%option_i32 = type { i1, i32 }

define internal fastcc i32 @main.test(%option_i32 %0) {
test_entry:
  %1 = extractvalue %option_i32 %0, 1
  ret i32 %1
}

define fastcc void @main() {
main_entry:
  %0 = call i32 @main.test(%option_i32 { i1 true, i32 1 })
  ret void
}
`,
		},
	}
	runTests(t, tests)
}

func TestArray(t *testing.T) {
	tests := []testCase{
		{
			name:  "array - i64 initialisation & access",
			input: "lib main pub fn main() { let a = [1, 2, 3_0, 3 + 1] let res = a[0] + a[1] }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%array = type { i64, ptr }

define fastcc void @main() {
main_entry:
  %0 = alloca %array, align 8
  %1 = alloca [4 x i64], align 8
  %2 = getelementptr inbounds %array, ptr %0, i32 0, i32 0
  store i64 4, ptr %2, align 8
  %3 = getelementptr inbounds [4 x i64], ptr %1, i64 0, i64 0
  store i64 1, ptr %3, align 8
  %4 = getelementptr inbounds [4 x i64], ptr %1, i64 0, i64 1
  store i64 2, ptr %4, align 8
  %5 = getelementptr inbounds [4 x i64], ptr %1, i64 0, i64 2
  store i64 30, ptr %5, align 8
  %6 = getelementptr inbounds [4 x i64], ptr %1, i64 0, i64 3
  store i64 4, ptr %6, align 8
  %7 = getelementptr inbounds %array, ptr %0, i32 0, i32 1
  store ptr %1, ptr %7, align 8
  %8 = getelementptr inbounds %array, ptr %0, i32 0, i32 1
  %9 = load ptr, ptr %8, align 8
  %10 = getelementptr inbounds [1 x i64], ptr %9, i64 0, i64 0
  %11 = load i64, ptr %10, align 8
  %12 = getelementptr inbounds %array, ptr %0, i32 0, i32 1
  %13 = load ptr, ptr %12, align 8
  %14 = getelementptr inbounds [1 x i64], ptr %13, i64 0, i64 1
  %15 = load i64, ptr %14, align 8
  %add = add i64 %11, %15
  ret void
}
`,
		},
		{
			name:  "f64 - array initialisation & access",
			input: "lib main pub fn main() { let t = test() } fn test() f64 { let a = [1.1,2.2,3.3] return a[0] - a[2] }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%array = type { i64, ptr }

define fastcc void @main() {
main_entry:
  %0 = call double @main.test()
  ret void
}

define internal fastcc double @main.test() {
test_entry:
  %0 = alloca %array, align 8
  %1 = alloca [3 x double], align 8
  %2 = getelementptr inbounds %array, ptr %0, i32 0, i32 0
  store i64 3, ptr %2, align 8
  %3 = getelementptr inbounds [3 x double], ptr %1, i64 0, i64 0
  store double 1.100000e+00, ptr %3, align 8
  %4 = getelementptr inbounds [3 x double], ptr %1, i64 0, i64 1
  store double 2.200000e+00, ptr %4, align 8
  %5 = getelementptr inbounds [3 x double], ptr %1, i64 0, i64 2
  store double 3.300000e+00, ptr %5, align 8
  %6 = getelementptr inbounds %array, ptr %0, i32 0, i32 1
  store ptr %1, ptr %6, align 8
  %7 = getelementptr inbounds %array, ptr %0, i32 0, i32 1
  %8 = load ptr, ptr %7, align 8
  %9 = getelementptr inbounds [1 x double], ptr %8, i64 0, i64 0
  %10 = load double, ptr %9, align 8
  %11 = getelementptr inbounds %array, ptr %0, i32 0, i32 1
  %12 = load ptr, ptr %11, align 8
  %13 = getelementptr inbounds [1 x double], ptr %12, i64 0, i64 2
  %14 = load double, ptr %13, align 8
  %fsub = fsub double %10, %14
  ret double %fsub
}
`,
		},
		{
			name:  "array - variable initialisation & access",
			input: "lib main pub fn main() { let a = 1 let b = 2 let c = [a, b, 1] let res = c[0] + c[len(c)-1] }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%array = type { i64, ptr }

define fastcc void @main() {
main_entry:
  %0 = alloca %array, align 8
  %1 = alloca [3 x i64], align 8
  %2 = getelementptr inbounds %array, ptr %0, i32 0, i32 0
  store i64 3, ptr %2, align 8
  %3 = getelementptr inbounds [3 x i64], ptr %1, i64 0, i64 0
  store i64 1, ptr %3, align 8
  %4 = getelementptr inbounds [3 x i64], ptr %1, i64 0, i64 1
  store i64 2, ptr %4, align 8
  %5 = getelementptr inbounds [3 x i64], ptr %1, i64 0, i64 2
  store i64 1, ptr %5, align 8
  %6 = getelementptr inbounds %array, ptr %0, i32 0, i32 1
  store ptr %1, ptr %6, align 8
  %7 = getelementptr inbounds %array, ptr %0, i32 0, i32 1
  %8 = load ptr, ptr %7, align 8
  %9 = getelementptr inbounds [1 x i64], ptr %8, i64 0, i64 0
  %10 = load i64, ptr %9, align 8
  %11 = getelementptr inbounds %array, ptr %0, i32 0, i32 1
  %12 = load ptr, ptr %11, align 8
  %13 = getelementptr inbounds %array, ptr %0, i32 0, i32 0
  %14 = load i64, ptr %13, align 8
  %sub = sub i64 %14, 1
  %15 = getelementptr inbounds [1 x i64], ptr %12, i64 0, i64 %sub
  %16 = load i64, ptr %15, align 8
  %add = add i64 %10, %16
  ret void
}
`,
		},
		{
			name:  "array of structs - index and dereference",
			input: "lib main struct test {x i64} pub fn main() { let a = [test{x:10}] let strct = a[0] let field = strct.x }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%array = type { i64, ptr }
%test = type { i64 }

define fastcc void @main() {
main_entry:
  %0 = alloca %array, align 8
  %1 = alloca [1 x %test], align 8
  %2 = getelementptr inbounds %array, ptr %0, i32 0, i32 0
  store i64 1, ptr %2, align 8
  %3 = alloca %test, align 8
  %4 = getelementptr inbounds %test, ptr %3, i32 0, i32 0
  store i64 10, ptr %4, align 8
  %5 = getelementptr inbounds [1 x %test], ptr %1, i64 0, i64 0
  store ptr %3, ptr %5, align 8
  %6 = getelementptr inbounds %array, ptr %0, i32 0, i32 1
  store ptr %1, ptr %6, align 8
  %7 = getelementptr inbounds %array, ptr %0, i32 0, i32 1
  %8 = load ptr, ptr %7, align 8
  %9 = getelementptr inbounds [1 x %test], ptr %8, i64 0, i64 0
  %10 = load ptr, ptr %9, align 8
  %11 = getelementptr inbounds %test, ptr %10, i32 0, i32 0
  %12 = load i64, ptr %11, align 8
  ret void
}
`,
		},
		{
			name:  "array of structs - index and direct dereference",
			input: "lib main struct test {x i64} pub fn main() { let a = [test{x:1}, test{x:2}] let res = a[0].x }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%array = type { i64, ptr }
%test = type { i64 }

define fastcc void @main() {
main_entry:
  %0 = alloca %array, align 8
  %1 = alloca [2 x %test], align 8
  %2 = getelementptr inbounds %array, ptr %0, i32 0, i32 0
  store i64 2, ptr %2, align 8
  %3 = alloca %test, align 8
  %4 = getelementptr inbounds %test, ptr %3, i32 0, i32 0
  store i64 1, ptr %4, align 8
  %5 = getelementptr inbounds [2 x %test], ptr %1, i64 0, i64 0
  store ptr %3, ptr %5, align 8
  %6 = alloca %test, align 8
  %7 = getelementptr inbounds %test, ptr %6, i32 0, i32 0
  store i64 2, ptr %7, align 8
  %8 = getelementptr inbounds [2 x %test], ptr %1, i64 0, i64 1
  store ptr %6, ptr %8, align 8
  %9 = getelementptr inbounds %array, ptr %0, i32 0, i32 1
  store ptr %1, ptr %9, align 8
  %10 = getelementptr inbounds %array, ptr %0, i32 0, i32 1
  %11 = load ptr, ptr %10, align 8
  %12 = getelementptr inbounds [1 x %test], ptr %11, i64 0, i64 0
  %13 = load ptr, ptr %12, align 8
  %14 = getelementptr inbounds %test, ptr %13, i32 0, i32 0
  %15 = load i64, ptr %14, align 8
  ret void
}
`,
		},
		{
			name:  "nested i64 array initialisation",
			input: `lib main pub fn main() { let t = test() } fn test() i64 { let a = [[1,2], [3,4]] let add = a[0][0] + a[1][1] return add }`,
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%array = type { i64, ptr }

define fastcc void @main() {
main_entry:
  %0 = call i64 @main.test()
  ret void
}

define internal fastcc i64 @main.test() {
test_entry:
  %0 = alloca %array, align 8
  %1 = alloca [2 x %array], align 8
  %2 = getelementptr inbounds %array, ptr %0, i32 0, i32 0
  store i64 2, ptr %2, align 8
  %3 = alloca %array, align 8
  %4 = alloca [2 x i64], align 8
  %5 = getelementptr inbounds %array, ptr %3, i32 0, i32 0
  store i64 2, ptr %5, align 8
  %6 = getelementptr inbounds [2 x i64], ptr %4, i64 0, i64 0
  store i64 1, ptr %6, align 8
  %7 = getelementptr inbounds [2 x i64], ptr %4, i64 0, i64 1
  store i64 2, ptr %7, align 8
  %8 = getelementptr inbounds %array, ptr %3, i32 0, i32 1
  store ptr %4, ptr %8, align 8
  %9 = getelementptr inbounds [2 x %array], ptr %1, i64 0, i64 0
  store ptr %3, ptr %9, align 8
  %10 = alloca %array, align 8
  %11 = alloca [2 x i64], align 8
  %12 = getelementptr inbounds %array, ptr %10, i32 0, i32 0
  store i64 2, ptr %12, align 8
  %13 = getelementptr inbounds [2 x i64], ptr %11, i64 0, i64 0
  store i64 3, ptr %13, align 8
  %14 = getelementptr inbounds [2 x i64], ptr %11, i64 0, i64 1
  store i64 4, ptr %14, align 8
  %15 = getelementptr inbounds %array, ptr %10, i32 0, i32 1
  store ptr %11, ptr %15, align 8
  %16 = getelementptr inbounds [2 x %array], ptr %1, i64 0, i64 1
  store ptr %10, ptr %16, align 8
  %17 = getelementptr inbounds %array, ptr %0, i32 0, i32 1
  store ptr %1, ptr %17, align 8
  %18 = getelementptr inbounds %array, ptr %0, i32 0, i32 1
  %19 = load ptr, ptr %18, align 8
  %20 = getelementptr inbounds [1 x %array], ptr %19, i64 0, i64 0
  %21 = load ptr, ptr %20, align 8
  %22 = getelementptr inbounds %array, ptr %21, i32 0, i32 1
  %23 = load ptr, ptr %22, align 8
  %24 = getelementptr inbounds [1 x i64], ptr %23, i64 0, i64 0
  %25 = load i64, ptr %24, align 8
  %26 = getelementptr inbounds %array, ptr %0, i32 0, i32 1
  %27 = load ptr, ptr %26, align 8
  %28 = getelementptr inbounds [1 x %array], ptr %27, i64 0, i64 1
  %29 = load ptr, ptr %28, align 8
  %30 = getelementptr inbounds %array, ptr %29, i32 0, i32 1
  %31 = load ptr, ptr %30, align 8
  %32 = getelementptr inbounds [1 x i64], ptr %31, i64 0, i64 1
  %33 = load i64, ptr %32, align 8
  %add = add i64 %25, %33
  ret i64 %add
}
`,
		},
		{
			name:  "1d slice",
			input: "lib main pub fn main() { let a = [1,2,3,4] let b = a[1:4]}",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%array = type { i64, ptr }

define fastcc void @main() {
main_entry:
  %0 = alloca %array, align 8
  %1 = alloca [4 x i64], align 8
  %2 = getelementptr inbounds %array, ptr %0, i32 0, i32 0
  store i64 4, ptr %2, align 8
  %3 = getelementptr inbounds [4 x i64], ptr %1, i64 0, i64 0
  store i64 1, ptr %3, align 8
  %4 = getelementptr inbounds [4 x i64], ptr %1, i64 0, i64 1
  store i64 2, ptr %4, align 8
  %5 = getelementptr inbounds [4 x i64], ptr %1, i64 0, i64 2
  store i64 3, ptr %5, align 8
  %6 = getelementptr inbounds [4 x i64], ptr %1, i64 0, i64 3
  store i64 4, ptr %6, align 8
  %7 = getelementptr inbounds %array, ptr %0, i32 0, i32 1
  store ptr %1, ptr %7, align 8
  %8 = getelementptr inbounds %array, ptr %0, i32 0, i32 1
  %9 = load ptr, ptr %8, align 8
  %10 = getelementptr inbounds [1 x i64], ptr %9, i64 1
  %11 = alloca %array, align 8
  %12 = getelementptr inbounds %array, ptr %11, i32 0, i32 0
  store i64 3, ptr %12, align 8
  %13 = getelementptr inbounds %array, ptr %11, i32 0, i32 1
  store ptr %10, ptr %13, align 8
  ret void
}
`,
		},
		{
			name:  "1d slice with index",
			input: "lib main pub fn main() { let a = [1,2,3,4] let a' = a[1:3] let val = a'[0] }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%array = type { i64, ptr }

define fastcc void @main() {
main_entry:
  %0 = alloca %array, align 8
  %1 = alloca [4 x i64], align 8
  %2 = getelementptr inbounds %array, ptr %0, i32 0, i32 0
  store i64 4, ptr %2, align 8
  %3 = getelementptr inbounds [4 x i64], ptr %1, i64 0, i64 0
  store i64 1, ptr %3, align 8
  %4 = getelementptr inbounds [4 x i64], ptr %1, i64 0, i64 1
  store i64 2, ptr %4, align 8
  %5 = getelementptr inbounds [4 x i64], ptr %1, i64 0, i64 2
  store i64 3, ptr %5, align 8
  %6 = getelementptr inbounds [4 x i64], ptr %1, i64 0, i64 3
  store i64 4, ptr %6, align 8
  %7 = getelementptr inbounds %array, ptr %0, i32 0, i32 1
  store ptr %1, ptr %7, align 8
  %8 = getelementptr inbounds %array, ptr %0, i32 0, i32 1
  %9 = load ptr, ptr %8, align 8
  %10 = getelementptr inbounds [1 x i64], ptr %9, i64 1
  %11 = alloca %array, align 8
  %12 = getelementptr inbounds %array, ptr %11, i32 0, i32 0
  store i64 2, ptr %12, align 8
  %13 = getelementptr inbounds %array, ptr %11, i32 0, i32 1
  store ptr %10, ptr %13, align 8
  %14 = getelementptr inbounds %array, ptr %11, i32 0, i32 1
  %15 = load ptr, ptr %14, align 8
  %16 = getelementptr inbounds [1 x i64], ptr %15, i64 0, i64 0
  %17 = load i64, ptr %16, align 8
  ret void
}
`,
		},
		{
			name: "assign expression to array",
			input: `lib main fn main() { test(1) }
			fn test(n i64) {
				let arr = make([]byte, 1) 
				use arr {
					arr[0] = byte((n % 10) + '0')
				}
			}`,
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%array = type { i64, ptr }

define fastcc void @main() {
main_entry:
  call void @main.test(i64 1)
  ret void
}

define internal fastcc void @main.test(i64 %0) {
test_entry:
  %1 = alloca i8, i64 1, align 4
  %2 = alloca %array, align 8
  %3 = alloca i64, align 8
  store i64 0, ptr %3, align 8
  %4 = getelementptr inbounds %array, ptr %2, i32 0, i32 0
  store i64 1, ptr %4, align 8
  %5 = getelementptr inbounds %array, ptr %2, i32 0, i32 1
  store ptr %1, ptr %5, align 8
  br label %6

6:                                                ; preds = %11, %test_entry
  %7 = load i64, ptr %3, align 8
  %8 = icmp ult i64 %7, 1
  br i1 %8, label %9, label %13

9:                                                ; preds = %6
  %10 = getelementptr inbounds [1 x i8], ptr %1, i64 0, i64 %7
  store i8 0, ptr %10, align 1
  br label %11

11:                                               ; preds = %9
  %12 = add i64 %7, 1
  store i64 %12, ptr %3, align 8
  br label %6

13:                                               ; preds = %6
  %srem = srem i64 %0, 10
  %add = add i64 %srem, 48
  %14 = trunc i64 %add to i8
  %15 = getelementptr inbounds %array, ptr %2, i32 0, i32 1
  %16 = load ptr, ptr %15, align 8
  %17 = getelementptr inbounds [1 x i8], ptr %16, i64 0, i64 0
  store i8 %14, ptr %17, align 1
  ret void
}
`,
		},
		{
			name: "assign char literal to byte array",
			input: `lib main fn main() {
    let arr = make([]byte, 1)      
    use arr {
		arr[0] = '0'
    }
}`,
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%array = type { i64, ptr }

define fastcc void @main() {
main_entry:
  %0 = alloca i8, i64 1, align 4
  %1 = alloca %array, align 8
  %2 = alloca i64, align 8
  store i64 0, ptr %2, align 8
  %3 = getelementptr inbounds %array, ptr %1, i32 0, i32 0
  store i64 1, ptr %3, align 8
  %4 = getelementptr inbounds %array, ptr %1, i32 0, i32 1
  store ptr %0, ptr %4, align 8
  br label %5

5:                                                ; preds = %10, %main_entry
  %6 = load i64, ptr %2, align 8
  %7 = icmp ult i64 %6, 1
  br i1 %7, label %8, label %12

8:                                                ; preds = %5
  %9 = getelementptr inbounds [1 x i8], ptr %0, i64 0, i64 %6
  store i8 0, ptr %9, align 1
  br label %10

10:                                               ; preds = %8
  %11 = add i64 %6, 1
  store i64 %11, ptr %2, align 8
  br label %5

12:                                               ; preds = %5
  %13 = getelementptr inbounds %array, ptr %1, i32 0, i32 1
  %14 = load ptr, ptr %13, align 8
  %15 = getelementptr inbounds [1 x i8], ptr %14, i64 0, i64 0
  store i8 48, ptr %15, align 1
  ret void
}
`,
		},
		// TODO:
		// {
		// 	name:  "index of slice expression",
		// 	input: "lib main pub fn main() { a = [1] val = a[0:1][0] }",
		// },
	}
	runTests(t, tests)
}

func TestForLoop(t *testing.T) {
	tests := []testCase{
		{
			name:  "for loop - simple",
			input: `lib main pub fn main() { for i = 0; i < 10; i++ { } }`,
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

define fastcc void @main() {
main_entry:
  %0 = alloca i64, align 8
  store i64 0, ptr %0, align 8
  br label %1

1:                                                ; preds = %4, %main_entry
  %2 = load i64, ptr %0, align 8
  %lt = icmp slt i64 %2, 10
  br i1 %lt, label %3, label %7

3:                                                ; preds = %1
  br label %4

4:                                                ; preds = %3
  %5 = load i64, ptr %0, align 8
  %6 = add i64 %5, 1
  store i64 %6, ptr %0, align 8
  br label %1

7:                                                ; preds = %1
  ret void
}
`,
		},
		{
			name:    "infinite",
			skipRun: true,
			input:   "lib main fn main () { for { } }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

define fastcc void @main() {
main_entry:
  br label %0

0:                                                ; preds = %0, %main_entry
  br label %0

1:                                                ; No predecessors!
  ret void
}
`,
		},
		{
			name:  "boolean",
			input: "lib main pub fn main() { let x = 2 for x < 0 { } }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

define fastcc void @main() {
main_entry:
  br label %0

0:                                                ; preds = %2, %main_entry
  br i1 false, label %1, label %3

1:                                                ; preds = %0
  br label %2

2:                                                ; preds = %1
  br label %0

3:                                                ; preds = %0
  ret void
}
`,
		},
		{
			name:  "for loop - iterate over array",
			input: `lib main pub fn main() { let arr = [1,2,3] for i = 0; i < len(arr); i++ { let a = arr[i] } }`,
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%array = type { i64, ptr }

define fastcc void @main() {
main_entry:
  %0 = alloca %array, align 8
  %1 = alloca [3 x i64], align 8
  %2 = getelementptr inbounds %array, ptr %0, i32 0, i32 0
  store i64 3, ptr %2, align 8
  %3 = getelementptr inbounds [3 x i64], ptr %1, i64 0, i64 0
  store i64 1, ptr %3, align 8
  %4 = getelementptr inbounds [3 x i64], ptr %1, i64 0, i64 1
  store i64 2, ptr %4, align 8
  %5 = getelementptr inbounds [3 x i64], ptr %1, i64 0, i64 2
  store i64 3, ptr %5, align 8
  %6 = getelementptr inbounds %array, ptr %0, i32 0, i32 1
  store ptr %1, ptr %6, align 8
  %7 = alloca i64, align 8
  store i64 0, ptr %7, align 8
  br label %8

8:                                                ; preds = %18, %main_entry
  %9 = load i64, ptr %7, align 8
  %10 = getelementptr inbounds %array, ptr %0, i32 0, i32 0
  %11 = load i64, ptr %10, align 8
  %lt = icmp slt i64 %9, %11
  br i1 %lt, label %12, label %21

12:                                               ; preds = %8
  %13 = getelementptr inbounds %array, ptr %0, i32 0, i32 1
  %14 = load ptr, ptr %13, align 8
  %15 = load i64, ptr %7, align 8
  %16 = getelementptr inbounds [1 x i64], ptr %14, i64 0, i64 %15
  %17 = load i64, ptr %16, align 8
  br label %18

18:                                               ; preds = %12
  %19 = load i64, ptr %7, align 8
  %20 = add i64 %19, 1
  store i64 %20, ptr %7, align 8
  br label %8

21:                                               ; preds = %8
  ret void
}
`,
		},
		{
			name:  "for loop - with conditional break",
			input: `lib main pub fn main() { for i = 0; i < 10; i++ { if i == 2 {  break } } }`,
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

define fastcc void @main() {
main_entry:
  %0 = alloca i64, align 8
  store i64 0, ptr %0, align 8
  br label %1

1:                                                ; preds = %5, %main_entry
  %2 = load i64, ptr %0, align 8
  %lt = icmp slt i64 %2, 10
  br i1 %lt, label %3, label %8

3:                                                ; preds = %1
  %4 = load i64, ptr %0, align 8
  %eq = icmp eq i64 %4, 2
  br i1 %eq, label %9, label %10

5:                                                ; preds = %10
  %6 = load i64, ptr %0, align 8
  %7 = add i64 %6, 1
  store i64 %7, ptr %0, align 8
  br label %1

8:                                                ; preds = %9, %1
  ret void

9:                                                ; preds = %3
  br label %8

10:                                               ; preds = %3
  br label %5
}
`,
		},
		{
			name:  "for loop - with conditional next",
			input: `lib main pub fn main() { for i = 0; i < 10; i++ { if i == 3 { next } } }`,
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

define fastcc void @main() {
main_entry:
  %0 = alloca i64, align 8
  store i64 0, ptr %0, align 8
  br label %1

1:                                                ; preds = %5, %main_entry
  %2 = load i64, ptr %0, align 8
  %lt = icmp slt i64 %2, 10
  br i1 %lt, label %3, label %8

3:                                                ; preds = %1
  %4 = load i64, ptr %0, align 8
  %eq = icmp eq i64 %4, 3
  br i1 %eq, label %9, label %10

5:                                                ; preds = %10, %9
  %6 = load i64, ptr %0, align 8
  %7 = add i64 %6, 1
  store i64 %7, ptr %0, align 8
  br label %1

8:                                                ; preds = %1
  ret void

9:                                                ; preds = %3
  br label %5

10:                                               ; preds = %3
  br label %5
}
`,
		},
		{
			name:  "classic, with setting var",
			input: "lib main pub fn main() { var x = 0 for i = 10; i > 0; i-- { x = i } }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

define fastcc void @main() {
main_entry:
  %0 = alloca i64, align 8
  store i64 0, ptr %0, align 8
  %1 = alloca i64, align 8
  store i64 10, ptr %1, align 8
  br label %2

2:                                                ; preds = %6, %main_entry
  %3 = load i64, ptr %1, align 8
  %gt = icmp sgt i64 %3, 0
  br i1 %gt, label %4, label %9

4:                                                ; preds = %2
  %5 = load i64, ptr %1, align 8
  store i64 %5, ptr %0, align 8
  br label %6

6:                                                ; preds = %4
  %7 = load i64, ptr %1, align 8
  %8 = sub i64 %7, 1
  store i64 %8, ptr %1, align 8
  br label %2

9:                                                ; preds = %2
  ret void
}
`,
		},
		{
			name:  "boolean loop",
			input: "lib main pub fn main() {var x = 1 for x < 10 { x = x + 1 } }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

define fastcc void @main() {
main_entry:
  %0 = alloca i64, align 8
  store i64 1, ptr %0, align 8
  br label %1

1:                                                ; preds = %5, %main_entry
  %2 = load i64, ptr %0, align 8
  %lt = icmp slt i64 %2, 10
  br i1 %lt, label %3, label %6

3:                                                ; preds = %1
  %4 = load i64, ptr %0, align 8
  %add = add i64 %4, 1
  store i64 %add, ptr %0, align 8
  br label %5

5:                                                ; preds = %3
  br label %1

6:                                                ; preds = %1
  ret void
}
`,
		},
		{
			name:  "return in loop",
			input: "lib main fn main() { var i = 0 for i < 10 { if i == 5 { return } i++ } }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

define fastcc void @main() {
main_entry:
  %0 = alloca i64, align 8
  store i64 0, ptr %0, align 8
  br label %1

1:                                                ; preds = %5, %main_entry
  %2 = load i64, ptr %0, align 8
  %lt = icmp slt i64 %2, 10
  br i1 %lt, label %3, label %6

3:                                                ; preds = %1
  %4 = load i64, ptr %0, align 8
  %eq = icmp eq i64 %4, 5
  br i1 %eq, label %7, label %8

5:                                                ; preds = %8
  br label %1

6:                                                ; preds = %1
  ret void

7:                                                ; preds = %3
  ret void

8:                                                ; preds = %3
  %9 = load i64, ptr %0, align 8
  %10 = add i64 %9, 1
  store i64 %10, ptr %0, align 8
  br label %5
}
`,
		},
		// TODO: add custom increment
		// {
		// 	name:  "for loop - custom increment",
		// 	input: `lib main pub fn main() { for i = 0; i < 10; i = i + 2  { } }`,
		// 	want:  ``,
		// },
	}
	runTests(t, tests)
}

func TestCopyExpression(t *testing.T) {
	tests := []testCase{
		{
			name:  "integer",
			input: "lib main fn main() { let x = test(2) } fn test(i i64) i64 { return i^ }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

define fastcc void @main() {
main_entry:
  %0 = call i64 @main.test(i64 2)
  ret void
}

define internal fastcc i64 @main.test(i64 %0) {
test_entry:
  ret i64 %0
}
`,
		},
		{
			name:  "struct",
			input: "lib main struct abc { a i64, b f64 } fn main() { let s = abc{a: 1, b: 2.3 } let s' = s^}",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%abc = type { i64, double }

define fastcc void @main() {
main_entry:
  %0 = alloca %abc, align 8
  %1 = getelementptr inbounds %abc, ptr %0, i32 0, i32 0
  store i64 1, ptr %1, align 8
  %2 = getelementptr inbounds %abc, ptr %0, i32 0, i32 1
  store double 2.300000e+00, ptr %2, align 8
  %3 = alloca %abc, align 8
  call void @llvm.memcpy.p0.p0.i64(ptr %3, ptr %0, i64 ptrtoint (ptr getelementptr (%abc, ptr null, i32 1) to i64), i1 false)
  ret void
}

; Function Attrs: nocallback nofree nounwind willreturn memory(argmem: readwrite)
declare void @llvm.memcpy.p0.p0.i64(ptr noalias nocapture writeonly, ptr noalias nocapture readonly, i64, i1 immarg) #0

attributes #0 = { nocallback nofree nounwind willreturn memory(argmem: readwrite) }
`,
		},
		{
			name:  "array",
			input: "lib main fn main() { let a = [1,2,3,4] let a' = a^ }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%array = type { i64, ptr }

define fastcc void @main() {
main_entry:
  %0 = alloca %array, align 8
  %1 = alloca [4 x i64], align 8
  %2 = getelementptr inbounds %array, ptr %0, i32 0, i32 0
  store i64 4, ptr %2, align 8
  %3 = getelementptr inbounds [4 x i64], ptr %1, i64 0, i64 0
  store i64 1, ptr %3, align 8
  %4 = getelementptr inbounds [4 x i64], ptr %1, i64 0, i64 1
  store i64 2, ptr %4, align 8
  %5 = getelementptr inbounds [4 x i64], ptr %1, i64 0, i64 2
  store i64 3, ptr %5, align 8
  %6 = getelementptr inbounds [4 x i64], ptr %1, i64 0, i64 3
  store i64 4, ptr %6, align 8
  %7 = getelementptr inbounds %array, ptr %0, i32 0, i32 1
  store ptr %1, ptr %7, align 8
  %8 = getelementptr inbounds %array, ptr %0, i32 0, i32 0
  %9 = load i64, ptr %8, align 8
  %10 = mul i64 ptrtoint (ptr getelementptr ([1 x i64], ptr null, i32 1) to i64), %9
  %11 = alloca %array, align 8
  %12 = alloca i64, i64 %10, align 8
  %13 = getelementptr inbounds %array, ptr %11, i32 0, i32 0
  store i64 %9, ptr %13, align 8
  %14 = getelementptr inbounds %array, ptr %11, i32 0, i32 1
  store ptr %12, ptr %14, align 8
  %15 = getelementptr inbounds %array, ptr %0, i32 0, i32 1
  %16 = load ptr, ptr %15, align 8
  call void @llvm.memcpy.p0.p0.i64(ptr %12, ptr %16, i64 %10, i1 false)
  ret void
}

; Function Attrs: nocallback nofree nounwind willreturn memory(argmem: readwrite)
declare void @llvm.memcpy.p0.p0.i64(ptr noalias nocapture writeonly, ptr noalias nocapture readonly, i64, i1 immarg) #0

attributes #0 = { nocallback nofree nounwind willreturn memory(argmem: readwrite) }
`,
		},
		// TODO: add tests copying string
	}
	runTests(t, tests)
}

func TestCopyUpdateExpression(t *testing.T) {
	tests := []testCase{
		{
			name:  "update array field",
			input: "lib main pub fn main() { let a = [0,1,2] let b = a^ { b[1] = 1 }}",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%array = type { i64, ptr }

define fastcc void @main() {
main_entry:
  %0 = alloca %array, align 8
  %1 = alloca [3 x i64], align 8
  %2 = getelementptr inbounds %array, ptr %0, i32 0, i32 0
  store i64 3, ptr %2, align 8
  %3 = getelementptr inbounds [3 x i64], ptr %1, i64 0, i64 0
  store i64 0, ptr %3, align 8
  %4 = getelementptr inbounds [3 x i64], ptr %1, i64 0, i64 1
  store i64 1, ptr %4, align 8
  %5 = getelementptr inbounds [3 x i64], ptr %1, i64 0, i64 2
  store i64 2, ptr %5, align 8
  %6 = getelementptr inbounds %array, ptr %0, i32 0, i32 1
  store ptr %1, ptr %6, align 8
  %7 = getelementptr inbounds %array, ptr %0, i32 0, i32 0
  %8 = load i64, ptr %7, align 8
  %9 = mul i64 ptrtoint (ptr getelementptr ([1 x i64], ptr null, i32 1) to i64), %8
  %10 = alloca %array, align 8
  %11 = alloca i64, i64 %9, align 8
  %12 = getelementptr inbounds %array, ptr %10, i32 0, i32 0
  store i64 %8, ptr %12, align 8
  %13 = getelementptr inbounds %array, ptr %10, i32 0, i32 1
  store ptr %11, ptr %13, align 8
  %14 = getelementptr inbounds %array, ptr %0, i32 0, i32 1
  %15 = load ptr, ptr %14, align 8
  call void @llvm.memcpy.p0.p0.i64(ptr %11, ptr %15, i64 %9, i1 false)
  %16 = getelementptr inbounds %array, ptr %10, i32 0, i32 1
  %17 = load ptr, ptr %16, align 8
  %18 = getelementptr inbounds [1 x i64], ptr %17, i64 0, i64 1
  store i64 1, ptr %18, align 8
  ret void
}

; Function Attrs: nocallback nofree nounwind willreturn memory(argmem: readwrite)
declare void @llvm.memcpy.p0.p0.i64(ptr noalias nocapture writeonly, ptr noalias nocapture readonly, i64, i1 immarg) #0

attributes #0 = { nocallback nofree nounwind willreturn memory(argmem: readwrite) }
`,
		},
		{
			name:  "update struct field",
			input: "lib main struct abc {a i64, b f64 } pub fn main() { let a = abc{a: 1, b: 2.3} let b = a^ { b.b = 1.1 } }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%abc = type { i64, double }

define fastcc void @main() {
main_entry:
  %0 = alloca %abc, align 8
  %1 = getelementptr inbounds %abc, ptr %0, i32 0, i32 0
  store i64 1, ptr %1, align 8
  %2 = getelementptr inbounds %abc, ptr %0, i32 0, i32 1
  store double 2.300000e+00, ptr %2, align 8
  %3 = alloca %abc, align 8
  call void @llvm.memcpy.p0.p0.i64(ptr %3, ptr %0, i64 ptrtoint (ptr getelementptr (%abc, ptr null, i32 1) to i64), i1 false)
  %4 = getelementptr inbounds %abc, ptr %3, i32 0, i32 1
  store double 1.100000e+00, ptr %4, align 8
  ret void
}

; Function Attrs: nocallback nofree nounwind willreturn memory(argmem: readwrite)
declare void @llvm.memcpy.p0.p0.i64(ptr noalias nocapture writeonly, ptr noalias nocapture readonly, i64, i1 immarg) #0

attributes #0 = { nocallback nofree nounwind willreturn memory(argmem: readwrite) }
`,
		},
		{
			name:  "update entire struct",
			input: "lib main struct abc {x i64} pub fn main() { let a = abc{x: 1} let b = a^ { b = abc{x: 2} } }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%abc = type { i64 }

define fastcc void @main() {
main_entry:
  %0 = alloca %abc, align 8
  %1 = getelementptr inbounds %abc, ptr %0, i32 0, i32 0
  store i64 1, ptr %1, align 8
  %2 = alloca %abc, align 8
  call void @llvm.memcpy.p0.p0.i64(ptr %2, ptr %0, i64 ptrtoint (ptr getelementptr (%abc, ptr null, i32 1) to i64), i1 false)
  %3 = alloca %abc, align 8
  %4 = getelementptr inbounds %abc, ptr %3, i32 0, i32 0
  store i64 2, ptr %4, align 8
  call void @llvm.memcpy.p0.p0.i64(ptr %2, ptr %3, i64 ptrtoint (ptr getelementptr (%abc, ptr null, i32 1) to i64), i1 false)
  ret void
}

; Function Attrs: nocallback nofree nounwind willreturn memory(argmem: readwrite)
declare void @llvm.memcpy.p0.p0.i64(ptr noalias nocapture writeonly, ptr noalias nocapture readonly, i64, i1 immarg) #0

attributes #0 = { nocallback nofree nounwind willreturn memory(argmem: readwrite) }
`,
		},
		{
			name:  "update array slice",
			input: "lib main pub fn main() { let a = [0,1,2] let b = a^ { b[1:3] = [6,7] } }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%array = type { i64, ptr }

define fastcc void @main() {
main_entry:
  %0 = alloca %array, align 8
  %1 = alloca [3 x i64], align 8
  %2 = getelementptr inbounds %array, ptr %0, i32 0, i32 0
  store i64 3, ptr %2, align 8
  %3 = getelementptr inbounds [3 x i64], ptr %1, i64 0, i64 0
  store i64 0, ptr %3, align 8
  %4 = getelementptr inbounds [3 x i64], ptr %1, i64 0, i64 1
  store i64 1, ptr %4, align 8
  %5 = getelementptr inbounds [3 x i64], ptr %1, i64 0, i64 2
  store i64 2, ptr %5, align 8
  %6 = getelementptr inbounds %array, ptr %0, i32 0, i32 1
  store ptr %1, ptr %6, align 8
  %7 = getelementptr inbounds %array, ptr %0, i32 0, i32 0
  %8 = load i64, ptr %7, align 8
  %9 = mul i64 ptrtoint (ptr getelementptr ([1 x i64], ptr null, i32 1) to i64), %8
  %10 = alloca %array, align 8
  %11 = alloca i64, i64 %9, align 8
  %12 = getelementptr inbounds %array, ptr %10, i32 0, i32 0
  store i64 %8, ptr %12, align 8
  %13 = getelementptr inbounds %array, ptr %10, i32 0, i32 1
  store ptr %11, ptr %13, align 8
  %14 = getelementptr inbounds %array, ptr %0, i32 0, i32 1
  %15 = load ptr, ptr %14, align 8
  call void @llvm.memcpy.p0.p0.i64(ptr %11, ptr %15, i64 %9, i1 false)
  %16 = alloca %array, align 8
  %17 = alloca [2 x i64], align 8
  %18 = getelementptr inbounds %array, ptr %16, i32 0, i32 0
  store i64 2, ptr %18, align 8
  %19 = getelementptr inbounds [2 x i64], ptr %17, i64 0, i64 0
  store i64 6, ptr %19, align 8
  %20 = getelementptr inbounds [2 x i64], ptr %17, i64 0, i64 1
  store i64 7, ptr %20, align 8
  %21 = getelementptr inbounds %array, ptr %16, i32 0, i32 1
  store ptr %17, ptr %21, align 8
  %22 = getelementptr inbounds %array, ptr %10, i32 0, i32 1
  %23 = load ptr, ptr %22, align 8
  %24 = getelementptr inbounds i64, ptr %23, i64 1
  %25 = getelementptr inbounds %array, ptr %16, i32 0, i32 0
  %26 = load i64, ptr %25, align 8
  %27 = getelementptr inbounds %array, ptr %16, i32 0, i32 1
  %28 = load ptr, ptr %27, align 8
  %29 = mul i64 ptrtoint (ptr getelementptr ([1 x i64], ptr null, i32 1) to i64), %26
  call void @llvm.memcpy.p0.p0.i64(ptr %24, ptr %28, i64 %29, i1 false)
  ret void
}

; Function Attrs: nocallback nofree nounwind willreturn memory(argmem: readwrite)
declare void @llvm.memcpy.p0.p0.i64(ptr noalias nocapture writeonly, ptr noalias nocapture readonly, i64, i1 immarg) #0

attributes #0 = { nocallback nofree nounwind willreturn memory(argmem: readwrite) }
`,
		},
	}
	runTests(t, tests)
}

func TestTypeDefinitions(t *testing.T) {
	tests := []testCase{
		{
			name:  "struct - definition, initialisation, access",
			input: "lib main struct abc { a i64 } type custom abc fn main() { let c = custom{a: 1} let val = c.a }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%custom = type { i64 }

define fastcc void @main() {
main_entry:
  %0 = alloca %custom, align 8
  %1 = getelementptr inbounds %custom, ptr %0, i32 0, i32 0
  store i64 1, ptr %1, align 8
  %2 = getelementptr inbounds %custom, ptr %0, i32 0, i32 0
  %3 = load i64, ptr %2, align 8
  ret void
}
`,
		},
	}
	runTests(t, tests)
}

func TestTypeCast(t *testing.T) {
	tests := []testCase{
		{
			name:  "cast byte and char subtraction to i64",
			input: `lib main fn main() { let s = "1" let n = i64(s[0] - '0') }`,
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%string = type { i64, ptr }

@0 = internal constant [1 x i8] c"1"
@1 = internal constant %string { i64 1, ptr @0 }

define fastcc void @main() {
main_entry:
  %0 = load ptr, ptr getelementptr inbounds (%string, ptr @1, i32 0, i32 1), align 8
  %1 = getelementptr inbounds [1 x i8], ptr %0, i64 0, i64 0
  %2 = load i8, ptr %1, align 1
  %sub = sub i8 %2, 48
  %3 = sext i8 %sub to i64
  ret void
}
`,
		},
		{
			name:  "cast byte to string",
			input: `lib main fn main() { let b = byte(0) let s = string(b) test(s) } fn test(s string) {}`,
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%string = type { i64, ptr }

define fastcc void @main() {
main_entry:
  %0 = alloca %string, align 8
  %1 = alloca [1 x i8], align 4
  %2 = getelementptr inbounds [1 x i8], ptr %1, i64 0, i64 0
  store i8 0, ptr %2, align 1
  %3 = getelementptr inbounds %string, ptr %0, i32 0, i32 0
  store i64 1, ptr %3, align 8
  %4 = getelementptr inbounds %string, ptr %0, i32 0, i32 1
  store ptr %1, ptr %4, align 8
  call void @main.test(ptr %0)
  ret void
}

define internal fastcc void @main.test(ptr %0) {
test_entry:
  ret void
}
`,
		},
	}
	runTests(t, tests)
}

func TestMatch(t *testing.T) {
	tests := []testCase{
		{
			name:  "expression, match by int",
			input: "lib module fn main() { let x = 1 let y = match x { case 1: 0 case _: 1 } }",
			want: `; ModuleID = 'module-ir'
source_filename = "module-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

define fastcc void @main() {
main_entry:
  switch i64 1, label %1 [
    i64 1, label %0
  ]

0:                                                ; preds = %main_entry
  br label %2

1:                                                ; preds = %main_entry
  br label %2

2:                                                ; preds = %1, %0
  %phi = phi i64 [ 0, %0 ], [ 1, %1 ]
  ret void
}
`,
		},
		{
			name:  "expression, match by float",
			input: "lib module fn main() { let y = match 1.1 { case 1.1: 0 case _: 1 } }",
			want: `; ModuleID = 'module-ir'
source_filename = "module-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

define fastcc void @main() {
main_entry:
  switch i64 4607632778762754458, label %1 [
    i64 4607632778762754458, label %0
  ]

0:                                                ; preds = %main_entry
  br label %2

1:                                                ; preds = %main_entry
  br label %2

2:                                                ; preds = %1, %0
  %phi = phi i64 [ 0, %0 ], [ 1, %1 ]
  ret void
}
`,
		},
		{
			name: "statement, return",
			input: `
				lib module
				fn main() {
					let r = test(1)
				}
				fn test(i i64) i64 {
					match i {
					case 1: return -i
					case 2: return i
					}
					return 0
				}`,
			want: `; ModuleID = 'module-ir'
source_filename = "module-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

define fastcc void @main() {
main_entry:
  %0 = call i64 @module.test(i64 1)
  ret void
}

define internal fastcc i64 @module.test(i64 %0) {
test_entry:
  switch i64 %0, label %3 [
    i64 1, label %1
    i64 2, label %2
  ]

1:                                                ; preds = %test_entry
  %neg = sub i64 0, %0
  ret i64 %neg

2:                                                ; preds = %test_entry
  ret i64 %0

3:                                                ; preds = %test_entry
  br label %4

4:                                                ; preds = %3
  ret i64 0
}
`,
		},
		{
			name:  "by enum",
			input: "lib main enum abc{x, y} fn main() { let v = abc.x match v { case abc.x: 1 case abc.y: 2 } }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

define fastcc void @main() {
main_entry:
  switch i8 0, label %2 [
    i8 0, label %0
    i8 1, label %1
  ]

0:                                                ; preds = %main_entry
  br label %3

1:                                                ; preds = %main_entry
  br label %3

2:                                                ; preds = %main_entry
  br label %3

3:                                                ; preds = %2, %1, %0
  ret void
}
`,
		},
		{
			name:  "by union of scalar types",
			input: "lib main union abc { i64, f64 } fn main() { let a = abc(1) let res = match a { case i64: 1 case f64: 2 case _: 0 } }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%main.abc = type { i64, [8 x i8] }

define fastcc void @main() {
main_entry:
  %0 = alloca %main.abc, align 8
  %1 = getelementptr inbounds %main.abc, ptr %0, i32 0, i32 0
  store i64 5234574298831366340, ptr %1, align 8
  %2 = getelementptr inbounds %main.abc, ptr %0, i32 0, i32 1
  store i64 1, ptr %2, align 8
  %3 = getelementptr inbounds %main.abc, ptr %0, i32 0, i32 0
  %4 = load i64, ptr %3, align 8
  switch i64 %4, label %11 [
    i64 5234574298831366340, label %5
    i64 8978123153534397908, label %8
  ]

5:                                                ; preds = %main_entry
  %6 = getelementptr inbounds %main.abc, ptr %0, i32 0, i32 1
  %7 = load i64, ptr %6, align 8
  br label %12

8:                                                ; preds = %main_entry
  %9 = getelementptr inbounds %main.abc, ptr %0, i32 0, i32 1
  %10 = load double, ptr %9, align 8
  br label %12

11:                                               ; preds = %main_entry
  br label %12

12:                                               ; preds = %11, %8, %5
  %phi = phi i64 [ 1, %5 ], [ 2, %8 ], [ 0, %11 ]
  ret void
}
`,
		},
		{
			name:  "by union and use scalar value",
			input: "lib main union a { i64 } fn main() { let v = a(10) let res = match v { case i64: v + 1 case _: 0 } }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%main.a = type { i64, [8 x i8] }

define fastcc void @main() {
main_entry:
  %0 = alloca %main.a, align 8
  %1 = getelementptr inbounds %main.a, ptr %0, i32 0, i32 0
  store i64 -8073169481221830107, ptr %1, align 8
  %2 = getelementptr inbounds %main.a, ptr %0, i32 0, i32 1
  store i64 10, ptr %2, align 8
  %3 = getelementptr inbounds %main.a, ptr %0, i32 0, i32 0
  %4 = load i64, ptr %3, align 8
  switch i64 %4, label %8 [
    i64 -8073169481221830107, label %5
  ]

5:                                                ; preds = %main_entry
  %6 = getelementptr inbounds %main.a, ptr %0, i32 0, i32 1
  %7 = load i64, ptr %6, align 8
  %add = add i64 %7, 1
  br label %9

8:                                                ; preds = %main_entry
  br label %9

9:                                                ; preds = %8, %5
  %phi = phi i64 [ %add, %5 ], [ 0, %8 ]
  ret void
}
`,
		},
		{
			name: "by against union of struct types",
			input: `lib main
					struct a { x bool }
					struct b { y f64 }
					union ab { a, b }
					fn main() {
						let v = ab(a{x: true})
						let res = match v { case a: 1 case b: 0 case _: -1 }
					}`,
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%a = type { i1 }
%main.ab = type { i64, [8 x i8] }

define fastcc void @main() {
main_entry:
  %0 = alloca %a, align 8
  %1 = getelementptr inbounds %a, ptr %0, i32 0, i32 0
  store i1 true, ptr %1, align 1
  %2 = alloca %main.ab, align 8
  %3 = getelementptr inbounds %main.ab, ptr %2, i32 0, i32 0
  store i64 -1983398940155033090, ptr %3, align 8
  %4 = getelementptr inbounds %main.ab, ptr %2, i32 0, i32 1
  call void @llvm.memcpy.p0.p0.i64(ptr %4, ptr %0, i64 ptrtoint (ptr getelementptr (%a, ptr null, i32 1) to i64), i1 false)
  %5 = getelementptr inbounds %main.ab, ptr %2, i32 0, i32 0
  %6 = load i64, ptr %5, align 8
  switch i64 %6, label %11 [
    i64 -1983398940155033090, label %7
    i64 -8075376313706817864, label %9
  ]

7:                                                ; preds = %main_entry
  %8 = getelementptr inbounds %main.ab, ptr %2, i32 0, i32 1
  br label %12

9:                                                ; preds = %main_entry
  %10 = getelementptr inbounds %main.ab, ptr %2, i32 0, i32 1
  br label %12

11:                                               ; preds = %main_entry
  br label %12

12:                                               ; preds = %11, %9, %7
  %phi = phi i64 [ 1, %7 ], [ 0, %9 ], [ -1, %11 ]
  ret void
}

; Function Attrs: nocallback nofree nounwind willreturn memory(argmem: readwrite)
declare void @llvm.memcpy.p0.p0.i64(ptr noalias nocapture writeonly, ptr noalias nocapture readonly, i64, i1 immarg) #0

attributes #0 = { nocallback nofree nounwind willreturn memory(argmem: readwrite) }
`,
		},
		{
			name:  "by union and use struct value",
			input: "lib main struct a { i i64, j i64 } union b { a } fn main() { let v = b(a{i: 3, j: 2}) let res = match v { case a: v.i + v.j case _: 0 }}",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%a = type { i64, i64 }
%main.b = type { i64, [16 x i8] }

define fastcc void @main() {
main_entry:
  %0 = alloca %a, align 8
  %1 = getelementptr inbounds %a, ptr %0, i32 0, i32 0
  store i64 3, ptr %1, align 8
  %2 = getelementptr inbounds %a, ptr %0, i32 0, i32 1
  store i64 2, ptr %2, align 8
  %3 = alloca %main.b, align 8
  %4 = getelementptr inbounds %main.b, ptr %3, i32 0, i32 0
  store i64 -1885140172373873918, ptr %4, align 8
  %5 = getelementptr inbounds %main.b, ptr %3, i32 0, i32 1
  call void @llvm.memcpy.p0.p0.i64(ptr %5, ptr %0, i64 ptrtoint (ptr getelementptr (%a, ptr null, i32 1) to i64), i1 false)
  %6 = getelementptr inbounds %main.b, ptr %3, i32 0, i32 0
  %7 = load i64, ptr %6, align 8
  switch i64 %7, label %14 [
    i64 -1885140172373873918, label %8
  ]

8:                                                ; preds = %main_entry
  %9 = getelementptr inbounds %main.b, ptr %3, i32 0, i32 1
  %10 = getelementptr inbounds %a, ptr %9, i32 0, i32 0
  %11 = load i64, ptr %10, align 8
  %12 = getelementptr inbounds %a, ptr %9, i32 0, i32 1
  %13 = load i64, ptr %12, align 8
  %add = add i64 %11, %13
  br label %15

14:                                               ; preds = %main_entry
  br label %15

15:                                               ; preds = %14, %8
  %phi = phi i64 [ %add, %8 ], [ 0, %14 ]
  ret void
}

; Function Attrs: nocallback nofree nounwind willreturn memory(argmem: readwrite)
declare void @llvm.memcpy.p0.p0.i64(ptr noalias nocapture writeonly, ptr noalias nocapture readonly, i64, i1 immarg) #0

attributes #0 = { nocallback nofree nounwind willreturn memory(argmem: readwrite) }
`,
		},
		{
			name:  "match byte with char",
			input: "lib main fn main() { let b = byte(0) match b { case '0': case _: } }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

define fastcc void @main() {
main_entry:
  switch i8 0, label %1 [
    i8 48, label %0
  ]

0:                                                ; preds = %main_entry
  br label %2

1:                                                ; preds = %main_entry
  br label %2

2:                                                ; preds = %1, %0
  ret void
}
`,
		},
		{
			name:  "reassignment in match",
			input: "lib main fn main() { test(1) } fn test(i i64) { var b = false match i { case _: b = true } let v = b }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

define fastcc void @main() {
main_entry:
  call void @main.test(i64 1)
  ret void
}

define internal fastcc void @main.test(i64 %0) {
test_entry:
  %1 = alloca i1, align 1
  store i1 false, ptr %1, align 1
  switch i64 %0, label %2 [
  ]

2:                                                ; preds = %test_entry
  store i1 true, ptr %1, align 1
  br label %3

3:                                                ; preds = %2
  %4 = load i1, ptr %1, align 1
  ret void
}
`,
		},
	}
	runTests(t, tests)
}

func TestBuiltInFn(t *testing.T) {
	tests := []testCase{
		// 		{
		// 			name:  "printf",
		// 			input: `lib main pub fn main() { printf("%d\n", 0) }`,
		// 			want: `; ModuleID = 'main-ir'
		// source_filename = "main-ir"

		// @main.str = private unnamed_addr constant [5 x i8] c"%d\\n\00", align 1

		// define fastcc void @main() {
		// main_entry:
		//   %0 = call i32 (ptr, ...) @main.printf(ptr @main.str, i64 0)
		//   ret void
		// }

		// declare i32 @main.printf(ptr, ...)
		// `,
		// 		},
		{
			name:  "len - array",
			input: `lib main pub fn main() { let a = [1,2] let l = len(a) }`,
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%array = type { i64, ptr }

define fastcc void @main() {
main_entry:
  %0 = alloca %array, align 8
  %1 = alloca [2 x i64], align 8
  %2 = getelementptr inbounds %array, ptr %0, i32 0, i32 0
  store i64 2, ptr %2, align 8
  %3 = getelementptr inbounds [2 x i64], ptr %1, i64 0, i64 0
  store i64 1, ptr %3, align 8
  %4 = getelementptr inbounds [2 x i64], ptr %1, i64 0, i64 1
  store i64 2, ptr %4, align 8
  %5 = getelementptr inbounds %array, ptr %0, i32 0, i32 1
  store ptr %1, ptr %5, align 8
  %6 = getelementptr inbounds %array, ptr %0, i32 0, i32 0
  %7 = load i64, ptr %6, align 8
  ret void
}
`,
		},
		{
			name:  "cap - array",
			input: `lib main pub fn main() { let a = [1,2] let c = cap(a) }`,
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%array = type { i64, ptr }

define fastcc void @main() {
main_entry:
  %0 = alloca %array, align 8
  %1 = alloca [2 x i64], align 8
  %2 = getelementptr inbounds %array, ptr %0, i32 0, i32 0
  store i64 2, ptr %2, align 8
  %3 = getelementptr inbounds [2 x i64], ptr %1, i64 0, i64 0
  store i64 1, ptr %3, align 8
  %4 = getelementptr inbounds [2 x i64], ptr %1, i64 0, i64 1
  store i64 2, ptr %4, align 8
  %5 = getelementptr inbounds %array, ptr %0, i32 0, i32 1
  store ptr %1, ptr %5, align 8
  ret void
}
`,
		},
		// needs to be seperate function otherwise llvm removes size() call
		{
			name:  "size - struct",
			input: "lib main struct abc {x i64, y i64} pub fn main() { let a = abc{x: 1, y: 1} let s = get_size(a) } fn get_size(s abc) i64 { return size(s) }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%abc = type { i64, i64 }

define fastcc void @main() {
main_entry:
  %0 = alloca %abc, align 8
  %1 = getelementptr inbounds %abc, ptr %0, i32 0, i32 0
  store i64 1, ptr %1, align 8
  %2 = getelementptr inbounds %abc, ptr %0, i32 0, i32 1
  store i64 1, ptr %2, align 8
  %3 = call i64 @main.get_size(ptr %0)
  ret void
}

define internal fastcc i64 @main.get_size(ptr %0) {
get_size_entry:
  ret i64 ptrtoint (ptr getelementptr (%abc, ptr null, i32 1) to i64)
}
`,
		},
		{
			name:  "size - struct literal",
			input: "lib main struct abc {x i64, y i64} pub fn main() { let s = get_size(abc{x: 1, y: 1}) } fn get_size(a abc) i64 { return size(a) }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%abc = type { i64, i64 }

define fastcc void @main() {
main_entry:
  %0 = alloca %abc, align 8
  %1 = getelementptr inbounds %abc, ptr %0, i32 0, i32 0
  store i64 1, ptr %1, align 8
  %2 = getelementptr inbounds %abc, ptr %0, i32 0, i32 1
  store i64 1, ptr %2, align 8
  %3 = call i64 @main.get_size(ptr %0)
  ret void
}

define internal fastcc i64 @main.get_size(ptr %0) {
get_size_entry:
  ret i64 ptrtoint (ptr getelementptr (%abc, ptr null, i32 1) to i64)
}
`,
		},
		// TODO: fix getting size of literal instead of variable
		{
			name:  "size - array",
			input: "lib main pub fn main() { let a = [1,2] let s = get_size(a) } fn get_size(a []i64) i64 { return size(a) }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%array = type { i64, ptr }

define fastcc void @main() {
main_entry:
  %0 = alloca %array, align 8
  %1 = alloca [2 x i64], align 8
  %2 = getelementptr inbounds %array, ptr %0, i32 0, i32 0
  store i64 2, ptr %2, align 8
  %3 = getelementptr inbounds [2 x i64], ptr %1, i64 0, i64 0
  store i64 1, ptr %3, align 8
  %4 = getelementptr inbounds [2 x i64], ptr %1, i64 0, i64 1
  store i64 2, ptr %4, align 8
  %5 = getelementptr inbounds %array, ptr %0, i32 0, i32 1
  store ptr %1, ptr %5, align 8
  %6 = call i64 @main.get_size(ptr %0)
  ret void
}

define internal fastcc i64 @main.get_size(ptr %0) {
get_size_entry:
  %1 = getelementptr inbounds %array, ptr %0, i32 0, i32 0
  %2 = load i64, ptr %1, align 8
  %3 = mul i64 ptrtoint (ptr getelementptr ([1 x i64], ptr null, i32 1) to i64), %2
  ret i64 %3
}
`,
		},
		{
			name:  "size - i64",
			input: "lib main pub fn main() { let a = 1 let s = get_size(a) } fn get_size(a i64) i64 { return size(a) }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

define fastcc void @main() {
main_entry:
  %0 = call i64 @main.get_size(i64 1)
  ret void
}

define internal fastcc i64 @main.get_size(i64 %0) {
get_size_entry:
  ret i64 ptrtoint (ptr getelementptr (i64, ptr null, i32 1) to i64)
}
`,
		},
		{
			name:  "make - []i64",
			input: "lib main fn main() { let arr = make([]i64,10) }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%array = type { i64, ptr }

define fastcc void @main() {
main_entry:
  %0 = alloca i64, i64 10, align 8
  %1 = alloca %array, align 8
  %2 = alloca i64, align 8
  store i64 0, ptr %2, align 8
  %3 = getelementptr inbounds %array, ptr %1, i32 0, i32 0
  store i64 10, ptr %3, align 8
  %4 = getelementptr inbounds %array, ptr %1, i32 0, i32 1
  store ptr %0, ptr %4, align 8
  br label %5

5:                                                ; preds = %10, %main_entry
  %6 = load i64, ptr %2, align 8
  %7 = icmp ult i64 %6, 10
  br i1 %7, label %8, label %12

8:                                                ; preds = %5
  %9 = getelementptr inbounds [1 x i64], ptr %0, i64 0, i64 %6
  store i64 0, ptr %9, align 8
  br label %10

10:                                               ; preds = %8
  %11 = add i64 %6, 1
  store i64 %11, ptr %2, align 8
  br label %5

12:                                               ; preds = %5
  ret void
}
`,
		},
		{
			name:  "validate, type def",
			input: "lib main type age i64 | age > 18 fn main() { let a = age(17) let valid = test(a) } fn test(a age) bool { return validate(a) }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

define fastcc void @main() {
main_entry:
  %0 = call i1 @main.test(i64 17)
  ret void
}

define internal fastcc i1 @main.test(i64 %0) {
test_entry:
  %1 = call i1 @main.__age(i64 %0)
  ret i1 %1
}

define fastcc i1 @main.__age(i64 %0) {
entry:
  %gt = icmp sgt i64 %0, 18
  %1 = select i1 %gt, i1 true, i1 false
  ret i1 %1
}
`,
		},
		{
			name:  "len of expression",
			input: "lib main struct abc { i []i64 } fn main() { let s = abc{i: [1]} len(s.i) }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%abc = type { %array }
%array = type { i64, ptr }

define fastcc void @main() {
main_entry:
  %0 = alloca %abc, align 8
  %1 = alloca %array, align 8
  %2 = alloca [1 x i64], align 8
  %3 = getelementptr inbounds %array, ptr %1, i32 0, i32 0
  store i64 1, ptr %3, align 8
  %4 = getelementptr inbounds [1 x i64], ptr %2, i64 0, i64 0
  store i64 1, ptr %4, align 8
  %5 = getelementptr inbounds %array, ptr %1, i32 0, i32 1
  store ptr %2, ptr %5, align 8
  %6 = getelementptr inbounds %abc, ptr %0, i32 0, i32 0
  store ptr %1, ptr %6, align 8
  %7 = getelementptr inbounds %abc, ptr %0, i32 0, i32 0
  %8 = load ptr, ptr %7, align 8
  %9 = getelementptr inbounds %array, ptr %8, i32 0, i32 0
  %10 = load i64, ptr %9, align 8
  ret void
}
`,
		},
	}
	runTests(t, tests)
}

func TestUseExpression(t *testing.T) {
	tests := []testCase{
		{
			name:  "make and use",
			input: "lib main fn main() { let arr = make([]i64,10) let arr' = use arr { arr[0] = 1 } }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%array = type { i64, ptr }

define fastcc void @main() {
main_entry:
  %0 = alloca i64, i64 10, align 8
  %1 = alloca %array, align 8
  %2 = alloca i64, align 8
  store i64 0, ptr %2, align 8
  %3 = getelementptr inbounds %array, ptr %1, i32 0, i32 0
  store i64 10, ptr %3, align 8
  %4 = getelementptr inbounds %array, ptr %1, i32 0, i32 1
  store ptr %0, ptr %4, align 8
  br label %5

5:                                                ; preds = %10, %main_entry
  %6 = load i64, ptr %2, align 8
  %7 = icmp ult i64 %6, 10
  br i1 %7, label %8, label %12

8:                                                ; preds = %5
  %9 = getelementptr inbounds [1 x i64], ptr %0, i64 0, i64 %6
  store i64 0, ptr %9, align 8
  br label %10

10:                                               ; preds = %8
  %11 = add i64 %6, 1
  store i64 %11, ptr %2, align 8
  br label %5

12:                                               ; preds = %5
  %13 = getelementptr inbounds %array, ptr %1, i32 0, i32 1
  %14 = load ptr, ptr %13, align 8
  %15 = getelementptr inbounds [1 x i64], ptr %14, i64 0, i64 0
  store i64 1, ptr %15, align 8
  ret void
}
`,
		},
		{
			name:  "passed and used",
			input: "lib main fn main() { let arr = make([]i64,10) let arr' = test(arr) } fn test(arr memory<[]i64>) []i64 {  return use arr { arr[0] = 1 } }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%array = type { i64, ptr }

define fastcc void @main() {
main_entry:
  %0 = alloca i64, i64 10, align 8
  %1 = alloca %array, align 8
  %2 = alloca i64, align 8
  store i64 0, ptr %2, align 8
  %3 = getelementptr inbounds %array, ptr %1, i32 0, i32 0
  store i64 10, ptr %3, align 8
  %4 = getelementptr inbounds %array, ptr %1, i32 0, i32 1
  store ptr %0, ptr %4, align 8
  br label %5

5:                                                ; preds = %10, %main_entry
  %6 = load i64, ptr %2, align 8
  %7 = icmp ult i64 %6, 10
  br i1 %7, label %8, label %12

8:                                                ; preds = %5
  %9 = getelementptr inbounds [1 x i64], ptr %0, i64 0, i64 %6
  store i64 0, ptr %9, align 8
  br label %10

10:                                               ; preds = %8
  %11 = add i64 %6, 1
  store i64 %11, ptr %2, align 8
  br label %5

12:                                               ; preds = %5
  %13 = call ptr @main.test(ptr %1)
  ret void
}

define internal fastcc ptr @main.test(ptr %0) {
test_entry:
  %1 = getelementptr inbounds %array, ptr %0, i32 0, i32 1
  %2 = load ptr, ptr %1, align 8
  %3 = getelementptr inbounds [1 x i64], ptr %2, i64 0, i64 0
  store i64 1, ptr %3, align 8
  ret ptr %0
}
`,
		},
		{
			name:  "use slice, literal",
			input: "lib main fn main() { let arr = make([]i64,10) let arr' = use arr { arr[0:3] = [1,2,3] } }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%array = type { i64, ptr }

define fastcc void @main() {
main_entry:
  %0 = alloca i64, i64 10, align 8
  %1 = alloca %array, align 8
  %2 = alloca i64, align 8
  store i64 0, ptr %2, align 8
  %3 = getelementptr inbounds %array, ptr %1, i32 0, i32 0
  store i64 10, ptr %3, align 8
  %4 = getelementptr inbounds %array, ptr %1, i32 0, i32 1
  store ptr %0, ptr %4, align 8
  %5 = alloca %array, align 8
  %6 = alloca [3 x i64], align 8
  br label %7

7:                                                ; preds = %12, %main_entry
  %8 = load i64, ptr %2, align 8
  %9 = icmp ult i64 %8, 10
  br i1 %9, label %10, label %14

10:                                               ; preds = %7
  %11 = getelementptr inbounds [1 x i64], ptr %0, i64 0, i64 %8
  store i64 0, ptr %11, align 8
  br label %12

12:                                               ; preds = %10
  %13 = add i64 %8, 1
  store i64 %13, ptr %2, align 8
  br label %7

14:                                               ; preds = %7
  %15 = getelementptr inbounds %array, ptr %5, i32 0, i32 0
  store i64 3, ptr %15, align 8
  %16 = getelementptr inbounds [3 x i64], ptr %6, i64 0, i64 0
  store i64 1, ptr %16, align 8
  %17 = getelementptr inbounds [3 x i64], ptr %6, i64 0, i64 1
  store i64 2, ptr %17, align 8
  %18 = getelementptr inbounds [3 x i64], ptr %6, i64 0, i64 2
  store i64 3, ptr %18, align 8
  %19 = getelementptr inbounds %array, ptr %5, i32 0, i32 1
  store ptr %6, ptr %19, align 8
  %20 = getelementptr inbounds %array, ptr %1, i32 0, i32 1
  %21 = load ptr, ptr %20, align 8
  %22 = getelementptr inbounds i64, ptr %21, i64 0
  %23 = getelementptr inbounds %array, ptr %5, i32 0, i32 0
  %24 = load i64, ptr %23, align 8
  %25 = getelementptr inbounds %array, ptr %5, i32 0, i32 1
  %26 = load ptr, ptr %25, align 8
  %27 = mul i64 ptrtoint (ptr getelementptr ([1 x i64], ptr null, i32 1) to i64), %24
  call void @llvm.memcpy.p0.p0.i64(ptr %22, ptr %26, i64 %27, i1 false)
  ret void
}

; Function Attrs: nocallback nofree nounwind willreturn memory(argmem: readwrite)
declare void @llvm.memcpy.p0.p0.i64(ptr noalias nocapture writeonly, ptr noalias nocapture readonly, i64, i1 immarg) #0

attributes #0 = { nocallback nofree nounwind willreturn memory(argmem: readwrite) }
`,
		},
		{
			name:  "use slice, variable",
			input: "lib main fn main() { let x = [1,2,3] let arr = make([]i64,10) let arr' = use arr { arr[0:3] = x } }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%array = type { i64, ptr }

define fastcc void @main() {
main_entry:
  %0 = alloca %array, align 8
  %1 = alloca [3 x i64], align 8
  %2 = getelementptr inbounds %array, ptr %0, i32 0, i32 0
  store i64 3, ptr %2, align 8
  %3 = getelementptr inbounds [3 x i64], ptr %1, i64 0, i64 0
  store i64 1, ptr %3, align 8
  %4 = getelementptr inbounds [3 x i64], ptr %1, i64 0, i64 1
  store i64 2, ptr %4, align 8
  %5 = getelementptr inbounds [3 x i64], ptr %1, i64 0, i64 2
  store i64 3, ptr %5, align 8
  %6 = getelementptr inbounds %array, ptr %0, i32 0, i32 1
  store ptr %1, ptr %6, align 8
  %7 = alloca i64, i64 10, align 8
  %8 = alloca %array, align 8
  %9 = alloca i64, align 8
  store i64 0, ptr %9, align 8
  %10 = getelementptr inbounds %array, ptr %8, i32 0, i32 0
  store i64 10, ptr %10, align 8
  %11 = getelementptr inbounds %array, ptr %8, i32 0, i32 1
  store ptr %7, ptr %11, align 8
  br label %12

12:                                               ; preds = %17, %main_entry
  %13 = load i64, ptr %9, align 8
  %14 = icmp ult i64 %13, 10
  br i1 %14, label %15, label %19

15:                                               ; preds = %12
  %16 = getelementptr inbounds [1 x i64], ptr %7, i64 0, i64 %13
  store i64 0, ptr %16, align 8
  br label %17

17:                                               ; preds = %15
  %18 = add i64 %13, 1
  store i64 %18, ptr %9, align 8
  br label %12

19:                                               ; preds = %12
  %20 = getelementptr inbounds %array, ptr %8, i32 0, i32 1
  %21 = load ptr, ptr %20, align 8
  %22 = getelementptr inbounds i64, ptr %21, i64 0
  %23 = getelementptr inbounds %array, ptr %0, i32 0, i32 0
  %24 = load i64, ptr %23, align 8
  %25 = getelementptr inbounds %array, ptr %0, i32 0, i32 1
  %26 = load ptr, ptr %25, align 8
  %27 = mul i64 ptrtoint (ptr getelementptr ([1 x i64], ptr null, i32 1) to i64), %24
  call void @llvm.memcpy.p0.p0.i64(ptr %22, ptr %26, i64 %27, i1 false)
  ret void
}

; Function Attrs: nocallback nofree nounwind willreturn memory(argmem: readwrite)
declare void @llvm.memcpy.p0.p0.i64(ptr noalias nocapture writeonly, ptr noalias nocapture readonly, i64, i1 immarg) #0

attributes #0 = { nocallback nofree nounwind willreturn memory(argmem: readwrite) }
`,
		},
		{
			name:    "use, full assign",
			skipRun: true,
			input:   "lib main fn main() { let x = [1,2,3] let arr = make([]i64,3) let arr' = use arr { arr = x } }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%array = type { i64, ptr }

define fastcc void @main() {
main_entry:
  %0 = alloca %array, align 8
  %1 = alloca [3 x i64], align 8
  %2 = getelementptr inbounds %array, ptr %0, i32 0, i32 0
  store i64 3, ptr %2, align 8
  %3 = getelementptr inbounds [3 x i64], ptr %1, i64 0, i64 0
  store i64 1, ptr %3, align 8
  %4 = getelementptr inbounds [3 x i64], ptr %1, i64 0, i64 1
  store i64 2, ptr %4, align 8
  %5 = getelementptr inbounds [3 x i64], ptr %1, i64 0, i64 2
  store i64 3, ptr %5, align 8
  %6 = getelementptr inbounds %array, ptr %0, i32 0, i32 1
  store ptr %1, ptr %6, align 8
  %7 = alloca i64, i64 3, align 8
  %8 = alloca %array, align 8
  %9 = alloca i64, align 8
  store i64 0, ptr %9, align 8
  %10 = getelementptr inbounds %array, ptr %8, i32 0, i32 0
  store i64 3, ptr %10, align 8
  %11 = getelementptr inbounds %array, ptr %8, i32 0, i32 1
  store ptr %7, ptr %11, align 8
  br label %12

12:                                               ; preds = %17, %main_entry
  %13 = load i64, ptr %9, align 8
  %14 = icmp ult i64 %13, 3
  br i1 %14, label %15, label %19

15:                                               ; preds = %12
  %16 = getelementptr inbounds [1 x i64], ptr %7, i64 0, i64 %13
  store i64 0, ptr %16, align 8
  br label %17

17:                                               ; preds = %15
  %18 = add i64 %13, 1
  store i64 %18, ptr %9, align 8
  br label %12

19:                                               ; preds = %12
  %20 = getelementptr inbounds %array, ptr %0, i32 0, i32 0
  %21 = load i64, ptr %20, align 8
  %22 = mul i64 ptrtoint (ptr getelementptr (i64, ptr null, i32 1) to i64), %21
  call void @llvm.memcpy.p0.p0.i64(ptr %8, ptr %0, i64 %22, i1 false)
  ret void
}

; Function Attrs: nocallback nofree nounwind willreturn memory(argmem: readwrite)
declare void @llvm.memcpy.p0.p0.i64(ptr noalias nocapture writeonly, ptr noalias nocapture readonly, i64, i1 immarg) #0

attributes #0 = { nocallback nofree nounwind willreturn memory(argmem: readwrite) }
`,
		},
		{
			name:  "size of array after use",
			input: "lib main fn main() { let arr = make([]i64,3) let arr' = use arr { arr = [1,2,3] } let s = size(arr') }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%array = type { i64, ptr }

define fastcc void @main() {
main_entry:
  %0 = alloca i64, i64 3, align 8
  %1 = alloca %array, align 8
  %2 = alloca i64, align 8
  store i64 0, ptr %2, align 8
  %3 = getelementptr inbounds %array, ptr %1, i32 0, i32 0
  store i64 3, ptr %3, align 8
  %4 = getelementptr inbounds %array, ptr %1, i32 0, i32 1
  store ptr %0, ptr %4, align 8
  %5 = alloca %array, align 8
  %6 = alloca [3 x i64], align 8
  br label %7

7:                                                ; preds = %12, %main_entry
  %8 = load i64, ptr %2, align 8
  %9 = icmp ult i64 %8, 3
  br i1 %9, label %10, label %14

10:                                               ; preds = %7
  %11 = getelementptr inbounds [1 x i64], ptr %0, i64 0, i64 %8
  store i64 0, ptr %11, align 8
  br label %12

12:                                               ; preds = %10
  %13 = add i64 %8, 1
  store i64 %13, ptr %2, align 8
  br label %7

14:                                               ; preds = %7
  %15 = getelementptr inbounds %array, ptr %5, i32 0, i32 0
  store i64 3, ptr %15, align 8
  %16 = getelementptr inbounds [3 x i64], ptr %6, i64 0, i64 0
  store i64 1, ptr %16, align 8
  %17 = getelementptr inbounds [3 x i64], ptr %6, i64 0, i64 1
  store i64 2, ptr %17, align 8
  %18 = getelementptr inbounds [3 x i64], ptr %6, i64 0, i64 2
  store i64 3, ptr %18, align 8
  %19 = getelementptr inbounds %array, ptr %5, i32 0, i32 1
  store ptr %6, ptr %19, align 8
  %20 = getelementptr inbounds %array, ptr %5, i32 0, i32 0
  %21 = load i64, ptr %20, align 8
  %22 = mul i64 ptrtoint (ptr getelementptr (i64, ptr null, i32 1) to i64), %21
  call void @llvm.memcpy.p0.p0.i64(ptr %1, ptr %5, i64 %22, i1 false)
  %23 = getelementptr inbounds %array, ptr %1, i32 0, i32 0
  %24 = load i64, ptr %23, align 8
  %25 = mul i64 ptrtoint (ptr getelementptr ([1 x i64], ptr null, i32 1) to i64), %24
  ret void
}

; Function Attrs: nocallback nofree nounwind willreturn memory(argmem: readwrite)
declare void @llvm.memcpy.p0.p0.i64(ptr noalias nocapture writeonly, ptr noalias nocapture readonly, i64, i1 immarg) #0

attributes #0 = { nocallback nofree nounwind willreturn memory(argmem: readwrite) }
`,
		},
	}
	runTests(t, tests)
}

func TestFunctions(t *testing.T) {
	tests := []testCase{
		{
			name:  "function - void",
			input: "lib main pub fn main() { test() } fn test() { 1 + 2 }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

define fastcc void @main() {
main_entry:
  call void @main.test()
  ret void
}

define internal fastcc void @main.test() {
test_entry:
  ret void
}
`,
		},
		{
			name:  "function - single arg and return",
			input: "lib main pub fn main() { let t = test(1) } fn test(a i64) i64 { return -a }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

define fastcc void @main() {
main_entry:
  %0 = call i64 @main.test(i64 1)
  ret void
}

define internal fastcc i64 @main.test(i64 %0) {
test_entry:
  %neg = sub i64 0, %0
  ret i64 %neg
}
`,
		},
		{
			name:  "function - pass struct on stack",
			input: `lib main struct A {a i64} pub fn main(){ let t = A{a: 1} let a = test(t) } fn test(t A) i64 { return t.a }`,
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%A = type { i64 }

define fastcc void @main() {
main_entry:
  %0 = alloca %A, align 8
  %1 = getelementptr inbounds %A, ptr %0, i32 0, i32 0
  store i64 1, ptr %1, align 8
  %2 = call i64 @main.test(ptr %0)
  ret void
}

define internal fastcc i64 @main.test(ptr %0) {
test_entry:
  %1 = getelementptr inbounds %A, ptr %0, i32 0, i32 0
  %2 = load i64, ptr %1, align 8
  ret i64 %2
}
`,
		},
		{
			name:  "function - pass array on stack",
			input: `lib main pub fn main() { let arr = [1,2,3] let el = test(arr) } fn test(a []i64) i64 { return a[1] }`,
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%array = type { i64, ptr }

define fastcc void @main() {
main_entry:
  %0 = alloca %array, align 8
  %1 = alloca [3 x i64], align 8
  %2 = getelementptr inbounds %array, ptr %0, i32 0, i32 0
  store i64 3, ptr %2, align 8
  %3 = getelementptr inbounds [3 x i64], ptr %1, i64 0, i64 0
  store i64 1, ptr %3, align 8
  %4 = getelementptr inbounds [3 x i64], ptr %1, i64 0, i64 1
  store i64 2, ptr %4, align 8
  %5 = getelementptr inbounds [3 x i64], ptr %1, i64 0, i64 2
  store i64 3, ptr %5, align 8
  %6 = getelementptr inbounds %array, ptr %0, i32 0, i32 1
  store ptr %1, ptr %6, align 8
  %7 = call i64 @main.test(ptr %0)
  ret void
}

define internal fastcc i64 @main.test(ptr %0) {
test_entry:
  %1 = getelementptr inbounds %array, ptr %0, i32 0, i32 1
  %2 = load ptr, ptr %1, align 8
  %3 = getelementptr inbounds [1 x i64], ptr %2, i64 0, i64 1
  %4 = load i64, ptr %3, align 8
  ret i64 %4
}
`,
		},
		{
			name:  "generic struct - pass to function",
			input: "lib main struct abc { a i64, b i64 } gen struct xyz { b i64 } fn main() { let s = abc{a: 1, b: 2} let res = test(s) } fn test(s xyz) i64 { return s.b }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%abc = type { i64, i64 }
%abstract = type { ptr, [1 x i64] }

define fastcc void @main() {
main_entry:
  %0 = alloca %abc, align 8
  %1 = getelementptr inbounds %abc, ptr %0, i32 0, i32 0
  store i64 1, ptr %1, align 8
  %2 = getelementptr inbounds %abc, ptr %0, i32 0, i32 1
  store i64 2, ptr %2, align 8
  %3 = alloca %abstract, align 8
  %4 = getelementptr inbounds %abstract, ptr %3, i32 0, i32 0
  store ptr %0, ptr %4, align 8
  %5 = getelementptr inbounds %abstract, ptr %3, i32 0, i32 1
  %6 = getelementptr inbounds %abc, ptr %0, i32 0, i32 1
  %7 = call i64 @unsafe.ptr_diff(ptr %6, ptr %0)
  %8 = getelementptr inbounds [1 x i64], ptr %5, i64 0
  store i64 %7, ptr %8, align 8
  %9 = call i64 @main.test(ptr %3)
  ret void
}

define internal fastcc i64 @main.test(ptr %0) {
test_entry:
  %1 = getelementptr inbounds %abstract, ptr %0, i32 0, i32 0
  %2 = load ptr, ptr %1, align 8
  %3 = getelementptr inbounds %abstract, ptr %0, i32 0, i32 1
  %4 = getelementptr inbounds [1 x i64], ptr %3, i64 0
  %5 = load i64, ptr %4, align 8
  %6 = getelementptr inbounds i8, ptr %2, i64 %5
  %7 = load i64, ptr %6, align 8
  ret i64 %7
}

define internal i64 @unsafe.ptr_diff(ptr %0, ptr %1) {
entry:
  %2 = ptrtoint ptr %0 to i64
  %3 = ptrtoint ptr %1 to i64
  %4 = sub i64 %3, %2
  ret i64 %4
}
`,
		},
		//		{
		//		name:  "function - return struct on heap",
		//		input: "lib main struct abc{x i64} pub fn main() { t = test() } fn test() abc { a = abc{x: 1} return a}",
		//		want: ``,
		//	},
		{
			name:  "return union",
			input: "lib main struct a { x i64 } union b { a } fn main() { let v = test() } fn test() b { let v = a{x: 1} return b(v) }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%main.b = type { i64, [8 x i8] }
%a = type { i64 }

define fastcc void @main() {
main_entry:
  %0 = alloca %main.b, align 8
  %1 = call ptr @main.test(ptr %0)
  ret void
}

define internal fastcc ptr @main.test(ptr %0) {
test_entry:
  %1 = alloca %a, align 8
  %2 = getelementptr inbounds %a, ptr %1, i32 0, i32 0
  store i64 1, ptr %2, align 8
  %3 = alloca %main.b, align 8
  %4 = getelementptr inbounds %main.b, ptr %3, i32 0, i32 0
  store i64 -1885140172373873918, ptr %4, align 8
  %5 = getelementptr inbounds %main.b, ptr %3, i32 0, i32 1
  call void @llvm.memcpy.p0.p0.i64(ptr %5, ptr %1, i64 ptrtoint (ptr getelementptr (%a, ptr null, i32 1) to i64), i1 false)
  call void @llvm.memcpy.p0.p0.i64(ptr %0, ptr %3, i64 ptrtoint (ptr getelementptr (%main.b, ptr null, i32 1) to i64), i1 false)
  ret ptr %0
}

; Function Attrs: nocallback nofree nounwind willreturn memory(argmem: readwrite)
declare void @llvm.memcpy.p0.p0.i64(ptr noalias nocapture writeonly, ptr noalias nocapture readonly, i64, i1 immarg) #0

attributes #0 = { nocallback nofree nounwind willreturn memory(argmem: readwrite) }
`,
		},
		{
			name:  "multiple primitive return types",
			input: "lib main fn test() i64, f64, bool { return 1, 2.0, false } pub fn main() { let a, let b, let c = test() }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

define internal fastcc { i64, double, i1 } @main.test() {
test_entry:
  ret { i64, double, i1 } { i64 1, double 2.000000e+00, i1 false }
}

define fastcc void @main() {
main_entry:
  %0 = call { i64, double, i1 } @main.test()
  %1 = extractvalue { i64, double, i1 } %0, 0
  %2 = extractvalue { i64, double, i1 } %0, 1
  %3 = extractvalue { i64, double, i1 } %0, 2
  ret void
}
`,
		},
		{
			name:  "return function",
			input: "lib main pub fn main() { let res = test() } fn test() i64 { return test2() } fn test2() i64 { return 2}",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

define fastcc void @main() {
main_entry:
  %0 = call i64 @main.test()
  ret void
}

define internal fastcc i64 @main.test() {
test_entry:
  %0 = call i64 @main.test2()
  ret i64 %0
}

define internal fastcc i64 @main.test2() {
test2_entry:
  ret i64 2
}
`,
		},
		{
			name:  "return struct",
			input: "lib main struct abc { x i64 } fn main() { let res = test() let v = res.x } fn test() abc { return abc{x: 11} }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%abc = type { i64 }

define fastcc void @main() {
main_entry:
  %0 = alloca %abc, align 8
  %1 = call ptr @main.test(ptr %0)
  %2 = getelementptr inbounds %abc, ptr %1, i32 0, i32 0
  %3 = load i64, ptr %2, align 8
  ret void
}

define internal fastcc ptr @main.test(ptr %0) {
test_entry:
  %1 = alloca %abc, align 8
  %2 = getelementptr inbounds %abc, ptr %1, i32 0, i32 0
  store i64 11, ptr %2, align 8
  call void @llvm.memcpy.p0.p0.i64(ptr %0, ptr %1, i64 ptrtoint (ptr getelementptr (%abc, ptr null, i32 1) to i64), i1 false)
  ret ptr %0
}

; Function Attrs: nocallback nofree nounwind willreturn memory(argmem: readwrite)
declare void @llvm.memcpy.p0.p0.i64(ptr noalias nocapture writeonly, ptr noalias nocapture readonly, i64, i1 immarg) #0

attributes #0 = { nocallback nofree nounwind willreturn memory(argmem: readwrite) }
`,
		},
		//	{
		//		name:  "function - variadic",
		//		input: "lib main fn test(args ...i64) i64 { return args[0] } pub fn main() { arg = test(0,1,2) }",
		//	},
		{
			name:  "return var",
			input: "lib main struct xyz {x i64} fn main() {} fn test() xyz { var x = xyz{x:1} return x }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%xyz = type { i64 }

define fastcc void @main() {
main_entry:
  ret void
}

define internal fastcc ptr @main.test(ptr %0) {
test_entry:
  %1 = alloca %xyz, align 8
  %2 = getelementptr inbounds %xyz, ptr %1, i32 0, i32 0
  store i64 1, ptr %2, align 8
  call void @llvm.memcpy.p0.p0.i64(ptr %0, ptr %1, i64 ptrtoint (ptr getelementptr (%xyz, ptr null, i32 1) to i64), i1 false)
  ret ptr %0
}

; Function Attrs: nocallback nofree nounwind willreturn memory(argmem: readwrite)
declare void @llvm.memcpy.p0.p0.i64(ptr noalias nocapture writeonly, ptr noalias nocapture readonly, i64, i1 immarg) #0

attributes #0 = { nocallback nofree nounwind willreturn memory(argmem: readwrite) }
`,
		},
		{
			name:  "return function with multiple return",
			input: "lib main struct abc {x i64} fn test1() abc, abc { return abc{x: 1}, abc{x: 2} } fn test2() abc, abc { return test1() } fn main() { test2() }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%abc = type { i64 }

define internal fastcc { ptr, ptr } @main.test1(ptr %0, ptr %1) {
test1_entry:
  %2 = alloca %abc, align 8
  %3 = getelementptr inbounds %abc, ptr %2, i32 0, i32 0
  store i64 1, ptr %3, align 8
  call void @llvm.memcpy.p0.p0.i64(ptr %0, ptr %2, i64 ptrtoint (ptr getelementptr (%abc, ptr null, i32 1) to i64), i1 false)
  %4 = alloca %abc, align 8
  %5 = getelementptr inbounds %abc, ptr %4, i32 0, i32 0
  store i64 2, ptr %5, align 8
  call void @llvm.memcpy.p0.p0.i64(ptr %1, ptr %4, i64 ptrtoint (ptr getelementptr (%abc, ptr null, i32 1) to i64), i1 false)
  %6 = alloca %abc, align 8
  %7 = getelementptr inbounds %abc, ptr %6, i32 0, i32 0
  store i64 1, ptr %7, align 8
  %8 = insertvalue { ptr, ptr } zeroinitializer, ptr %6, 0
  %9 = alloca %abc, align 8
  %10 = getelementptr inbounds %abc, ptr %9, i32 0, i32 0
  store i64 2, ptr %10, align 8
  %11 = insertvalue { ptr, ptr } %8, ptr %9, 1
  ret { ptr, ptr } %11
}

define internal fastcc { ptr, ptr } @main.test2(ptr %0, ptr %1) {
test2_entry:
  %2 = alloca %abc, align 8
  %3 = alloca %abc, align 8
  %4 = call { ptr, ptr } @main.test1(ptr %2, ptr %3)
  %5 = extractvalue { ptr, ptr } %4, 0
  call void @llvm.memcpy.p0.p0.i64(ptr %0, ptr %5, i64 ptrtoint (ptr getelementptr (%abc, ptr null, i32 1) to i64), i1 false)
  %6 = extractvalue { ptr, ptr } %4, 1
  call void @llvm.memcpy.p0.p0.i64(ptr %1, ptr %6, i64 ptrtoint (ptr getelementptr (%abc, ptr null, i32 1) to i64), i1 false)
  %7 = alloca %abc, align 8
  %8 = alloca %abc, align 8
  %9 = call { ptr, ptr } @main.test1(ptr %7, ptr %8)
  %10 = extractvalue { ptr, ptr } %9, 0
  %11 = insertvalue { ptr, ptr } zeroinitializer, ptr %10, 0
  %12 = extractvalue { ptr, ptr } %9, 1
  %13 = insertvalue { ptr, ptr } %11, ptr %12, 1
  ret { ptr, ptr } %13
}

define fastcc void @main() {
main_entry:
  %0 = alloca %abc, align 8
  %1 = alloca %abc, align 8
  %2 = call { ptr, ptr } @main.test2(ptr %0, ptr %1)
  ret void
}

; Function Attrs: nocallback nofree nounwind willreturn memory(argmem: readwrite)
declare void @llvm.memcpy.p0.p0.i64(ptr noalias nocapture writeonly, ptr noalias nocapture readonly, i64, i1 immarg) #0

attributes #0 = { nocallback nofree nounwind willreturn memory(argmem: readwrite) }
`,
		},
		{
			name:  "pass struct and return field of type struct",
			input: "lib main struct a{v b} struct b{i i64} fn test(s a) b { return s.v } fn main() { let s = a{v: b{i:1}} test(s) }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%a = type { %b }
%b = type { i64 }

define internal fastcc ptr @main.test(ptr %0, ptr %1) {
test_entry:
  %2 = getelementptr inbounds %a, ptr %0, i32 0, i32 0
  %3 = load ptr, ptr %2, align 8
  call void @llvm.memcpy.p0.p0.i64(ptr %1, ptr %3, i64 ptrtoint (ptr getelementptr (%b, ptr null, i32 1) to i64), i1 false)
  ret ptr %1
}

define fastcc void @main() {
main_entry:
  %0 = alloca %a, align 8
  %1 = alloca %b, align 8
  %2 = getelementptr inbounds %b, ptr %1, i32 0, i32 0
  store i64 1, ptr %2, align 8
  %3 = getelementptr inbounds %a, ptr %0, i32 0, i32 0
  store ptr %1, ptr %3, align 8
  %4 = alloca %b, align 8
  %5 = call ptr @main.test(ptr %0, ptr %4)
  ret void
}

; Function Attrs: nocallback nofree nounwind willreturn memory(argmem: readwrite)
declare void @llvm.memcpy.p0.p0.i64(ptr noalias nocapture writeonly, ptr noalias nocapture readonly, i64, i1 immarg) #0

attributes #0 = { nocallback nofree nounwind willreturn memory(argmem: readwrite) }
`,
		},
		{
			name:  "return function and call",
			input: "lib main fn main() { let func = test2() func() } fn test1() i64, bool { return 0, false } fn test2()fn()i64,bool { return test1 }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

define fastcc void @main() {
main_entry:
  %0 = call ptr @main.test2()
  %1 = call { i64, i1 } %0()
  ret void
}

define internal fastcc { i64, i1 } @main.test1() {
test1_entry:
  ret { i64, i1 } zeroinitializer
}

define internal fastcc ptr @main.test2() {
test2_entry:
  ret ptr @main.test1
}
`,
		},
		{
			name: "return function and call, optional",
			input: `lib main
				type abc fn()i64,bool
				fn main() { let func = ?(test2()) func() }
				fn test1() i64, bool { return 0, false }
				fn test2() ?abc { return test1 }
			`,
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%option_ptr = type { i1, ptr }

define fastcc void @main() {
main_entry:
  %0 = call %option_ptr @main.test2()
  %1 = extractvalue %option_ptr %0, 1
  %2 = call { i64, i1 } %1()
  ret void
}

define internal fastcc { i64, i1 } @main.test1() {
test1_entry:
  ret { i64, i1 } zeroinitializer
}

define internal fastcc %option_ptr @main.test2() {
test2_entry:
  ret %option_ptr { i1 true, ptr @main.test1 }
}
`,
		},
	}
	runTests(t, tests)
}

func TestFunctionAttributes(t *testing.T) {
	tests := []testCase{
		{
			name:    "extern c",
			skipRun: true,
			input:   "lib main @extern(c) pub fn test()",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

declare void @test()
`,
		},
		{
			name:  "extern c - call",
			input: `lib main fn main() { let s = []byte("12") write(1, &s, 2) } @extern(c) pub fn write(fd u64, buf *[]byte, size i64) i64`,
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%string = type { i64, ptr }
%array = type { i64, ptr }

@0 = internal constant [2 x i8] c"12"
@1 = internal constant %string { i64 2, ptr @0 }

define fastcc void @main() {
main_entry:
  %0 = load ptr, ptr getelementptr inbounds (%array, ptr @1, i32 0, i32 1), align 8
  %1 = call i64 @write(i64 1, ptr %0, i64 2)
  ret void
}

declare i64 @write(i64, ptr, i64)
`,
		},
	}
	runTests(t, tests)
}

func TestDashToC(t *testing.T) {
	tests := []testCase{
		{
			name:  "*memory<[]byte> to char*",
			input: "lib main @extern(c) pub fn read(buf *memory<[]byte>) fn main() { let b = make([]byte, 1) read(&b) }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%array = type { i64, ptr }

declare void @read(ptr)

define fastcc void @main() {
main_entry:
  %0 = alloca i8, i64 1, align 4
  %1 = alloca %array, align 8
  %2 = alloca i64, align 8
  store i64 0, ptr %2, align 8
  %3 = getelementptr inbounds %array, ptr %1, i32 0, i32 0
  store i64 1, ptr %3, align 8
  %4 = getelementptr inbounds %array, ptr %1, i32 0, i32 1
  store ptr %0, ptr %4, align 8
  br label %5

5:                                                ; preds = %10, %main_entry
  %6 = load i64, ptr %2, align 8
  %7 = icmp ult i64 %6, 1
  br i1 %7, label %8, label %12

8:                                                ; preds = %5
  %9 = getelementptr inbounds [1 x i8], ptr %0, i64 0, i64 %6
  store i8 0, ptr %9, align 1
  br label %10

10:                                               ; preds = %8
  %11 = add i64 %6, 1
  store i64 %11, ptr %2, align 8
  br label %5

12:                                               ; preds = %5
  %13 = getelementptr inbounds %array, ptr %1, i32 0, i32 1
  %14 = load ptr, ptr %13, align 8
  call void @read(ptr %14)
  ret void
}
`,
		},
	}
	runTests(t, tests)
}

func TestAnonymousFunctions(t *testing.T) {
	tests := []testCase{
		{
			name:  "anonymous fn - no args, no return",
			input: "lib main pub fn main() { let test = fn() { } test() }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

define fastcc void @main() {
main_entry:
  call void @__main_test()
  ret void
}

define internal fastcc void @__main_test() {
__main_test_entry:
  ret void
}
`,
		},
		{
			name:  "single arg and return",
			input: "lib main pub fn main() { let test = fn(a i64) i64 { return a } let t = test(1) + 1 }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

define fastcc void @main() {
main_entry:
  %0 = call i64 @__main_test(i64 1)
  %add = add i64 %0, 1
  ret void
}

define internal fastcc i64 @__main_test(i64 %0) {
__main_test_entry:
  ret i64 %0
}
`,
		},
		{
			name:  "args, scalar return",
			input: "lib main pub fn main() { let add = fn(a i64, b i64) i64 { return a + b } let res = add(1,-1) }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

define fastcc void @main() {
main_entry:
  %0 = call i64 @__main_add(i64 1, i64 -1)
  ret void
}

define internal fastcc i64 @__main_add(i64 %0, i64 %1) {
__main_add_entry:
  %add = add i64 %0, %1
  ret i64 %add
}
`,
		},
		{
			name:  "multi return",
			input: "lib main pub fn main() { let test = fn() i64, f64 { return 1, 2.0 } let x, let y = test() }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

define fastcc void @main() {
main_entry:
  %0 = call { i64, double } @__main_test()
  %1 = extractvalue { i64, double } %0, 0
  %2 = extractvalue { i64, double } %0, 1
  ret void
}

define internal fastcc { i64, double } @__main_test() {
__main_test_entry:
  ret { i64, double } { i64 1, double 2.000000e+00 }
}
`,
		},
	}
	runTests(t, tests)
}

func TestHigherOrderFns(t *testing.T) {
	tests := []testCase{
		{
			name:  "no arguments, single return, scalar",
			input: "lib main pub fn main() { let res = a(b)} fn a(f fn() i64) i64 { return f() } fn b() i64 { return 2 }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

define fastcc void @main() {
main_entry:
  %0 = call i64 @main.a(ptr @main.b)
  ret void
}

define internal fastcc i64 @main.a(ptr %0) {
a_entry:
  %1 = call i64 %0()
  ret i64 %1
}

define internal fastcc i64 @main.b() {
b_entry:
  ret i64 2
}
`,
		},
	}
	runTests(t, tests)
}

func TestHigherOrderAnonymousFunctions(t *testing.T) {
	tests := []testCase{
		{
			name:    "",
			skipRun: true,
			input: `lib main
fn main() {				
	let concat = fn(a, b string) string {
        return a + b
    }   

    let do = fn(l, r string, f fn(string, string)string) string {
        return f(l, r)
    }
    let res = do("1","2",concat)
}`,
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%string = type { i64, ptr }

@0 = internal constant [1 x i8] c"1"
@1 = internal constant %string { i64 1, ptr @0 }
@2 = internal constant [1 x i8] c"2"
@3 = internal constant %string { i64 1, ptr @2 }

define fastcc void @main() {
main_entry:
  %0 = call ptr @__main_do(ptr @1, ptr @3, ptr @__main_concat)
  ret void
}

define internal fastcc ptr @__main_concat(ptr %0, ptr %1) {
__main_concat_entry:
  %2 = getelementptr inbounds %string, ptr %0, i32 0, i32 0
  %3 = load i64, ptr %2, align 8
  %4 = getelementptr inbounds %string, ptr %1, i32 0, i32 0
  %5 = load i64, ptr %4, align 8
  %6 = add i64 %3, %5
  %7 = alloca i8, i64 %6, align 4
  %8 = alloca %string, align 8
  %9 = getelementptr inbounds %string, ptr %8, i32 0, i32 0
  store i64 %6, ptr %9, align 8
  %10 = getelementptr inbounds %string, ptr %8, i32 0, i32 1
  store ptr %7, ptr %10, align 8
  %11 = call fastcc ptr @runtime.str_concat2(ptr %8, ptr %0, ptr %1)
  ret ptr %11
}

declare fastcc ptr @runtime.str_concat2(ptr, ptr, ptr)

define internal fastcc ptr @__main_do(ptr %0, ptr %1, ptr %2) {
__main_do_entry:
  %3 = call ptr %2(ptr %0, ptr %1)
  ret ptr %3
}
`,
		},
	}
	runTests(t, tests)
}

func TestEnums(t *testing.T) {
	tests := []testCase{
		{
			name:  "enum - initialise, pass and return from fn",
			input: "lib main enum status { unknown, stopped } pub fn main() { let u = test(status.unknown) } fn test(s status) status { return s }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%status = type { i8 }

define fastcc void @main() {
main_entry:
  %0 = call %status @main.test(%status zeroinitializer)
  ret void
}

define internal fastcc %status @main.test(%status %0) {
test_entry:
  ret %status %0
}
`,
		},
		{
			name:  "enum - return directly from fn",
			input: "lib main enum status { unknown } fn test() status { return status.unknown } pub fn main() { let status = test() }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%status = type { i8 }

define internal fastcc %status @main.test() {
test_entry:
  ret %status zeroinitializer
}

define fastcc void @main() {
main_entry:
  %0 = call %status @main.test()
  ret void
}
`,
		},
		{
			name:  "field equality",
			input: "lib main enum status{offline, online} fn main() { test() } fn test() bool { return status.offline == status.online }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

define fastcc void @main() {
main_entry:
  %0 = call i1 @main.test()
  ret void
}

define internal fastcc i1 @main.test() {
test_entry:
  ret i1 false
}
`,
		},
	}
	runTests(t, tests)
}

func TestUnion(t *testing.T) {
	tests := []testCase{
		{
			name:  "union of scalar types",
			input: "lib main union abc { i64, f64 } fn main() { let a = abc(1) }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%main.abc = type { i64, [8 x i8] }

define fastcc void @main() {
main_entry:
  %0 = alloca %main.abc, align 8
  %1 = getelementptr inbounds %main.abc, ptr %0, i32 0, i32 0
  store i64 5234574298831366340, ptr %1, align 8
  %2 = getelementptr inbounds %main.abc, ptr %0, i32 0, i32 1
  store i64 1, ptr %2, align 8
  ret void
}
`,
		},
		{
			name:  "mixed union",
			input: "lib main struct a {x bool} union ab { a i64 } fn main() { let v = ab(a{x: true}) }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

%a = type { i1 }
%main.ab = type { i64, [8 x i8] }

define fastcc void @main() {
main_entry:
  %0 = alloca %a, align 8
  %1 = getelementptr inbounds %a, ptr %0, i32 0, i32 0
  store i1 true, ptr %1, align 1
  %2 = alloca %main.ab, align 8
  %3 = getelementptr inbounds %main.ab, ptr %2, i32 0, i32 0
  store i64 -1983398940155033090, ptr %3, align 8
  %4 = getelementptr inbounds %main.ab, ptr %2, i32 0, i32 1
  call void @llvm.memcpy.p0.p0.i64(ptr %4, ptr %0, i64 ptrtoint (ptr getelementptr (%a, ptr null, i32 1) to i64), i1 false)
  ret void
}

; Function Attrs: nocallback nofree nounwind willreturn memory(argmem: readwrite)
declare void @llvm.memcpy.p0.p0.i64(ptr noalias nocapture writeonly, ptr noalias nocapture readonly, i64, i1 immarg) #0

attributes #0 = { nocallback nofree nounwind willreturn memory(argmem: readwrite) }
`,
		},
	}
	runTests(t, tests)
}

func TestDefer(t *testing.T) {
	tests := []testCase{
		{
			name:  "defer block with single return",
			input: "lib main pub fn main() { let t = test() } fn test() i64 { defer { let l = log() } return 2 } fn log() i64 { return 1 }",
			want: `; ModuleID = 'main-ir'
source_filename = "main-ir"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-linux-unknown"

define fastcc void @main() {
main_entry:
  %0 = call i64 @main.test()
  ret void
}

define internal fastcc i64 @main.test() {
test_entry:
  %0 = call i64 @main.log()
  ret i64 2
}

define internal fastcc i64 @main.log() {
log_entry:
  ret i64 1
}
`,
		},
	}

	runTests(t, tests)
}

// ------- //
// Helpers //
// ------- //

type testCase struct {
	name  string
	input string
	want  string
	// whether test case should not be run
	// using LLVM function execution engine
	skipRun bool
}

func runTests(t *testing.T, tests []testCase) {
	// for _, tt := range tests {
	// 	parser := GetParser(tt.input)
	// 	ast := parser.ParseLibrary()
	// 	// TODO: check synatx errors
	// 	tsm := transformer.New()
	// 	tsm.Tranform(ast)

	// 	s := semantic.New()
	// 	s.Analyse(ast)
	// 	if len(s.Errors()) != 0 {
	// 		t.Errorf("semantic analysis: %s", s.Errors())
	// 	}

	// 	t.Run(tt.name, func(t *testing.T) {

	// 		c := New(&Config{
	// 			Triple:    NewTriple(AARCH64, LINUX, UNKNOWN),
	// 			Mode:      DEBUG,
	// 			ModuleTag: "test",
	// 		})
	// 		ir, err := c.GenerateIR(ast)
	// 		if err != nil {
	// 			t.Errorf("unable to compile %s: %s", tt.input, err)
	// 		}

	// 		// TODO: validate header by slicing string?
	// 		//
	// 		// ; ModuleID = 'main-ir'
	// 		// source_filename = "main-ir"
	// 		// target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
	// 		// target triple = "aarch64-linux-unknown"

	// 		if tt.want != ir {
	// 			t.Errorf("want %s but got %s", tt.want, ir)
	// 		}
	// 	})
	// 	if tt.skipRun {
	// 		continue
	// 	}
	// 	t.Run(tt.name+"-exec", func(t *testing.T) {
	// 		c := New(&Config{
	// 			Mode:      DEBUG,
	// 			ModuleTag: "exec",
	// 		})
	// 		if err := c.GenerateAndExec(ast); err != nil {
	// 			t.Errorf("unable to run code: %s", err)
	// 		}
	// 	})
	// }
}

func GetParser(input string) *parser.Parser {
	lcfg := &lexer.Config{SkipComments: true}
	l := lexer.New("", input, lcfg)
	return parser.New(l)
}
