package generator

import (
	"tinygo.org/x/go-llvm"
)

// Docs: https://llvm.org/docs/LangRef.html#llvm-memcpy-intrinsic
// @llvm.memcpy.p0.p0.i64(ptr <dest>, ptr <src>, i64 <len>, i1 <isvolatile>)
func (g *Generator) createCallMemCopy(destPtr, srcPtr llvm.Value, size llvm.Value, isVolatile llvm.Value) llvm.Value {
	fn := g.mod.NamedFunction("llvm.memcpy.p0.p0.i64")
	voidType := g.ctx.VoidType()
	fnType := llvm.FunctionType(voidType, []llvm.Type{destPtr.Type(), srcPtr.Type(), size.Type(), isVolatile.Type()}, false)
	// create memcopy if doesnt exist
	if fn.IsNil() {
		i64Type := g.ctx.Int64Type()
		i1Type := g.ctx.Int1Type()
		ptrType := llvm.PointerType(g.ctx.Int8Type(), 0)

		fnType = llvm.FunctionType(voidType, []llvm.Type{ptrType, ptrType, i64Type, i1Type}, false)
		fn = llvm.AddFunction(g.mod, "llvm.memcpy.p0.p0.i64", fnType)
	}

	return g.builder.CreateCall(fnType, fn, []llvm.Value{destPtr, srcPtr, size, isVolatile}, "")
}
