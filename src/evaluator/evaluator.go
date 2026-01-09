package evaluator

import (
	"fmt"
	"hash/fnv"
	"maps"
	"path"
	"reflect"
	"strconv"
	"strings"

	"dash-lang.io/src/ast"
	"dash-lang.io/src/token"
	"dash-lang.io/src/types"
)

type state uint8

const (
	NONE state = iota
	BREAK
	NEXT
	ERROR
)

// Debug flag to enable/disable debug output
var isDebug = false

// Global error type descriptors to avoid recomputation
var (
	errDescIndexOutOfBounds = generateTypeDescriptor("runtime.index_out_of_bounds")
	errForceUnwrapNull      = generateTypeDescriptor("runtime.force_unwrap_null")
)

type Error struct {
	descriptor uint32
	Err        string
	// optional
	Args map[string]any
}

type Return struct {
	Values []any
}

type Optional struct {
	isValid bool
	value   any
}

type Any struct {
	descriptor uint32
	value      any
}

type Union struct {
	// hash of library name + type name
	descriptor uint32
	value      any
}

type Pointer struct {
	id    uint64 // unique identifier for this pointer
	value any    // the value being pointed to
}

type Function struct {
	ctx       *Context
	arguments []*ast.ParameterStatement
	body      *ast.BlockStatement
}

type Evaluator struct {
	// Contains all libraries used by program
	libs map[string]*ast.Library
	// Keeps track of computed contextes for
	// each library with 'prev' == nil
	ctxs map[string]*Context
	// Counter for generating unique pointer IDs
	nextPointerID uint64
	// Maps variable locations to existing pointers (for pointer identity)
	// Key is a hash of context+variable path
	pointerCache map[uint64]*Pointer
	state        state
	// can be *Error, *Return or nil
	stateVar any
}

func New(libs map[string]*ast.Library) *Evaluator {
	return &Evaluator{
		libs:         libs,
		ctxs:         make(map[string]*Context),
		pointerCache: make(map[uint64]*Pointer),
	}
}

func (e *Evaluator) InitialiseLib(lib *ast.Library, ctx *Context) {
	// For all imports of current library ensure
	// context is aware of imports and initialise
	// those imported libraries if not already done
	for _, n := range lib.Nodes {
		imp, ok := n.(*ast.UseStatement)
		if !ok {
			continue
		}
		libPath := imp.Name.TokenLiteral()
		lib, ok := e.libs[libPath]
		if !ok {
			panic("")
		}

		normalisedLibName := path.Base(libPath)
		ctx.imps.Set(normalisedLibName, lib)

		if _, ok := e.ctxs[normalisedLibName]; !ok {
			// Create context with the imported library's path
			ctxLib := NewContextWith(nil, libPath)
			e.eval(lib, ctxLib)
			e.ctxs[normalisedLibName] = ctxLib
		}
	}

	// Pass 1: initialise types, functions, enums, etc.
	// This must happen before assignments so type casts work correctly
	for _, n := range lib.Nodes {
		switch n := n.(type) {
		case *ast.UnionStatement:
			ctx.typs.Set(n.Name.TokenLiteral(), n.T)
		case *ast.TypeDefinitionStatement:
			ctx.typs.Set(n.Name.TokenLiteral(), n.T)
		case *ast.TypeAliasStatement:
			ctx.typs.Set(n.Name.TokenLiteral(), n.T)
		case *ast.StructStatement:
			ctx.vars.Set(n.Name.TokenLiteral(), n.T)
		case *ast.EnumStatement:
			e.initialiseEnumStatement(n, ctx)
		case *ast.ErrorStatement:
			typeName := ctx.libPath + "." + n.Name.String()
			typeDesc := generateTypeDescriptor(typeName)
			err := &Error{
				descriptor: typeDesc,
				Err:        typeName,
			}
			ctx.Set(n.Name.String(), err)
		case *ast.FunctionExpression:
			fn := &Function{
				arguments: n.Arguments,
				body:      n.Body,
				ctx:       ctx,
			}
			ctx.Set(n.Name.Value, fn)
		}
	}

	// Pass 2: evaluate assignments after all types are registered
	for _, n := range lib.Nodes {
		if assgn, ok := n.(*ast.AssignmentStatement); ok {
			e.evalAssignmentStatement(assgn, ctx)
		}
	}
}

func (e *Evaluator) Eval(n ast.Node, ctx *Context) (result any) {
	defer recoverPanic(&result)

	result = e.eval(n, ctx)
	if e.state == ERROR {
		// reset error after evaluation
		err := e.stateVar
		e.stateVar = nil
		e.state = NONE
		return err
	}
	return
}

func (e *Evaluator) eval(n ast.Node, ctx *Context) (result any) {
	if e.state == ERROR {
		return e.stateVar
	}

	switch n := n.(type) {
	// Recursively initialise libraries top down
	case *ast.Library:
		// if the library has not been
		// initialised yet we do this here
		if _, ok := e.ctxs[n.Name.TokenLiteral()]; ok {
			return nil
		}

		e.InitialiseLib(n, ctx)
		var last any
		for _, n := range n.Nodes {
			switch n := n.(type) {
			case *ast.UseStatement, *ast.UnionStatement,
				*ast.TypeDefinitionStatement, *ast.StructStatement,
				*ast.EnumStatement, *ast.ErrorStatement:
				// skip as already initialised
			default:
				last = e.eval(n, ctx)
			}
		}
		return last
	case *ast.FunctionExpression:
		fn := &Function{
			arguments: n.Arguments,
			body:      n.Body,
			ctx:       ctx,
		}
		ctx.Set(n.Name.Value, fn)
		return fn
	case *ast.FunctionCallExpression:
		// check if it's a built-in function
		if res, ok := e.evalBuiltinFunction(n, ctx); ok {
			return res
		}
		// check if its a custom type cast
		if res, ok := e.evalCustomTypeCast(n, ctx); ok {
			return res
		}
		return e.evalFunctionCall(n, ctx)
	case *ast.TypeCastExpression:
		return e.evalTypeCastExpression(n, ctx)
	case *ast.AssignmentStatement:
		return e.evalAssignmentStatement(n, ctx)
	case *ast.BlockStatement:
		return e.evalBlockStatement(n, ctx)
	case *ast.IfElseExpression:
		return e.evalIfElseExpression(n, ctx)
	case *ast.ForStatement:
		return e.evalForStatement(n, ctx)
	case *ast.MatchExpressionStatement:
		return e.evalMatchExpressionStatement(n, ctx)
	case *ast.TryExpression:
		return e.evalTryExpression(n, ctx)
	case *ast.CatchExpression:
		return e.evalCatchExpression(n, ctx)
	case *ast.RaiseStatement:
		return e.evalRaiseStatement(n, ctx)
	case *ast.KeywordStatement:
		return e.evalKeywordStatement(n)
	case *ast.ReturnStatement:
		return e.evalReturnStatement(n, ctx)
	case *ast.DotExpression:
		return e.evalDotExpression(n, ctx)
	case *ast.PrefixExpression:
		return e.evalPrefixExpression(n, ctx)
	case *ast.InfixExpression:
		return e.evalInfixExpression(n, ctx)
	case *ast.PostfixExpression:
		e.evalPostfixExpression(n, ctx)
	case *ast.Identifier:
		val, ok := ctx.Get(n.Value)
		if !ok {
			panic("this is a compiler bug. please report")
		}
		return val

	case *ast.TypeLiteral:
		// for type literals in match expressions, return the type name
		return n.String()
	case *ast.StructLiteral:
		return e.evalStructLiteral(n, ctx)
	case *ast.ArrayLiteral:
		vals := make([]any, len(n.Values))
		for i, val := range n.Values {
			vals[i] = e.eval(val, ctx)
		}
		return vals
	case *ast.StringLiteral:
		return n.TokenLiteral()
	case *ast.CharacterLiteral:
		switch t := n.Type().(type) {
		case *types.Int:
			return e.evalIntCast(t, n.Value)
		case *types.Byte:
			return uint8(n.Value)
		case *types.Char:
			return uint32(n.Value)
		default:
			panic("this is a compiler error. please report")
		}
	case *ast.IntegerLiteral:
		underlyingType := types.GetUnderlyingType(n.Type())
		switch t := underlyingType.(type) {
		case *types.Int:
			return e.evalIntCast(t, n.Value)
		case *types.Byte:
			return e.evalByteCast(n.Value)
		case *types.Char:
			return e.evalCharCast(n.Value)
		case *types.Any:
			return int64(n.Value)
		case *types.Generic:
			return int64(n.Value)
		default:
			panic("this is a compiler error. please report")
		}
	case *ast.HexLiteral:
		underlyingType := types.GetUnderlyingType(n.Type())
		switch t := underlyingType.(type) {
		case *types.Int:
			return e.evalIntCast(t, n.Value)
		default:
			panic("this is a compiler error. please report")
		}
	case *ast.BooleanLiteral:
		return n.Value
	case *ast.FloatLiteral:
		return n.Value
	case *ast.NullLiteral:
		return Optional{isValid: false}
	case *ast.Comment:
		return nil
	case nil:
		panic("eval failed as node was nil")
	default:
		e.addError(n, fmt.Errorf("unknown node type %T", n))
	}
	return nil
}

func (e *Evaluator) initialiseEnumStatement(n *ast.EnumStatement, stk *Context) {
	fields := make(map[string]any)
	for i, field := range n.Fields {
		fields[field.Value] = int64(i)
	}
	stk.Set(n.Name.Value, fields)
}

