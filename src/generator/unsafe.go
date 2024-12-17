// NOTE: this will be part of 'unsafe' module
package generator

import "tinygo.org/x/go-llvm"

var (
	unsafe_ptr_diff = "unsafe.ptr_diff"
)

//	pub fn ptr_diff<T any>(p1, p2 *T) u64 {
//	    let offset = ptr(p2) - ptr(p1)
//	    return u64(offset)
//	}
func (g *Generator) createPtrDiff() llvm.Value {
	oldBuilder := g.builder
	oldEntry := g.fnEntry
	g.builder = g.ctx.NewBuilder()
	defer g.builder.Dispose()
	g.fnNameScope.Push(unsafe_ptr_diff)
	defer g.fnNameScope.Pop()

	// Create function
	ptrType := llvm.PointerType(g.ctx.Int8Type(), 0)
	fnType := llvm.FunctionType(g.ctx.Int64Type(), []llvm.Type{ptrType, ptrType}, false)
	fn := llvm.AddFunction(g.mod, unsafe_ptr_diff, fnType)
	fn.SetLinkage(llvm.InternalLinkage)

	entry := llvm.AddBasicBlock(fn, "entry")
	g.builder.SetInsertPointAtEnd(entry)

	uint1 := g.builder.CreatePtrToInt(fn.Param(0), g.ctx.Int64Type(), "")
	uint2 := g.builder.CreatePtrToInt(fn.Param(1), g.ctx.Int64Type(), "")

	diff := g.builder.CreateSub(uint2, uint1, "")
	g.builder.CreateRet(diff)

	g.builder = oldBuilder
	g.fnEntry = oldEntry

	g.fnSt.Set(unsafe_ptr_diff, Function{Ptr: fn, Type: fnType})

	return fn
}

// Creates call to unsafe function that calculates offset between
// an element pointer (p1) and the base pointer (p2)
func (g *Generator) createCallPtrDiff(p1, p2 llvm.Value) llvm.Value {
	// Create '%offset = call i64 @llvm.ptrdiff.p0.p0(ptr %element_ptr, ptr %base_ptr)'
	fnInfo, _ := g.fnSt.Get(unsafe_ptr_diff)
	return g.builder.CreateCall(fnInfo.Type, fnInfo.Ptr, []llvm.Value{p1, p2}, "")
}
