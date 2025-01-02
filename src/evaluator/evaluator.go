package evaluator

import (
	"fmt"
	"strconv"
	"strings"

	"dash-lang.io/src/ast"
	"dash-lang.io/src/token"
	"dash-lang.io/src/types"
)

type keyword uint8

const (
	BREAK keyword = iota
	NEXT
)

type Optional struct {
	isValid bool
	value   any
}

type Function struct {
	stk       *Stack
	arguments []*ast.ParameterStatement
	body      *ast.BlockStatement
}

func Run(n ast.Node, stk *Stack) any {
	switch n := n.(type) {
	case *ast.Library:
	case *ast.Evaluator:
		var last any
		for _, n := range n.Enums {
			initialiseEnumStatement(n, stk)
		}
		for _, n := range n.Nodes {
			last = Run(n, stk)
		}
		return last
	case *ast.FunctionExpression:
		if n.Name.Value == "main" {
			evalMainFunction(n, stk)
		} else {
			fn := &Function{
				arguments: n.Arguments,
				body:      n.Body,
				stk:       stk,
			}
			stk.Set(n.Name.Value, fn)
			return fn
		}
	case *ast.FunctionCallExpression:
		// check if it's a built-in function
		if n.TokenLiteral() == "len" {
			return evalLen(n.Arguments, stk)
		} else if n.TokenLiteral() == "println" {
			return evalPrintln(n.Arguments, stk)
		} else if n.TokenLiteral() == "make" {
			return evalMake(n.Arguments, stk)
		}
		return evalFunctionCall(n, stk)
	case *ast.AssignmentStatement:
		evalAssignmentStatement(n, stk)
	case *ast.BlockStatement:
		return evalBlockStatement(n, stk)
	case *ast.IfElseExpression:
		return evalIfElseExpression(n, stk)
	case *ast.ForStatement:
		return evalForStatement(n, stk)
	case *ast.MatchExpressionStatement:
		return evalMatchExpressionStatement(n, stk)
	case *ast.KeywordStatement:
		return evalKeywordStatement(n)
	case *ast.ReturnStatement:
		return evalReturnStatement(n, stk)
	case *ast.UseExpression:
		return evalUseExpression(n, stk)
	case *ast.DotExpression:
		return evalDotExpression(n, stk)
	case *ast.IndexExpression:
		return evalIndexExpression(n, stk)
	case *ast.PrefixExpression:
		return evalPrefixExpression(n, stk)
	case *ast.InfixExpression:
		return evalInfixExpression(n, stk)
	case *ast.PostfixExpression:
		evalPostfixExpression(n, stk)
	case *ast.Identifier:
		val, ok := stk.Get(n.Value)
		if !ok {
			fmt.Println(n, "not found")
			panic("this is a compiler bug. please report")
		}
		return val

	case *ast.StructLiteral:
		return evalStructLiteral(n, stk)
	case *ast.ArrayLiteral:
		vals := make([]any, len(n.Values))
		for i, val := range n.Values {
			vals[i] = Run(val, stk)
		}
		return vals
	case *ast.StringLiteral:
		return n.Value
	case *ast.IntegerLiteral:
		return n.Value
	case *ast.BooleanLiteral:
		return n.Value
	case *ast.FloatLiteral:
		return n.Value
	case *ast.NullLiteral:
		return Optional{isValid: false}
	case nil:
		panic("eval failed as node was nil")
	default:
		addError(n, fmt.Errorf("unknown node type %T", n))
	}
	return nil
}

func initialiseEnumStatement(n *ast.EnumStatement, stk *Stack) {
	fields := make(map[string]any)
	for i, field := range n.Fields {
		fields[field.Value] = int64(i)
	}
	stk.Set(n.Name.Value, fields)
}

func evalMainFunction(n *ast.FunctionExpression, stk *Stack) {
	Run(n.Body, stk)
}

// returns list of function call results
func evalFunctionCall(n *ast.FunctionCallExpression, stk *Stack) any {
	var fn *Function
	if n.Func == nil {
		fn_, _ := stk.Get(n.TokenLiteral())
		fn = fn_.(*Function)
	} else {
		fn = Run(n.Func, stk).(*Function)
	}

	newStk := NewStack(fn.stk)

	// evaluate arguments and set values in fresh symbol table
	for i, arg := range n.Arguments {
		fnArgName := fn.arguments[i].Name.Value
		newStk.Set(fnArgName, Run(arg, stk))
	}
	res := Run(fn.body, newStk)

	return res
}