// returns list of function call results
func (e *Evaluator) evalFunctionCall(n *ast.FunctionCallExpression, stk *Context) any {
	_fn, ok := stk.Get(n.TokenLiteral())
	fn, ok := _fn.(*Function)
	if !ok {
		// Check if this is a custom type cast (type definition, union, etc.)
		if _, exists := stk.GetType(n.TokenLiteral()); exists {
			if len(n.Arguments) == 1 {
				val := e.eval(n.Arguments[0], stk)
				val = unwrapFunctionResult(val, 0)
				return unwrapAny(val)
			}
			panic("type cast requires exactly one argument: " + n.TokenLiteral())
		}
		panic("not a function: " + n.TokenLiteral())
	}

	newCtx := NewContext(fn.ctx)

	// evaluate arguments and set values in fresh symbol table
	for i, arg := range n.Arguments {
		fnArgName := fn.arguments[i].Name.Value
		argValue := e.eval(arg, stk)
		argValue = unwrapFunctionResult(argValue, 0)

		if shouldWrapTypeInAny(fn.arguments[i].Type) {
			argValue = e.evalToAny(argValue)
		}

		newCtx.Set(fnArgName, argValue)
	}
	res := e.eval(fn.body, newCtx)
	if e.state == ERROR {
		assertType[*Error](e.stateVar)
		return nil
	}


	if ret, ok := res.(*Return); ok {
		for i, retType := range n.ReturnTypes {
			if shouldWrapTypeInAny(retType) && i < len(ret.Values) {
				if _, isAlreadyAny := ret.Values[i].(*Any); !isAlreadyAny {
					ret.Values[i] = e.evalToAny(ret.Values[i])
				}
			}
		}
		return ret
	}
	return &Return{Values: []any{res}}
}

// The goal of type casts for now is to only support the minimum number of operations
// to be able to bootstrap the compiler in dash
func (e *Evaluator) evalTypeCastExpression(n *ast.TypeCastExpression, stk *Context) any {
	val := e.eval(n.Argument, stk)
	val = unwrapFunctionResult(val, 0)
	val = unwrapAny(val)
	switch t := n.Typ.(type) {
	case *types.Int:
		return e.evalIntCast(t, val)
	case *types.Byte:
		return e.evalByteCast(val)
	case *types.Char:
		return e.evalCharCast(val)
	case *types.String:
		return e.evalStringCast(val)
	case *types.Array:
		return e.evalArrayCast(t, val)

	}
	return val
}

// Int casting

func (e *Evaluator) evalIntCast(t *types.Int, v any) any {
	switch t.Signed + t.Width {
	case 8:
		return e.toUint8(v)
	case 16:
		return e.toUint16(v)
	case 32:
		return e.toUint32(v)
	case 64:
		return e.toUint64(v)
	case 9, 17, 33, 65:
		return e.toInt64(v)
	}
	panic("invalid int cast")
}
func (e *Evaluator) toUint8(v any) uint8 {
	v = unwrapAny(v)
	switch v := v.(type) {
	case bool:
		if !v {
			return 0
		}
		return 1
	case uint8:
		return v
	case uint16:
		return uint8(v)
	case uint32:
		return uint8(v)
	case uint64:
		return uint8(v)
	case int8:
		return uint8(v)
	case int16:
		return uint8(v)
	case int32:
		return uint8(v)
	case int64:
		return uint8(v)
	}
	panic("invalid cast to u8")
}

func (e *Evaluator) toUint16(v any) uint16 {
	v = unwrapAny(v)
	switch v := v.(type) {
	case bool:
		if !v {
			return 0
		}
		return 1
	case uint8:
		return uint16(v)
	case uint16:
		return v
	case uint32:
		return uint16(v)
	case uint64:
		return uint16(v)
	case int8:
		return uint16(v)
	case int16:
		return uint16(v)
	case int32:
		return uint16(v)
	case int64:
		return uint16(v)
	}
	panic("invalid cast to u16")
}

func (e *Evaluator) toUint32(v any) uint32 {
	v = unwrapAny(v)
	switch v := v.(type) {
	case bool:
		if !v {
			return 0
		}
		return 1
	case uint8:
		return uint32(v)
	case uint16:
		return uint32(v)
	case uint32:
		return v
	case uint64:
		return uint32(v)
	case int8:
		return uint32(v)
	case int16:
		return uint32(v)
	case int32:
		return uint32(v)
	case int64:
		return uint32(v)
	}
	panic("invalid cast to u32")
}

func (e *Evaluator) toUint64(v any) uint64 {
	switch v := v.(type) {
	case bool:
		if !v {
			return 0
		}
		return 1
	case uint8:
		return uint64(v)
	case uint16:
		return uint64(v)
	case uint32:
		return uint64(v)
	case uint64:
		return v
	case int8:
		return uint64(v)
	case int16:
		return uint64(v)
	case int32:
		return uint64(v)
	case int64:
		return uint64(v)
	}
	panic(fmt.Sprintf("invalid cast to u64 from %T", v))
}

func (e *Evaluator) toInt64(v any) int64 {
	v = unwrapAny(v)
	switch v := v.(type) {
	case bool:
		if !v {
			return 0
		}
		return 1
	case uint8:
		return int64(v)
	case uint16:
		return int64(v)
	case uint32:
		return int64(v)
	case uint64:
		return int64(v)
	case int8:
		return int64(v)
	case int16:
		return int64(v)
	case int32:
		return int64(v)
	case int64:
		return v
	}
	panic(fmt.Sprintf("invalid cast to i64 from %T", v))
}

func (e *Evaluator) toFloat64(v any) float64 {
	switch v := v.(type) {
	case uint8:
		return float64(v)
	case uint16:
		return float64(v)
	case uint32:
		return float64(v)
	case uint64:
		return float64(v)
	case int8:
		return float64(v)
	case int16:
		return float64(v)
	case int32:
		return float64(v)
	case int64:
		return float64(v)
	case float64:
		return v
	case float32:
		return float64(v)
	}
	panic(fmt.Sprintf("invalid cast to float64 from %T", v))
}

// Byte casting

func (e *Evaluator) evalByteCast(v any) any {
	switch v := v.(type) {
	case uint8:
		return v
	case uint16:
		return uint8(v)
	case uint32:
		return uint8(v)
	case uint64:
		return uint8(v)
	case int8:
		return uint8(v)
	case int16:
		return uint8(v)
	case int32:
		return uint8(v)
	case int64:
		return uint8(v)
	}
	panic("invalid cast to byte")
}

// Char casting
func (e *Evaluator) evalCharCast(v any) any {
	switch v := v.(type) {
	case uint8:
		return uint32(v)
	case uint16:
		return uint32(v)
	case uint32:
		return v
	case uint64:
		return uint32(v)
	case int8:
		return uint32(v)
	case int16:
		return uint32(v)
	case int32:
		return uint32(v)
	case int64:
		return uint32(v)
	}
	panic("invalid cast to char")
}

// String casting
func (e *Evaluator) evalStringCast(v any) any {
	switch v := v.(type) {
	case string:
		return v
	case byte:
		return string(v)
	case []any:
		arr := make([]byte, len(v))
		for i, el := range v {
			switch el := el.(type) {
			case uint8:
				arr[i] = el
			default:
				arr[i] = byte(el.(int32))
			}
		}
		return string(arr)
	case []uint8:
		return string(v)
	}
	panic("invalid cast to string")
}

// Array casting

func (e *Evaluator) evalArrayCast(t *types.Array, v any) any {
	v = unwrapFunctionResult(v, 0)
	v = unwrapAny(v)
	switch t.T.(type) {
	case *types.Byte:
		str := v.(string)
		newArr := make([]byte, len(str))
		for i := range str {
			newArr[i] = str[i]
		}
		return newArr
	case *types.Int:
		switch v := v.(type) {
		case string:
			// we assume semsis caught any issues e.g. []u64(str)
			newArr := make([]uint8, len(v))
			for i := range v {
				newArr[i] = v[i]
			}
			return newArr
		case []byte:
			// we assume semsis caught any issues e.g. []u64(str)
			newArr := make([]uint8, len(v))
			for i := range v {
				newArr[i] = v[i]
			}
			return newArr
		case []any:
			return v
		}
	}
	panic("invalid array cast")
}

// always returns nil
func (e *Evaluator) evalAssignmentStatement(n *ast.AssignmentStatement, ctx *Context) any {
	// TODO: iterate over if function reached we need to handle that
	// TODO: data can be assigned to identifiers, struct fields, array indices, slices
	// g. if in use expression

	for i, val := range n.Values {
		switch val.Type().(type) {
		case *types.Multi:
			res := e.eval(val, ctx).(*Return)
			for j := range len(res.Values) {
				setOrUpdateForAssignment(n.Declerations[i+j], n.VarNameAt(i+j), res.Values[j], ctx)
			}
		case *types.ImportedNamed:
			res := e.eval(val, ctx)
			if ret, ok := res.(*Return); ok {
				for j := range len(ret.Values) {
					setOrUpdateForAssignment(n.Declerations[i+j], n.VarNameAt(i+j), ret.Values[j], ctx)
				}
			} else {
				setOrUpdateForAssignment(n.Declerations[i], n.VarNameAt(i), res, ctx)
			}
		default:
			res := e.eval(val, ctx)
			res = unwrapFunctionResult(res, 0)
			switch decl := n.Declerations[i].(type) {
			case *ast.Identifier:
				ctx.SetAll(n.VarNameAt(i), res)
			case *ast.DeclarationStatement:
				ctx.Set(n.VarNameAt(i), res)
			case *ast.DotExpression:
				rootIdent := decl.TokenLiteral()
				fieldPath := getFieldPath(decl)
				currentValue, _ := ctx.Get(rootIdent)
				newStruct := updateStructField(currentValue, fieldPath, res)
				ctx.SetAll(rootIdent, newStruct)
			}
		}
	}
	return nil
}

