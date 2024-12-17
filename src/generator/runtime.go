package generator

import (
	"dash-lang.io/src/types"
	"tinygo.org/x/go-llvm"
)

func (g *Generator) createCallExit(code llvm.Value) {
	name := "runtime.exit"
	fnInfo, ok := g.fnSt.Get(name)
	if !ok {
		fnType := llvm.FunctionType(g.ctx.VoidType(), []llvm.Type{g.getLLVMType(&types.ConstI64)}, false)
		fnPtr := llvm.AddFunction(g.mod, name, fnType)
		fnPtr.SetLinkage(llvm.ExternalLinkage)

		g.fnSt.SetIn(0, name, Function{Type: fnType, Ptr: fnPtr})

		fnInfo.Ptr = fnPtr
		fnInfo.Type = fnType
	}

	g.builder.CreateCall(fnInfo.Type, fnInfo.Ptr, []llvm.Value{code}, "")
	g.builder.CreateUnreachable()
	return
}

// Creates the function call (and the function header if not exists) for the runtime function
// 'runtime.str_concat2'
func (g *Generator) createCallStrConcat2(alloca, s1, s2 llvm.Value) llvm.Value {
	name := "runtime.str_concat2"
	fnInfo, ok := g.fnSt.Get(name)
	if !ok {
		// generate function header
		strPtrTypeLLVM := llvm.PointerType(g.getLLVMType(&types.ConstString), 0)
		arrPtrTypeLLVM := llvm.PointerType(g.getLLVMType(&types.Array{T: &types.ConstU8}), 0)
		fnType := llvm.FunctionType(strPtrTypeLLVM, []llvm.Type{arrPtrTypeLLVM, strPtrTypeLLVM, strPtrTypeLLVM}, false)
		fnPtr := llvm.AddFunction(g.mod, name, fnType)
		fnPtr.SetLinkage(llvm.ExternalLinkage)
		fnPtr.SetFunctionCallConv(llvm.FastCallConv)

		g.fnSt.SetIn(0, name, Function{Type: fnType, Ptr: fnPtr})

		fnInfo.Ptr = fnPtr
		fnInfo.Type = fnType
	}
	call := g.builder.CreateCall(fnInfo.Type, fnInfo.Ptr, []llvm.Value{alloca, s1, s2}, "")
	call.SetInstructionCallConv(llvm.FastCallConv)
	return call
}

// Creates the function call (and the function header if not exists) for the runtime function
// 'runtime.str_cmp'
func (g *Generator) createCallStrCmp(s1, s2 llvm.Value) llvm.Value {
	name := "runtime.str_cmp"
	fnInfo, ok := g.fnSt.Get(name)
	if !ok {
		// generate function header
		strPtrTypeLLVM := llvm.PointerType(g.getLLVMType(&types.ConstString), 0)
		fnType := llvm.FunctionType(g.ctx.Int1Type(), []llvm.Type{strPtrTypeLLVM, strPtrTypeLLVM}, false)
		fnPtr := llvm.AddFunction(g.mod, name, fnType)
		fnPtr.SetLinkage(llvm.ExternalLinkage)
		fnPtr.SetFunctionCallConv(llvm.FastCallConv)

		g.fnSt.SetIn(0, name, Function{Type: fnType, Ptr: fnPtr})

		fnInfo.Ptr = fnPtr
		fnInfo.Type = fnType
	}
	call := g.builder.CreateCall(fnInfo.Type, fnInfo.Ptr, []llvm.Value{s1, s2}, "")
	call.SetInstructionCallConv(llvm.FastCallConv)
	return call
}
