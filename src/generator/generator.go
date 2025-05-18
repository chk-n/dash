// This file contains core methods for converting AST to LLVM IR.
package generator

import (
	"crypto/sha256"
	"encoding/binary"
	"strings"

	"dash-lang.io/src/ast"
	"dash-lang.io/src/internal"
	"dash-lang.io/src/token"
	"dash-lang.io/src/types"

	"tinygo.org/x/go-llvm"
)

func init() {
	llvm.InitializeAllTargets()
	llvm.InitializeAllTargetMCs()
	llvm.InitializeAllTargetInfos()
	llvm.InitializeAllAsmParsers()
	llvm.InitializeAllAsmPrinters()
}

var (
	empty = llvm.Value{}
)

type Mode int

const (
	DEBUG = iota
	REPL
)

type Config struct {
	// Default: host operating system
	Triple *Triple
	// Default: debug
	Mode Mode
	// can be used to prepend llvm module name
	// with additional information e.g.
	// 'repl', 'jit', 'test'
	ModuleTag string
}

type scope uint8

const (
	GLOBAL scope = iota
	FUNCTION
	COPY_UPDATE
	USE
	FOR
	MATCH
)

type Generator struct {
	// target
	triple *Triple

	// compiler mode
	mode Mode

	// name of librabry being compiled
	library string

	// module tag. See 'ModuleTag' in Config for more info
	tag string

	ctx           llvm.Context
	mod           llvm.Module
	builder       llvm.Builder
	fnEntry       llvm.BasicBlock
	targetMachine llvm.TargetMachine
	targetData    llvm.TargetData

	// symbol tables
	varSt *internal.StackedSymTab[Variable]
	fnSt  *internal.StackedSymTab[Function]
	// lookup table to fetch type guard when required
	typeGuardSt *internal.Cache[string, ast.Expression]
	// keep track of loop blocks used by 'break' and
	// 'next' to jump to correct block
	loopScope *internal.Stack[LoopData]
	// caches, to avoid duplicates
	typeCache      *internal.Cache[string, llvm.Type]
	attributeCache *internal.Cache[attribute, llvm.Attribute]
	// names of current chain of nested functions
	// used to generate name for anonymous functions
	fnNameScope *internal.Stack[string]
	// keeps track of current fn type being generated
	fnTypeScope *internal.Stack[*types.Function]
	// keeps track of current function being generated
	fnScope *internal.Stack[llvm.Value]
	// keeps track of the current scope
	scope *internal.Stack[scope]
}

func New(cfg *Config) *Generator {
	return &Generator{
		triple:         cfg.Triple,
		mode:           cfg.Mode,
		tag:            cfg.ModuleTag,
		varSt:          internal.NewStackedSymbolTable[Variable](),
		fnSt:           internal.NewStackedSymbolTable[Function](),
		typeGuardSt:    internal.NewCache[string, ast.Expression](),
		typeCache:      internal.NewCache[string, llvm.Type](),
		attributeCache: internal.NewCache[attribute, llvm.Attribute](),
		loopScope:      internal.NewStack[LoopData](),
		scope:          internal.NewStack[scope](),
		fnScope:        internal.NewStack[llvm.Value](),
		fnNameScope:    internal.NewStack[string](),
		fnTypeScope:    internal.NewStack[*types.Function](),
	}
}

// Takes a dash library and converts it to an llvm.Module
func (g *Generator) GenerateLibrary(lib *ast.Library) (llvm.Module, error) {
	if _, err := g.generate(lib); err != nil {
		return llvm.Module{}, err
	}

	return g.mod, nil
}

// Takes a dash library, generates the IR and executes it using the llvm execution engine
// NOTE: public main function is required!
func (g *Generator) GenerateAndExec(lib *ast.Library) error {
	g.tag = "exec"
	v, err := g.generate(lib)
	if err != nil {
		return err
	}

	defer g.mod.Dispose()

	if err := llvm.VerifyModule(g.mod, llvm.ReturnStatusAction); err != nil {
		return err
	}

	if _, err = g.run(v); err != nil {
		return err
	}

	return nil
}

func (g *Generator) GenerateIR(lib *ast.Library) (string, error) {
	g.tag = "ir"
	if _, err := g.generate(lib); err != nil {
		return "", err
	}
	defer g.mod.Dispose()

	if err := llvm.VerifyModule(g.mod, llvm.ReturnStatusAction); err != nil {
		return "", err
	}

	return g.mod.String(), nil
}

func NewTargetMachine(cfg *Config) (llvm.TargetMachine, error) {
	var triple string
	if cfg.Triple != nil {
		triple = cfg.Triple.String()
	} else {
		triple = llvm.DefaultTargetTriple()
	}

	// Lookup target
	target, err := llvm.GetTargetFromTriple(triple)
	if err != nil {
		return llvm.TargetMachine{}, err
	}

	// Create a target machine
	cpu := "generic"
	features := ""
	machine := target.CreateTargetMachine(triple, cpu, features, llvm.CodeGenLevelDefault, llvm.RelocDefault, llvm.CodeModelDefault)
	return machine, nil
}

func (g *Generator) setTarget() error {
	// Set triple or use default
	var triple string
	if g.triple != nil {
		triple = g.triple.String()
	} else {
		triple = llvm.DefaultTargetTriple()
	}

	g.mod.SetTarget(triple)

	// Lookup target
	target, err := llvm.GetTargetFromTriple(triple)
	if err != nil {
		return err
	}

	// Create a target machine
	cpu := "generic"
	features := ""
	g.targetMachine = target.CreateTargetMachine(triple, cpu, features, llvm.CodeGenLevelDefault, llvm.RelocDefault, llvm.CodeModelDefault)
	g.targetData = g.targetMachine.CreateTargetData()
	// Set the target data layout for the module
	g.mod.SetDataLayout(g.targetData.String())

	return nil
}

func (g *Generator) generate(lib *ast.Library) (llvm.Value, error) {
	g.library = lib.Name.String()
	modName := g.library
	if g.tag != "" {
		modName += "-" + g.tag
	}
	g.ctx = llvm.NewContext()
	g.mod = g.ctx.NewModule(modName)
	g.setTarget()

	g.scope.Push(GLOBAL)
	defer g.scope.Pop()

	var v llvm.Value
	// Build symbol table for enums
	for _, e := range lib.Enums {
		enumType := g.getLLVMType(e.T)
		fieldType := g.getInternalEnumType(e.T.Size)
		for i, f := range e.Fields {
			name := e.Name.String() + "." + f.Value
			fields := []llvm.Value{llvm.ConstInt(fieldType, uint64(i), false)}
			value := llvm.ConstNamedStruct(enumType, fields)
			g.varSt.Set(name, Variable{Type: enumType, Ptr: value})
		}
		g.varSt.Set(e.Name.String(), Variable{Type: enumType})
	}

	for _, td := range lib.TypeDefinitions {
		g.getLLVMType(td.Type())
		if td.Guard != nil {
			g.typeGuardSt.Set(td.Name.String(), td.Guard)
		}
	}
	for _, ta := range lib.TypeAliases {
		g.getLLVMType(ta.Type())
	}
	for _, st := range lib.Structs {
		g.getLLVMType(st.Type())
	}
	for _, gs := range lib.GenericStructs {
		g.getLLVMType(gs.Type())
	}

	for _, v := range lib.GlobalVariables {
		// BUG: for structs it attempts to stack allocate..
		g.buildInternal(v, "")
		// val.SetLinkage(llvm.)
	}

	for _, un := range lib.Unions {
		g.getLLVMType(un.T)
	}

	for _, fn := range lib.Functions {
		fnPtr, fnType := g.createFunctionHeader(fn)
		// store function in global symbol table
		g.fnSt.Set(fn.Name.String(), Function{
			Type:       fnType,
			Ptr:        fnPtr,
			TypeDash:   fn.Type(),
			Attributes: fn.Attributes,
		})
	}

	for _, fn := range lib.Functions {
		res := g.buildInternal(fn, "")
		if fn.Name.Value == "main" {
			v = res
		}
	}

	// fmt.Println(g.mod.String())

	return v, nil
}