// Performs a context set but propagates the set up context chain if its a reassignment
func setOrUpdateForAssignment(assgn ast.Node, name string, res any, ctx *Context) {
	switch decl := assgn.(type) {
	case *ast.Identifier:
		ctx.SetAll(name, res)
	case *ast.DotExpression:
		rootIdent := decl.TokenLiteral()
		fieldPath := getFieldPath(decl)
		currentValue, _ := ctx.Get(rootIdent)
		newStruct := updateStructField(currentValue, fieldPath, res)
		ctx.SetAll(rootIdent, newStruct)
	default:
		ctx.Set(name, res)
	}
}

func (e *Evaluator) evalIfElseExpression(n *ast.IfElseExpression, stk *Context) any {
	stk.Scope()
	defer stk.Unscope()

	for _, c := range n.Conditionals {
		// This means else block matched
		if c.Condition == nil {
			return e.eval(c.Block, stk)
		}
		val := e.eval(c.Condition, stk)
		var cond bool
		if res, ok := val.(*Return); ok {
			cond = res.Values[0].(bool)
		} else if res, ok := val.(bool); ok {
			cond = res
		} else {
			panic("this is a compiler error. please report")
		}
		if cond {
			return e.eval(c.Block, stk)
		}
	}
	return nil
}

func (e *Evaluator) evalForStatement(n *ast.ForStatement, stk *Context) any {
	stk.Scope()
	defer stk.Unscope()
	// classic for loop
	if n.Assignment != nil {
		return e.evalForClassic(n, stk)
	}

	// conditional loop
	if n.Condition != nil && n.Change == nil {
		return e.evalForConditional(n, stk)
	}

	// infinite loop
	if n.Condition == nil {
		return e.evalForInfinite(n, stk)
	}

	panic("this is a compiler error. please report")
}

func (e *Evaluator) evalForClassic(n *ast.ForStatement, stk *Context) any {
	e.evalAssignmentStatement(n.Assignment, stk)
outer:
	for {
		cond := e.eval(n.Condition, stk)
		if e.state == ERROR {
			return nil
		}
		cond = unwrapFunctionResult(cond, 0)
		if !cond.(bool) {
			break
		}
		exp := e.eval(n.Block, stk)
		if _, ok := exp.(*Return); ok {
			return exp
		}
		switch e.state {
		case ERROR:
			return nil
		case BREAK:
			e.state = NONE
			break outer
		case NEXT:
			e.state = NONE
			e.eval(n.Change, stk)
			continue
		}
		e.eval(n.Change, stk)
	}
	return nil
}

func (e *Evaluator) evalForConditional(n *ast.ForStatement, stk *Context) any {
outer:
	for {
		cond := e.eval(n.Condition, stk)
		if e.state == ERROR {
			return nil
		}
		cond = unwrapFunctionResult(cond, 0)
		if !cond.(bool) {
			break
		}

		exp := e.eval(n.Block, stk)
		if _, ok := exp.(*Return); ok {
			return exp
		}
		switch e.state {
		case ERROR:
			return nil
		case BREAK:
			e.state = NONE
			break outer
		case NEXT:
			e.state = NONE
			continue
		}
	}
	return nil
}

func (e *Evaluator) evalForInfinite(n *ast.ForStatement, stk *Context) any {
outer:
	for {
		exp := e.eval(n.Block, stk)
		if _, ok := exp.(*Return); ok {
			return exp
		}
		switch e.state {
		case ERROR:
			return nil
		case BREAK:
			e.state = NONE
			break outer
		case NEXT:
			e.state = NONE
			continue
		}
	}
	return nil
}

func (e *Evaluator) evalMatchExpressionStatement(n *ast.MatchExpressionStatement, stk *Context) any {
	var scrutinee any
	var bindingName string

	// Handle assignment pattern: match v = expr
	// Only evaluate the right side and bind v to the result
	if infixExp, ok := n.Scrutinee.(*ast.InfixExpression); ok && infixExp.Operator == "=" {
		if ident, ok := infixExp.Left.(*ast.Identifier); ok {
			bindingName = ident.Value
			scrutinee = e.eval(infixExp.Right, stk)
		}
	} else {
		scrutinee = e.eval(n.Scrutinee, stk)
	}
	scrutinee = unwrapFunctionResult(scrutinee, 0)

	stk.Scope()
	defer stk.Unscope()

	typ := types.GetUnderlyingType(n.Scrutinee.Type())
	// Unwrap ImportedNamed to get the actual type for matching
	if imported, ok := typ.(*types.ImportedNamed); ok {
		typ = imported.Typ
	}

	// Check if type is 'any' or a generic with 'any' constraint
	isAnyType := false
	if _, ok := typ.(*types.Any); ok {
		isAnyType = true
	} else if genType, ok := typ.(*types.Generic); ok {
		for _, constraint := range genType.Constraints {
			if _, isAny := constraint.(*types.Any); isAny {
				isAnyType = true
				break
			}
		}
	}

	if isAnyType {
		// Handle matching against 'any' type
		anyVal, ok := scrutinee.(*Any)
		if !ok {
			panic("matching against non-any type")
		}

		for _, c := range n.Cases {
			// check each predicate in the case
			for _, pred := range c.Predicates {
				// For primitive types we check if predicate is TypeLiteral to get
				// type name as semantic analyzer may set type to scrutinee type
				var typeName string
				if typeLit, ok := pred.(*ast.TypeLiteral); ok {
					typeName = typeLit.TokenLiteral()
				} else {
					typeName = normaliseTypeDescriptorName(pred.Type(), stk.libPath, e)
				}
				caseDescriptor := generateTypeDescriptor(typeName)

				if caseDescriptor == anyVal.descriptor {
					return e.evalMatchCase(c, stk)
				}
			}
		}
	} else if _, ok := typ.(*types.Union); ok {
		// Unwrap Any if the value came from a generic function
		if anyVal, ok := scrutinee.(*Any); ok {
			scrutinee = anyVal.value
		}
		unionVal, ok := scrutinee.(*Union)
		if !ok {
			panic(fmt.Sprintf("matching against non-union type (got %T)", scrutinee))
		}

		for _, c := range n.Cases {
			// check each predicate in the case
			for _, pred := range c.Predicates {
				// Determine the type name for descriptor generation
				var typeName string
				if typeLit, ok := pred.(*ast.TypeLiteral); ok {
					typeName = typeLit.TokenLiteral()
				} else {
					typeName = normaliseTypeDescriptorName(pred.Type(), stk.libPath, e)
				}
				caseDescriptor := generateTypeDescriptor(typeName)

				if caseDescriptor == unionVal.descriptor {
					// Update binding to the inner value
					if bindingName != "" {
						stk.Set(bindingName, unionVal.value)
					}
					return e.evalMatchCase(c, stk)
				}
			}
		}
	} else if _, ok := typ.(*types.Error); ok {
		errVal, ok := scrutinee.(*Error)
		if !ok {
			panic("matching against non-error type")
		}

		for _, c := range n.Cases {
			// check each predicate in the case
			for range c.Predicates {
				typeName := errVal.Err
				caseDescriptor := generateTypeDescriptor(typeName)

				if caseDescriptor == errVal.descriptor {
					// update binding to the underlying error type
					if bindingName != "" {
						stk.Set(bindingName, errVal.Args)
					}
					return e.evalMatchCase(c, stk)
				}
			}
		}
	} else {
		for _, c := range n.Cases {
			// Check each predicate in the case
			for _, pred := range c.Predicates {
				predValue := e.eval(pred, stk)
				if e.evalInfixEqual(predValue, scrutinee).(bool) {
					return e.evalMatchCase(c, stk)
				}
			}
		}
	}

	if n.Default != nil {
		return e.evalMatchCase(n.Default, stk)
	}

	panic("this is a compiler error. please report")

}
func (e *Evaluator) evalMatchCase(c *ast.MatchCase, stk *Context) any {
	var last any
	for _, stmt := range c.Body {
		last = e.eval(stmt, stk)
		if _, ok := last.(*Return); ok {
			return last
		}
		if e.state == BREAK || e.state == NEXT {
			return nil
		}
	}
	return last
}

// returns last expression in block if any otherwise nil
func (e *Evaluator) evalBlockStatement(n *ast.BlockStatement, stk *Context) any {
	var exp any
	for _, stmt := range n.Statements {
		exp = e.eval(stmt, stk)
		switch e.state {
		case BREAK, NEXT, ERROR:
			return nil
		}
		// Handle function call results (unwrap *Return to get value)
		if ret, ok := exp.(*Return); ok {
			switch stmt.(type) {
			case *ast.TryExpression:
				if e.state == ERROR {
					return nil
				}
				exp = unwrapFunctionResult(ret, 0)
			case *ast.FunctionCallExpression:
				exp = unwrapFunctionResult(ret, 0)
			case *ast.CatchExpression:
				exp = unwrapFunctionResult(ret, 0)
			default:
				return exp
			}
		}
	}
	return exp
}

func (e *Evaluator) evalKeywordStatement(n *ast.KeywordStatement) any {
	switch n.Token.Type {
	case token.BREAK:
		e.state = BREAK
		return nil
	case token.NEXT:
		e.state = NEXT
		return nil
	default:
		panic("this is a compiler error please report")
	}
}

func (e *Evaluator) evalReturnStatement(n *ast.ReturnStatement, stk *Context) any {
	var vals []any
	for i := range n.Values {
		res := e.eval(n.Values[i], stk)
		switch n.Values[i].(type) {
		case *ast.FunctionCallExpression, *ast.TryExpression, *ast.MatchExpressionStatement, *ast.CatchExpression:
			if ret, ok := res.(*Return); ok {
				for j := range ret.Values {
					vals = append(vals, unwrapFunctionResult(ret, j))
				}
			} else {
				vals = append(vals, res)
			}

		default:
			vals = append(vals, res)
		}
	}
	return &Return{Values: vals}
}