// always returns nil
func evalAssignmentStatement(n *ast.AssignmentStatement, stk *Stack) {
	// TODO: iterate over if function reached we need to handle that
	// TODO: data can be assigned to identifiers, struct fields, array indices, slices
	// g. if in use expression
	for i, val := range n.Values {
		switch val := val.(type) {
		case *ast.FunctionCallExpression:
			res := Run(val, stk).([]any)
			for j := range val.ReturnTypes {
				decl := n.Declerations[i+j].Assignee
				stk.Set(decl.String(), res[i+j])
			}
		case *ast.CopyUpdateExpression:
			decl := n.Declerations[i].Assignee
			res := evalCopyUpdateExpression(decl.String(), val, stk)
			stk.Set(decl.String(), res)
		default:
			decl := n.Declerations[i].Assignee
			switch exp := decl.(type) {
			case *ast.IndexExpression:
				evalAssignmentToArrayIndex(exp, val, stk)
			case *ast.SliceExpression:
				evalAssignmentToArraySlice(exp, val, stk)
			case *ast.DotExpression:
				evalAssignmentToStructField(exp, val, stk)
			default:
				res := Run(val, stk)
				stk.Set(decl.String(), res)
			}
		}
	}
}

func evalAssignmentToArrayIndex(exp *ast.IndexExpression, val ast.Expression, stk *Stack) {
	res := Run(val, stk)
	// we know that LHS has to be an identifier as
	// assigning to index of array is only possible
	// within an use expression
	ident := exp.Left.(*ast.Identifier)

	arr, ok := stk.Get(ident.Value)
	if !ok {
		panic("this is a compiler error. please report")
	}
	idx := Run(exp.Indices[0], stk).(int64)
	arr.([]any)[idx] = res
}

func evalAssignmentToArraySlice(exp *ast.SliceExpression, val ast.Expression, stk *Stack) {
	res := Run(val, stk)
	// we know that LHS has to be an identifier as
	// assigning to index of array is only possible
	// within an use expression
	ident := exp.Left.(*ast.Identifier)

	arr, ok := stk.Get(ident.Value)
	if !ok {
		panic("this is a compiler error. please report")
	}
	rng := Run(exp.Indices[0], stk).([]any)
	start := rng[0].(int64)
	end := rng[1].(int64)
	copy(arr.([]any)[start:end], res.([]any))
}

func evalAssignmentToStructField(exp *ast.DotExpression, val ast.Expression, stk *Stack) {
	res := Run(val, stk)

	strct := Run(exp.Left, stk).(map[string]any)

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

func evalIfElseExpression(n *ast.IfElseExpression, stk *Stack) any {
	stk.Scope()
	defer stk.Unscope()

	for _, c := range n.Conditionals {
		cond := Run(c.Condition, stk)
		if cond.(bool) {
			return Run(c.Block, stk)
		}
	}
	return nil
}

func evalForStatement(n *ast.ForStatement, stk *Stack) any {
	// classic for loop
	if n.Assignment != nil {
		evalAssignmentStatement(n.Assignment, stk)

		for {
			cond := Run(n.Condition, stk)
			if !cond.(bool) {
				break
			}
			exp := Run(n.Block, stk)
			if exp == BREAK {
				break
			} else if exp == NEXT {
				Run(n.Change, stk)
				continue
			}
			Run(n.Change, stk)
		}
		return nil
	}

	// conditional loop
	if n.Condition != nil && n.Change == nil {
		for {
			cond := e.Run(n.Condition)
			if !cond.(bool) {
				break
			}

			e.Run(n.Block)
		}
		return nil
	}

	// infinite loop
	if n.Condition == nil {
		for {
			exp := e.Run(n.Block)
			if exp == BREAK {
				break
			}
		}
		return nil
	}

	return nil
}

func evalMatchExpressionStatement(n *ast.MatchExpressionStatement, stk *Stack) any {
	scrutinee := Run(n.Scrutinee, stk)

	if _, ok := n.Scrutinee.(*ast.FunctionCallExpression); ok {
		scrutinee = unwrapFunctionResult(scrutinee, 0)
	}

	stk.Scope()
	defer stk.Unscope()

	// TODO: handle multiple predicates in one case
	for _, c := range n.Cases {
		predValue := Run(c.Predicate, stk)
		if predValue == scrutinee {
			return evalMatchCase(c, stk)
		}
	}

	if n.Default != nil {
		return evalMatchCase(n.Default, stk)
	}

	panic("this is a compiler error. please report")

}
func evalMatchCase(c *ast.MatchCase, stk *Stack) any {
	var last any
	for _, stmt := range c.Body {
		last = Run(stmt, stk)
	}
	return last
}

// returns last expression in block if any otherwise nil
func evalBlockStatement(n *ast.BlockStatement, stk *Stack) any {
	var exp any
	for _, stmt := range n.Statements {
		exp = Run(stmt, stk)
	}
	return exp
}

func evalKeywordStatement(n *ast.KeywordStatement) any {
	switch n.Token.Type {
	case token.BREAK:
		return BREAK
	case token.NEXT:
		return NEXT
	default:
		addError(n, fmt.Errorf("unknown keyword %s", n.Token.Literal))
	}
	return nil
}

func evalReturnStatement(n *ast.ReturnStatement, stk *Stack) any {
	vals := make([]any, len(n.Values))
	for i := range n.Values {
		res := Run(n.Values[i], stk)
		switch val := n.Values[i].(type) {
		case *ast.FunctionCallExpression:
			if len(val.ReturnTypes) == 1 {
				vals[i] = unwrapFunctionResult(res, 0)
			} else {
				for j := range val.ReturnTypes {
					vals[i+j] = unwrapFunctionResult(res, j)
				}
			}
		default:
			vals[i] = res
		}
	}
	return vals
}

func evalUseExpression(n *ast.UseExpression, stk *Stack) any {
	arr := Run(n.Ident, stk)

	stk.Scope()
	defer stk.Unscope()

	stk.Set(n.Ident.Value, arr)

	Run(n.Block, stk)
	arr, _ = stk.Get(n.Ident.Value)
	return arr
}

func evalCopyUpdateExpression(newVar string, n *ast.CopyUpdateExpression, stk *Stack) any {
	orig := Run(n.Ident, stk)

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
		addError(n, fmt.Errorf("copy-update only works on arrays and structs"))
		return nil
	}

	stk.Set(newVar, cpy)

	Run(n.Block, stk)

	return cpy
}