func (g *Generator) buildInternal(node ast.Node, name string) llvm.Value {
	switch n := node.(type) {
	case *ast.FunctionExpression:
		oldBuilder := g.builder
		oldEntry := g.fnEntry
		g.builder = g.ctx.NewBuilder()
		defer g.builder.Dispose()

		fnName := n.Name.String()
		g.fnNameScope.Push(fnName)
		defer g.fnNameScope.Pop()
		g.fnTypeScope.Push(n.Type().(*types.Function))
		defer g.fnTypeScope.Pop()

		var fn Function
		if n.IsAnonymous {
			// handle anonymous function
			fn.Ptr, fn.Type = g.createFunctionHeader(n)
			fnName = "__" + strings.Join(g.fnNameScope.GetAll(), "_")
			fn.TypeDash = n.Type()
			fn.Ptr.SetName(fnName)

			g.fnSt.Set(fnName, fn)
		} else {
			var ok bool
			fn, ok = g.fnSt.Get(fnName)
			if !ok {
				panic("this is a compiler error. pleasse report")
			}
		}
		g.fnScope.Push(fn.Ptr)
		defer g.fnScope.Pop()
		g.scope.Push(FUNCTION)
		defer g.scope.Pop()

		// scope in to ensure fn arguments and body only available within function
		g.varSt.Scope()
		defer g.varSt.Unscope()
		g.fnSt.Scope()
		defer g.fnSt.Unscope()

		// register arguments in symbol tables
		typ := n.Type().(*types.Function)
		argTypes := g.getLLVMTypes(typ.Arg)
		for i, at := range argTypes {
			// No value should be undefined in Dash

			// TODO: set attributes e.g. pointers cant be null
			// fn.AddAttributeAtIndex(i, g.get_llvm_attribute(noUndefined))
			// fn.AddAttributeAtIndex(i, g.get_llvm_attribute(readOnly))

			// set appropriate symbol tables
			argName := n.Arguments[i].Name.String()
			switch at.TypeKind() {
			case llvm.FunctionTypeKind:
				g.fnSt.Set(argName, Function{Type: at, Ptr: fn.Ptr.Param(i), TypeDash: typ.GetArgumentTypeAt(i)})
			default:
				g.varSt.Set(argName, Variable{Type: at, Ptr: fn.Ptr.Param(i)})
			}
		}

		var last llvm.Value
		if n.Body != nil {
			// Create a basic block in the function and set the builder's insert point to it
			g.fnEntry = llvm.AddBasicBlock(fn.Ptr, fnName+"_entry")
			g.builder.SetInsertPointAtEnd(g.fnEntry)
			last = g.buildInternal(n.Body, "")
		}
		// if repl mode print value of last instruction
		if g.mode == REPL && !n.IsAnonymous {
			if last.IsNil() {
				// do nothing
			} else if last.Type().TypeKind() == llvm.PointerTypeKind {
				// TODO: fix as this throws error
				// exp, ok := n.Body.Statements[len(n.Body.Statements)-1].(ast.Expression)
				// if ok {
				// 	lastType := g.get_llvm_type(exp.Type())
				// 	last = g.curBuilder.CreateLoad(lastType, last, "")
				// 	g.compileReplPrint(last)
				// }
			} else {
				g.compileReplPrint(last)
			}
		}
		// add exit if main
		if fnName == "main" && last.IsAReturnInst().IsNil() {
			// exitCode0 := llvm.ConstInt(g.ctx.Int64Type(), 0, false)
			// g.createCallExit(exitCode0)
			g.builder.CreateRetVoid()
		} else if len(n.ReturnValues) == 0 && last.IsAReturnInst().IsNil() {
			g.builder.CreateRetVoid()
		}

		g.builder = oldBuilder
		g.fnEntry = oldEntry
		return fn.Ptr
	case *ast.FunctionCallExpression:
		if v, ok := g.generateTypeCast(n); ok {
			return v
		}
		// check if function is built-in
		if v, ok := g.compileBuiltinFunc(n); ok {
			return v
		}

		fnName := n.TokenLiteral()
		// anonymous functions get generated like normal functions except their name is changed
		if n.IsAnonymousFn && g.fnNameScope.Len() >= 1 {
			fnName = "__" + strings.Join(g.fnNameScope.GetAll(), "_") + "_" + fnName
		}
		fnInfo, ok := g.fnSt.Get(fnName)
		if !ok {
			panic("this is a compiler error. please report")
		}

		var args []llvm.Value
		fnType := fnInfo.TypeDash.(*types.Function)
		isExternC := hasAttribute(fnInfo.Attributes, ast.ExternC)
		if isExternC {
			for _, a := range n.Arguments {
				val := g.buildInternal(a, "")
				args = append(args, g.generateDashToC(a.Type(), val))
			}
		} else {
			for i, a := range n.Arguments {
				// if function argument is of type abstract struct we need to change
				// how the struct is represented so it works without types being known
				switch t1 := fnType.GetArgumentTypeAt(i).(type) {
				case *types.Optional:
					args = append(args, g.generateOptionalFromLiteral(a, t1))
				case *types.AbstractStruct:
					switch t2 := a.Type().(type) {
					// compile conversion required to go from struct to abstract struct
					case *types.Struct:
						args = append(args, g.compileStructToAbstract(a, t2, t1))
					case *types.AbstractStruct:
						// TODO: abstract struct to abstract struct
					default:
						panic("this is a compiler error. please report")
					}
				default:
					args = append(args, g.buildInternal(a, ""))
				}
			}
		}
		for _, rt := range n.ReturnTypes {
			switch rt.(type) {
			case *types.Struct, *types.Union:
				varInfo, ok := g.varSt.Get(name)

				llvmType := g.getLLVMType(rt)
				var ptr llvm.Value
				g.doInEntry(func() {
					ptr = g.builder.CreateAlloca(llvmType, "")
				})
				args = append(args, ptr)
				// NOTE: ngl this is definitely a bug, meaning we did not
				// get 'name' right
				if !ok {
					g.varSt.Set(name, Variable{Type: varInfo.Type, Ptr: ptr, IsVar: varInfo.IsVar})
				}
			}
		}

		call := g.builder.CreateCall(fnInfo.Type, fnInfo.Ptr, args, "")
		if isExternC {
			call.SetFunctionCallConv(llvm.CCallConv)
		}
		return call
	case *ast.TypeCastExpression:
		v := g.buildInternal(n.Argument, "")
		switch t := n.Typ.(type) {
		case *types.String:
			switch n.Argument.Type().(type) {
			case *types.Byte:
				// TODO: convert to string
				stringType := g.getLLVMType(n.Typ)
				intType := g.ctx.Int64Type()
				ptr := g.builder.CreateAlloca(stringType, "")
				size := llvm.ConstInt(intType, 1, false)

				internalType := g.makeArrayType(&types.Array{T: n.Argument.Type()}, 1)
				internal := g.builder.CreateAlloca(internalType, "")
				indices := []llvm.Value{
					llvm.ConstInt(intType, uint64(0), false),
					llvm.ConstInt(intType, uint64(0), false),
				}
				firstIndex := g.builder.CreateInBoundsGEP(internalType, internal, indices, "")
				g.builder.CreateStore(v, firstIndex)

				firstField := g.builder.CreateStructGEP(stringType, ptr, 0, "")
				g.builder.CreateStore(size, firstField)

				secondField := g.builder.CreateStructGEP(stringType, ptr, 1, "")
				g.builder.CreateStore(internal, secondField)

				return ptr
			// case *types.Char
			// TODO: converting char to string

			// for arrays we just return as semsis validated type
			case *types.Array:
				return v
			default:
				panic("this is a compiler error. please report")
			}
		case *types.Int:
			typ := g.getLLVMType(n.Typ)
			if t.Signed == 0 {
				return g.builder.CreateZExt(v, typ, "")
			}
			return g.builder.CreateSExt(v, typ, "")
		case *types.Byte:
			typ := g.getLLVMType(&types.ConstU8)
			return g.builder.CreateTrunc(v, typ, "")
		}
		return v
	case *ast.MatchExpressionStatement:
		g.scope.Push(MATCH)
		defer g.scope.Pop()

		var blocks []llvm.BasicBlock
		for range n.Cases {
			blocks = append(blocks, llvm.AddBasicBlock(g.builder.GetInsertBlock().Parent(), ""))
		}
		defaultBlock := llvm.AddBasicBlock(g.builder.GetInsertBlock().Parent(), "")
		end := llvm.AddBasicBlock(g.builder.GetInsertBlock().Parent(), "")

		// TODO: check if infix
		var scrutinee llvm.Value
		scrutineeName := ""
		switch exp := n.Scrutinee.(type) {
		case *ast.InfixExpression:
			scrutineeName = exp.Left.String()
			scrutinee = g.buildInternal(exp.Right, "")
		default:
			scrutineeName = exp.String()
			scrutinee = g.buildInternal(n.Scrutinee, "")
		}

		// TODO: in the future this will be in a function
		// when matching against error types, structs etc.
		scrutineeVal := scrutinee
		switch lt := n.Scrutinee.Type().(type) {
		case *types.Union:
			unionType := g.getLLVMType(lt)
			fieldPtr := g.builder.CreateStructGEP(unionType, scrutinee, 0, "")
			scrutineeVal = g.builder.CreateLoad(g.ctx.Int64Type(), fieldPtr, "")
		case *types.Enum:
			scrutineeVal = g.builder.CreateExtractValue(scrutinee, 0, "")
		case *types.Float:
			var t llvm.Type
			switch lt.Width {
			case 32:
				t = g.ctx.Int32Type()
			case 64:
				t = g.ctx.Int64Type()
			}
			scrutineeVal = g.builder.CreateBitCast(scrutinee, t, "")
		}
		swtch := g.builder.CreateSwitch(scrutineeVal, defaultBlock, len(blocks))
		for i, c := range n.Cases {
			var pattern llvm.Value
			switch l := c.Predicate.(type) {
			case *ast.TypeLiteral:
				// TODO: generate type descriptor
				id := g.library + n.Scrutinee.Type().String() + l.Type().String()
				pattern = g.generateTypeDescriptor(id)
			case *ast.Identifier:
				// if struct type name used to match in case
				// then its a union match
				if _, ok := l.Type().(*types.Struct); ok {
					id := g.library + n.Scrutinee.Type().String() + l.Type().String()
					pattern = g.generateTypeDescriptor(id)
				} else {
					pattern = g.buildInternal(c.Predicate, "")
				}
			default:
				switch lt := c.Predicate.Type().(type) {
				case *types.Union:
				case *types.Enum:
					v := g.buildInternal(c.Predicate, "")
					pattern = g.builder.CreateExtractValue(v, 0, "")
				case *types.Float:
					v := g.buildInternal(c.Predicate, "")
					var t llvm.Type
					switch lt.Width {
					case 32:
						t = g.ctx.Int32Type()
					case 64:
						t = g.ctx.Int64Type()
					}
					pattern = g.builder.CreateBitCast(v, t, "")
				// ints, bool and byte
				default:
					pattern = g.buildInternal(c.Predicate, "")
				}
			}
			swtch.AddCase(pattern, blocks[i])
		}

		// generate IR for body in each case
		var phiVals []llvm.Value
		for i, c := range n.Cases {
			block := blocks[i]
			g.builder.SetInsertPointAtEnd(block)

			g.varSt.Scope()

			// Literals shouldnt be stored in symbol table.
			// This generation and symbol table update is
			// only needed for unions so code in case block
			// can access underyling data
			_, isLiteral := n.Scrutinee.(ast.Literal)
			_, isUnion := n.Scrutinee.Type().(*types.Union)
			if !isLiteral && isUnion {
				unionType := g.getLLVMType(n.Scrutinee.Type())
				caseType := g.getLLVMType(c.Predicate.Type())
				underlyingVal := g.generateUnionDataAccess(scrutinee, unionType, caseType)
				if scrutineeName == "" {
					panic("this is a compiler error. please report!")
				}
				// TODO: store this in LHS
				g.varSt.Set(scrutineeName, Variable{Ptr: underlyingVal, Type: caseType})
			}

			var last llvm.Value
			for _, node := range c.Body {
				last = g.buildInternal(node, "")
			}
			phiVals = append(phiVals, last)

			// compile jump to end
			lastInst := g.builder.GetInsertBlock().LastInstruction()
			if lastInst.IsAReturnInst().IsNil() && lastInst.IsABranchInst().IsNil() {
				g.builder.CreateBr(end)
			}

			g.varSt.Unscope()

		}
		g.builder.SetInsertPointAtEnd(defaultBlock)
		// if default block not nil generate body otherwise
		// jump from default to end block
		if n.Default != nil && len(n.Default.Body) != 0 {
			var last llvm.Value
			for _, node := range n.Default.Body {
				last = g.buildInternal(node, "")
			}
			phiVals = append(phiVals, last)
		}
		lastInst := g.builder.GetInsertBlock().LastInstruction()
		if lastInst.IsAReturnInst().IsNil() && lastInst.IsABranchInst().IsNil() {
			g.builder.CreateBr(end)
		}
		blocks = append(blocks, defaultBlock)

		g.builder.SetInsertPointAtEnd(end)
		// if match used as expression compile phi instruction
		if !n.IsStatement {
			phi := g.builder.CreatePHI(phiVals[0].Type(), "phi")
			phi.AddIncoming(phiVals, blocks)
			return phi
		}
		return swtch
	case *ast.ForStatement:
		g.scope.Push(FOR)
		defer g.scope.Pop()

		if n.Assignment == nil && n.Condition == nil && n.Change == nil {
			return g.generateForInfinity(n)
		} else if n.Assignment == nil && n.Change == nil {
			return g.generateForBoolean(n)
		}

		g.buildInternal(n.Assignment, "")

		// Create main blocks in logical order
		loop := llvm.AddBasicBlock(g.builder.GetInsertBlock().Parent(), "")
		body := llvm.AddBasicBlock(g.builder.GetInsertBlock().Parent(), "")
		incr := llvm.AddBasicBlock(g.builder.GetInsertBlock().Parent(), "")
		after := llvm.AddBasicBlock(g.builder.GetInsertBlock().Parent(), "")

		g.loopScope.Push(LoopData{EndBlock: &after, IncrBlock: &incr})

		g.builder.CreateBr(loop)

		// Compile loop condition
		g.builder.SetInsertPointAtEnd(loop)
		cond := g.buildInternal(n.Condition, "")
		g.builder.CreateCondBr(cond, body, after)

		// Compile loop body
		g.builder.SetInsertPointAtEnd(body)
		// Create the loop body
		var last llvm.Value
		if n.Block != nil {
			last = g.buildInternal(n.Block, "")
		}

		g.builder.CreateBr(incr)

		// Compile increment or decrement
		g.builder.SetInsertPointAtEnd(incr)
		g.buildInternal(n.Change, "")
		g.builder.CreateBr(loop)

		// Set the insert point to the after block
		g.builder.SetInsertPointAtEnd(after)

		g.loopScope.Pop()
		return last
	case *ast.ForRangeStatement:
		panic("not implemented")
	case *ast.CopyExpression:
		return g.compileCopyExpression(n.Ident.String(), n.Type())
	case *ast.CopyUpdateExpression:
		g.scope.Push(COPY_UPDATE)
		defer g.scope.Pop()

		varPtr := g.compileCopyExpression(n.Ident.String(), n.Type())
		// set vartab with data from copy using variable name in assignemnt
		llvmType := g.getLLVMType(n.Type())
		// NOTE: name used as key as in body user updates ident in assignment
		g.varSt.Set(name, Variable{Type: llvmType, Ptr: varPtr})

		g.buildInternal(n.Block, "")

		return varPtr
	case *ast.UseExpression:
		g.scope.Push(USE)
		defer g.scope.Pop()
		// oldBlock := g.builder.GetInsertBlock()
		// useBlock := llvm.AddBasicBlock(g.builder.GetInsertBlock().Parent(), "")

		// g.loopScope.Push(LoopData{EndBlock: &oldBlock})

		// g.builder.CreateBr(useBlock)
		// g.builder.SetInsertPointAtEnd(useBlock)

		varPtr := g.buildInternal(n.Ident, "")
		llvmType := g.getLLVMType(n.Type())
		g.varSt.Set(n.Ident.String(), Variable{Type: llvmType, Ptr: varPtr})

		g.buildInternal(n.Block, "")

		// g.builder.CreateBr(oldBlock)
		// g.builder.SetInsertPointAtEnd(oldBlock)

		return varPtr
	case *ast.ArrayLiteral:
		// arrays are represented using struct: { len u64, ptr [ n x T ]}
		structType := g.getLLVMType(n.T)

		arrT, _ := n.T.(*types.Array)
		size := len(n.Values)
		arrayType := g.makeArrayType(arrT, size)

		// compile dynamic array
		var structPtr llvm.Value
		var arrayPtr llvm.Value
		g.doInEntry(func() {
			structPtr = g.builder.CreateAlloca(structType, "")
			arrayPtr = g.builder.CreateAlloca(arrayType, "")
		})

		// create length value
		length := llvm.ConstInt(g.ctx.Int64Type(), uint64(size), false)

		// set 1st struct field to length
		field1Ptr := g.builder.CreateStructGEP(structType, structPtr, 0, "")
		g.builder.CreateStore(length, field1Ptr)

		// compile array literal
		intType := g.getLLVMType(&types.ConstI64)
		intV := llvm.ConstInt(intType, uint64(0), false)
		for i, v := range n.Values {
			val := g.buildInternal(v, name)
			indices := []llvm.Value{intV, llvm.ConstInt(intType, uint64(i), false)}
			fieldPtr := g.builder.CreateInBoundsGEP(arrayType, arrayPtr, indices, "")
			g.builder.CreateStore(val, fieldPtr)
		}

		// set 2nd struct field to arrayPtr
		field2Ptr := g.builder.CreateStructGEP(structType, structPtr, 1, "")
		g.builder.CreateStore(arrayPtr, field2Ptr)
		// g.varSt.Set(name, Variable{Type: structType, Ptr: structPtr})
		return structPtr

		// NOTE: for simplicities sake we will just compile one array type (as struct)
		// In future we can have two ways to compile arrays to offer lower memory footprint
		// of fixed-size attars. But it would most likely require change to IndexExpression
		// converting it to recursive (DFS) expression to allow for parsing dynamic arrays
		// containing fixed size arrays.

		// } else {
		// 	// compile fixed-size array
		// 	var arrayPtr llvm.Value
		// 	g.doInEntry(func() {
		// 		arrayPtr = g.curBuilder.CreateAlloca(arrayType, "")
		// 	})

		// 	basePtr := llvm.ConstInt(g.get_llvm_type(&types.I64), 0, false)
		// 	indices := []llvm.Value{basePtr, {}}
		// 	for i, v := range n.Values {
		// 		val := g.compile(v, name)
		// 		indices[1] = llvm.ConstInt(g.get_llvm_type(&types.I64), uint64(i), false)
		// 		fieldPtr := g.curBuilder.CreateInBoundsGEP(arrayType, arrayPtr, indices, "")
		// 		g.curBuilder.CreateStore(val, fieldPtr)
		// 	}
		// 	g.vartab.Set(name, Variable{ Type: arrayType, Ptr: arrayPtr, Size: len(n.Values)})
		// 	return arrayPtr
		// }

	case *ast.IndexExpression:
		// if ok then it is most likely an array ident e.g. a = []

		fieldPtr := g.buildInternal(n.Left, "")
		// TODO: validate index within len using 1st struct field
		structType := g.getLLVMType(n.GetTypeAt(0))

		intType := g.getLLVMType(&types.ConstI64)
		intV := llvm.ConstInt(intType, uint64(0), false)
		for i, indexExp := range n.Indices {
			// Create GEP to 2nd struct field (which is array or struct (which means nested array))
			structFieldPtr := g.builder.CreateStructGEP(structType, fieldPtr, 1, "")
			// this loads the pointer to memory not the actual array
			ptrToArr := g.builder.CreateLoad(structFieldPtr.Type(), structFieldPtr, "")
			indices := []llvm.Value{intV, g.buildInternal(indexExp, "")}
			// We are cheating here by going out of bounds
			// but we check bounds using len
			arrType := g.makeArrayType(n.GetTypeAt(i), 1)
			fieldPtr = g.builder.CreateInBoundsGEP(arrType, ptrToArr, indices, "")
			if i != len(n.Indices)-1 {
				fieldPtr = g.builder.CreateLoad(fieldPtr.Type(), fieldPtr, "")
			}
		}

		fieldType := g.getLLVMType(n.Type())
		// if aggregate data structure only load pointer
		if fieldType.TypeKind() == llvm.StructTypeKind || fieldType.TypeKind() == llvm.ArrayTypeKind {
			fieldType = llvm.PointerType(fieldType, 0)
			return g.builder.CreateLoad(fieldType, fieldPtr, "")
		}
		return g.builder.CreateLoad(fieldType, fieldPtr, "")
		// return fieldPtr

		// NOTE: as only one type of array literal we can ignore below code
		// } else {
		// 	fieldPtr := list.Ptr
		// 	fieldType := list.Type
		// 	indices := make([]llvm.Value, 2)
		// 	// add base pointer
		// 	indices[0] = llvm.ConstInt(g.get_llvm_type(&types.I64), 0, false)
		// 	for i, indexExp := range n.Indices {
		// 		// val, ok := g.eval.IArith(indexExp)
		// 		// if ok {
		// 		// 	if val < 0 {
		// 		// 		val = list.Size - val
		// 		// 	}
		// 		// 	indices[1] = llvm.ConstInt(g.get_llvm_type(&types.I64), uint64(val), false)
		// 		// } else {
		// 		indices[1] = g.compile(indexExp, "")
		// 		// }

		// 		fieldType = g.get_llvm_type(n.GetTypeAt(i))
		// 		fieldPtr = g.curBuilder.CreateInBoundsGEP(fieldType, fieldPtr, indices, "")
		// 	}

		// 	fieldType = g.get_llvm_type(n.Type())

		// 	return g.curBuilder.CreateLoad(fieldType, fieldPtr, "")
		// }
	case *ast.DotExpression:
		var varPtr llvm.Value
		switch left := n.Left.(type) {
		// In case we have index expression that yields a struct or enum,
		// we need to first compile the array/list access
		case *ast.IndexExpression:
			varPtr = g.buildInternal(left, name)
		case *ast.FunctionCallExpression:
			// TODO:
		default:
			// TODO: this check is not required for enums as we get info later on
			// * we store enum type in vartab when parsing enum but this is only done to avoid panicking here
			v, ok := g.varSt.Get(n.Left.String())
			if !ok {
				panic("variable not found " + name)
			}
			varPtr = v.Ptr
		}

		switch left := n.Left.Type().(type) {
		case *types.Enum:
			v, ok := g.varSt.Get(n.String())
			if !ok {
				panic("variable not found " + name)
			}
			return v.Ptr
		case *types.Definition:
			// 1:1 copy of *types.Struct case as we need to be able
			// to compile type definitions of struct definitions.
			strct, ok := left.Underlying.(*types.Struct)
			if !ok {
				panic("compiler error. please report")
			}
			structPtr := varPtr
			structType := g.getLLVMType(n.Left.Type())

			var fieldType types.TypeSpec
			var fieldIndex int
			switch r := n.Right.(type) {
			// case for named fields
			case *ast.Identifier:
				fieldType, fieldIndex, _ = strct.GetTypeByField(r.String())

			// case for unnamed fields
			case *ast.IntegerLiteral:
				fieldType, _ = strct.GetTypeByIndex(int(r.Value))
				fieldIndex = int(r.Value)
			}

			fieldPtr := g.builder.CreateStructGEP(structType, structPtr, fieldIndex, "")

			fieldTypeLLVM := g.getLLVMType(fieldType)
			return g.builder.CreateLoad(fieldTypeLLVM, fieldPtr, "")
		case *types.Struct:
			var fieldType types.TypeSpec
			var fieldIndex int
			switch r := n.Right.(type) {
			// case for named fields
			case *ast.Identifier:
				fieldType, fieldIndex, _ = left.GetTypeByField(r.String())

			// case for unnamed fields
			case *ast.IntegerLiteral:
				fieldType, _ = left.GetTypeByIndex(int(r.Value))
				fieldIndex = int(r.Value)
			}
			llvmType := g.getLLVMType(left)
			return g.generateStructFieldAccess(varPtr, llvmType, fieldType, fieldIndex)
		case *types.Pointer:
			switch left := left.T.(type) {
			case *types.Struct:
				var fieldType types.TypeSpec
				var fieldIndex int
				switch r := n.Right.(type) {
				// case for named fields
				case *ast.Identifier:
					fieldType, fieldIndex, _ = left.GetTypeByField(r.String())

				// case for unnamed fields
				case *ast.IntegerLiteral:
					fieldType, _ = left.GetTypeByIndex(int(r.Value))
					fieldIndex = int(r.Value)
				}
				llvmType := g.getLLVMType(left)
				return g.generateStructFieldAccess(varPtr, llvmType, fieldType, fieldIndex)
			case *types.Enum:
				panic("this is a compiler error. please report")
			}
		case *types.AbstractStruct:
			// Calculate pointer to base pointer
			abstractTypeLLVM := g.getLLVMType(left)
			ptrToBasePtr := g.builder.CreateStructGEP(abstractTypeLLVM, varPtr, 0, "")
			basePtr := g.builder.CreateLoad(g.getLLVMType(&types.Pointer{T: &types.ConstI64}), ptrToBasePtr, "")
			// Calculate pointer to offset array
			arrayPtr := g.builder.CreateStructGEP(abstractTypeLLVM, varPtr, 1, "")

			// Calculate pointer to offset of field of underlying struct and load offset
			fieldType, fieldIndex, _ := left.GetTypeByField(n.Right.String())
			intType := g.ctx.Int64Type()
			indices := []llvm.Value{llvm.ConstInt(intType, uint64(fieldIndex), false)}
			arrayTypeLLVM := g.makeArrayType(&types.Array{T: &types.ConstI64}, len(left.Ts))
			arrayElemPtr := g.builder.CreateInBoundsGEP(arrayTypeLLVM, arrayPtr, indices, "")
			offset := g.builder.CreateLoad(intType, arrayElemPtr, "")

			// Calculate pointer to element using base ptr and offset and load field
			indices = []llvm.Value{offset}
			i8Type := g.getLLVMType(&types.ConstI8)
			fieldPtr := g.builder.CreateInBoundsGEP(i8Type, basePtr, indices, "")

			fieldTypeLLVM := g.getLLVMType(fieldType)
			return g.builder.CreateLoad(fieldTypeLLVM, fieldPtr, "")
		}
		panic("this is a compiler error. please report")
	case *ast.SliceExpression:
		fieldPtr := g.buildInternal(n.Left, "")
		structType := g.getLLVMType(n.Left.Type())
		// Create GEP to 2nd struct field (which is array or struct (which means nested array))
		structFieldPtr := g.builder.CreateStructGEP(structType, fieldPtr, 1, "")
		// This loads the pointer to memory not the actual array
		ptrToArr := g.builder.CreateLoad(structFieldPtr.Type(), structFieldPtr, "")

		infix, ok := n.Indices[0].(*ast.InfixExpression)
		if !ok {
			panic("start and end range need to be defined")
		}
		// Calculate new length
		rangeStart := g.buildInternal(infix.Left, "")
		rangeEnd := g.buildInternal(infix.Right, "")
		newLen := g.builder.CreateSub(rangeEnd, rangeStart, "")

		// Get new pointer offset
		indices := []llvm.Value{rangeStart}
		arrayType := g.makeArrayType(n.Type(), 1)
		newPtr := g.builder.CreateInBoundsGEP(arrayType, ptrToArr, indices, "")

		// Create new array header
		newStructPtr := empty
		g.doInEntry(func() {
			newStructPtr = g.builder.CreateAlloca(structType, "")
		})
		// Set length
		structLenFieldPtr := g.builder.CreateStructGEP(structType, newStructPtr, 0, "")
		g.builder.CreateStore(newLen, structLenFieldPtr)
		// Set underlying array ptr
		structPtrFieldPtr := g.builder.CreateStructGEP(structType, newStructPtr, 1, "")
		g.builder.CreateStore(newPtr, structPtrFieldPtr)

		return newStructPtr
	case *ast.IfElseExpression:
		return g.compileIfElseExpression(n)
	case *ast.BlockStatement:
		g.varSt.Scope()
		defer g.varSt.Unscope()

		var last llvm.Value
		for _, s := range n.Statements {
			last = g.buildInternal(s, "")
		}
		return last
	case *ast.StructLiteral:
		structType := g.getLLVMType(n.T)

		if g.scope.GetLast() == GLOBAL {
			vals := []llvm.Value{}
			for _, f := range n.Fields {
				val := g.buildInternal(f.Value, name)
				vals = append(vals, val)
			}
			strct := llvm.ConstNamedStruct(structType, vals)
			global := llvm.AddGlobal(g.mod, structType, "x")
			global.SetInitializer(strct)
			global.SetGlobalConstant(true)
			global.SetLinkage(llvm.InternalLinkage)
			return global
		}

		var structPtr llvm.Value
		g.doInEntry(func() {
			structPtr = g.builder.CreateAlloca(structType, "")
		})
		for _, f := range n.Fields {
			val := g.buildInternal(f.Value, name)
			fieldPtr := g.builder.CreateStructGEP(structType, structPtr, f.Index, "")
			g.builder.CreateStore(val, fieldPtr)
		}

		return structPtr
	case *ast.ReturnStatement:
		// Return statement lowers variables and literals return as
		// optional types to the coresponding optional struct literal
		retTypes := n.ReturnTypes()
		if len(retTypes) == 0 {
			return g.builder.CreateRetVoid()
		} else if len(retTypes) == 1 {
			fnType := g.fnTypeScope.GetLast()
			typ := fnType.GetReturnTypeAt(0)
			switch t := typ.(type) {
			case *types.Struct, *types.Union:
				offset := len(fnType.Arg)
				src := g.buildInternal(n.Values[0], "")

				fn := g.fnScope.GetLast()
				dst := fn.Param(offset)

				size := g.getWidth(g.getLLVMType(t))
				false := llvm.ConstInt(g.ctx.Int1Type(), 0, false)
				g.createCallMemCopy(dst, src, size, false)

				return g.builder.CreateRet(dst)
			case *types.Optional:
				v := g.generateOptionalFromLiteral(n.Values[0], t)
				return g.builder.CreateRet(v)
			default:
				v := g.buildInternal(n.Values[0], "")
				return g.builder.CreateRet(v)
			}
		} else {
			fnType := g.fnTypeScope.GetLast()
			var typs []llvm.Type
			cnt := 0
			for _, v := range n.Values {
				switch t := v.Type().(type) {
				case *types.Struct, *types.Union:
					offset := len(fnType.Arg) + cnt
					src := g.buildInternal(v, "")

					fn := g.fnScope.GetLast()
					dst := fn.Param(offset)

					size := g.getWidth(g.getLLVMType(t))
					false := llvm.ConstInt(g.ctx.Int1Type(), 0, false)
					g.createCallMemCopy(dst, src, size, false)

					cnt++
					typs = append(typs, dst.Type())
				case *types.Optional:
					typs = append(typs, g.getLLVMType(t))
				case *types.Multi:
					fn := g.fnScope.GetLast()
					ptr := g.buildInternal(v, "")
					for j, t := range t.Ts {
						src := g.builder.CreateExtractValue(ptr, j, "")
						offset := len(fnType.Arg) + cnt

						dst := fn.Param(offset)

						size := g.getWidth(g.getLLVMType(t))
						false := llvm.ConstInt(g.ctx.Int1Type(), 0, false)
						g.createCallMemCopy(dst, src, size, false)

						cnt++
						typs = append(typs, dst.Type())
					}
				default:
					typs = append(typs, g.getLLVMType(v.Type()))
				}
			}
			// This could occur if we return multiple structs
			if len(typs) == 0 {
				return g.builder.CreateRetVoid()
			}

			strctType := g.ctx.StructType(typs, false)
			v := llvm.ConstNull(strctType)
			for i, ret := range n.Values {
				switch t := ret.Type().(type) {
				case *types.Multi:
					ptr := g.buildInternal(ret, "")
					// TODO: extract
					j := i
					for range t.Ts {
						el := g.builder.CreateExtractValue(ptr, j-i, "")
						v = g.builder.CreateInsertValue(v, el, j, "")
						j++
					}

				case *types.Optional:
					opt := g.generateOptionalFromLiteral(n.Values[0], t)
					v = g.builder.CreateInsertValue(v, opt, i, "")
				default:
					ptr := g.buildInternal(ret, "")
					v = g.builder.CreateInsertValue(v, ptr, i, "")
				}
			}
			return g.builder.CreateRet(v)
		}
	case *ast.KeywordStatement:
		switch n.Token.Type {
		case token.BREAK:
			// jump to end block
			block := g.loopScope.GetLast()
			return g.builder.CreateBr(*block.EndBlock)
		case token.NEXT:
			// jump to loop block
			block := g.loopScope.GetLast()
			return g.builder.CreateBr(*block.IncrBlock)
		}
		panic("invalid keyword")
	case *ast.Identifier:
		v, ok := g.varSt.Get(n.String())
		if !ok {
			f, ok := g.fnSt.Get(n.String())
			if !ok {
				panic("identifier not found " + n.String())
			}
			return f.Ptr
		}
		// if we have a var we need to load it
		if v.IsVar {
			switch n.T.(type) {
			case *types.Struct, *types.String, *types.Array,
				*types.Pointer, *types.Optional, *types.Union:
				return v.Ptr
			}
			return g.builder.CreateLoad(v.Type, v.Ptr, "")
		}
		return v.Ptr
	case *ast.AssignmentStatement:
		for i := 0; i < len(n.Values); i++ {
			decl := n.Declerations[i]
			switch val := n.Values[i].(type) {
			case *ast.FunctionCallExpression:
				if len(val.ReturnTypes) == 1 {
					llvmVal := g.buildInternal(val, decl.Assignee.String())
					g.generateAssignment(decl, val.Type(), llvmVal)
				} else {
					fnReturnVals := g.buildInternal(val, decl.Assignee.String())
					for j := range val.ReturnTypes {
						decl := n.Declerations[i+j]
						llvmVal := g.builder.CreateExtractValue(fnReturnVals, j, "")
						g.generateAssignment(decl, val.ReturnTypes[j], llvmVal)
					}
				}
				// TODO: struct field dereference
				// case *ast.StructLiteral
			default:
				llvmVal := g.buildInternal(val, decl.Assignee.String())
				g.generateAssignment(decl, val.Type(), llvmVal)
			}
		}
		return empty
	case *ast.PrefixExpression:
		r := g.buildInternal(n.Right, "")
		switch n.Token.Type {
		case token.MINUS:
			switch r.Type() {
			case g.ctx.Int8Type(), g.ctx.Int16Type(), g.ctx.Int32Type(), g.ctx.Int64Type():
				return g.builder.CreateNeg(r, "neg")
			case g.ctx.FloatType(), g.ctx.DoubleType():
				return g.builder.CreateFNeg(r, "fneg")
			}
		case token.BANG:
			return g.builder.CreateNot(r, "not")
		case token.AMPERSAND:
			// switch n.Right.(type) {
			// // As StructLiteral generation returns pointer
			// // we can simply return 'r'
			// case *ast.StructLiteral:
			// 	return r
			// }
			// for types less than 64bits we simply return the value

			if r.Type().TypeKind() == llvm.PointerTypeKind {
				return r
			}

			// allocate value to stack and return the pointer
			var valPtr llvm.Value
			llvmType := g.getLLVMType(n.Right.Type())
			g.doInEntry(func() {
				valPtr = g.builder.CreateAlloca(llvmType, "")
			})
			g.builder.CreateStore(r, valPtr)

			// store dereference so we avoid generating another stack entry if called again,
			// as this would cause both pointers to be different. logically
			// g.varSt.Set(n.String(), Variable{Type: llvmType, Ptr: valPtr})

			return valPtr
		case token.ASTERISK:
			switch t := n.Right.Type().(type) {
			case *types.Pointer:
				// We never want to fully dereference aggregate data type. See:
				// https://llvm.org/docs/Frontend/PerformanceTips.html#avoid-creating-values-of-aggregate-type
				switch t.T.(type) {
				case *types.Struct, *types.Array, *types.Union:
					return r
				case *types.Int, *types.Float, *types.Bool,
					*types.Byte, *types.Char:

					ptr := g.builder.CreateLoad(r.Type(), r, "")
					llvmType := g.getLLVMType(n.Type())
					return g.builder.CreateLoad(llvmType, ptr, "")
				}
			default:
				panic("this is a compiler error. please report")
			}

			if r.Type().TypeKind() != llvm.PointerTypeKind {
				panic("this is a compiler error. please report")
			}

			llvmType := g.getLLVMType(n.Type())
			return g.builder.CreateLoad(llvmType, r, "")
		case token.OPTIONAL:
			return g.builder.CreateExtractValue(r, 1, "")
		}
	case *ast.InfixExpression:
		return g.compileInfix(n)
	case *ast.PostfixExpression:
		switch n.Token.Type {
		case token.INCR:
			val, ok := g.varSt.Get(n.Left.String())
			if !ok {
				panic("this is a compiler error. please report")
			}
			// Increment the loop variable
			i := g.builder.CreateLoad(val.Type, val.Ptr, "")
			loopVarInc := g.builder.CreateAdd(i, llvm.ConstInt(val.Type, 1, false), "")
			g.builder.CreateStore(loopVarInc, val.Ptr)
			return val.Ptr
		case token.DECR:
			val, ok := g.varSt.Get(n.Left.String())
			if !ok {
				panic("this is a compiler error. please report")
			}
			// Decrement loop variable
			i := g.builder.CreateLoad(val.Type, val.Ptr, "")
			loopVarInc := g.builder.CreateSub(i, llvm.ConstInt(val.Type, 1, false), "")
			g.builder.CreateStore(loopVarInc, val.Ptr)
			return val.Ptr
		}
	case *ast.BooleanLiteral:
		t := g.ctx.Int1Type()
		if n.Value {
			return llvm.ConstInt(t, 1, false)
		}
		return llvm.ConstInt(t, 0, false)
	case *ast.FloatLiteral:
		t := g.ctx.DoubleType()
		return llvm.ConstFloat(t, n.Value)
	case *ast.IntegerLiteral:
		t := g.getLLVMType(n.T)
		return llvm.ConstInt(t, uint64(n.Value), false)
	case *ast.CharacterLiteral:
		t := g.getLLVMType(n.Type())
		return llvm.ConstInt(t, uint64(n.Value), false)
	case *ast.StringLiteral:
		stringType := g.getLLVMType(n.Type())
		elType := g.ctx.Int8Type()
		var values []llvm.Value
		// count is required as using len(n.Values) will give
		// wrong llvm type as we filter escape sequences out
		var cnt int
		for i := 0; i < len(n.Value); i++ {
			if i < len(n.Value)-1 && n.Value[i] == '\\' && n.Value[i+1] == 'n' {
				values = append(values, llvm.ConstInt(elType, uint64('\n'), false))
				i++
			} else {
				values = append(values, llvm.ConstInt(elType, uint64(n.Value[i]), false))
			}
			cnt++
		}
		arrType := llvm.ArrayType(elType, cnt)
		constArray := llvm.ConstArray(arrType, values)
		globalArr := llvm.AddGlobal(g.mod, arrType, "")
		if len(n.Value) == 0 {
			globalArr.SetInitializer(llvm.ConstPointerNull(arrType))
		} else {
			globalArr.SetInitializer(constArray)
		}
		globalArr.SetGlobalConstant(true)
		globalArr.SetLinkage(llvm.InternalLinkage)

		length := llvm.ConstInt(g.ctx.Int64Type(), uint64(len(n.Value)), false)
		constStruct := llvm.ConstNamedStruct(stringType, []llvm.Value{length, globalArr})
		globalStruct := llvm.AddGlobal(g.mod, stringType, "")
		globalStruct.SetInitializer(constStruct)
		globalStruct.SetGlobalConstant(true)
		// TODO: if escapes then set to external
		globalStruct.SetLinkage(llvm.InternalLinkage)

		return globalStruct

	case *ast.NullLiteral:
		// It is expected for semsis to catch null
		// if n.Type() == &types.ConstNull {
		// 	panic("this is a compiler error. please report")
		// }
		optTypeLLVM := g.getLLVMType(n.Type())

		// Create null pointer for the value field
		false := llvm.ConstInt(g.ctx.Int1Type(), 0, false)
		agg := g.builder.CreateInsertValue(llvm.ConstNull(optTypeLLVM), false, 0, "")
		nullPtr := llvm.ConstPointerNull(g.ctx.Int64Type())
		agg = g.builder.CreateInsertValue(agg, nullPtr, 1, "")
		return agg
	case *ast.Comment:
		return empty
	case *ast.Unreachable:
		g.builder.CreateUnreachable()
	}

	panic("ast did not match: " + node.String())
}