func (e *Evaluator) evalDotExpression(n *ast.DotExpression, stk *Context) any {
	// case 1: library access
	if leftIdent, ok := n.Left.(*ast.Identifier); ok {
		if libCtx, isLibrary := e.ctxs[leftIdent.Value]; isLibrary {
			return e.evalLibraryAccess(libCtx, n.Right, n.Type(), stk)
		}
	}

	// case 2: local access
	leftValue := e.eval(n.Left, stk)
	if leftValue == nil {
		panic("this is a compiler error. please report")
	}
	if ret, ok := leftValue.(*Return); ok {
		leftValue = unwrapFunctionResult(ret, 0)
	}
	// Auto-dereference pointers for field access
	leftValue = unwrapPointer(leftValue)

	return e.evalLocalAccess(leftValue, n.Right, stk)
}

func (e *Evaluator) evalLibraryAccess(libCtx *Context, right ast.Expression, exprType types.Type, stk *Context) any {
	switch right := right.(type) {
	case *ast.Identifier:
		// lib.variable or lib.enum access
		if val, ok := libCtx.Get(right.Value); ok {
			return val
		}

	case *ast.FunctionCallExpression:
		name := right.TokenLiteral()

		// Check if it's a type cast: lib.Type(value)
		if _, ok := libCtx.GetType(name); ok {
			res := e.eval(right.Arguments[0], stk)
			// Check if it's a union type (including imported unions)
			// Use exprType from the DotExpression since right.T may be nil
			isUnion := false
			if _, ok := exprType.(*types.Union); ok {
				isUnion = true
			} else if imported, ok := exprType.(*types.ImportedNamed); ok {
				if _, ok := imported.Typ.(*types.Union); ok {
					isUnion = true
				}
			}
			if isUnion {
				argType := right.Arguments[0].Type()
				argType = types.StripMultiType(argType)
				// Use argType (the variant type) for descriptor, not exprType (the union type)
				typeName := normaliseTypeDescriptorName(argType, libCtx.libPath, e)
				descriptor := generateTypeDescriptor(typeName)

				res = &Union{
					descriptor: descriptor,
					value:      res,
				}
			}
			return res
		}

		// Otherwise it's a function call: lib.function(args...)
		if fnVal, ok := libCtx.Get(name); ok {
			fn := fnVal.(*Function)
			return e.evalFunction(fn, right.Arguments, stk)
		}
	}
	panic("this is a compiler error. please report")
}

func (e *Evaluator) evalLocalAccess(leftValue any, right ast.Expression, stk *Context) any {
	switch right := right.(type) {
	case *ast.Identifier:
		// named field access: struct.field, enum.field
		if fields, ok := leftValue.(map[string]any); ok {
			if val, exists := fields[right.Value]; exists {
				return val
			}
		}
		return nil

	case *ast.IntegerLiteral:
		// unnamed struct field access by index: struct.0
		if fields, ok := leftValue.(map[string]any); ok {
			key := fmt.Sprintf("%d", right.Value)
			if val, exists := fields[key]; exists {
				return val
			}
		}
		return nil

	case *ast.FunctionCallExpression:
		// This assumes the left side evaluated to a function directly
		if fn, ok := leftValue.(*Function); ok {
			return e.evalFunction(fn, right.Arguments, stk)
		}
		panic("this is a compiler error. please report")
		// case *ast.TryExpression:
		// 	call := right.Right.(*ast.FunctionCallExpression)
		// 	if fn, ok := leftValue.(*Function); ok {
		// 		return e.somefun(fn, call.Arguments, stk)
		// 	}
	}
	return nil
}

func indexArray(arr any, idx int) any {
	switch arr := arr.(type) {
	case []any:
		if idx < 0 || idx >= len(arr) {
			// TODO: add dash error handling
			panic("index out of bounds")
		}
		return arr[idx]
	case []uint8:
		if idx < 0 || idx >= len(arr) {
			// TODO: add dash error handling
			panic("index out of bounds")
		}
		return arr[idx]
	case string:
		return arr[idx]
	default:
		panic("this is a compiler error. please report")

	}
}

// ----------------- //
// Prefix expression //
// ----------------- //

func (e *Evaluator) evalPrefixExpression(n *ast.PrefixExpression, stk *Context) any {
	// Special handling for address-of op
	if n.Token.Type == token.AMPERSAND {
		return e.evalPrefixAddressOf(n.Right, stk)
	}

	val := e.eval(n.Right, stk)
	val = unwrapFunctionResult(val, 0)
	var err error
	switch n.Token.Type {
	case token.MINUS:
		val, err = e.evalPrefixMinus(val)
	case token.BANG:
		val, err = e.evalPrefixNot(val)
	case token.BNOT:
		val, err = e.evalPrefixBitwiseNot(val)
	case token.OPTIONAL:
		val = e.evalPrefixOptional(val)
	case token.ASTERISK:
		val = e.evalPrefixDereference(val)
	}

	if err != nil {
		e.addError(n, err)
	}
	return val
}

func (e *Evaluator) evalPrefixMinus(v any) (any, error) {
	switch v := v.(type) {
	case int8:
		return -v, nil
	case int16:
		return -v, nil
	case int32:
		return -v, nil
	case int64:
		return -v, nil
	case uint8:
		return -v, nil
	case uint16:
		return -v, nil
	case uint32:
		return -v, nil
	case uint64:
		return -v, nil
	case float32:
		return -v, nil
	case float64:
		return -v, nil
	default:
		return nil, fmt.Errorf("cannot apply prefix minus to type %T", v)
	}
}

func (e *Evaluator) evalPrefixNot(v any) (any, error) {
	switch v := v.(type) {
	case bool:
		return !v, nil
	case *Return:
		return !v.Values[0].(bool), nil
	default:
		return nil, fmt.Errorf("cannot apply not to type %T", v)
	}
}

func (e *Evaluator) evalPrefixBitwiseNot(v any) (any, error) {
	switch v := v.(type) {
	case int64:
		return ^v, nil
	case uint8:
		return ^v, nil
	case uint32:
		return ^v, nil
	case uint64:
		return ^v, nil
	default:
		return nil, fmt.Errorf("cannot apply bitwise NOT to type %T", v)
	}
}

func (e *Evaluator) evalPrefixOptional(v any) any {
	if opt, ok := v.(Optional); ok {
		if !opt.isValid {
			e.setError(&Error{descriptor: errForceUnwrapNull, Err: "runtime.force_unwrap_null"})
			return nil
		}
		return opt.value
	}
	if v == nil {
		panic("this is a compiler error. please report")
	}
	return v
}

func (e *Evaluator) evalPrefixAddressOf(expr ast.Expression, ctx *Context) any {
	var locationID uint64
	switch expr := expr.(type) {
	case *ast.Identifier:
		// For identifiers, hash the context + variable name to ensure
		// &x always returns the same pointer
		locationID = computeLocationHash(ctx, expr.Value)

	default:
		// For other expressions evaluate and create a new pointer
		val := e.eval(expr, ctx)
		ptr := &Pointer{
			id:    e.nextPointerID,
			value: val,
		}
		e.nextPointerID++
		return ptr
	}

	if ptr, exists := e.pointerCache[locationID]; exists {
		return ptr
	}

	// Create a new pointer for this location
	val := e.eval(expr, ctx)
	ptr := &Pointer{
		id:    locationID,
		value: val,
	}
	e.pointerCache[locationID] = ptr
	return ptr
}

func (e *Evaluator) evalPrefixDereference(v any) any {
	ptr, ok := v.(*Pointer)
	if !ok {
		panic(fmt.Sprintf("cannot dereference non-pointer type %T", v))
	}
	return ptr.value
}

// ---------------- //
// Infix expression //
// ---------------- //

func (e *Evaluator) evalInfixExpression(n *ast.InfixExpression, ctx *Context) any {
	l := e.eval(n.Left, ctx)
	if ret, ok := l.(*Return); ok {
		l = unwrapFunctionResult(ret, 0)
	}
	l = unwrapAny(l)
	r := e.eval(n.Right, ctx)
	if ret, ok := r.(*Return); ok {
		r = unwrapFunctionResult(ret, 0)
	}
	r = unwrapAny(r)

	var val any
	var err error
	switch n.Token.Type {
	// Arithmetic
	case token.PLUS:
		val, err = e.evalInfixAdd(l, r)
	case token.MINUS:
		val, err = e.evalInfixSub(l, r)
	case token.ASTERISK:
		val, err = e.evalInfixMul(l, r)
	case token.SLASH:
		val, err = e.evalInfixDiv(l, r)
	case token.MOD:
		val, err = e.evalInfixMod(l, r)
	// Logical
	case token.AND:
		val, err = e.evalInfixAnd(l, r)
	case token.OR:
		val, err = e.evalInfixOr(l, r)
	// Relational
	case token.GT:
		val, err = e.evalInfixGreater(l, r)
	case token.GTE:
		val, err = e.evalInfixGreaterEqual(l, r)
	case token.LT:
		val, err = e.evalInfixLess(l, r)
	case token.LTE:
		val, err = e.evalInfixLessEqual(l, r)
	// Equality
	case token.EQ:
		val = e.evalInfixEqual(l, r)
	case token.NEQ:
		val = !(e.evalInfixEqual(l, r).(bool))
	// Optional
	case token.NULL_COALESCE:
		val = e.evalInfixNullCoalesce(l, r)
	// Assignment
	case token.ASSIGN:
		val = e.evalInfixAssign(n, r, ctx)
	// Bitwise
	case token.LSHIFT:
		val, err = e.evalInfixLeftShift(l, r)
	case token.RSHIFT:
		val, err = e.evalInfixRightShift(l, r)
	case token.AMPERSAND:
		val, err = e.evalInfixBitwiseAnd(l, r)
	case token.BAR:
		val, err = e.evalInfixBitwiseOr(l, r)
	case token.CARET:
		val, err = e.evalInfixBitwiseXor(l, r)
	// Special
	case token.COLON:
		val = []any{l, r}
	case token.PIPE:
		panic("pipe operator not implemented")
	}

	if err != nil {
		e.addError(n, err)
	}

	return val
}

