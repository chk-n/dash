package evaluator

import (
	"fmt"
	"hash/fnv"
	"strconv"
	"strings"

	"dash-lang.io/src/ast"
	"dash-lang.io/src/token"
	"dash-lang.io/src/types"
)

// TODO: adjust evaluator to check symbol tables when dot expression used
// on non struct or enum variable
type keyword uint8

const (
	BREAK keyword = iota
	NEXT
)

type Error struct {
	Err string
	// optional args
	Args []any
}

type Return struct {
	Values []any
}

type Optional struct {
	isValid bool
	value   any
}

type Union struct {
	// hash of library name + type name
	descriptor uint32
	value      any
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
}

func New(libs map[string]*ast.Library) *Evaluator {
	return &Evaluator{
		libs: libs,
		ctxs: make(map[string]*Context),
	}
}

func (e *Evaluator) InitialiseLib(n *ast.Library, ctx *Context) {

	// For all imports of current library ensure
	// context is aware of imports and initialise
	// those imported libraries if not already done
	for _, n := range n.Nodes {
		imp, ok := n.(*ast.UseStatement)
		if !ok {
			continue
		}
		libName := imp.Name.TokenLiteral()
		lib, ok := e.libs[libName]
		if !ok {
			panic("")
		}
		ctx.imps.Set(libName, lib)

		if _, ok := e.ctxs[libName]; !ok {
			ctxLib := NewContext(nil)
			e.Eval(lib, ctxLib)
			e.ctxs[libName] = ctxLib
		}
	}
	// TODO: handle generic structs

	// initialise types
	for _, n := range n.Nodes {
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
			ctx.vars.Set(n.Name.String(), n.Name.TokenLiteral())
		}
	}
}