// Builds function and ensures fn arguments are in proper symbol tables
func (g *Generator) createFunctionHeader(f *ast.FunctionExpression) (llvm.Value, llvm.Type) {
	name := f.Name.String()

	fnType := g.getLLVMType(f.Type())

	// All functions are defined using <lib name>.<fn name>
	if name != "main" && !f.HasAttribute(ast.ExternC) {
		name = g.library + "." + name
	}
	fn := llvm.AddFunction(g.mod, name, fnType)
	if f.HasAttribute(ast.ExternC) {
		fn.SetFunctionCallConv(llvm.CCallConv)
	} else {
		fn.SetFunctionCallConv(llvm.FastCallConv)
	}

	// NOTE: as we currently return the pointer it is not
	// required to set sret(<ty>) attribute
	// offset := len(f.Arguments)
	// for i, rt := range f.ReturnValues {
	// 	switch rt.T.(type) {
	// 	case *types.Struct:
	// 		typ := g.getLLVMType(rt.T)
	// 		fn.AddAttributeAtIndex(offset+i, g.getAttributeSret(typ))
	// 	}
	// }

	if f.Public {
		fn.SetLinkage(llvm.ExternalLinkage)
	} else {
		// Use 'Internal' to ensure fn shows as a local symbol in object file
		// https://llvm.org/docs/LangRef.html#linkage-types
		fn.SetLinkage(llvm.InternalLinkage)
	}

	return fn, fnType
}