func (e *Evaluator) evalInfixAdd(l, r any) (res any, err error) {
	switch l := l.(type) {
	case int64:
		var r_ int64
		r_, err = castTo[int64](r)
		res = l + r_
	case uint8:
		var r_ uint8
		r_, err = castTo[uint8](r)
		res = l + r_
	case uint16:
		var r_ uint16
		r_, err = castTo[uint16](r)
		res = l + r_
	case uint32:
		var r_ uint32
		r_, err = castTo[uint32](r)
		res = l + r_
	case uint64:
		var r_ uint64
		r_, err = castTo[uint64](r)
		res = l + r_
	case float64:
		var r_ float64
		r_, err = castTo[float64](r)
		res = l + r_
	case string:
		var r_ string
		r_, err = castTo[string](r)
		res = l + r_
	}

	return
}

func (e *Evaluator) evalInfixSub(l, r any) (res any, err error) {
	switch l := l.(type) {
	case int64:
		var r_ int64
		r_, err = castTo[int64](r)
		res = l - r_
	case uint8:
		var r_ uint8
		r_, err = castTo[uint8](r)
		res = l - r_
	case uint16:
		var r_ uint16
		r_, err = castTo[uint16](r)
		res = l - r_
	case uint32:
		var r_ uint32
		r_, err = castTo[uint32](r)
		res = l - r_
	case uint64:
		var r_ uint64
		r_, err = castTo[uint64](r)
		res = l - r_
	case float64:
		var r_ float64
		r_, err = castTo[float64](r)
		res = l - r_
	}
	return
}

func (e *Evaluator) evalInfixMul(l, r any) (res any, err error) {
	switch l := l.(type) {
	case int64:
		var r_ int64
		r_, err = castTo[int64](r)
		res = l * r_
	case uint8:
		var r_ uint8
		r_, err = castTo[uint8](r)
		res = l * r_
	case uint16:
		var r_ uint16
		r_, err = castTo[uint16](r)
		res = l * r_
	case uint32:
		var r_ uint32
		r_, err = castTo[uint32](r)
		res = l * r_
	case uint64:
		var r_ uint64
		r_, err = castTo[uint64](r)
		res = l * r_
	case float64:
		var r_ float64
		r_, err = castTo[float64](r)
		res = l * r_
	}
	return
}

func (e *Evaluator) evalInfixDiv(l, r any) (res any, err error) {
	switch l := l.(type) {
	case int64:
		var r_ int64
		r_, err = castTo[int64](r)
		if r_ == 0 {
			return nil, fmt.Errorf("division by zero")
		}
		res = l / r_
	case uint8:
		var r_ uint8
		r_, err = castTo[uint8](r)
		if r_ == 0 {
			return nil, fmt.Errorf("division by zero")
		}
		res = l / r_
	case uint16:
		var r_ uint16
		r_, err = castTo[uint16](r)
		if r_ == 0 {
			return nil, fmt.Errorf("division by zero")
		}
		res = l / r_
	case uint32:
		var r_ uint32
		r_, err = castTo[uint32](r)
		if r_ == 0 {
			return nil, fmt.Errorf("division by zero")
		}
		res = l / r_
	case uint64:
		var r_ uint64
		r_, err = castTo[uint64](r)
		if r_ == 0 {
			return nil, fmt.Errorf("division by zero")
		}
		res = l / r_
	case float64:
		var r_ float64
		r_, err = castTo[float64](r)
		if r_ == 0 {
			return nil, fmt.Errorf("division by zero")
		}
		res = l / r_
	}
	return
}

func (e *Evaluator) evalInfixMod(l, r any) (res any, err error) {
	switch l := l.(type) {
	case int64:
		var r_ int64
		r_, err = castTo[int64](r)
		if r_ == 0 {
			return nil, fmt.Errorf("modulo by zero")
		}
		res = l % r_
	case uint8:
		var r_ uint8
		r_, err = castTo[uint8](r)
		if r_ == 0 {
			return nil, fmt.Errorf("modulo by zero")
		}
		res = l % r_
	case uint16:
		var r_ uint16
		r_, err = castTo[uint16](r)
		if r_ == 0 {
			return nil, fmt.Errorf("modulo by zero")
		}
		res = l % r_
	case uint32:
		var r_ uint32
		r_, err = castTo[uint32](r)
		if r_ == 0 {
			return nil, fmt.Errorf("modulo by zero")
		}
		res = l % r_
	case uint64:
		var r_ uint64
		r_, err = castTo[uint64](r)
		if r_ == 0 {
			return nil, fmt.Errorf("modulo by zero")
		}
		res = l % r_
	default:
		err = fmt.Errorf("unsupported type for modulo %T", l)
	}
	return
}

func (e *Evaluator) evalInfixLeftShift(l, r any) (res any, err error) {
	switch l := l.(type) {
	case uint8:
		r_, _ := castTo[uint8](r)
		res = l << r_
	case uint16:
		r_, _ := castTo[uint16](r)
		res = l << r_
	case uint32:
		r_, _ := castTo[uint32](r)
		res = l << r_
	case uint64:
		r_, _ := castTo[uint64](r)
		res = l << r_
	case int64:
		r_, _ := castTo[int64](r)
		res = l << r_
	default:
		err = fmt.Errorf("unsupported type for left shift %T", l)
	}
	return
}

func (e *Evaluator) evalInfixRightShift(l, r any) (res any, err error) {
	switch l := l.(type) {
	case uint8:
		r_, _ := castTo[uint8](r)
		res = l >> r_
	case uint16:
		r_, _ := castTo[uint16](r)
		res = l >> r_
	case uint32:
		r_, _ := castTo[uint32](r)
		res = l >> r_
	case uint64:
		r_, _ := castTo[uint64](r)
		res = l >> r_
	case int64:
		r_, _ := castTo[int64](r)
		res = l >> r_
	default:
		err = fmt.Errorf("unsupported type for left shift %T", l)
	}
	return
}

func (e *Evaluator) evalInfixBitwiseAnd(l, r any) (res any, err error) {
	// Preserve the type of the left operand for the result
	switch l := l.(type) {
	case int64:
		r64, err := castTo[int64](r)
		if err != nil {
			return nil, err
		}
		res = l & r64
	case uint64:
		r64, err := castTo[uint64](r)
		if err != nil {
			return nil, err
		}
		res = l & r64
	case uint8:
		r8, err := castTo[uint8](r)
		if err != nil {
			return nil, err
		}
		res = l & r8
	case uint32:
		r32, err := castTo[uint32](r)
		if err != nil {
			return nil, err
		}
		res = l & r32
	default:
		return nil, fmt.Errorf("unsupported type for bitwise AND %T", l)
	}
	return
}

func (e *Evaluator) evalInfixBitwiseOr(l, r any) (res any, err error) {
	// Preserve the type of the left operand for the result
	switch l := l.(type) {
	case int64:
		r64, err := castTo[int64](r)
		if err != nil {
			return nil, err
		}
		res = l | r64
	case uint64:
		r64, err := castTo[uint64](r)
		if err != nil {
			return nil, err
		}
		res = l | r64
	case uint8:
		r8, err := castTo[uint8](r)
		if err != nil {
			return nil, err
		}
		res = l | r8
	case uint32:
		r32, err := castTo[uint32](r)
		if err != nil {
			return nil, err
		}
		res = l | r32
	default:
		return nil, fmt.Errorf("unsupported type for bitwise OR %T", l)
	}
	return
}

func (e *Evaluator) evalInfixBitwiseXor(l, r any) (res any, err error) {
	// Preserve the type of the left operand for the result
	switch l := l.(type) {
	case int64:
		r64, err := castTo[int64](r)
		if err != nil {
			return nil, err
		}
		res = l ^ r64
	case uint64:
		r64, err := castTo[uint64](r)
		if err != nil {
			return nil, err
		}
		res = l ^ r64
	case uint8:
		r8, err := castTo[uint8](r)
		if err != nil {
			return nil, err
		}
		res = l ^ r8
	case uint32:
		r32, err := castTo[uint32](r)
		if err != nil {
			return nil, err
		}
		res = l ^ r32
	default:
		return nil, fmt.Errorf("unsupported type for bitwise XOR %T", l)
	}
	return
}

func (e *Evaluator) evalInfixAnd(l, r any) (res any, err error) {
	switch l := l.(type) {
	case bool:
		var r_ bool
		r_, err = castTo[bool](r)
		res = l && r_
	}
	return
}

func (e *Evaluator) evalInfixOr(l, r any) (res any, err error) {
	switch l := l.(type) {
	case bool:
		var r_ bool
		r_, err = castTo[bool](r)
		res = l || r_
	}
	return
}

func (e *Evaluator) evalInfixLess(l, r any) (res any, err error) {
	switch l := l.(type) {
	case int64:
		var r_ int64
		r_, err = castTo[int64](r)
		res = l < r_
	case uint8:
		var r_ uint8
		r_, err = castTo[uint8](r)
		res = l < r_
	case uint16:
		var r_ uint16
		r_, err = castTo[uint16](r)
		res = l < r_
	case uint32:
		var r_ uint32
		r_, err = castTo[uint32](r)
		res = l < r_
	case uint64:
		var r_ uint64
		r_, err = castTo[uint64](r)
		res = l < r_
	case float64:
		var r_ float64
		r_, err = castTo[float64](r)
		res = l < r_
	}
	return
}

func (e *Evaluator) evalInfixGreater(l, r any) (res any, err error) {
	switch l := l.(type) {
	case int64:
		var r_ int64
		r_, err = castTo[int64](r)
		res = l > r_
	case uint8:
		var r_ uint8
		r_, err = castTo[uint8](r)
		res = l > r_
	case uint16:
		var r_ uint16
		r_, err = castTo[uint16](r)
		res = l > r_
	case uint32:
		var r_ uint32
		r_, err = castTo[uint32](r)
		res = l > r_
	case uint64:
		var r_ uint64
		r_, err = castTo[uint64](r)
		res = l > r_
	case float64:
		var r_ float64
		r_, err = castTo[float64](r)
		res = l > r_
	}
	return
}