func evalDotExpression(n *ast.DotExpression, stk *Stack) any {
	obj := Run(n.Left, stk)

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

func evalIndexExpression(n *ast.IndexExpression, stk *Stack) any {
	arr := Run(n.Left, stk)

	// evaluate all indices
	indices := make([]int, len(n.Indices))
	for i, idx := range n.Indices {
		val := Run(idx, stk)
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

func evalPrefixExpression(n *ast.PrefixExpression, stk *Stack) any {
	val := Run(n.Right, stk)
	var err error
	switch n.Token.Type {
	case token.MINUS:
		val, err = evalPrefixMinus(val)
	case token.NOT:
		val, err = evalPrefixNot(val)
	// case token.AMPERSAND:
	// case token.ASTERISK:
	case token.OPTIONAL:
		if _, ok := n.Right.(*ast.FunctionCallExpression); ok {
			val = unwrapFunctionResult(val, 0)
		}
		val = evalPrefixOptional(val)
	}

	if err != nil {
		addError(n, err)
	}
	return val
}

func evalPrefixMinus(v any) (any, error) {
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

func evalPrefixNot(v any) (any, error) {
	switch v := v.(type) {
	case bool:
		return !v, nil
	default:
		return nil, fmt.Errorf("cannot apply unary minus to type %T", v)
	}
}

func evalPrefixOptional(v any) any {
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

func evalInfixExpression(n *ast.InfixExpression, stk *Stack) any {
	l := Run(n.Left, stk)
	if _, ok := n.Left.(*ast.FunctionCallExpression); ok {
		l = unwrapFunctionResult(l, 0)
	}
	r := Run(n.Right, stk)
	if _, ok := n.Right.(*ast.FunctionCallExpression); ok {
		r = unwrapFunctionResult(r, 0)
	}

	var val any
	var err error
	switch n.Token.Type {
	// Arithmetic
	case token.PLUS:
		val, err = evalInfixAdd(l, r)
	case token.MINUS:
		val, err = evalInfixSub(l, r)
	case token.ASTERISK:
		val, err = evalInfixMul(l, r)
	case token.SLASH:
		val, err = evalInfixDiv(l, r)
	case token.MOD:
		val, err = evalInfixMod(l, r)
	// Logical
	case token.AND:
		val, err = evalInfixAnd(l, r)
	case token.OR:
		val, err = evalInfixOr(l, r)
	// Relational
	case token.GT:
		val, err = evalInfixGreater(l, r)
	case token.GTE:
		val, err = evalInfixGreaterEqual(l, r)
	case token.LT:
		val, err = evalInfixLess(l, r)
	case token.LTE:
		val, err = evalInfixLessEqual(l, r)
	// Equality
	case token.EQ:
		val = evalInfixEqual(l, r)
	case token.NEQ:
		val = evalInfixNotEqual(l, r)
	// Optional
	case token.NULL_COALESCE:
		val = evalInfixNullCoalesce(l, r)
	// Special
	case token.COLON:
		val = []any{l, r}
	}

	if err != nil {
		addError(n, err)
	}

	return val
}

func evalInfixAdd(l, r any) (res any, err error) {
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

func evalInfixSub(l, r any) (res any, err error) {
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

func evalInfixMul(l, r any) (res any, err error) {
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

func evalInfixDiv(l, r any) (res any, err error) {
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

func evalInfixMod(l, r any) (res any, err error) {
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

func evalInfixAnd(l, r any) (res any, err error) {
	switch l := l.(type) {
	case bool:
		var r_ bool
		r_, err = castTo[bool](r)
		res = l && r_
	}
	return
}

func evalInfixOr(l, r any) (res any, err error) {
	switch l := l.(type) {
	case bool:
		var r_ bool
		r_, err = castTo[bool](r)
		res = l || r_
	}
	return
}

func evalInfixLess(l, r any) (res any, err error) {
	switch l := l.(type) {
	case int64:
		var r_ int64
		r_, err = castTo[int64](r)
		res = l < r_
	case float64:
		var r_ float64
		r_, err = castTo[float64](r)
		res = l < r_
	}
	return
}

func evalInfixGreater(l, r any) (res any, err error) {
	switch l := l.(type) {
	case int64:
		var r_ int64
		r_, err = castTo[int64](r)
		res = l > r_
	case float64:
		var r_ float64
		r_, err = castTo[float64](r)
		res = l > r_
	}
	return
}

func evalInfixLessEqual(l, r any) (res any, err error) {
	switch l := l.(type) {
	case int64:
		var r_ int64
		r_, err = castTo[int64](r)
		res = l <= r_
	case float64:
		var r_ float64
		r_, err = castTo[float64](r)
		res = l <= r_
	}
	return
}

func evalInfixGreaterEqual(l, r any) (res any, err error) {
	switch l := l.(type) {
	case int64:
		var r_ int64
		r_, err = castTo[int64](r)
		res = l >= r_
	case float64:
		var r_ float64
		r_, err = castTo[float64](r)
		res = l >= r_
	}
	return
}

func evalInfixEqual(l, r any) any {
	if l_, ok := l.(Optional); ok {
		l = l_.value
	}
	if r_, ok := r.(Optional); ok {
		r = r_.value
	}
	return l == r
}

func evalInfixNotEqual(l, r any) any {
	if l_, ok := l.(Optional); ok {
		l = l_.value
	}
	if r_, ok := r.(Optional); ok {
		r = r_.value
	}
	return l != r
}

func evalInfixNullCoalesce(l, r any) any {
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

// ------------------ //
// Postfix expression //
// ------------------ //

func evalPostfixExpression(n *ast.PostfixExpression, stk *Stack) {
	left := Run(n.Left, stk)
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

func evalStructLiteral(n *ast.StructLiteral, stk *Stack) map[string]any {
	strct := make(map[string]any)

	for _, field := range n.Fields {
		var name string
		// we distinguish between named fields and unnamed
		if field.Name != nil {
			name = field.Name.Value
		} else {
			name = fmt.Sprintf("%d", field.Index)
		}
		strct[name] = Run(field.Value, stk)
	}
	return strct
}

// ------------------ //
// Built-in functions //
// ------------------ //

func evalLen(args []ast.Expression, stk *Stack) any {
	if len(args) != 1 {
		panic("this is a compiler error. please report")
	}

	var val any
	switch args[0].(type) {
	case *ast.FunctionCallExpression:
		val = Run(args[0], stk).([]any)[0]
	default:
		val = Run(args[0], stk)
	}

	switch v := val.(type) {
	case []any:
		return int64(len(v))
	default:
		panic("this is a compiler error. please report")
	}
}

func evalPrintln(args []ast.Expression, stk *Stack) any {
	values := make([]string, 0, len(args))

	// evaluate each argument and convert to string
	for _, arg := range args {
		switch args[0].(type) {
		case *ast.FunctionCallExpression:
			rets := Run(arg, stk).([]any)
			for _, ret := range rets {
				values = append(values, valueToString(ret))
			}
		default:
			val := Run(arg, stk)
			values = append(values, valueToString(val))
		}
	}

	n, err := fmt.Println(strings.Join(values, " "))
	if err != nil {
		panic("error when printing to console: " + strings.Join(values, " "))
	}
	return n
}

func evalMake(args []ast.Expression, stk *Stack) any {

	typ, _ := args[0].(*ast.TypeLiteral)
	arrayType, _ := typ.T.(*types.Array)

	size := Run(args[1], stk)
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

	return []any{arr}
}

// ------- //
// Helpers //
// ------- //

func addError(n ast.Node, err error) {
	pos := n.Pos()
	fmt.Printf("[ERROR] Eval failed at %d:%d - %s\n", pos.Line(), pos.Column(), err)
}

func unwrapFunctionResult(res any, idx int) any {
	returnValues, ok := res.([]any)
	if !ok {
		return res
	}
	if len(returnValues) == 0 {
		return nil
	}
	if 0 < idx || idx > len(returnValues)-1 {
		panic("this is a compiler error. please report")
	}

	return returnValues[idx]
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

	// BUG: we can not rely on order here so tests may occasionally fail
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