func (g *Generator) generateForInfinity(n *ast.ForStatement) llvm.Value {
	// Create main blocks in logical order
	body := llvm.AddBasicBlock(g.builder.GetInsertBlock().Parent(), "")
	after := llvm.AddBasicBlock(g.builder.GetInsertBlock().Parent(), "")

	g.loopScope.Push(LoopData{EndBlock: &after, IncrBlock: &body})

	g.builder.CreateBr(body)

	// Compile body
	g.builder.SetInsertPointAtEnd(body)
	g.buildInternal(n.Block, "")

	// TODO: if last instruction is not return, or break
	g.builder.CreateBr(body)

	g.builder.SetInsertPointAtEnd(after)

	return empty
}

func (g *Generator) generateForBoolean(n *ast.ForStatement) llvm.Value {
	// Create main blocks in logical order
	loop := llvm.AddBasicBlock(g.builder.GetInsertBlock().Parent(), "")
	body := llvm.AddBasicBlock(g.builder.GetInsertBlock().Parent(), "")
	incr := llvm.AddBasicBlock(g.builder.GetInsertBlock().Parent(), "")
	after := llvm.AddBasicBlock(g.builder.GetInsertBlock().Parent(), "")

	g.loopScope.Push(LoopData{EndBlock: &after, IncrBlock: &incr})

	// jump to for loop
	g.builder.CreateBr(loop)

	// compile boolean condition
	g.builder.SetInsertPointAtEnd(loop)
	cond := g.buildInternal(n.Condition, "")
	g.builder.CreateCondBr(cond, body, after)

	// Compile body
	g.builder.SetInsertPointAtEnd(body)
	g.buildInternal(n.Block, "")
	g.builder.CreateBr(incr)

	// lastInst := body.LastInstruction()
	g.builder.SetInsertPointAtEnd(incr)
	g.builder.CreateBr(loop)

	g.builder.SetInsertPointAtEnd(after)

	g.loopScope.Pop()

	return empty
}

func (g *Generator) compileCopyExpression(ident string, typ types.TypeSpec) llvm.Value {
	varInfo, ok := g.varSt.Get(ident)
	if !ok {
		panic("this is a compiler error. please report")
	}
	switch typ := typ.(type) {
	case *types.Array:
		// As arrays are implemented as structs { len u64, ptr ptr }
		// we need to perform a deep copy of the struct manually

		oldStructPtr := varInfo.Ptr
		structType := varInfo.Type

		var size llvm.Value
		var length llvm.Value
		var structPtr llvm.Value
		var arrayPtr llvm.Value
		g.doInEntry(func() {
			// length and size need to go with alloca otherwise allocas
			// reference length and size but are defined before
			oldLenFieldPtr := g.builder.CreateStructGEP(structType, oldStructPtr, 0, "")
			length = g.builder.CreateLoad(g.ctx.Int64Type(), oldLenFieldPtr, "")
			arrayType := g.makeArrayType(typ, 1)
			size = g.builder.CreateMul(g.getWidth(arrayType), length, "")

			structPtr = g.builder.CreateAlloca(structType, "")
			internalType := g.getLLVMType(typ.T)
			arrayPtr = g.builder.CreateArrayAlloca(internalType, size, "")
		})

		// copy length from old array to new
		newLenPtr := g.builder.CreateStructGEP(structType, structPtr, 0, "")
		g.builder.CreateStore(length, newLenPtr)

		// set new array ptr to new header
		newArrayPtr := g.builder.CreateStructGEP(structType, structPtr, 1, "")
		g.builder.CreateStore(arrayPtr, newArrayPtr)

		// memory not volatile so we set to 'false'
		false := llvm.ConstInt(g.ctx.Int1Type(), 0, false)

		// copy old struct to new one
		oldArrayFieldPtr := g.builder.CreateStructGEP(structType, oldStructPtr, 1, "")
		ptrType := g.getLLVMType(&types.Pointer{T: typ.T})
		oldArrayPtr := g.builder.CreateLoad(ptrType, oldArrayFieldPtr, "")
		g.createCallMemCopy(arrayPtr, oldArrayPtr, size, false)

		return structPtr
	case *types.Struct:
		// TODO: check if escapes
		var destPtr llvm.Value
		g.doInEntry(func() {
			destPtr = g.builder.CreateAlloca(varInfo.Type, "")
		})

		size := g.getWidth(varInfo.Type)

		false := llvm.ConstInt(g.ctx.Int1Type(), 0, false)

		g.createCallMemCopy(destPtr, varInfo.Ptr, size, false)
		return destPtr
	// case *types.Pointer:
	// case *types.String:
	default:
		// for ints, floats, bool, byte and char just return underlying data
		// as there is no need to make copy.
		return varInfo.Ptr
	}
}

// Compiles if else expressions and statements
func (g *Generator) compileIfElseExpression(n *ast.IfElseExpression) llvm.Value {
	var blocks []llvm.BasicBlock
	blocks = append(blocks, g.builder.GetInsertBlock())
	for i := 0; i < len(n.Conditionals)*2; i++ {
		b := llvm.AddBasicBlock(g.builder.GetInsertBlock().Parent(), "")
		blocks = append(blocks, b)
		// skip adding conditional block for else
		if n.Conditionals[i/2].Token.Type == token.ELSE {
			i++
		}
	}

	var vals []llvm.Value
	var phiBlocks []llvm.BasicBlock

	for i := 0; i < len(n.Conditionals); i++ {
		cond := n.Conditionals[i]

		var condb llvm.BasicBlock
		var bodyb llvm.BasicBlock
		var nextb llvm.BasicBlock
		if cond.Token.Type == token.ELSE {
			bodyb = blocks[i*2]
			nextb = blocks[i*2+1]
		} else {
			condb = blocks[i*2]
			g.builder.SetInsertPointAtEnd(condb)
			bodyb = blocks[i*2+1]
			nextb = blocks[i*2+2]
			condition := g.buildInternal(cond.Condition, "")
			g.builder.CreateCondBr(condition, bodyb, nextb)
		}

		g.builder.SetInsertPointAtEnd(bodyb)

		v := g.buildInternal(cond.Block, "")
		if v.IsAReturnInst().IsNil() && v.IsABranchInst().IsNil() {
			g.builder.CreateBr(blocks[len(blocks)-1])
			// TODO: if this branch runs it means if else if expression did not have
			// an else clause this means we return an optional value
			if i*2+2 == len(blocks)-1 {
				vals = append(vals, v)
				phiBlocks = append(phiBlocks, condb)
			}
			vals = append(vals, v)
			phiBlocks = append(phiBlocks, bodyb)
		}
		g.builder.SetInsertPointAtEnd(nextb)

	}
	if len(vals) > 0 && !n.IsStatement {
		phi := g.builder.CreatePHI(vals[0].Type(), "phi")
		phi.AddIncoming(vals, phiBlocks)
		return phi
	}

	// return empty llvm value as we are generating a if else statement
	return empty
}

func (g *Generator) generateAssignment(decl *ast.DeclarationStatement, valType types.TypeSpec, val llvm.Value) {
	name := decl.Assignee.String()
	declType := decl.Assignee.Type()
	underlying := types.GetUnderlyingType(valType)
	if _, ok := underlying.(*types.Function); ok {
		llvmType := g.getLLVMType(underlying)
		g.fnSt.Set(decl.Assignee.String(), Function{Type: llvmType, Ptr: val, TypeDash: underlying})
	}
	if decl.Token.Type == token.VAR {
		typ := g.getLLVMType(declType)

		switch underlying.(type) {
		// structs and arrays are already stack allocated
		case *types.Struct, *types.Array, *types.Union:
			g.varSt.Set(name, Variable{Type: typ, Ptr: val, IsVar: true})
		default:
			var ptr llvm.Value
			g.doInEntry(func() {
				ptr = g.builder.CreateAlloca(val.Type(), "")
			})
			g.builder.CreateStore(val, ptr)
			g.varSt.Set(name, Variable{Type: typ, Ptr: ptr, IsVar: true})
		}
	} else if decl.Token.Type == token.LET {
		typ := g.getLLVMType(declType)
		g.varSt.Set(name, Variable{Type: typ, Ptr: val})
	} else {
		g.generateReassignment(decl, valType, val)
	}
}