func (e *Evaluator) evalInfixLessEqual(l, r any) (res any, err error) {
	switch l := l.(type) {
	case int64:
		var r_ int64
		r_, err = castTo[int64](r)
		res = l <= r_
	case uint8:
		var r_ uint8
		r_, err = castTo[uint8](r)
		res = l <= r_
	case uint16:
		var r_ uint16
		r_, err = castTo[uint16](r)
		res = l <= r_
	case uint32:
		var r_ uint32
		r_, err = castTo[uint32](r)
		res = l <= r_
	case uint64:
		var r_ uint64
		r_, err = castTo[uint64](r)
		res = l <= r_
	case float64:
		var r_ float64
		r_, err = castTo[float64](r)
		res = l <= r_
	}
	return
}

func (e *Evaluator) evalInfixGreaterEqual(l, r any) (res any, err error) {
	switch l := l.(type) {
	case int64:
		var r_ int64
		r_, err = castTo[int64](r)
		res = l >= r_
	case uint8:
		var r_ uint8
		r_, err = castTo[uint8](r)
		res = l >= r_
	case uint16:
		var r_ uint16
		r_, err = castTo[uint16](r)
		res = l >= r_
	case uint32:
		var r_ uint32
		r_, err = castTo[uint32](r)
		res = l >= r_
	case uint64:
		var r_ uint64
		r_, err = castTo[uint64](r)
		res = l >= r_
	case float64:
		var r_ float64
		r_, err = castTo[float64](r)
		res = l >= r_
	}
	return
}

func (e *Evaluator) evalInfixEqual(l, r any) any {
	if l_, ok := l.(Optional); ok {
		l = l_.value
	}
	if r_, ok := r.(Optional); ok {
		r = r_.value
	}

	// special handling for errors,
	// where we compare type descriptor
	// and then field by field
	if lErr, ok := l.(*Error); ok {
		if rErr, ok := r.(*Error); ok {
			if lErr.descriptor != rErr.descriptor {
				return false
			}

			if lErr.Err != rErr.Err {
				return false
			}

			if len(lErr.Args) != len(rErr.Args) {
				return false
			}

			for k := range lErr.Args {
				if !(e.evalInfixEqual(lErr.Args[k], rErr.Args[k]).(bool)) {
					return false
				}
			}
			return true
		}
		return false
	}
	if _, ok := r.(*Error); ok {
		return false
	}

	// special handling for dash pointers
	if lPtr, ok := l.(*Pointer); ok {
		if rPtr, ok := r.(*Pointer); ok {
			return lPtr.id == rPtr.id
		}
		return false
	}
	if _, ok := r.(*Pointer); ok {
		return false
	}

	return l == r
}

func (e *Evaluator) evalInfixNullCoalesce(l, r any) any {
	if opt, ok := l.(Optional); ok {
		// NOTE: we can optimise by only computing r if not valid
		if !opt.isValid {
			return r
		}
		return opt.value
	}
	if l == nil {
		panic("this is a compiler error. please report")
	}
	return l
}

// Handle assignment operations in contexts where expressions were expected
// e.g. for loop increment
func (e *Evaluator) evalInfixAssign(n *ast.InfixExpression, r any, ctx *Context) any {
	// left exp has to be an identifier
	ident, ok := n.Left.(*ast.Identifier)
	if !ok {
		panic("assignment to non-identifier in infix expression")
	}

	ctx.SetAll(ident.Value, r)

	return r
}

// ------------------ //
// Postfix expression //
// ------------------ //

func (e *Evaluator) evalPostfixExpression(n *ast.PostfixExpression, stk *Context) {
	left := e.eval(n.Left, stk)

	var isIncr bool
	switch n.Token.Type {
	case token.INCR:
		isIncr = true
	case token.DECR:
		isIncr = false
	default:
		panic("this is a compiler error. please report")
	}

	var newVal any
	switch v := left.(type) {
	case uint8:
		if isIncr {
			newVal = v + 1
		} else {
			newVal = v - 1
		}
	case uint16:
		if isIncr {
			newVal = v + 1
		} else {
			newVal = v - 1
		}
	case uint32:
		if isIncr {
			newVal = v + 1
		} else {
			newVal = v - 1
		}
	case uint64:
		if isIncr {
			newVal = v + 1
		} else {
			newVal = v - 1
		}
	case int64:
		if isIncr {
			newVal = v + 1
		} else {
			newVal = v - 1
		}
	default:
		panic("this is a compiler error. please report")
	}

	stk.Set(n.Left.String(), newVal)
}

// -------- //
// Literals //
// -------- //

func (e *Evaluator) evalStructLiteral(n *ast.StructLiteral, stk *Context) any {
	strct := make(map[string]any)
	if n.Copy != nil {
		copyVal := e.eval(n.Copy, stk)
		// If the copy source is a pointer, unwrap it
		if ptr, ok := copyVal.(*Pointer); ok {
			copyVal = ptr.value
		}
		s := copyVal.(map[string]any)
		maps.Copy(strct, s)
	}

	for _, field := range n.Fields {
		var name string
		// we distinguish between named fields and unnamed
		if field.Name != nil {
			name = field.Name.Value
		} else {
			name = fmt.Sprintf("%d", field.Index)
		}
		val := e.eval(field.Value, stk)
		// Unwrap Return structs to get the actual value
		val = unwrapFunctionResult(val, 0)

		// Cast the value to the correct type based on the field's type
		if field.T != nil {
			underlyingType := types.GetUnderlyingType(field.T)
			switch t := underlyingType.(type) {
			case *types.Int:
				val = e.evalIntCast(t, val)
			case *types.Byte:
				val = e.evalByteCast(val)
			case *types.Char:
				val = e.evalCharCast(val)
			}
			// Unwrap Any if field type is not 'any' type
			// This handles values from generic functions that are wrapped in Any{}
			// but need to be stored as concrete types in struct fields
			val = unwrapAny(val)
		}

		strct[name] = val
	}
	if _, ok := n.T.(*types.Error); ok {
		typeName := normaliseTypeDescriptorName(n.T, stk.libPath, e)
		typeDesc := generateTypeDescriptor(typeName)
		return &Error{
			descriptor: typeDesc,
			Err:        typeName,
			Args:       strct,
		}
	}
	return strct
}

// -------------- //
// Error handling //
// -------------- //

func (e *Evaluator) evalTryExpression(n *ast.TryExpression, ctx *Context) any {
	return e.eval(n.Right, ctx)
}

func (e *Evaluator) evalCatchExpression(n *ast.CatchExpression, ctx *Context) any {
	result := e.eval(n.Left, ctx)

	if e.state == ERROR {
		// We save the caught error so it is accessible
		// within catch block and clear it globally to
		// avoid premature return. the catch block can
		// either set a default, return something else
		// or raise the error causing it to be propagated
		caughtErr := e.stateVar
		e.stateVar = nil
		e.state = NONE
		ctx.Set(n.Ident.Value, caughtErr)

		catchResult := e.evalBlockStatement(n.Block, ctx)
		return catchResult
	}

	return result
}

func (e *Evaluator) evalRaiseStatement(n *ast.RaiseStatement, ctx *Context) any {
	// Evaluate the error expression to get the actual error value/name
	errVal := e.eval(n.Error, ctx)
	assertType[*Error](errVal)
	e.setError(errVal.(*Error))

	return &Return{Values: []any{}}
}

// ------------------ //
// Built-in functions //
// ------------------ //

func (e *Evaluator) evalBuiltinFunction(n *ast.FunctionCallExpression, ctx *Context) (any, bool) {
	switch n.TokenLiteral() {
	case "len":
		return e.evalLen(n.Arguments, ctx), true
	case "println":
		return e.evalPrintln(n.Arguments, ctx), true
	case "make":
		return e.evalMake(n.Arguments, ctx), true
	case "assert":
		return e.evalAssert(n.Arguments, ctx), true
	case "append":
		return e.evalAppend(n.Arguments, ctx), true
	case "put":
		return e.evalPut(n.Arguments, ctx), true
	case "get":
		return e.evalGet(n.Arguments, ctx), true
	case "slice":
		return e.evalSlice(n.Arguments, ctx), true
	default:
		return nil, false
	}
}

func (e *Evaluator) evalCustomTypeCast(n *ast.FunctionCallExpression, ctx *Context) (any, bool) {
	if _, ok := ctx.GetType(n.TokenLiteral()); !ok {
		return nil, false
	}

	res := e.eval(n.Arguments[0], ctx)
	// if cast is a union we need to also
	// generate a type descriptor
	if unionType, ok := n.T.(*types.Union); ok {
		argType := n.Arguments[0].Type()

		// if argType is a generic type (like E), use the first union variant
		if _, isGeneric := argType.(*types.Generic); isGeneric {
			if len(unionType.Ts) > 0 {
				argType = unionType.Ts[0]
			}
		}

		typeName := normaliseTypeDescriptorName(argType, ctx.libPath, e)
		descriptor := generateTypeDescriptor(typeName)

		res = &Union{
			descriptor: descriptor,
			value:      res,
		}
	}
	return &Return{Values: []any{res}}, true
}

func (e *Evaluator) evalLen(args []ast.Expression, stk *Context) any {
	if len(args) != 1 {
		panic("this is a compiler error. please report")
	}

	var val any
	switch args[0].(type) {
	case *ast.FunctionCallExpression:
		val = unwrapFunctionResult(e.eval(args[0], stk), 0)
	default:
		val = e.eval(args[0], stk)
	}

	val = unwrapFunctionResult(val, 0)

	var n int64
	switch v := val.(type) {
	case []any:
		n = int64(len(v))
	case []uint8:
		n = int64(len(v))
	case string:
		n = int64(len(v))
	default:
		panic("this is a compiler error. please report")
	}
	return &Return{Values: []any{n}}
}