func (e *Evaluator) Eval(n ast.Node, ctx *Context) any {
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
			case *ast.UnionStatement, *ast.TypeDefinitionStatement,
				*ast.StructStatement, *ast.EnumStatement,
				*ast.ErrorStatement:
				// skip as already initialised
			default:
				last = e.Eval(n, ctx)
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
		if n.TokenLiteral() == "len" {
			return e.evalLen(n.Arguments, ctx)
		} else if n.TokenLiteral() == "println" {
			return e.evalPrintln(n.Arguments, ctx)
		} else if n.TokenLiteral() == "make" {
			return e.evalMake(n.Arguments, ctx)
		} else if n.TokenLiteral() == "assert" {
			return e.evalAssert(n.Arguments, ctx)
		}
		// If ok is true then it is a custom type cast
		if _, ok := ctx.typs.Get(n.TokenLiteral()); ok {
			res := e.Eval(n.Arguments[0], ctx)
			if _, ok := n.T.(*types.Union); ok {
				typeName := n.Arguments[0].Type().Ident()
				descriptor := generateTypeDescriptor(typeName)

				res = &Union{
					descriptor: descriptor,
					value:      res,
				}
			}
			return &Return{Values: []any{res}}
		}
		return e.evalFunctionCall(n, ctx)
	case *ast.TypeCastExpression:
		return e.evalTypeCastExpression(n, ctx)
	case *ast.AssignmentStatement:
		e.evalAssignmentStatement(n, ctx)
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
	case *ast.RaiseStatement:
		return e.evalRaiseStatement(n, ctx)
	case *ast.KeywordStatement:
		return e.evalKeywordStatement(n)
	case *ast.ReturnStatement:
		return e.evalReturnStatement(n, ctx)
	case *ast.UseExpression:
		return e.evalUseExpression(n, ctx)
	case *ast.DotExpression:
		return e.evalDotExpression(n, ctx)
	case *ast.SliceExpression:
		return e.evalSliceExpression(n, ctx)
	case *ast.IndexExpression:
		return e.evalIndexExpression(n, ctx)
	case *ast.PrefixExpression:
		return e.evalPrefixExpression(n, ctx)
	case *ast.InfixExpression:
		return e.evalInfixExpression(n, ctx)
	case *ast.PostfixExpression:
		e.evalPostfixExpression(n, ctx)
	case *ast.Identifier:
		val, ok := ctx.Get(n.Value)
		if !ok {
			panic("this is a compiler bug. please report: " + n.Value)
		}
		return val

	case *ast.TypeLiteral:
		switch t := n.T.(type) {
		case *types.Int:
			return int64(0)
		case *types.Float:
			return float64(0)
		case *types.Struct:
			return "strct." + t.Name
		}
	case *ast.StructLiteral:
		return e.evalStructLiteral(n, ctx)
	case *ast.ArrayLiteral:
		vals := make([]any, len(n.Values))
		for i, val := range n.Values {
			vals[i] = e.Eval(val, ctx)
		}
		return vals
	case *ast.StringLiteral:
		return n.TokenLiteral()
	case *ast.CharacterLiteral:
		return rune(n.Value)
	case *ast.IntegerLiteral:
		return n.Value
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

func (e *Evaluator) evalMainFunction(n *ast.FunctionExpression, stk *Context) {
	e.Eval(n.Body, stk)
}

// returns list of function call results
func (e *Evaluator) evalFunctionCall(n *ast.FunctionCallExpression, stk *Context) any {
	_fn, _ := stk.Get(n.TokenLiteral())
	fn := _fn.(*Function)

	newCtx := NewContext(fn.ctx)

	// evaluate arguments and set values in fresh symbol table
	for i, arg := range n.Arguments {
		fnArgName := fn.arguments[i].Name.Value
		newCtx.Set(fnArgName, e.Eval(arg, stk))
	}
	res := e.Eval(fn.body, newCtx)
	if _, ok := res.(*Return); ok {
		return res
	}
	return &Return{Values: []any{res}}
}

// The goal of type casts for now is to only support the minimum number of operations
// to be able to bootstrap the compiler in dash
func (e *Evaluator) evalTypeCastExpression(n *ast.TypeCastExpression, stk *Context) any {
	val := e.Eval(n.Argument, stk)
	switch t := n.Typ.(type) {
	case *types.Int:
		return e.evalIntCast(t, val)
	case *types.Byte:
		return e.evalByteCast(t, val)
	// case *types.Char:
	case *types.String:
		return e.evalStringCast(t, val)
	case *types.Array:
		return e.evalArrayCast(t, val)

	}
	return val
}

// Int casting

func (e *Evaluator) evalIntCast(t *types.Int, v any) any {
	switch t.Signed + t.Width {
	// 8, 9
	case 8:
		// return toUint8(v)
	// 16, 17
	// 32, 33
	case 64:
		// toUint64()
	case 65:
		return e.toInt64(v)
	}
	panic("invalid int cast")
}

func (e *Evaluator) toInt64(v any) any {
	switch v := v.(type) {
	case byte:
		return int64(v)
	case rune:
		return int64(v)
		// TODO i8 to i32
		// TODO u8 to u64
	}
	panic("invalid cast to i64")
}

// Byte casting

func (e *Evaluator) evalByteCast(t *types.Byte, v any) any {
	switch v := v.(type) {
	case int64:
		return byte(v)
	}
	panic("invalid cast to byte")
}

// Char casting

// String casting
func (e *Evaluator) evalStringCast(t *types.String, v any) any {
	switch v := v.(type) {
	case byte:
		return string(v)
	case []any:
		arr := make([]byte, len(v))
		for i, el := range v {
			arr[i] = byte(el.(int32))
		}
		return string(arr)
	}
	panic("invalid cast to string")
}

// Array casting

func (e *Evaluator) evalArrayCast(t *types.Array, v any) any {
	switch t.T.(type) {
	case *types.Byte:
		str := v.(string)
		newArr := make([]any, len(str))
		for i, ch := range str {
			newArr[i] = ch
		}
		return newArr
	}
	panic("invalid array cast")
}

// always returns nil
func (e *Evaluator) evalAssignmentStatement(n *ast.AssignmentStatement, ctx *Context) {
	// TODO: iterate over if function reached we need to handle that
	// TODO: data can be assigned to identifiers, struct fields, array indices, slices
	// g. if in use expression

	for i, val := range n.Values {
		switch val := val.(type) {
		case *ast.TryExpression:
			fn := val.Right.(*ast.FunctionCallExpression)
			res := e.Eval(val, ctx).(*Return)
			for j := range fn.ReturnTypes {
				setOrUpdateForAssignment(n.Declerations[i+j], n.VarNameAt(i+j), res.Values[i+j], ctx)
			}
		case *ast.FunctionCallExpression:
			res := e.Eval(val, ctx).(*Return)
			for j := range val.ReturnTypes {
				setOrUpdateForAssignment(n.Declerations[i+j], n.VarNameAt(i+j), res.Values[i+j], ctx)
			}
		case *ast.CopyUpdateExpression:
			res := e.evalCopyUpdateExpression(n.VarNameAt(i), val, ctx)
			setOrUpdateForAssignment(n.Declerations[i], n.VarNameAt(i), res, ctx)
		default:
			switch decl := n.Declerations[i].(type) {
			case *ast.Identifier:
				res := e.Eval(val, ctx)
				ctx.SetAll(n.VarNameAt(i), res)
			case *ast.DeclarationStatement:
				res := e.Eval(val, ctx)
				ctx.Set(n.VarNameAt(i), res)
			case *ast.IndexExpression:
				e.evalAssignmentToArrayIndex(decl, val, ctx)
			case *ast.SliceExpression:
				e.evalAssignmentToArraySlice(decl, val, ctx)
			case *ast.DotExpression:
				e.evalAssignmentToStructField(decl, val, ctx)
			}
		}
	}
}

// Performs a context set but propagates the set up context chain if its a reassignment
func setOrUpdateForAssignment(assgn ast.Node, name string, res any, ctx *Context) {
	if _, ok := assgn.(*ast.Identifier); ok {
		ctx.SetAll(name, res)
	} else {
		ctx.Set(name, res)
	}
}

func (e *Evaluator) evalAssignmentToArrayIndex(exp *ast.IndexExpression, val ast.Expression, stk *Context) {
	res := e.Eval(val, stk)
	// we know that LHS has to be an identifier as
	// assigning to index of array is only possible
	// within an use expression
	ident := exp.Left.(*ast.Identifier)

	arr, ok := stk.Get(ident.Value)
	if !ok {
		panic("this is a compiler error. please report")
	}
	idx := e.Eval(exp.Indices[0], stk).(int64)
	arr.([]any)[idx] = res
}

func (e *Evaluator) evalAssignmentToArraySlice(exp *ast.SliceExpression, val ast.Expression, stk *Context) {
	res := e.Eval(val, stk)
	// we know that LHS has to be an identifier as
	// assigning to index of array is only possible
	// within an use expression
	ident := exp.Left.(*ast.Identifier)

	arr, ok := stk.Get(ident.Value)
	if !ok {
		panic("this is a compiler error. please report")
	}
	rng := e.Eval(exp.Indices[0], stk).([]any)
	start := rng[0].(int64)
	end := rng[1].(int64)
	copy(arr.([]any)[start:end], res.([]any))
}

func (e *Evaluator) evalAssignmentToStructField(exp *ast.DotExpression, val ast.Expression, stk *Context) {
	res := e.Eval(val, stk)

	strct := e.Eval(exp.Left, stk).(map[string]any)

	// we know exp.Right must be an identifier or integer literal for struct fields
	var field string
	switch right := exp.Right.(type) {
	case *ast.Identifier:
		field = right.Value
	case *ast.IntegerLiteral:
		field = fmt.Sprintf("%d", right.Value)
	default:
		panic("this is a compiler error. please report")
	}

	strct[field] = res

}

func (e *Evaluator) evalIfElseExpression(n *ast.IfElseExpression, stk *Context) any {
	stk.Scope()
	defer stk.Unscope()

	for _, c := range n.Conditionals {
		// This means else block matched
		if c.Condition == nil {
			return e.Eval(c.Block, stk)
		}
		val := e.Eval(c.Condition, stk)
		var cond bool
		if res, ok := val.(*Return); ok {
			cond = res.Values[0].(bool)
		} else if res, ok := val.(bool); ok {
			cond = res
		} else {
			panic("this is a compiler error. please report")
		}
		if cond {
			return e.Eval(c.Block, stk)
		}
	}
	return nil
}

func (e *Evaluator) evalForStatement(n *ast.ForStatement, stk *Context) any {
	// classic for loop
	if n.Assignment != nil {
		e.evalAssignmentStatement(n.Assignment, stk)

		for {
			cond := e.Eval(n.Condition, stk)
			if !cond.(bool) {
				break
			}
			exp := e.Eval(n.Block, stk)
			if _, ok := exp.(*Return); ok {
				return exp
			} else if exp == BREAK {
				break
			} else if exp == NEXT {
				e.Eval(n.Change, stk)
				continue
			}
			e.Eval(n.Change, stk)
		}
		return nil
	}

	// conditional loop
	if n.Condition != nil && n.Change == nil {
		for {
			cond := e.Eval(n.Condition, stk)
			if !cond.(bool) {
				break
			}

			exp := e.Eval(n.Block, stk)
			if _, ok := exp.(*Return); ok {
				return exp
			} else if exp == BREAK {
				break
			} else if exp == NEXT {
				continue
			}
		}
		return nil
	}

	// infinite loop
	if n.Condition == nil {
		for {
			exp := e.Eval(n.Block, stk)
			if _, ok := exp.(*Return); ok {
				return exp
			} else if exp == BREAK {
				break
			} else if exp == NEXT {
				continue
			}
		}
		return nil
	}

	return nil
}

func (e *Evaluator) evalMatchExpressionStatement(n *ast.MatchExpressionStatement, stk *Context) any {
	scrutinee := e.Eval(n.Scrutinee, stk)

	if _, ok := n.Scrutinee.(*ast.FunctionCallExpression); ok {
		scrutinee = unwrapFunctionResult(scrutinee, 0)
	}

	stk.Scope()
	defer stk.Unscope()

	// TODO: handle multiple predicates in one case
	if _, ok := n.Scrutinee.Type().(*types.Union); ok {
		unionVal, ok := scrutinee.(*Union)
		if !ok {
			panic("matching against non-union type")
		}

		for _, c := range n.Cases {
			// Get type name from predicate
			typeName := c.Predicate.String()
			// Hash it for comparison
			caseDescriptor := generateTypeDescriptor(typeName)

			// Match descriptors
			if caseDescriptor == unionVal.descriptor {
				return e.evalMatchCase(c, stk)
			}
		}
	} else {
		for _, c := range n.Cases {
			predValue := e.Eval(c.Predicate, stk)
			if predValue == scrutinee {
				return e.evalMatchCase(c, stk)
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
		last = e.Eval(stmt, stk)
	}
	return last
}

// returns last expression in block if any otherwise nil
func (e *Evaluator) evalBlockStatement(n *ast.BlockStatement, stk *Context) any {
	var exp any
	for _, stmt := range n.Statements {
		exp = e.Eval(stmt, stk)
		if _, ok := exp.(*Return); ok {
			return exp
		} else if exp == BREAK || exp == NEXT {
			return exp
		}
	}
	return exp
}

func (e *Evaluator) evalKeywordStatement(n *ast.KeywordStatement) any {
	switch n.Token.Type {
	case token.BREAK:
		return BREAK
	case token.NEXT:
		return NEXT
	default:
		e.addError(n, fmt.Errorf("unknown keyword %s", n.Token.Literal))
	}
	return nil
}

func (e *Evaluator) evalReturnStatement(n *ast.ReturnStatement, stk *Context) any {
	var vals []any
	for i := range n.Values {
		res := e.Eval(n.Values[i], stk)
		switch val := n.Values[i].(type) {
		case *ast.FunctionCallExpression:
			if len(val.ReturnTypes) == 1 {
				vals = append(vals, unwrapFunctionResult(res, 0))
			} else {
				for j := range val.ReturnTypes {
					vals = append(vals, unwrapFunctionResult(res, j))
				}
			}
		default:
			vals = append(vals, res)
		}
	}
	return &Return{Values: vals}
}

func (e *Evaluator) evalUseExpression(n *ast.UseExpression, stk *Context) any {
	arr := e.Eval(n.Ident, stk)

	stk.Scope()
	defer stk.Unscope()

	stk.Set(n.Ident.Value, arr)

	e.Eval(n.Block, stk)
	arr, _ = stk.Get(n.Ident.Value)
	return arr
}

func (e *Evaluator) evalCopyUpdateExpression(newVar string, n *ast.CopyUpdateExpression, stk *Context) any {
	orig := e.Eval(n.Ident, stk)

	// make a deep copy
	var cpy any
	switch v := orig.(type) {
	case []any:
		newArr := make([]any, len(v))
		copy(newArr, v)
		cpy = newArr

	case map[string]any:
		newMap := make(map[string]any, len(v))
		for k, val := range v {
			newMap[k] = val
		}
		cpy = newMap

	default:
		e.addError(n, fmt.Errorf("copy-update only works on arrays and structs"))
		return nil
	}

	stk.Set(newVar, cpy)

	e.Eval(n.Block, stk)

	return cpy
}

func (e *Evaluator) evalDotExpression(n *ast.DotExpression, stk *Context) any {
	obj := e.Eval(n.Left, stk)

	switch right := n.Right.(type) {
	// handles named structs and enums
	case *ast.Identifier:
		if fields, ok := obj.(map[string]any); ok {
			if val, exists := fields[right.Value]; exists {
				return val
			}
		}
	// handles unnamed structs
	case *ast.IntegerLiteral:
		if fields, ok := obj.(map[string]any); ok {
			key := fmt.Sprintf("%d", right.Value)
			if val, exists := fields[key]; exists {
				return val
			}
		}
	}

	return nil
}

func (e *Evaluator) evalSliceExpression(n *ast.SliceExpression, ctx *Context) any {

	arr, ok := e.Eval(n.Left, ctx).([]any)
	if !ok {
		panic("this is a compiler error. plese report")
	}

	rng := n.Indices[0].(*ast.InfixExpression)

	start := e.Eval(rng.Left, ctx).(int64)
	end := e.Eval(rng.Right, ctx).(int64)

	return arr[start:end]

}

func (e *Evaluator) evalIndexExpression(n *ast.IndexExpression, stk *Context) any {
	arr := e.Eval(n.Left, stk)

	// evaluate all indices
	indices := make([]int, len(n.Indices))
	for i, idx := range n.Indices {
		val := e.Eval(idx, stk)
		idx, err := castTo[int64](val)
		if err != nil {
			panic("this is a compiler error. plese report")
		}
		indices[i] = int(idx)
	}

	// perform indexing and handle multiple dimensions
	curr := arr
	for _, idx := range indices {
		slice, ok := curr.([]any)
		if !ok {
			panic("this is a compiler error. plese report")
		}

		if idx < 0 || idx >= len(slice) {
			// TODO: add dash error handling
			panic("index out of bounds")
		}

		curr = slice[idx]
	}
	return curr
}

// ----------------- //
// Prefix expression //
// ----------------- //

func (e *Evaluator) evalPrefixExpression(n *ast.PrefixExpression, stk *Context) any {
	val := e.Eval(n.Right, stk)
	var err error
	switch n.Token.Type {
	case token.MINUS:
		val, err = e.evalPrefixMinus(val)
	case token.BANG:
		val, err = e.evalPrefixNot(val)
	case token.OPTIONAL:
		if _, ok := n.Right.(*ast.FunctionCallExpression); ok {
			val = unwrapFunctionResult(val, 0)
		}
		val = e.evalPrefixOptional(val)
	case token.AMPERSAND, token.ASTERISK:
		return val
	}

	if err != nil {
		e.addError(n, err)
	}
	return val
}

func (e *Evaluator) evalPrefixMinus(v any) (any, error) {
	switch v := v.(type) {
	case int:
		return -v, nil
	case int8:
		return -v, nil
	case int16:
		return -v, nil
	case int32:
		return -v, nil
	case int64:
		return -v, nil
	case uint:
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

func (e *Evaluator) evalPrefixOptional(v any) any {
	if opt, ok := v.(Optional); ok {
		if !opt.isValid {
			// NOTE: here Dash error handling would kick in
			panic("attempted to force unwrap null value")
		}
		return opt.value
	}
	if v == nil {
		panic("this is a compiler error. please report")
	}
	return v
}

// ---------------- //
// Infix expression //
// ---------------- //

func (e *Evaluator) evalInfixExpression(n *ast.InfixExpression, ctx *Context) any {
	l := e.Eval(n.Left, ctx)
	if _, ok := n.Left.(*ast.FunctionCallExpression); ok {
		l = unwrapFunctionResult(l, 0)
	}
	r := e.Eval(n.Right, ctx)
	if _, ok := n.Right.(*ast.FunctionCallExpression); ok {
		r = unwrapFunctionResult(r, 0)
	}

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
		val = e.evalInfixNotEqual(l, r)
	// Optional
	case token.NULL_COALESCE:
		val = e.evalInfixNullCoalesce(l, r)
	// Special
	case token.COLON:
		val = []any{l, r}
	case token.PIPE:
		val = e.evalInfixPipe(l, r, ctx)
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
	default:
		err = fmt.Errorf("unsupported type for modulo %T", l)
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
	case float64:
		var r_ float64
		r_, err = castTo[float64](r)
		res = l < r_
	case byte:
		var r_ byte
		r_, err = castTo[byte](r)
		res = l < r_
	case rune:
		var r_ rune
		r_, err = castTo[rune](r)
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
	case float64:
		var r_ float64
		r_, err = castTo[float64](r)
		res = l > r_
	case byte:
		var r_ byte
		r_, err = castTo[byte](r)
		res = l > r_
	case rune:
		var r_ rune
		r_, err = castTo[rune](r)
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
	case float64:
		var r_ float64
		r_, err = castTo[float64](r)
		res = l <= r_
	case byte:
		var r_ byte
		r_, err = castTo[byte](r)
		res = l <= r_
	case rune:
		var r_ rune
		r_, err = castTo[rune](r)
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
	case float64:
		var r_ float64
		r_, err = castTo[float64](r)
		res = l >= r_
	case byte:
		var r_ byte
		r_, err = castTo[byte](r)
		res = l >= r_
	case rune:
		var r_ rune
		r_, err = castTo[rune](r)
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
	return l == r
}

func (e *Evaluator) evalInfixNotEqual(l, r any) any {
	if l_, ok := l.(Optional); ok {
		l = l_.value
	}
	if r_, ok := r.(Optional); ok {
		r = r_.value
	}
	return l != r
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

// The left expression result becomes the first argument to the right function
func (e *Evaluator) evalInfixPipe(l, r any, ctx *Context) any {
	// If left is a function result, unwrap it to get the actual value
	if ret, ok := l.(*Return); ok {
		l = unwrapFunctionResult(ret, 0)
	}

	switch rightExpr := r.(type) {
	// case: value |> functionName
	case *ast.Identifier:
		fnVal, ok := ctx.Get(rightExpr.Value)
		if !ok {
			return &Error{Err: fmt.Sprintf("undefined function: %s", rightExpr.Value)}
		}

		fn, ok := fnVal.(*Function)
		if !ok {
			return &Error{Err: fmt.Sprintf("%s is not a function", rightExpr.Value)}
		}

		newCtx := NewContext(fn.ctx)

		// set arguments of 'r' function
		if ret, ok := l.(*Return); ok {
			for i, val := range ret.Values {
				paramName := fn.arguments[i].Name.Value
				newCtx.Set(paramName, val)
			}
		} else {
			paramName := fn.arguments[0].Name.Value
			newCtx.Set(paramName, l)
		}

		return e.Eval(fn.body, newCtx)
	}
	return nil
}

// ------------------ //
// Postfix expression //
// ------------------ //

func (e *Evaluator) evalPostfixExpression(n *ast.PostfixExpression, stk *Context) {
	left := e.Eval(n.Left, stk)
	val, ok := left.(int64)
	if !ok {
		panic("this is a compiler error. please report")
	}

	switch n.Token.Type {
	case token.INCR:
		stk.Set(n.Left.String(), val+1)
	case token.DECR:
		stk.Set(n.Left.String(), val-1)
	default:
		panic("this is a compiler error. please report")
	}
}

// -------- //
// Literals //
// -------- //

func (e *Evaluator) evalStructLiteral(n *ast.StructLiteral, stk *Context) map[string]any {
	strct := make(map[string]any)

	for _, field := range n.Fields {
		var name string
		// we distinguish between named fields and unnamed
		if field.Name != nil {
			name = field.Name.Value
		} else {
			name = fmt.Sprintf("%d", field.Index)
		}
		strct[name] = e.Eval(field.Value, stk)
	}
	return strct
}

// -------------- //
// Error handling //
// -------------- //

func (e *Evaluator) evalTryExpression(n *ast.TryExpression, ctx *Context) any {
	res := e.Eval(n.Right, ctx)
	if res == nil {
		return nil
	}

	// propagate error
	ret := res.(*Return)
	if len(ret.Values) > 0 {
		if err, ok := ret.Values[0].(*Error); ok {
			newErr := &Error{Err: err.Err}
			return &Return{Values: []any{newErr}}
		}
	}

	return ret
}

func (e *Evaluator) evalRaiseStatement(n *ast.RaiseStatement, ctx *Context) any {
	// Evaluate the error expression to get the actual error value/name
	errVal := e.Eval(n.Error, ctx)

	// If it's a simple error identifier, use its string representation directly
	var errName string
	switch err := errVal.(type) {
	case *ast.Identifier:
		errName = err.Value
	default:
		errName = n.Error.String()
	}

	// Return an Error that represents the raised error
	// This Error will be caught by try/catch blocks up the stack
	return &Return{Values: []any{&Error{Err: errName}}}
}

// ------------------ //
// Built-in functions //
// ------------------ //

func (e *Evaluator) evalLen(args []ast.Expression, stk *Context) any {
	if len(args) != 1 {
		panic("this is a compiler error. please report")
	}

	var val any
	switch args[0].(type) {
	case *ast.FunctionCallExpression:
		val = unwrapFunctionResult(e.Eval(args[0], stk), 0)
	default:
		val = e.Eval(args[0], stk)
	}

	switch v := val.(type) {
	case []any:
		return int64(len(v))
	case []uint8:
		return int64(len(v))
	case string:
		return int64(len(v))
	default:
		panic("this is a compiler error. please report")
	}
}

func (e *Evaluator) evalPrintln(args []ast.Expression, stk *Context) any {
	values := make([]string, 0, len(args))

	// evaluate each argument and convert to string
	for _, arg := range args {
		a := e.Eval(arg, stk)
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

	typ, _ := args[0].(*ast.TypeLiteral)
	arrayType, _ := typ.T.(*types.Array)

	size := e.Eval(args[1], stk)
	sizeVal, _ := size.(int64)
	if sizeVal < 0 {
		panic("this is a compiler error. please report")
	}

	// create array with default values
	arr := make([]any, sizeVal)
	var defaultVal any
	switch arrayType.T.(type) {
	case *types.Int:
		defaultVal = int64(0)
	case *types.Float:
		defaultVal = float64(0)
	case *types.String:
		defaultVal = ""
	case *types.Bool:
		defaultVal = false
	case *types.Byte:
		defaultVal = uint8(0)
	case *types.Char:
		defaultVal = uint32(0)
	case *types.Struct, *types.Enum, *types.Union:
		defaultVal = map[string]any{}
	default:
		defaultVal = nil
	}

	for i := range arr {
		arr[i] = defaultVal
	}

	return &Return{Values: []any{arr}}
}

func (e *Evaluator) evalAssert(args []ast.Expression, ctx *Context) any {

	if e.Eval(args[0], ctx).(bool) {
		return nil
	}
	msg := e.Eval(args[1], ctx).(string)
	return &Return{Values: []any{&Error{Err: msg}}}
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

func castTo[T any](v any) (T, error) {
	val, ok := v.(T)
	if !ok {
		return val, fmt.Errorf("unable to cast %v to type", v)
	}
	return val, nil
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