func (g *Generator) generateReassignment(decl *ast.DeclarationStatement, valType types.TypeSpec, val llvm.Value) {
	varInfo, ok := g.varSt.Get(decl.Assignee.String())
	if ok && varInfo.IsVar {
		// NOTE: This condition captures a wider set of cases than required.
		// We only want to store if the for loop updates a variable defined
		// using `var` outside of the loops scope.
		// if g.scope.IsIn(FOR) || g.scope.IsIn(MATCH) {
		underlyingType := types.GetUnderlyingType(decl.Assignee.Type())
		// for aggregate types we need to ensure all underlying data is copied
		// not just pointer.
		switch uT := underlyingType.(type) {
		case *types.Array:
			firstPtr := g.builder.CreateStructGEP(varInfo.Type, val, 0, "")
			length := g.builder.CreateLoad(g.ctx.Int64Type(), firstPtr, "")
			elementWidth := g.getWidth(g.getLLVMType(uT.InternalType()))
			size := g.builder.CreateMul(elementWidth, length, "")
			false := llvm.ConstInt(g.ctx.Int1Type(), 0, false)
			g.createCallMemCopy(varInfo.Ptr, val, size, false)
		case *types.String:
			panic("not implemented yet.")
		case *types.Struct, *types.Union:
			size := g.getWidth(g.getLLVMType(valType))
			false := llvm.ConstInt(g.ctx.Int1Type(), 0, false)
			g.createCallMemCopy(varInfo.Ptr, val, size, false)
		default:
			g.builder.CreateStore(val, varInfo.Ptr)
		}
		// } else {
		// 	g.varSt.Set(decl.Assignee.String(), Variable{Type: val.Type(), Ptr: val, IsVar: true})
		// }
		return
	}
	if !g.scope.IsIn(USE) && !g.scope.IsIn(COPY_UPDATE) {
		panic("this is a compiler error. please report")
	}
	switch left := decl.Assignee.(type) {
	// For identifier assignment in copy_update or use expression
	// we simply store pointer of RHS to LHS.
	// TODO: this process can be optimised as we currently generate
	// instructions to copy RHS before this block for COPY_UPDATE
	case *ast.Identifier:
		varInfo, ok := g.varSt.Get(left.Value)
		if !ok {
			panic("this is a compiler error. please report")
		}

		var size llvm.Value
		switch t := left.Type().(type) {
		case *types.Array:
			// TODO: bounds check
			firstPtr := g.builder.CreateStructGEP(varInfo.Type, val, 0, "")
			length := g.builder.CreateLoad(g.ctx.Int64Type(), firstPtr, "")
			elementWidth := g.getWidth(g.getLLVMType(t.InternalType()))
			size = g.builder.CreateMul(elementWidth, length, "")
		default:
			size = g.getWidth(varInfo.Type)
		}

		false := llvm.ConstInt(g.ctx.Int1Type(), 0, false)
		g.createCallMemCopy(varInfo.Ptr, val, size, false)

	case *ast.IndexExpression:
		ptr := g.compileGEPArray(left)
		g.builder.CreateStore(val, ptr)
	case *ast.DotExpression:
		ptr := g.compileGEPStruct(left)
		g.builder.CreateStore(val, ptr)

	// In case of slice we know RHS must be an array due to semsis.
	case *ast.SliceExpression:
		ptr := g.compileGEPSlice(left)

		arrayType := g.makeArrayType(left.Type(), 1)

		llvmType := g.getLLVMType(valType)
		firstPtr := g.builder.CreateStructGEP(llvmType, val, 0, "")
		length := g.builder.CreateLoad(g.ctx.Int64Type(), firstPtr, "")
		secondPtr := g.builder.CreateStructGEP(llvmType, val, 1, "")
		ptrType := llvm.PointerType(arrayType, 0)
		array := g.builder.CreateLoad(ptrType, secondPtr, "")

		// copy compact array to pointer
		size := g.builder.CreateMul(g.getWidth(arrayType), length, "")
		false := llvm.ConstInt(g.ctx.Int1Type(), 0, false)
		g.createCallMemCopy(ptr, array, size, false)
	}
	return
}

func (g *Generator) generateStructFieldAccess(structPtr llvm.Value, structType llvm.Type, fieldType types.TypeSpec, fieldIndex int) llvm.Value {
	fieldPtr := g.builder.CreateStructGEP(structType, structPtr, fieldIndex, "")
	switch fieldType.(type) {
	case *types.String, *types.Struct, *types.Union, *types.Array:
		fieldTypeLLVM := llvm.PointerType(g.getLLVMType(fieldType), 0)
		return g.builder.CreateLoad(fieldTypeLLVM, fieldPtr, "")
		// return fieldPtr
	default:
		fieldTypeLLVM := g.getLLVMType(fieldType)
		return g.builder.CreateLoad(fieldTypeLLVM, fieldPtr, "")
	}
}

// Generates relevant IR to be able to go from struct to abstract struct
func (g *Generator) compileStructToAbstract(exp ast.Expression, structType *types.Struct, abstractType *types.AbstractStruct) llvm.Value {
	basePtr := g.buildInternal(exp, "")

	// Create function
	g.createPtrDiff()

	// TODO: if escapes allocate to heap

	abstractTypeLLVM := g.getLLVMType(abstractType)
	var structPtr llvm.Value
	g.doInEntry(func() {
		// { base ptr, offsets [n x i64] }
		structPtr = g.builder.CreateAlloca(abstractTypeLLVM, "")
	})

	//  add basePtr to struct
	firstFieldPtr := g.builder.CreateStructGEP(abstractTypeLLVM, structPtr, 0, "")
	g.builder.CreateStore(basePtr, firstFieldPtr)

	// add offsets to struct
	structTypeLLVM := g.getLLVMType(structType)
	secondFieldPtr := g.builder.CreateStructGEP(abstractTypeLLVM, structPtr, 1, "")
	secondFieldType := g.makeArrayType(&types.Array{T: &types.ConstI64}, len(abstractType.Ts))
	intType := g.ctx.Int64Type()
	for i, sf := range structType.Ts {
		for j, af := range abstractType.Ts {
			if sf.Name == af.Name {
				elemPtr := g.builder.CreateStructGEP(structTypeLLVM, basePtr, i, "")
				offset := g.createCallPtrDiff(elemPtr, basePtr)

				indices := []llvm.Value{llvm.ConstInt(intType, uint64(j), false)}
				arrayElemPtr := g.builder.CreateInBoundsGEP(secondFieldType, secondFieldPtr, indices, "")

				g.builder.CreateStore(offset, arrayElemPtr)
			}
		}
	}

	return structPtr

}

// Generates optional value for an identifier or literal unless alreafy optional
// If expression is NullLiteral or of type Optional we skip conversion and buildInternal
func (g *Generator) generateOptionalFromLiteral(exp ast.Expression, typ *types.Optional) llvm.Value {
	// if argument already optional we skip conversion
	if _, ok := exp.Type().(*types.Optional); ok {
		return g.buildInternal(exp, "")
	} else if _, ok := exp.(*ast.NullLiteral); ok {
		return g.buildInternal(exp, "")
	}

	optTypeLLVM := g.getLLVMType(typ)
	true := llvm.ConstInt(g.ctx.Int1Type(), 1, false)
	val := g.buildInternal(exp, "")

	agg := g.builder.CreateInsertValue(llvm.ConstNull(optTypeLLVM), true, 0, "")
	agg = g.builder.CreateInsertValue(agg, val, 1, "")

	return agg
}

func (g *Generator) compileInfix(n *ast.InfixExpression) llvm.Value {
	l := g.buildInternal(n.Left, "")
	r := g.buildInternal(n.Right, "")

	// There are two types of infix expressions:
	// * immutable: e.g. arithmetic operations
	// * pseudo mutable: e.g. reassignments in CopyUpdateExpressions
	switch n.Token.Type {
	case token.PIPE:
		// TODO:
	case token.NULL_COALESCE:
		// semsis validates that LHS optional value so we can
		// directly use 'l'
		isSet := g.builder.CreateExtractValue(l, 0, "")
		val := g.builder.CreateExtractValue(l, 1, "")

		true := llvm.ConstInt(g.ctx.Int1Type(), 1, false)
		cmp := g.builder.CreateICmp(llvm.IntEQ, true, isSet, "")
		return g.builder.CreateSelect(cmp, val, r, "")
	case token.PLUS:
		switch n.Left.Type().(type) {
		case *types.Int, *types.Byte, *types.Char:
			return g.builder.CreateAdd(l, r, "add")
		case *types.Float:
			return g.builder.CreateFAdd(l, r, "fadd")
		case *types.String:
			strTypeLLVM := g.getLLVMType(&types.ConstString)

			len1Ptr := g.builder.CreateStructGEP(strTypeLLVM, l, 0, "")
			len1 := g.builder.CreateLoad(g.ctx.Int64Type(), len1Ptr, "")

			len2Ptr := g.builder.CreateStructGEP(strTypeLLVM, r, 0, "")
			len2 := g.builder.CreateLoad(g.ctx.Int64Type(), len2Ptr, "")
			// get length of both strings

			totalLength := g.builder.CreateAdd(len1, len2, "")

			// allocate enough space for both strings
			var newStrPtr llvm.Value
			var arrayPtr llvm.Value
			g.doInEntry(func() {
				arrayPtr = g.builder.CreateArrayAlloca(g.ctx.Int8Type(), totalLength, "")
				newStrPtr = g.builder.CreateAlloca(strTypeLLVM, "")
			})

			// initialise new string header
			field1Ptr := g.builder.CreateStructGEP(strTypeLLVM, newStrPtr, 0, "")
			g.builder.CreateStore(totalLength, field1Ptr)

			field2Ptr := g.builder.CreateStructGEP(strTypeLLVM, newStrPtr, 1, "")
			g.builder.CreateStore(arrayPtr, field2Ptr)

			return g.createCallStrConcat2(newStrPtr, l, r)
		default:
			panic("this is a compiler error. please report")
		}

	case token.MINUS:
		switch l.Type() {
		case g.ctx.FloatType(), g.ctx.DoubleType():
			return g.builder.CreateFSub(l, r, "fsub")
		}
		return g.builder.CreateSub(l, r, "sub")

	case token.ASTERISK:
		switch l.Type() {
		case g.ctx.FloatType(), g.ctx.DoubleType():
			return g.builder.CreateFMul(l, r, "fmul")
		}
		return g.builder.CreateMul(l, r, "mul")
	case token.SLASH:
		switch l.Type() {
		case g.ctx.Int8Type(), g.ctx.Int16Type(), g.ctx.Int32Type(), g.ctx.Int64Type():
			// TODO: handle unisgned
			return g.builder.CreateSDiv(l, r, "sdiv")
		case g.ctx.FloatType(), g.ctx.DoubleType():
			return g.builder.CreateFDiv(l, r, "fdiv")
		}
	case token.MOD:
		switch l.Type() {
		case g.ctx.FloatType(), g.ctx.DoubleType():
			return g.builder.CreateFRem(l, r, "frem")
		}
		return g.builder.CreateSRem(l, r, "srem")
	case token.EQ:
		switch n.Left.Type().(type) {
		// handle case where both sides are null
		case nil:
			switch n.Right.Type().(type) {
			case nil:
				return llvm.ConstInt(g.ctx.Int1Type(), 1, false)
			}
			// semsis should have swapped NullLiteral with RHS
			panic("this is a compiler error. please report")
		case *types.Optional:
			switch n.Right.Type().(type) {
			case *types.Null:
				isDefined := g.builder.CreateExtractValue(l, 0, "")
				null := llvm.ConstInt(g.ctx.Int1Type(), 0, false)
				return g.builder.CreateICmp(llvm.IntEQ, isDefined, null, "")
			case *types.Optional:
				// TODO: extract val from LHS optional
				// TODO: extract val from RHS optional
				// TODO: generate correct comparison
				// return g.generateOptionalCmp(l, r, t)
				panic("this is a compiler error. please report")
			default:
				fieldDefined := g.builder.CreateExtractValue(l, 0, "")
				true := llvm.ConstInt(g.ctx.Int1Type(), 1, false)
				resDefined := g.builder.CreateICmp(llvm.IntEQ, fieldDefined, true, "")
				fieldVal := g.builder.CreateExtractValue(l, 1, "")
				resValueMatch := g.builder.CreateICmp(llvm.IntEQ, fieldVal, r, "")
				return g.builder.CreateAnd(resDefined, resValueMatch, "")
			}
		case *types.Array:
			// TODO: ensure semsis catches comparison between different types
			// TODO: call runtime.arr_cmp
		case *types.Struct:
			// TODO: how should we handle this?
		case *types.Enum:
			l = g.builder.CreateExtractValue(l, 0, "")
			r = g.builder.CreateExtractValue(r, 0, "")
			return g.builder.CreateICmp(llvm.IntEQ, l, r, "eq")
		case *types.Pointer:
			return g.builder.CreateICmp(llvm.IntEQ, l, r, "eq")
		case *types.Int, *types.Bool, *types.Byte, *types.Char:
			return g.builder.CreateICmp(llvm.IntEQ, l, r, "eq")
		case *types.Float:
			return g.builder.CreateFCmp(llvm.FloatOEQ, l, r, "feq")
		case *types.String:
			return g.createCallStrCmp(l, r)
		}
	case token.NEQ:
		switch n.Left.Type().(type) {
		case nil:
			switch n.Right.Type().(type) {
			case nil:
				return llvm.ConstInt(g.ctx.Int1Type(), 0, false)
			}
			// semsis should have swapped NullLiteral with RHS
			panic("this is a compiler error. please report")
		case *types.Optional:
			switch n.Right.Type().(type) {
			case *types.Null:
				isDefined := g.builder.CreateExtractValue(l, 0, "")
				null := llvm.ConstInt(g.ctx.Int1Type(), 0, false)
				return g.builder.CreateICmp(llvm.IntNE, isDefined, null, "")
			case *types.Optional:
				// TODO: extract val from LHS optional
				// TODO: extract val from RHS optional
				// TODO: generate correct comparison
				// return g.generateOptionalCmp(l, r, t)
				panic("this is a compiler error. please report")
			default:
				isDefined := g.builder.CreateExtractValue(l, 0, "")
				null := llvm.ConstInt(g.ctx.Int1Type(), 0, false)
				resNotDefined := g.builder.CreateICmp(llvm.IntEQ, isDefined, null, "")
				val := g.builder.CreateExtractValue(l, 1, "")
				resValueNotMatch := g.builder.CreateICmp(llvm.IntNE, val, r, "")
				return g.builder.CreateOr(resNotDefined, resValueNotMatch, "")
			}
		case *types.Array:
			// TODO: ensure semsis catches comparison between different types
			// TODO: call runtime.arr_cmp
		case *types.Struct:
			// TODO: how should we handle this?
		case *types.Enum:
			l = g.builder.CreateExtractValue(l, 0, "")
			r = g.builder.CreateExtractValue(r, 0, "")
			return g.builder.CreateICmp(llvm.IntNE, l, r, "ne")
		case *types.Pointer:
			return g.builder.CreateICmp(llvm.IntNE, l, r, "ne")
		case *types.Int, *types.Bool, *types.Byte, *types.Char:
			return g.builder.CreateICmp(llvm.IntNE, l, r, "ne")
		case *types.Float:
			return g.builder.CreateFCmp(llvm.FloatONE, l, r, "fne")
		case *types.String:
			one := llvm.ConstInt(g.ctx.Int1Type(), 1, false)
			return g.builder.CreateXor(g.createCallStrCmp(l, r), one, "")
		}
	case token.GT:
		switch n.Left.Type().(type) {
		case *types.Enum:
			l = g.builder.CreateExtractValue(l, 0, "")
			r = g.builder.CreateExtractValue(r, 0, "")
			return g.builder.CreateICmp(llvm.IntSGT, l, r, "gt")
		case *types.Float:
			return g.builder.CreateFCmp(llvm.FloatOGT, l, r, "fgt")
		case *types.Int, *types.Bool, *types.Byte, *types.Char, *types.Dirty, *types.Definition:
			return g.builder.CreateICmp(llvm.IntSGT, l, r, "gt")
		}
	case token.LT:
		switch n.Left.Type().(type) {
		case *types.Enum:
			l = g.builder.CreateExtractValue(l, 0, "")
			r = g.builder.CreateExtractValue(r, 0, "")
			return g.builder.CreateICmp(llvm.IntSLT, l, r, "lt")
		case *types.Float:
			return g.builder.CreateFCmp(llvm.FloatOLT, l, r, "flt")
		case *types.Int, *types.Bool, *types.Byte, *types.Char, *types.Dirty, *types.Definition:
			return g.builder.CreateICmp(llvm.IntSLT, l, r, "lt")
		}
	case token.GTE:
		switch n.Left.Type().(type) {
		case *types.Enum:
			l = g.builder.CreateExtractValue(l, 0, "")
			r = g.builder.CreateExtractValue(r, 0, "")
			return g.builder.CreateICmp(llvm.IntSGE, l, r, "ge")
		case *types.Float:
			return g.builder.CreateFCmp(llvm.FloatOGE, l, r, "fge")
		case *types.Int, *types.Bool, *types.Byte, *types.Char, *types.Dirty, *types.Definition:
			return g.builder.CreateICmp(llvm.IntSGE, l, r, "ge")
		}
	case token.LTE:
		switch n.Left.Type().(type) {
		case *types.Enum:
			l = g.builder.CreateExtractValue(l, 0, "")
			r = g.builder.CreateExtractValue(r, 0, "")
			return g.builder.CreateICmp(llvm.IntSLE, l, r, "le")
		case *types.Float:
			return g.builder.CreateFCmp(llvm.FloatOLE, l, r, "fle")
		case *types.Int, *types.Bool, *types.Byte, *types.Char, *types.Dirty, *types.Definition:
			return g.builder.CreateICmp(llvm.IntSLE, l, r, "le")
		}
	case token.OR:
		return g.builder.CreateOr(l, r, "or")
	case token.AND:
		return g.builder.CreateAnd(l, r, "and")
	case token.ASSIGN:
		panic("unreachable")
	default:
		panic("unknown infix expression. this is a compiler error. please report.")

	}
	panic("unknown infix expression. this is a compiler error. please report.")
}