func (e *Evaluator) evalPrintln(args []ast.Expression, stk *Context) any {
	values := make([]string, 0, len(args))

	// evaluate each argument and convert to string
	for _, arg := range args {
		a := e.eval(arg, stk)
		switch a := a.(type) {
		case *Return:
			rets := a.Values
			for _, ret := range rets {
				values = append(values, valueToString(ret))
			}
		default:
			values = append(values, valueToString(a))
		}
	}

	n, err := fmt.Println(strings.Join(values, " "))
	if err != nil {
		panic("error when printing to console: " + strings.Join(values, " "))
	}
	return n
}

func (e *Evaluator) evalMake(args []ast.Expression, stk *Context) any {

	var arr []any
	if len(args) == 2 {
		size := e.eval(args[1], stk)
		sizeVal, _ := size.(int64)
		if sizeVal < 0 {
			panic("this is a compiler error. please report")
		}
		arr = make([]any, sizeVal)
	} else {
		len := e.eval(args[1], stk)
		lenVal, _ := len.(int64)
		if lenVal < 0 {
			panic("this is a compiler error. please report")
		}
		size := e.eval(args[2], stk)
		sizeVal, _ := size.(int64)
		if sizeVal < 0 || lenVal > sizeVal {
			panic("this is a compiler error. please report")
		}
		arr = make([]any, lenVal, sizeVal)
	}

	return &Return{Values: []any{arr}}
}

func (e *Evaluator) evalAssert(args []ast.Expression, ctx *Context) any {
	var cond bool
	val := e.eval(args[0], ctx)
	if ret, ok := val.(*Return); ok {
		cond = ret.Values[0].(bool)
	} else {
		cond = val.(bool)
	}

	if !cond {
		msg := e.eval(args[1], ctx)
		// avoid overwritting error when evaluating args[1]
		if e.state == ERROR {
			return nil
		}
		msg = unwrapFunctionResult(msg, 0)
		typeDesc := generateTypeDescriptor("std.assert")
		e.setError(&Error{descriptor: typeDesc, Err: msg.(string)})
	}
	return nil
}

func (e *Evaluator) evalAppend(args []ast.Expression, ctx *Context) any {
	arr := e.eval(args[0], ctx)
	arr = unwrapFunctionResult(arr, 0)

	var newArr any
	switch arr := arr.(type) {
	case []any:
		val := e.eval(args[1], ctx)
		val = unwrapFunctionResult(val, 0)
		arrCopy := make([]any, len(arr))
		copy(arrCopy, arr)
		if anyArr, ok := val.([]any); ok {
			newArr = append(arrCopy, anyArr...)
		} else if byteArr, ok := val.([]uint8); ok {
			nArr := make([]uint8, len(arr))
			for i, el := range arr {
				switch el := el.(type) {
				case int32:
					nArr[i] = uint8(el)
				case uint8:
					nArr[i] = el
				}
			}
			newArr = append(nArr, byteArr...)
		} else {
			newArr = append(arrCopy, val)
		}
	case []uint8:
		val := e.eval(args[1], ctx)
		val = unwrapFunctionResult(val, 0)
		arrCopy := make([]uint8, len(arr))
		copy(arrCopy, arr)
		if byteArr, ok := val.([]uint8); ok {
			newArr = append(arrCopy, byteArr...)
		} else {
			switch val := val.(type) {
			case int32:
				newArr = append(arrCopy, uint8(val))
			case uint8:
				newArr = append(arrCopy, val)
			}
		}
	case string:
		val := e.eval(args[1], ctx)
		val = unwrapFunctionResult(val, 0)
		if strVal, ok := val.(string); ok {
			newArr = arr + strVal
		} else if byteVal, ok := val.(uint8); ok {
			newArr = arr + string([]byte{byteVal})
		} else if charVal, ok := val.(int32); ok {
			newArr = arr + string(charVal)
		} else if charVal, ok := val.(uint32); ok {
			newArr = arr + string(rune(charVal))
		}
	default:
		panic("this is a compiler error. please report")
	}
	return &Return{Values: []any{newArr}}
}

func (e *Evaluator) evalPut(args []ast.Expression, ctx *Context) any {
	// Handle array/map case: put(arr, idx, val)
	arr := e.eval(args[0], ctx)
	idxVal := e.eval(args[1], ctx)
	idxVal = unwrapFunctionResult(idxVal, 0)
	idx := e.toInt64(idxVal)
	val := e.eval(args[2], ctx)
	val = unwrapFunctionResult(val, 0)

	var newArr any
	switch arr := arr.(type) {
	case []any:
		if valArr, ok := val.([]any); ok {
			if idx < 0 || idx+int64(len(valArr)) > int64(len(arr)) {
				debugPrintf("[DEBUG] at %d runtime.index_out_of_bounds, arr:%v, idx:%d\n", args[0].Pos().Line(), arr, idx)
				e.setError(&Error{descriptor: errDescIndexOutOfBounds, Err: "runtime.index_out_of_bounds"})
				return &Return{Values: []any{}}
			}
			newArr = make([]any, len(arr))
			copy(newArr.([]any), arr)
			for i, v := range valArr {
				newArr.([]any)[idx+int64(i)] = v
			}
		} else if valArr, ok := val.([]uint8); ok {
			if idx < 0 || idx+int64(len(valArr)) > int64(len(arr)) {
				debugPrintf("[DEBUG] at %d runtime.index_out_of_bounds, arr:%v, idx:%d\n", args[0].Pos().Line(), arr, idx)
				e.setError(&Error{descriptor: errDescIndexOutOfBounds, Err: "runtime.index_out_of_bounds"})
				return &Return{Values: []any{}}
			}
			newArr = make([]any, len(arr))
			copy(newArr.([]any), arr)
			for i, v := range valArr {
				newArr.([]any)[idx+int64(i)] = v
			}
		} else {
			if idx < 0 || idx >= int64(len(arr)) {
				debugPrintf("[DEBUG] at %d runtime.index_out_of_bounds, arr:%v, idx:%d\n", args[0].Pos().Line(), arr, idx)
				e.setError(&Error{descriptor: errDescIndexOutOfBounds, Err: "runtime.index_out_of_bounds"})
				return &Return{Values: []any{}}
			}
			newArr = make([]any, len(arr))
			copy(newArr.([]any), arr)
			newArr.([]any)[idx] = val
		}
	case []uint8:
		if valArr, ok := val.([]uint8); ok {
			if idx < 0 || idx+int64(len(valArr)) > int64(len(arr)) {
				debugPrintf("[DEBUG] at %d runtime.index_out_of_bounds, arr:%v, idx:%d\n", args[0].Pos().Line(), arr, idx)
				e.setError(&Error{descriptor: errDescIndexOutOfBounds, Err: "runtime.index_out_of_bounds"})
				return &Return{Values: []any{}}
			}
			newArr = make([]uint8, len(arr))
			copy(newArr.([]uint8), arr)
			for i, v := range valArr {
				newArr.([]uint8)[idx+int64(i)] = v
			}
		} else {
			if idx < 0 || idx >= int64(len(arr)) {
				debugPrintf("[DEBUG] at %d runtime.index_out_of_bounds, arr:%v, idx:%d\n", args[0].Pos().Line(), arr, idx)
				e.setError(&Error{descriptor: errDescIndexOutOfBounds, Err: "runtime.index_out_of_bounds"})
				return &Return{Values: []any{}}
			}
			newArr = make([]uint8, len(arr))
			copy(newArr.([]uint8), arr)
			switch val := val.(type) {
			case int32:
				newArr.([]uint8)[idx] = uint8(val)
			case uint8:
				newArr.([]uint8)[idx] = val
			default:
				panic("this is a compiler error. please report")
			}
		}
	default:
		panic("this is a compiler error. please report")
	}
	return &Return{Values: []any{newArr}}
}

func (e *Evaluator) evalGet(args []ast.Expression, ctx *Context) any {
	arr := e.eval(args[0], ctx)
	idxVal := e.eval(args[1], ctx)
	idxVal = unwrapFunctionResult(idxVal, 0)
	idx := e.toInt64(idxVal)

	arr = unwrapFunctionResult(arr, 0)
	arr = unwrapAny(arr)
	var result any
	switch arr := arr.(type) {
	case []any:
		if idx < 0 || idx >= int64(len(arr)) {
			debugPrintf("[DEBUG] at %d idx:%d\n", args[0].Pos().Line(), idx)
			e.setError(&Error{descriptor: errDescIndexOutOfBounds, Err: "runtime.index_out_of_bounds"})
			return &Return{Values: []any{}}
		}
		result = arr[idx]
	case []uint8:
		if idx < 0 || idx >= int64(len(arr)) {
			debugPrintf("[DEBUG] at %d idx:%d\n", args[0].Pos().Line(), idx)
			e.setError(&Error{descriptor: errDescIndexOutOfBounds, Err: "runtime.index_out_of_bounds"})
			return &Return{Values: []any{}}
		}
		result = arr[idx]
	case string:
		if idx < 0 || idx >= int64(len(arr)) {
			debugPrintf("[DEBUG] at %d idx:%d\n", args[0].Pos().Line(), idx)
			e.setError(&Error{descriptor: errDescIndexOutOfBounds, Err: "runtime.index_out_of_bounds"})
			return &Return{Values: []any{}}
		}
		result = arr[idx]
	default:
		panic("this is a compiler error. please report")
	}
	return &Return{Values: []any{result}}
}