// Generates the pointer to specific index of array
func (g *Generator) compileGEPArray(exp *ast.IndexExpression) llvm.Value {
	arrayInfo, ok := g.varSt.Get(exp.Left.String())
	if !ok {
		panic("this is a compiler error. please report")
	}
	arrayType := arrayInfo.Type
	arrayPtr := arrayInfo.Ptr

	intType := g.getLLVMType(&types.ConstI64)
	intV := llvm.ConstInt(intType, 0, false)
	for i, indexExp := range exp.Indices {
		// Create GEP to 2nd struct field (which is array or struct (which means nested array))
		structFieldPtr := g.builder.CreateStructGEP(arrayType, arrayPtr, 1, "")
		// this loads the pointer to memory not the actual array
		ptrToArr := g.builder.CreateLoad(structFieldPtr.Type(), structFieldPtr, "")
		indices := []llvm.Value{intV, g.buildInternal(indexExp, "")}
		// We are cheating here by going out of bounds
		// but we check bounds using len
		arrType := g.makeArrayType(exp.GetTypeAt(i), 1)
		arrayPtr = g.builder.CreateInBoundsGEP(arrType, ptrToArr, indices, "")
		if i != len(exp.Indices)-1 {
			arrayPtr = g.builder.CreateLoad(arrayPtr.Type(), arrayPtr, "")
		}
	}

	// ptrType := g.get_llvm_type(&types.Pointer{T: &types.I64})
	return arrayPtr
	// return g.curBuilder.CreateLoad(ptrType, arrayPtr, "")
}

func (g *Generator) compileGEPStruct(exp *ast.DotExpression) llvm.Value {
	structInfo, ok := g.varSt.Get(exp.Left.String())
	if !ok {
		panic("this is a compiler error. please report")
	}
	structType := structInfo.Type
	structPtr := structInfo.Ptr

	switch left := exp.Left.Type().(type) {
	case *types.Struct:
		var fieldIndex int
		switch r := exp.Right.(type) {
		// case for named fields
		case *ast.Identifier:
			_, fieldIndex, _ = left.GetTypeByField(r.String())

		// case for unnamed fields
		case *ast.IntegerLiteral:
			fieldIndex = int(r.Value)
		}

		return g.builder.CreateStructGEP(structType, structPtr, fieldIndex, "")
	default:
		panic("this is a compiler error. please report")
	}
}

// Returns a pointer to the offset in array according to beginning of slice range
func (g *Generator) compileGEPSlice(exp *ast.SliceExpression) llvm.Value {
	sliceInfo, ok := g.varSt.Get(exp.Left.String())
	if !ok {
		panic("this is a compiler error. please report")
	}
	sliceType := sliceInfo.Type
	slicePtr := sliceInfo.Ptr

	// Load ptr to underlying array from array header
	structFieldPtr := g.builder.CreateStructGEP(sliceType, slicePtr, 1, "")
	ptrToArr := g.builder.CreateLoad(structFieldPtr.Type(), structFieldPtr, "")

	infix, ok := exp.Indices[0].(*ast.InfixExpression)
	if !ok {
		panic("start and end range need to be defined")
	}
	// Get new pointer offset
	rangeStart := g.buildInternal(infix.Left, "")
	indices := []llvm.Value{rangeStart}

	// TODO:
	var internalType llvm.Type
	switch t := exp.Type().(type) {
	case *types.Array:
		internalType = g.getLLVMType(t.T)
	case *types.String:
		internalType = g.getLLVMType(&types.ConstU8)
	case *types.Definition:
		switch t := t.Underlying.(type) {
		case *types.Array:
			internalType = g.getLLVMType(t.T)
		case *types.String:
			internalType = g.getLLVMType(&types.ConstU8)
		default:
			panic("this is a compiler error. please report")
		}
	default:
		panic("this is a compiler error. please report")
	}
	return g.builder.CreateInBoundsGEP(internalType, ptrToArr, indices, "")
}

// Compiles compact array without array header e.g. llvm [n x T]
// returning the pointer to the array and the length if it was
// array literal.
func (g *Generator) compileCompactArray(n *ast.ArrayLiteral) (llvm.Value, int) {
	// TODO: check if escapes
	arrayType := g.makeArrayType(n.Type(), len(n.Values))

	// allocate to stack
	var arrayPtr llvm.Value
	g.doInEntry(func() {
		arrayPtr = g.builder.CreateAlloca(arrayType, "")
	})

	// compile array literal
	intType := g.getLLVMType(&types.ConstI64)
	intV := llvm.ConstInt(intType, uint64(0), false)
	for i, v := range n.Values {
		val := g.buildInternal(v, "")
		indices := []llvm.Value{intV, llvm.ConstInt(intType, uint64(i), false)}
		fieldPtr := g.builder.CreateInBoundsGEP(arrayType, arrayPtr, indices, "")
		g.builder.CreateStore(val, fieldPtr)
	}

	return arrayPtr, len(n.Values)
}

func (g *Generator) generateDashToC(typ types.TypeSpec, val llvm.Value) llvm.Value {
	// TODO: compile expression

	switch t := typ.(type) {
	case *types.Pointer:
		switch innerT := t.T.(type) {
		case *types.Array, *types.String:
			arrTypeLLVM := g.getLLVMType(t.T)
			eleT := g.getLLVMType(&types.ConstU8)
			internalArrType := llvm.PointerType(llvm.ArrayType(eleT, 1), 0)
			internalArrPtr := g.builder.CreateStructGEP(arrTypeLLVM, val, 1, "")
			// this loads the pointer to memory not the actual array
			ptrToArr := g.builder.CreateLoad(internalArrType, internalArrPtr, "")
			return ptrToArr
		case *types.Memory:
			// As type is pointer to memory we need to convert this to
			// pointer to underlying type e.g. '*memory<[]byte>' would become
			// '*[]byte' which in terms of C types would be 'char*'
			// BUG: causes stack overflow
			newT := &types.Pointer{T: innerT.T}
			return g.generateDashToC(newT, val)
		case *types.Struct:
			panic("struct to c not supported yet")
		// all other types are not generated as pointer
		default:
			panic(t.String() + " to c type not supported yet")
		}
		// switch innerT := t.T.(type) {
		// // do nothing as type already matches c
		// case *types.Int, *types.Float, *types.Bool, *types.Byte:
		// 	return val
		// case *types.Array, *types.String:
		// 	// extract internal array
		// 	arrTypeLLVM := g.getLLVMType(innerT)
		// 	internalArrPtr := g.builder.CreateStructGEP(arrTypeLLVM, val, 1, "")
		// 	// this loads the pointer to memory not the actual array
		// 	eleT := g.getLLVMType(&types.ConstU8)
		// 	internalArrType := llvm.PointerType(llvm.ArrayType(eleT, 1), 0)
		// 	ptrToArr := g.builder.CreateLoad(internalArrType, internalArrPtr, "")
		// 	return ptrToArr
		// case *types.Memory:
		// 	// TODO:
		// 	arrTypeLLVM := g.getLLVMType(innerT.T)
		// 	internalArrPtr := g.builder.CreateStructGEP(arrTypeLLVM, val, 1, "")
		// 	// this loads the pointer to memory not the actual array
		// 	eleT := g.getLLVMType(&types.ConstU8)
		// 	internalArrType := llvm.PointerType(llvm.ArrayType(eleT, 1), 0)
		// 	ptrToArr := g.builder.CreateLoad(internalArrType, internalArrPtr, "")
		// 	return ptrToArr
		// default:
		// 	// we cant convert other types to c
		// 	panic("this is a compiler error. please report")
		// }
	// do nothing as type already matches c
	case *types.Int, *types.Float, *types.Bool, *types.Byte:
		return val
	case *types.String:
		// extract internal array
		arrTypeLLVM := g.getLLVMType(t)
		// Create GEP to 2nd struct field (which is array or struct (which means nested array))
		internalArrPtr := g.builder.CreateStructGEP(arrTypeLLVM, val, 1, "")
		// this loads the pointer to memory not the actual array
		eleT := g.getLLVMType(&types.ConstU8)
		internalArrType := llvm.PointerType(llvm.ArrayType(eleT, 1), 0)
		ptrToArr := g.builder.CreateLoad(internalArrType, internalArrPtr, "")
		return ptrToArr
	case *types.Array:
		// extract internal array
		arrTypeLLVM := g.getLLVMType(t)
		internalArrType := g.makeArrayType(t, 1)
		// Create GEP to 2nd struct field (which is array or struct (which means nested array))
		internalArrPtr := g.builder.CreateStructGEP(arrTypeLLVM, val, 1, "")
		// this loads the pointer to memory not the actual array
		ptrToArr := g.builder.CreateLoad(internalArrType, internalArrPtr, "")
		return ptrToArr
	case *types.Memory:
		return g.generateDashToC(t.T, val)
	case *types.Struct:
		panic("struct to c not supported yet")
	default:
		// we cant convert other types to c
		panic("this is a compiler error. please report")
	}
}

func (g *Generator) generateTypeCast(n *ast.FunctionCallExpression) (llvm.Value, bool) {
	// TODO: check if type definition
	if _, ok := g.typeCache.Get(n.TokenLiteral()); !ok {
		return empty, false
	}

	switch t := n.Type().(type) {
	case *types.Union:
		v := g.buildInternal(n.Arguments[0], "")
		// generate type descriptor
		id := g.library + t.Name + n.Arguments[0].Type().Ident()
		typeDesc := g.generateTypeDescriptor(id)

		llvmType := g.getLLVMType(t)
		var ptr llvm.Value
		g.doInEntry(func() {
			ptr = g.builder.CreateAlloca(llvmType, "")
		})
		firstField := g.builder.CreateStructGEP(llvmType, ptr, 0, "")
		g.builder.CreateStore(typeDesc, firstField)

		secondField := g.builder.CreateStructGEP(llvmType, ptr, 1, "")
		argType := g.getLLVMType(n.Arguments[0].Type())
		if argType.TypeKind() == llvm.StructTypeKind {
			width := g.getWidth(argType)
			false := llvm.ConstInt(g.ctx.Int1Type(), 0, false)
			g.createCallMemCopy(secondField, v, width, false)
		} else {
			g.builder.CreateStore(v, secondField)
		}
		return ptr, true
	default:
		return g.buildInternal(n.Arguments[0], ""), true
	}
}

func (g *Generator) generateTypeDescriptor(id string) llvm.Value {
	h := sha256.New()
	h.Write([]byte(id))

	sum := h.Sum(nil)

	typeDesc := binary.BigEndian.Uint64(sum[:8])

	return llvm.ConstInt(g.ctx.Int64Type(), typeDesc, false)
}

func (g *Generator) generateUnionDataAccess(unionPtr llvm.Value, unionType, caseType llvm.Type) llvm.Value {
	underylingPtr := g.builder.CreateStructGEP(unionType, unionPtr, 1, "")

	// TODO: cast underlying ptr data to caseType
	switch caseType.TypeKind() {
	case llvm.StructTypeKind:
		return underylingPtr
	default:
		return g.builder.CreateLoad(caseType, underylingPtr, "")
	}
}

// ----------------- //
// Compile built-ins //
// ----------------- //

func (g *Generator) compileBuiltinFunc(n *ast.FunctionCallExpression) (llvm.Value, bool) {
	switch n.Token.Literal {
	case "printf":
		var args []llvm.Value
		for _, arg := range n.Arguments {
			args = append(args, g.buildInternal(arg, ""))
		}
		g.compilePrintf(args)
		return empty, true
	case "len":
		return g.compileLenArray(n), true
	case "cap":
		return g.compileCapArray(n), true
	case "size":
		return g.compileSize(n), true
	case "make":
		return g.compileMakeArrayOnStack(n), true
	case "validate":
		return g.generateCallValidate(n), true
	}
	return empty, false
}

func (g *Generator) compileLenArray(n *ast.FunctionCallExpression) llvm.Value {
	// for len([1,2,3,4]), kinda pointless but whatever
	// if _, ok := n.Arguments[0].(*ast.ArrayLiteral); ok {
	// 	g.buildInternal(n.Arguments[0], "")
	// }
	if _, ok := n.Arguments[0].(*ast.StringLiteral); ok {
		// strPtr := g.buildInternal(n.Arguments[0], "")
		// typeLLVM := g.getLLVMType(n.Arguments[0].Type())
		// fieldPtr := g.curBuilder.CreateStructGEP(typeLLVM, strPtr, 0, "")
		// return g.curBuilder.CreateLoad(g.ctx.Int64Type(), fieldPtr, "")
	}

	v := g.buildInternal(n.Arguments[0], "")
	// list, ok := g.varSt.Get(n.Arguments[0].TokenLiteral())
	// if !ok {
	// 	panic("unknown identifier: " + n.Arguments[0].String())
	// }

	arrayTypeLLVM := g.getLLVMType(n.Arguments[0].Type())
	fieldPtr := g.builder.CreateStructGEP(arrayTypeLLVM, v, 0, "")
	return g.builder.CreateLoad(g.ctx.Int64Type(), fieldPtr, "")
}

func (g *Generator) compileCapArray(n *ast.FunctionCallExpression) llvm.Value {
	// case: cap([1,2,3,4])
	if _, ok := n.Arguments[0].(*ast.ArrayLiteral); ok {
		g.buildInternal(n.Arguments[0], "")
	}

	return llvm.ConstInt(g.ctx.Int64Type(), uint64(0), false)
}

func (g *Generator) compileSize(n *ast.FunctionCallExpression) llvm.Value {
	// https://nondot.org/sabre/LLVMNotes/SizeOf-OffsetOf-VariableSizedStructs.txt
	arg := n.Arguments[0]
	g.buildInternal(arg, "")

	// case 1: array literal passed to 'size'
	if lit, ok := arg.(*ast.ArrayLiteral); ok {
		typ := g.makeArrayType(lit.T, len(lit.Values))
		return g.getWidth(typ)
	}

	t, ok := g.varSt.Get(arg.String())
	if !ok {
		panic("this is a compiler error. please report")
	}

	// case 2: array type
	switch internalT := arg.Type().(type) {
	case *types.Array:
		// get width of array if it had 1 element
		width := g.getWidth(g.makeArrayType(internalT, 1))
		llvmType := g.getLLVMType(internalT)
		// get length field
		lenPtr := g.builder.CreateStructGEP(llvmType, t.Ptr, 0, "")
		int64Type := g.getLLVMType(&types.ConstI64)
		len := g.builder.CreateLoad(int64Type, lenPtr, "")

		return g.builder.CreateMul(width, len, "")
	}

	// TODO: maybe for abstract structs we convert from abstract to struct and then return size
	// otherwise to know the size of the pointer fields and array seems kind of pointless

	// case 3: all other types
	return g.getWidth(t.Type)
}

// TODO: consider making initialisation a function instead of inlining everything directly
// Compiles stack allocation of array of given type and size with initial value if defined
func (g *Generator) compileMakeArrayOnStack(fn *ast.FunctionCallExpression) llvm.Value {
	arrayType := fn.Arguments[0].Type().(*types.Array)
	arrayTypeLLVM := g.makeArrayType(arrayType, 1)
	headerTypeLLVM := g.getLLVMType(arrayType)

	length := g.buildInternal(fn.Arguments[1], "")
	var internalArrPtr llvm.Value
	var headerPtr llvm.Value
	var iPtr llvm.Value
	g.doInEntry(func() {
		elementTypeLLVM := g.getLLVMType(arrayType.T)
		internalArrPtr = g.builder.CreateArrayAlloca(elementTypeLLVM, length, "")
		headerPtr = g.builder.CreateAlloca(headerTypeLLVM, "")
		iPtr = g.builder.CreateAlloca(g.ctx.Int64Type(), "")
	})

	// initialise i at 0
	zero := llvm.ConstInt(g.ctx.Int64Type(), 0, false)
	g.builder.CreateStore(zero, iPtr)

	// set 1st struct field to length
	field1Ptr := g.builder.CreateStructGEP(headerTypeLLVM, headerPtr, 0, "")
	g.builder.CreateStore(length, field1Ptr)

	// set 2nd struct field to arrayPtr
	field2Ptr := g.builder.CreateStructGEP(headerTypeLLVM, headerPtr, 1, "")
	g.builder.CreateStore(internalArrPtr, field2Ptr)

	// determine initial value
	var initValue llvm.Value
	if len(fn.Arguments) != 3 {
		llvmType := g.getLLVMType(arrayType.T)
		initValue = llvm.ConstNull(llvmType)
	} else {
		initValue = g.buildInternal(fn.Arguments[2], "")
	}

	// Initialise array with initValue dynamically

	loop := llvm.AddBasicBlock(g.builder.GetInsertBlock().Parent(), "")
	body := llvm.AddBasicBlock(g.builder.GetInsertBlock().Parent(), "")
	incr := llvm.AddBasicBlock(g.builder.GetInsertBlock().Parent(), "")
	after := llvm.AddBasicBlock(g.builder.GetInsertBlock().Parent(), "")

	g.builder.CreateBr(loop)

	// Compile loop condition
	g.builder.SetInsertPointAtEnd(loop)
	i := g.builder.CreateLoad(g.ctx.Int64Type(), iPtr, "")
	cond := g.builder.CreateICmp(llvm.IntULT, i, length, "")
	g.builder.CreateCondBr(cond, body, after)

	// Store init value in array
	g.builder.SetInsertPointAtEnd(body)
	indices := []llvm.Value{llvm.ConstInt(g.ctx.Int64Type(), uint64(0), false), i}
	elementPtr := g.builder.CreateInBoundsGEP(arrayTypeLLVM, internalArrPtr, indices, "")
	g.builder.CreateStore(initValue, elementPtr)
	g.builder.CreateBr(incr)

	// Compile increment
	g.builder.SetInsertPointAtEnd(incr)
	one := llvm.ConstInt(g.ctx.Int64Type(), 1, false)
	newI := g.builder.CreateAdd(i, one, "")
	g.builder.CreateStore(newI, iPtr)
	g.builder.CreateBr(loop)

	// Set the insert point to the after block
	g.builder.SetInsertPointAtEnd(after)

	return headerPtr
}

// Compiles the fail fast version of builtin 'validate' returning bool where false means
// data is not valid
func (g *Generator) generateCallValidate(n *ast.FunctionCallExpression) llvm.Value {
	arg := n.Arguments[0]
	argVal := g.buildInternal(arg, "")

	name := g.library + ".__" + arg.Type().Ident()
	fnInfo, ok := g.fnSt.Get(name)
	if !ok {
		fnType := &types.Function{Arg: []types.TypeSpec{arg.Type()}, Ret: []types.TypeSpec{n.Type()}}
		fnInfo.Ptr, fnInfo.Type = g.generateValidate(name, arg.Type().Ident(), fnType)

		g.fnSt.Set(name, fnInfo)
	}

	args := []llvm.Value{argVal}
	return g.builder.CreateCall(fnInfo.Type, fnInfo.Ptr, args, "")
}

// Dynamically generates validation function for a given type definition using its guard
func (g *Generator) generateValidate(fnName, tdName string, fnType *types.Function) (llvm.Value, llvm.Type) {

	oldBuilder := g.builder
	oldEntry := g.fnEntry
	g.builder = g.ctx.NewBuilder()
	defer g.builder.Dispose()
	g.fnNameScope.Push(fnName)
	defer g.fnNameScope.Pop()

	g.varSt.Scope()
	defer g.varSt.Unscope()

	var fn llvm.Value
	var typ llvm.Type
	typ = g.getLLVMType(fnType)

	fn = llvm.AddFunction(g.mod, fnName, typ)
	fn.SetFunctionCallConv(llvm.FastCallConv)
	fn.SetLinkage(llvm.ExternalLinkage)
	entry := llvm.AddBasicBlock(fn, "entry")
	g.builder.SetInsertPointAtEnd(entry)

	// The reason we set variable table using the type definition is so that we
	// can compile the guard (as it uses the type definition name as a 'variable').
	g.varSt.Set(tdName, Variable{Ptr: fn.Param(0), Type: fn.Param(0).Type()})

	guard, ok := g.typeGuardSt.Get(tdName)
	if !ok {
		panic("this is a compiler error. please report")
	}

	cond := g.buildInternal(guard, "")
	true := llvm.ConstInt(g.ctx.Int1Type(), 1, false)
	false := llvm.ConstInt(g.ctx.Int1Type(), 0, false)
	isValid := g.builder.CreateSelect(cond, true, false, "")
	g.builder.CreateRet(isValid)

	g.builder = oldBuilder
	g.fnEntry = oldEntry

	return fn, typ
}

// ---------------- //
// Extern functions //
// ---------------- //

// Declares function signature for c printf function
func (g *Generator) compilePrintf(args []llvm.Value) {

	printfType := llvm.FunctionType(g.ctx.Int32Type(), []llvm.Type{llvm.PointerType(g.ctx.Int8Type(), 0)}, true)

	printfFn := g.mod.NamedFunction("printf")
	if printfFn.IsNil() {
		printfFn = llvm.AddFunction(g.mod, "printf", printfType)
	}

	// Call printf
	g.builder.CreateCall(printfType, printfFn, args, "")
}

// ---------------- //
// Helper functions //
// ---------------- //

// Returns width as i64 llvm type
func (g *Generator) getWidth(typ llvm.Type) llvm.Value {
	nullPtr := llvm.ConstPointerNull(llvm.PointerType(g.ctx.Int8Type(), 0))
	int32Type := g.getLLVMType(&types.ConstI32)
	indices := []llvm.Value{
		llvm.ConstInt(int32Type, 1, false),
	}
	ptr := g.builder.CreateGEP(typ, nullPtr, indices, "")
	int64Type := g.getLLVMType(&types.ConstI64)
	return g.builder.CreatePtrToInt(ptr, int64Type, "")
}

func (g *Generator) compileReplPrint(v llvm.Value) {
	if v.IsNil() {
		return
	}

	// Create format string
	p := g.llvmTypeToFlag(v.Type().TypeKind())
	str := g.builder.CreateGlobalStringPtr(p, "fmt")

	g.compilePrintf([]llvm.Value{str, v})

}

// Ensures llvm operation defined in closure run in entry block of function
// Can be useful to make sure that all allocas occur in entry block
// See: https://llvm.org/docs/Frontend/PerformanceTips.html#use-of-allocas
func (g *Generator) doInEntry(fn func()) {
	// if !g.scope.IsIn(FOR) {
	oldEntry := g.builder.GetInsertBlock()
	// ensure allocas occur at the very top. If we use SetInsertPointAtEnd
	// we risk placing alloca after a jump
	lastInst := g.fnEntry.LastInstruction()
	if !lastInst.IsABranchInst().IsNil() || !lastInst.IsASwitchInst().IsNil() {
		g.builder.SetInsertPointBefore(lastInst)
	} else {
		g.builder.SetInsertPointAtEnd(g.fnEntry)
	}
	fn()
	g.builder.SetInsertPointAtEnd(oldEntry)
	// } else {
	// fn()
	// }
}

func (g *Generator) run(fn llvm.Value) (llvm.GenericValue, error) {
	if err := llvm.InitializeNativeTarget(); err != nil {
		return llvm.GenericValue{}, err
	}
	// Create execution engine
	// https://llvm.org/docs/MCJITDesignAndImplementation.html
	opts := llvm.MCJITCompilerOptions{}
	engine, err := llvm.NewMCJITCompiler(g.mod, opts)
	if err != nil {
		return llvm.GenericValue{}, err
	}

	// Execute main function in module
	val := engine.RunFunction(fn, []llvm.GenericValue{})
	// TODO: we probably need to validate val
	return val, nil
}

// Converts multiple types to llvm types using cached types where possible
func (g *Generator) getLLVMTypes(ts []types.TypeSpec) []llvm.Type {
	_ts := make([]llvm.Type, len(ts))
	for i, t := range ts {
		_ts[i] = g.getLLVMType(t)
	}
	return _ts
}

type attribute uint8

const (
	noUndefined attribute = iota
	// attribute may only be applied to pointer typed parameters
	nonNull
	// only used for fn arguments that are returned
	argReturned
	// no modifications will be made to argument.
	// Applies to all arguments as Dash is an immutable language
	readOnly
	// function does not raise exception
	noUnwind
)

func (g *Generator) getAttributeSret(typ llvm.Type) llvm.Attribute {
	return g.ctx.CreateTypeAttribute(llvm.AttributeKindID("sret"), typ)
}

func (g *Generator) getLLVMAttribute(aId attribute) llvm.Attribute {
	a, ok := g.attributeCache.Get(aId)
	if ok {
		return a
	}

	a = g.makeLLVMAttribute(aId)

	g.attributeCache.Set(aId, a)
	return a
}

func (g *Generator) makeLLVMAttribute(attr attribute) llvm.Attribute {
	switch attr {
	case noUndefined:
		return g.ctx.CreateEnumAttribute(llvm.AttributeKindID("noundef"), 0)
	case nonNull:
		return g.ctx.CreateEnumAttribute(llvm.AttributeKindID("nonnull"), 0)
	case argReturned:
		return g.ctx.CreateEnumAttribute(llvm.AttributeKindID("returned"), 0)
	case readOnly:
		return g.ctx.CreateEnumAttribute(llvm.AttributeKindID("readonly"), 0)
	default:
		panic("make_llvm_attribute(), unknown attribute")
	}
}