func (e *Evaluator) evalSlice(args []ast.Expression, ctx *Context) any {
	arr := e.eval(args[0], ctx)
	arr = unwrapFunctionResult(arr, 0)
	startVal := e.eval(args[1], ctx)
	startVal = unwrapFunctionResult(startVal, 0)
	start := e.toInt64(startVal)
	endVal := e.eval(args[2], ctx)
	endVal = unwrapFunctionResult(endVal, 0)
	end := e.toInt64(endVal)

	var newArr any
	switch arr := arr.(type) {
	case []any:
		if end == -1 {
			end = int64(len(arr))
		}
		if start < 0 || end > int64(len(arr)) || start > end {
			debugPrintf("[DEBUG] at %d runtime.index_out_of_bounds, arr:%v, start:%d end:%d\n", args[0].Pos().Line(), arr, start, end)
			e.setError(&Error{descriptor: errDescIndexOutOfBounds, Err: "runtime.index_out_of_bounds"})
			return &Return{Values: []any{}}
		}
		newArr = arr[start:end]
	case []uint8:
		if end == -1 {
			end = int64(len(arr))
		}
		if start < 0 || end > int64(len(arr)) || start > end {
			debugPrintf("[DEBUG] at %d runtime.index_out_of_bounds, arr:%v, start:%d end:%d\n", args[0].Pos().Line(), arr, start, end)
			e.setError(&Error{descriptor: errDescIndexOutOfBounds, Err: "runtime.index_out_of_bounds"})
			return &Return{Values: []any{}}
		}
		newArr = arr[start:end]
	case string:
		if end == -1 {
			end = int64(len(arr))
		}
		if start < 0 || end > int64(len(arr)) || start > end {
			debugPrintf("[DEBUG] at %d runtime.index_out_of_bounds, arr:%v, start:%d end:%d\n", args[0].Pos().Line(), arr, start, end)
			e.setError(&Error{descriptor: errDescIndexOutOfBounds, Err: "runtime.index_out_of_bounds"})
			return &Return{Values: []any{}}
		}
		newArr = arr[start:end]
	default:
		panic("this is a compiler error. please report")
	}
	return &Return{Values: []any{newArr}}
}

func (e *Evaluator) evalFunction(fn *Function, args []ast.Expression, ctx *Context) any {
	newCtx := NewContext(fn.ctx)

	// evaluate arguments and set values in fresh context
	for i, arg := range args {
		fnArgName := fn.arguments[i].Name.Value
		argValue := e.eval(arg, ctx)
		argValue = unwrapFunctionResult(argValue, 0)

		if shouldWrapTypeInAny(fn.arguments[i].Type) {
			argValue = e.evalToAny(argValue)
		}

		newCtx.Set(fnArgName, argValue)
	}
	res := e.eval(fn.body, newCtx)
	if _, ok := res.(*Return); ok {
		return res
	}
	return &Return{Values: []any{res}}
}

func (e *Evaluator) addError(n ast.Node, err error) {
	pos := n.Pos()
	fmt.Printf("[ERROR] Eval failed at %d:%d - %s\n", pos.Line(), pos.Column(), err)
}

// ------- //
// Helpers //
// ------- //

func generateTypeDescriptor(typeName string) uint32 {
	hash := fnv.New32a()
	hash.Write([]byte(typeName))
	return hash.Sum32()
}

// normaliseTypeDescriptorName creates a consistent type descriptor name.
// For primitives: returns just the type name (e.g., "u32")
// For user-defined types: returns "library/path.type_name" (e.g., "dash/intern.my_type")
// Pointer prefixes are stripped.
func normaliseTypeDescriptorName(t types.Type, libPath string, e *Evaluator) string {
	t = types.StripPointerType(t)

	if imported, ok := t.(*types.ImportedNamed); ok {
		// use full library path from evaluator contexts
		typeName := imported.Typ.Ident()
		if libCtx, ok := e.ctxs[imported.Lib]; ok {
			return libCtx.libPath + "." + typeName
		}
		panic("this is a compiler error. please report")
	}

	typeName := t.Ident()

	if types.IsPrimitiveType(t) {
		return typeName
	}

	return libPath + "." + typeName
}

// convertGoTypeToDash converts Go type names to Dash type names
func convertGoTypeToDash(goTypeName string) string {
	switch goTypeName {
	case "uint32":
		return "u32"
	case "uint64":
		return "u64"
	case "uint8":
		return "u8"
	case "uint16":
		return "u16"
	case "int64":
		return "i64"
	case "int32":
		return "i32"
	case "int16":
		return "i16"
	case "int8":
		return "i8"
	case "float64":
		return "f64"
	case "float32":
		return "f32"
	case "bool":
		return "bool"
	case "string":
		return "string"
	default:
		return goTypeName
	}
}

func unwrapFunctionResult(res any, idx int) any {
	ret, ok := res.(*Return)
	if !ok {
		return res
	}
	if len(ret.Values) == 0 {
		return nil
	}

	if idx < 0 || idx >= len(ret.Values) {
		panic("this is a compiler error. please report")
	}

	return ret.Values[idx]
}

func unwrapAny(val any) any {
	a, ok := val.(*Any)
	if !ok {
		return val
	}
	return a.value
}

func unwrapPointer(val any) any {
	ptr, ok := val.(*Pointer)
	if !ok {
		return val
	}
	return ptr.value
}

// computeLocationHash generates a hash for a variable location
func computeLocationHash(ctx *Context, varName string) uint64 {
	h := fnv.New64a()

	// we include context pointer to distinguish between variables
	// defined in different scopes within program
	key := fmt.Sprintf("%p%s", ctx, varName)
	h.Write([]byte(key))

	return h.Sum64()
}

func castTo[T any](v any) (T, error) {
	val, ok := v.(T)
	if !ok {
		var zero T
		debugPrintf("[DEBUG castTo] Failed: v=%v (type %T) -> target type %T\n", v, v, zero)
		return val, fmt.Errorf("unable to cast %v to type", v)
	}
	return val, nil
}

// shouldWrapTypeInAny returns true if a type requires Any{} wrapping at runtime
func shouldWrapTypeInAny(t types.Type) bool {
	if t == nil {
		return false
	}

	switch typ := t.(type) {
	case *types.Optional:
		return shouldWrapTypeInAny(typ.T)
	case *types.Pointer:
		return shouldWrapTypeInAny(typ.T)
	case *types.Array:
		return shouldWrapTypeInAny(typ.T)
	case *types.Any:
		return true
	case *types.Generic:
		if len(typ.Constraints) == 0 {
			return true
		}
		for _, constraint := range typ.Constraints {
			if _, isAny := constraint.(*types.Any); isAny {
				return true
			}
		}
	}

	return false
}

// evalToAny converts a value to Any{} with type descriptor if not already Any
func (e *Evaluator) evalToAny(v any) *Any {
	if anyVal, ok := v.(*Any); ok {
		return anyVal
	}

	if _, ok := v.(map[string]any); ok {
		panic("structs are not supported in evalToAny")
	}

	goTypeName := reflect.TypeOf(v).String()
	dashTypeName := convertGoTypeToDash(goTypeName)
	descriptor := generateTypeDescriptor(dashTypeName)

	return &Any{
		descriptor: descriptor,
		value:      v,
	}
}

func (e *Evaluator) setError(err *Error) {
	e.state = ERROR
	e.stateVar = err
}

// converts an Error to a string representation
func (e *Error) String() string {
	var b strings.Builder
	b.WriteString(e.Err)

	if len(e.Args) > 0 {
		b.WriteString("{")
		first := true
		for k, v := range e.Args {
			if !first {
				b.WriteString(", ")
			}
			b.WriteString(k)
			b.WriteString(": ")
			b.WriteString(valueToString(v))
			first = false
		}
		b.WriteString("}")
	}

	return b.String()
}

// getFieldPath extracts the field path from a potentially nested DotExpression.
// For example: s.a.b -> returns ["a", "b"]
func getFieldPath(expr *ast.DotExpression) []string {
	var path []string
	current := expr
	for {
		if ident, ok := current.Right.(*ast.Identifier); ok {
			path = append([]string{ident.Value}, path...)
		} else if intLit, ok := current.Right.(*ast.IntegerLiteral); ok {
			path = append([]string{fmt.Sprintf("%d", intLit.Value)}, path...)
		}
		if next, ok := current.Left.(*ast.DotExpression); ok {
			current = next
		} else {
			break
		}
	}
	return path
}

// updateStructField creates a copy of the struct with the field at the given path updated.
// It handles nested structs by recursively copying and updating.
func updateStructField(value any, path []string, newValue any) map[string]any {
	value = unwrapPointer(value)
	strct := value.(map[string]any)
	result := make(map[string]any)
	maps.Copy(result, strct)

	if len(path) == 1 {
		// Base case: set the field directly
		result[path[0]] = newValue
	} else {
		// Recursive case: copy the nested struct and update it
		result[path[0]] = updateStructField(result[path[0]], path[1:], newValue)
	}

	return result
}

func valueToString(v any) string {
	switch v := v.(type) {
	case string:
		return v
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(v)
	case *Error:
		return v.String()
	case []any:
		var b strings.Builder
		b.WriteString("[")
		for i, elem := range v {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(valueToString(elem))
		}
		b.WriteString("]")
		return b.String()
	case map[string]any:
		var b strings.Builder
		b.WriteString("{")
		first := true
		for k, v := range v {
			if !first {
				b.WriteString(", ")
			}
			b.WriteString(k)
			b.WriteString(": ")
			b.WriteString(valueToString(v))
			first = false
		}
		b.WriteString("}")
		return b.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}

func debugPrintf(format string, args ...any) {
	if isDebug {
		fmt.Printf(format, args...)
	}
}
func assertType[T any](t any) {
	if _, ok := t.(T); !ok {
		var zero T
		rsn := fmt.Sprintf("expected expression to be of type: %T, but got %T", zero, t)
		panic(rsn)
	}
}