// Returns type from cache or if not found generates new
func (g *Generator) getLLVMType(t types.TypeSpec) llvm.Type {
	name := t.String()

	// Only make keep one copy of 'array' or 'abstract' (abstract struct)
	// as its always the same no matter what underlying type.
	switch t := t.(type) {
	case *types.Array:
		name = "array"
	case *types.AbstractStruct:
		name = "abstract"

	// as optional of aggregate types uses a pointer for second field
	// we need to change name used for cache lookup so we dont generate
	// spurious types in IR
	case *types.Optional:
		switch t.T.(type) {
		case *types.Array, *types.Struct, *types.String, *types.AbstractStruct, *types.Function:
			name = "optional_ptr"
		}
	}

	// If rype already processed return directly
	v, ok := g.typeCache.Get(name)
	if ok {
		return v
	}

	llvm_type := g.makeLLVMType(t)

	g.typeCache.Set(name, llvm_type)

	return llvm_type

}

// Generates new llvm type. DO NOT USE
func (g *Generator) makeLLVMType(t types.TypeSpec) llvm.Type {
	switch t := t.(type) {
	case *types.Null:
		var llvmType llvm.Type
		typs := []llvm.Type{g.ctx.Int1Type()}
		llvmType = g.ctx.StructCreateNamed("option_ptr")
		typs = append(typs, llvm.PointerType(g.ctx.Int8Type(), 0))

		llvmType.StructSetBody(typs, false)
		return llvmType

	case *types.Int:
		switch t.Width {
		case 8:
			return g.ctx.Int8Type()
		case 16:
			return g.ctx.Int16Type()
		case 32:
			return g.ctx.Int32Type()
		case 64:
			return g.ctx.Int64Type()
		case 128:
			return g.ctx.IntType(128)
		case 256:
			return g.ctx.IntType(256)
		}
	case *types.Float:
		switch t.Width {
		case 32:
			return g.ctx.FloatType()
		case 64:
			return g.ctx.DoubleType()
		}
		return g.ctx.Int8Type()
	case *types.Bool:
		return g.ctx.Int1Type()
	case *types.Byte:
		return g.ctx.Int8Type()
	case *types.Char:
		return g.ctx.Int32Type()
	case *types.String:
		eleT := g.getLLVMType(&types.ConstU8)
		llvmType := g.ctx.StructCreateNamed("string")
		types := []llvm.Type{
			g.ctx.Int64Type(), llvm.PointerType(eleT, 0),
		}
		llvmType.StructSetBody(types, false)
		return llvmType
	case *types.Optional:
		var llvmType llvm.Type
		typs := []llvm.Type{g.ctx.Int1Type()}
		underlying := types.GetUnderlyingType(t.T)
		switch underlying.(type) {
		case *types.Array, *types.Struct, *types.String, *types.AbstractStruct, *types.Function:
			llvmType = g.ctx.StructCreateNamed("option_ptr")
			typs = append(typs, llvm.PointerType(g.getLLVMType(t.T), 0))
		default:
			llvmType = g.ctx.StructCreateNamed("option_" + t.T.Ident())
			typs = append(typs, g.getLLVMType(t.T))
		}

		llvmType.StructSetBody(typs, false)
		return llvmType
	case *types.Array:
		// if t.Size == 0 {
		// dynamic sized array
		llvmType := g.ctx.StructCreateNamed("array")
		eleT := g.getLLVMType(t.T)

		types := []llvm.Type{
			g.ctx.Int64Type(), llvm.PointerType(eleT, 0),
		}
		llvmType.StructSetBody(types, false)
		return llvmType
		// } else {
		// 	// fixed-size array
		// 	eleT := g.get_llvm_type(t.T)
		// 	return llvm.ArrayType(eleT, t.Size)
		// }
	case *types.Struct:
		// Named struct required otherwise recursive structs (self-referencing)
		// are not possible see: https://llvm.org/docs/LangRef.html#structure-type
		llvmType := g.ctx.StructCreateNamed(t.Name)
		g.typeCache.Set(t.String(), llvmType)
		var types []llvm.Type
		for _, f := range t.Ts {
			types = append(types, g.getLLVMType(f.T))
		}
		llvmType.StructSetBody(types, false)

		return llvmType
	case *types.AbstractStruct:
		// { base ptr, offsets [n x i64] }
		llvmType := g.ctx.StructCreateNamed("abstract")
		g.typeCache.Set("abstract", llvmType)

		types := make([]llvm.Type, 2)
		i64Type := g.ctx.Int64Type()
		types[0] = llvm.PointerType(i64Type, 0)
		types[1] = llvm.ArrayType(i64Type, len(t.Ts))

		llvmType.StructSetBody(types, false)
		return llvmType
	case *types.Enum:
		llvmType := g.ctx.StructCreateNamed(t.Name)
		g.typeCache.Set(t.String(), llvmType)

		types := []llvm.Type{g.getInternalEnumType(t.Size)}

		llvmType.StructSetBody(types, false)
		return llvmType
	case *types.Union:
		// NOTE: aligning to 64 bits might be wasteful?

		// Here we create a generic union type and new
		// types for each underlying type in the union.
		// The underlying union type has a type descriptor
		// all the of the same fields of the original type
		// The generic type contains a type descriptor and
		// padding. This type should be used when allocating
		// space for union and when passing or returning
		// the union. The underlying union types should
		// only be used when we know for certain that the
		// generic union type desc and the underlying union
		// type desc match.
		// Example:
		// ```
		// struct abc {
		//     x i64
		// }
		//
		// union xyz {
		//     abc
		// }
		//
		// // underlying union type
		// struct union_abc {
		//     desc u32
		//     x    i64
		// }
		//
		// // generic union type
		// struct union_xyz {
		//    desc    u32
		//    padding [ 8 x byte ]
		// }
		// ```
		unionName := g.library + "." + t.Name
		typeDescSize := g.targetData.TypeAllocSize(g.ctx.Int32Type())
		maxSize := typeDescSize
		maxAlign := uint64(8)
		for _, ut := range t.Ts {
			unionFieldName := unionName + "." + ut.Ident()
			unionFieldType := g.getLLVMType(ut)
			// generate the underlying union type
			g.getUnionType(unionFieldName, unionFieldType)

			typeSize := g.targetData.TypeAllocSize(unionFieldType)
			typeAlign := uint64(g.targetData.ABITypeAlignment(unionFieldType))

			if typeSize > maxSize {
				maxSize = typeSize
			}
			if typeAlign > maxAlign {
				maxAlign = typeAlign
			}
		}

		// Align size to next alignment boundary.
		// For example if size is 6 bytes but alignment
		// has to be multiple of 8 then we add 2 bytes
		// Original:   [1][2][3][4][5][6]
		//                             ^ Size = 6
		// Padded:     [1][2][3][4][5][6][-][-]
		//                                ^ Add padding
		totalSize := maxSize
		if totalSize%maxAlign != 0 {
			totalSize = ((totalSize + maxAlign - 1) / maxAlign) * maxAlign
		}

		types := []llvm.Type{
			// type descriptor
			g.ctx.Int64Type(),
			// enough bytes for data
			llvm.ArrayType(g.ctx.Int8Type(), int(totalSize)),
		}

		if t, ok := g.typeCache.Get(unionName); ok {
			return t
		}

		llvmType := g.ctx.StructCreateNamed(unionName)
		llvmType.StructSetBody(types, false)

		g.typeCache.Set(unionName, llvmType)

		return llvmType
	case *types.Memory:
		return g.getLLVMType(t.T)
	case *types.Definition:
		switch ut := t.Underlying.(type) {
		case *types.Array:
			// Copied from *types.Array llvm type generation above.
			// We use type definition name instead of 'array'
			llvmType := g.ctx.StructCreateNamed(t.Name)
			eleT := g.getLLVMType(ut.T)
			types := []llvm.Type{
				g.ctx.Int64Type(), llvm.PointerType(eleT, 0),
			}
			llvmType.StructSetBody(types, false)
			return llvmType
		case *types.Struct:
			// Copied from *types.Struct llvm type generation above.
			// We use type definition name instead of old struct name
			llvmType := g.ctx.StructCreateNamed(t.Name)
			g.typeCache.Set(t.String(), llvmType)
			var types []llvm.Type
			for _, f := range ut.Ts {
				types = append(types, g.getLLVMType(f.T))
			}
			llvmType.StructSetBody(types, false)

			return llvmType
		default:
			// BUG:
			// The fact that scalar type definitions get compiled like normal types
			// means that linking against a function accepting a type definition and
			// passing the underlying type wont raise any errors, potentially leading
			// to unexpected behaviour. If everything is statically compiled semsis
			// will enforce it
			return g.getLLVMType(t.Underlying)
		}
	case *types.Alias:
		return g.getLLVMType(t.Underlying)
	case *types.Function:
		// Argument types
		var argTypes []llvm.Type
		for _, arg := range t.Arg {
			argTypes = append(argTypes, g.getLLVMArgRetType(arg))
		}
		// Return types
		// If struct is returned we also add an argument
		// as struct will be allocated by caller.
		// NOTE: we currently waste a register by returing pointer
		// to escaped struct. Eventhough caller already has the pointer
		retType := g.ctx.VoidType()
		switch len(t.Ret) {
		case 0:
			retType = g.ctx.VoidType()
		case 1:
			retType = g.getLLVMArgRetType(t.Ret[0])
			switch t.Ret[0].(type) {
			case *types.Struct, *types.Union:
				argTypes = append(argTypes, retType)
			}

		default:
			var typs []llvm.Type
			for _, rt := range t.Ret {
				retType := g.getLLVMArgRetType(rt)
				switch rt.(type) {
				case *types.Struct, *types.Union:
					argTypes = append(argTypes, retType)
				default:
				}
				typs = append(typs, retType)
			}
			retType = g.ctx.StructType(typs, false)
		}

		// variadic is always false as Dash uses list to pass variable arguments
		return llvm.FunctionType(retType, argTypes, false)
	case *types.Pointer:
		llvmType := g.getLLVMType(t.T)
		return llvm.PointerType(llvmType, 0)
	case *types.Dirty:
		return g.getLLVMType(t.T)
	case *types.UnknownNamed:
		panic("named type found in compiler: " + t.String())
	}
	panic("make_llvm_type() unsupported type: " + t.String())
}

func (g *Generator) getUnionType(name string, typ llvm.Type) llvm.Type {
	if t, ok := g.typeCache.Get(name); ok {
		return t
	}

	llvmType := g.ctx.StructCreateNamed(name)
	typs := []llvm.Type{g.ctx.Int32Type()}

	// if we have a struct type we flatten it into union struct
	if typ.TypeKind() == llvm.StructTypeKind || typ.TypeKind() == llvm.PointerTypeKind {
		for _, fieldType := range typ.StructElementTypes() {
			typs = append(typs, fieldType)
		}
	} else {
		typs = append(typs, typ)
	}

	llvmType.StructSetBody(typs, false)

	g.typeCache.Set(name, llvmType)
	return llvmType
}

func (g *Generator) makeArrayType(t types.TypeSpec, size int) llvm.Type {
	switch t := t.(type) {
	// TODO: dirty type should probably only exist
	// within semsis scope
	case *types.Dirty:
		return g.makeArrayType(t.T, size)
	case *types.Array:
		eleT := g.getLLVMType(t.T)
		return llvm.ArrayType(eleT, size)
	case *types.String:
		eleT := g.getLLVMType(&types.ConstByte)
		return llvm.ArrayType(eleT, size)
	case *types.Definition:
		return g.makeArrayType(t.Underlying, size)
	}
	panic("makeArrayType() unsupported type: " + t.String())
}

func (g *Generator) getInternalEnumType(size int) llvm.Type {
	if size <= 255 {
		return g.ctx.Int8Type()
	} else {
		return g.ctx.Int16Type()
	}
}

func (g *Generator) getLLVMArgRetType(typ types.TypeSpec) llvm.Type {
	switch t := typ.(type) {
	case *types.Array, *types.String, *types.Struct,
		*types.AbstractStruct, *types.Function, *types.Union:
		return llvm.PointerType(g.getLLVMType(t), 0)
	case *types.Definition:
		// If underlying type of type def array or struct we pass as
		// pointer like the primitive aggregate types.
		return g.getLLVMArgRetType(t.Underlying)
	case *types.Dirty:
		return g.getLLVMArgRetType(t.T)
	case *types.Memory:
		return llvm.PointerType(g.getLLVMType(t.T), 0)

	default:
		return g.getLLVMType(t)

	}
}

// e.g. for int: %d, float: %g, string: %s
func (g *Generator) llvmTypeToFlag(t llvm.TypeKind) string {
	switch t {
	case llvm.IntegerTypeKind, llvm.ArrayTypeKind:
		return "%d\n"
	case llvm.FloatTypeKind, llvm.DoubleTypeKind:
		return "%g\n"
	case llvm.PointerTypeKind, llvm.FunctionTypeKind:
		return "%s\n"
	default:
		return "%s\n"
	}
}

// BUG: if alloca depends on instruction above DO NOT MOVE IT
// otherwise we get: 'Instruction does not dominate all uses!'
func promoteEntryBlockAllocas(fn llvm.Value) {
	// Skip if not a function or has no blocks
	if fn.IsNil() || fn.IsAFunction().IsNil() {
		return
	}
	entry := fn.EntryBasicBlock()
	if entry.IsNil() {
		return
	}

	// Collect allocas while removing them from entry block
	var allocas []llvm.Value
	curr := entry.FirstInstruction()
	for !curr.IsNil() {
		next := llvm.NextInstruction(curr)
		if !curr.IsAAllocaInst().IsNil() {
			allocas = append(allocas, curr)
			curr.RemoveFromParentAsInstruction()
		}
		curr = next
	}

	// Nothing to do if no allocas found
	if len(allocas) == 0 {
		return
	}

	// Get the first instruction in the block to insert before
	// If block is now empty due to only having allocas, we'll insert at end
	firstInst := entry.FirstInstruction()
	builder := fn.GlobalParent().Context().NewBuilder()
	defer builder.Dispose()

	if firstInst.IsNil() {
		builder.SetInsertPointAtEnd(entry)
	} else {
		builder.SetInsertPointBefore(firstInst)
	}

	// Insert allocas in original order at the top
	for i := 0; i < len(allocas); i++ {
		builder.Insert(allocas[i])
	}

}

type Frame struct {
	Ctx     llvm.Context
	Builder llvm.Builder
	FnEntry llvm.BasicBlock
}

type Variable struct {
	Type  llvm.Type
	Ptr   llvm.Value
	IsVar bool
}

type Function struct {
	Type       llvm.Type
	Ptr        llvm.Value
	TypeDash   types.TypeSpec
	Attributes []ast.Attribute
}

type StructField struct {
	Index int
}

type FunctionData struct {
	Name string
	Enty *llvm.BasicBlock
}

type LoopData struct {
	EndBlock  *llvm.BasicBlock
	IncrBlock *llvm.BasicBlock
}

func hasAttribute(attrs []ast.Attribute, a ast.AttributeType) bool {
	for _, attr := range attrs {
		if attr.Equal(a) {
			return true
		}
	}
	return false
}
