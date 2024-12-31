package evaluator

import (
	"fmt"
	"strconv"
	"strings"

	"dash-lang.io/src/ast"
	"dash-lang.io/src/internal"
	"dash-lang.io/src/token"
	"dash-lang.io/src/types"
)

type keyword uint8

const (
	BREAK keyword = iota
	NEXT
)

type Evaluator struct {
	vars *internal.StackedSymTab[any]
	fns  *internal.StackedSymTab[*ast.FunctionExpression]
}

func New() *Evaluator {
	return &Evaluator{
		vars: internal.NewStackedSymbolTable[any](),
	}
}

func (e *Evaluator) Run(n ast.Node) any {
	switch n := n.(type) {
	case *ast.Library:
	case *ast.Evaluator:
		var last any
		for _, n := range n.Enums {
			e.initialiseEnumStatement(n)
		}
		for _, n := range n.Nodes {
			last = e.Run(n)
		}
		return last
	case *ast.FunctionExpression:
		if n.Name.Value == "main" {
			e.evalMainFunction(n)
		}
	case *ast.FunctionCallExpression:
		return e.evalFunctionCall(n)
	case *ast.AssignmentStatement:
		e.evalAssignmentStatement(n)
	case *ast.BlockStatement:
		return e.evalBlockStatement(n)
	case *ast.IfElseExpression:
		return e.evalIfElseExpression(n)
	case *ast.ForStatement:
		return e.evalForStatement(n)
	case *ast.KeywordStatement:
		return e.evalKeywordStatement(n)
	case *ast.ReturnStatement:
		return e.evalReturnStatement(n)
	case *ast.DotExpression:
		return e.evalDotExpression(n)
	case *ast.IndexExpression:
		return e.evalIndexExpression(n)
	case *ast.PrefixExpression:
		return e.evalPrefixExpression(n)
	case *ast.InfixExpression:
		return e.evalInfixExpression(n)
	case *ast.PostfixExpression:
		e.evalPostfixExpression(n)
	case *ast.Identifier:
		val, ok := e.vars.Get(n.Value)
		if !ok {
			// this is a serious issue
			panic("this is a compiler bug. please report")
		}
		return val

	case *ast.StructLiteral:
		return e.evalStructLiteral(n)
	case *ast.ArrayLiteral:
		vals := make([]any, len(n.Values))
		for i, val := range n.Values {
			vals[i] = e.Run(val)
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
	case nil:
		panic("eval failed as node was nil")
	default:
		e.addError(n, fmt.Errorf("unknown node type %T", n))
	}
	return nil
}

func (e *Evaluator) initialiseEnumStatement(n *ast.EnumStatement) {
	fields := make(map[string]any)
	for i, field := range n.Fields {
		fields[field.Value] = int64(i)
	}
	e.vars.Set(n.Name.Value, fields)
}

func (e *Evaluator) evalMainFunction(n *ast.FunctionExpression) {
	e.Run(n.Body)
}

// returns list of function call results
func (e *Evaluator) evalFunctionCall(n *ast.FunctionCallExpression) any {
	// First check if it's a built-in function
	if n.TokenLiteral() == "len" {
		return e.evalLen(n.Arguments)
	} else if n.TokenLiteral() == "println" {
		return e.evalPrintln(n.Arguments)
	} else if n.TokenLiteral() == "make" {
		return e.evalMake(n.Arguments)
	}
	// create new scope during function execution
	e.vars.Scope()
	defer e.vars.Unscope()
	// evaluate arguments and set values in fresh symbol table
	for i, arg := range n.Arguments {
		fnArgName := n.Func.Arguments[i].Name.Value
		e.vars.Set(fnArgName, e.Run(arg))
	}

	return e.Run(n.Func.Body)
}

// -------------------- //
// Assignment Statement //
// -------------------- //

// always returns nil
func (e *Evaluator) evalAssignmentStatement(n *ast.AssignmentStatement) {
	// TODO: iterate over if function reached we need to handle that
	// TODO: data can be assigned to identifiers, struct fields, array indices, slices
	// e.g. if in use expression
	for i, val := range n.Values {
		switch val := val.(type) {
		case *ast.FunctionCallExpression:
			res := e.Run(val).([]any)
			for j := range val.ReturnTypes {
				decl := n.Declerations[i+j].Assignee
				e.vars.Set(decl.String(), res[i+j])
			}
		default:
			res := e.Run(val)
			decl := n.Declerations[i].Assignee
			e.vars.Set(decl.String(), res)
		}
	}
}

// ----------------- //
// IfElse Expression //
// ----------------- //

func (e *Evaluator) evalIfElseExpression(n *ast.IfElseExpression) any {
	for _, c := range n.Conditionals {
		cond := e.Run(c.Condition)
		if cond.(bool) {
			return e.Run(c.Block)
		}
	}
	return nil
}

func (e *Evaluator) evalForStatement(n *ast.ForStatement) any {
	// classic for loop
	if n.Assignment != nil {
		e.evalAssignmentStatement(n.Assignment)

		for {
			cond := e.Run(n.Condition)
			if !cond.(bool) {
				break
			}
			// execute block
			e.Run(n.Block)

			// execute change
			e.Run(n.Change)
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

	// range loop
	// if n.Condition == nil && n.Assignment == nil && n.Change == nil {
	// 	// Range loop is handled by ForRangeStatement
	// 	// This is an infinite loop
	// 	for {
	// 		e.Run(n.Block)
	// 	}
	// }

	return nil
}

// returns last expression in block if any otherwise nil
func (e *Evaluator) evalBlockStatement(n *ast.BlockStatement) any {
	var exp any
	for _, stmt := range n.Statements {
		exp = e.Run(stmt)
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

func (e *Evaluator) evalReturnStatement(n *ast.ReturnStatement) any {
	vals := make([]any, len(n.Values))
	for i := range n.Values {
		vals[i] = e.Run(n.Values[i])
	}
	return vals
}

func (e *Evaluator) evalDotExpression(n *ast.DotExpression) any {
	obj := e.Run(n.Left)

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

func (e *Evaluator) evalIndexExpression(n *ast.IndexExpression) any {
	arr := e.Run(n.Left)

	// Evaluate all indices
	indices := make([]int, len(n.Indices))
	for i, idx := range n.Indices {
		val := e.Run(idx)
		idx, err := castTo[int64](val)
		if err != nil {
			e.addError(n, fmt.Errorf("this is a compiler error plese report"))
			return nil
		}
		indices[i] = int(idx)
	}

	// Handle multi-dimensional indexing
	curr := arr
	for _, idx := range indices {
		slice, ok := curr.([]any)
		if !ok {
			e.addError(n, fmt.Errorf("this is a compiler error plese report"))
			return nil
		}

		if idx < 0 || idx >= len(slice) {
			e.addError(n, fmt.Errorf("index out of bounds"))
			return nil
		}

		curr = slice[idx]
	}
	return curr
}

// ----------------- //
// Prefix expression //
// ----------------- //

func (e *Evaluator) evalPrefixExpression(n *ast.PrefixExpression) any {
	val := e.Run(n.Right)
	var err error
	switch n.Token.Type {
	case token.MINUS:
		val, err = e.evalPrefixMinus(val)
	case token.NOT:
		val, err = e.evalPrefixNot(val)
		// case token.AMPERSAND:
		// case token.ASTERISK:
		// case token.OPTIONAL:
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
	default:
		return nil, fmt.Errorf("cannot apply unary minus to type %T", v)
	}
}

// ---------------- //
// Infix expression //
// ---------------- //

func (e *Evaluator) evalInfixExpression(n *ast.InfixExpression) any {
	l := e.Run(n.Left)
	r := e.Run(n.Right)

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
		val, err = e.evalInfixEqual(l, r)
	case token.NEQ:
		val, err = e.evalInfixNotEqual(l, r)
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
	}
	return
}

func (e *Evaluator) evalInfixEqual(l, r any) (res any, err error) {
	switch l := l.(type) {
	case int64:
		var r_ int64
		r_, err = castTo[int64](r)
		res = l == r_
	case float64:
		var r_ float64
		r_, err = castTo[float64](r)
		res = l == r_
	case string:
		var r_ string
		r_, err = castTo[string](r)
		res = l == r_
	}
	return
}

func (e *Evaluator) evalInfixNotEqual(l, r any) (res any, err error) {
	switch l := l.(type) {
	case int64:
		var r_ int64
		r_, err = castTo[int64](r)
		res = l != r_
	case float64:
		var r_ float64
		r_, err = castTo[float64](r)
		res = l != r_
	case string:
		var r_ string
		r_, err = castTo[string](r)
		res = l != r_
	}
	return
}

// ------------------ //
// Postfix expression //
// ------------------ //

func (e *Evaluator) evalPostfixExpression(n *ast.PostfixExpression) {
	left := e.Run(n.Left)
	val, ok := left.(int64)
	if !ok {
		e.addError(n, fmt.Errorf("postfix operation only works on integers"))
		return
	}

	switch n.Token.Type {
	case token.INCR:
		e.vars.Set(n.Left.String(), val+1)
	case token.DECR:
		e.vars.Set(n.Left.String(), val-1)
	default:
		e.addError(n, fmt.Errorf("unknown postfix operator %s", n.Token.Literal))
		return
	}
}

// -------- //
// Literals //
// -------- //

func (e *Evaluator) evalStructLiteral(n *ast.StructLiteral) map[string]any {
	strct := make(map[string]any)

	for _, field := range n.Fields {
		var name string
		// we distinguish between named fields and unnamed
		if field.Name != nil {
			name = field.Name.Value
		} else {
			name = fmt.Sprintf("%d", field.Index)
		}
		strct[name] = e.Run(field.Value)
	}
	return strct
}

// ------------------ //
// Built-in functions //
// ------------------ //

func (e *Evaluator) evalLen(args []ast.Expression) any {
	if len(args) != 1 {
		panic("len() requires exactly one argument")
	}

	var val any
	switch args[0].(type) {
	case *ast.FunctionCallExpression:
		val = e.Run(args[0]).([]any)[0]
	default:
		val = e.Run(args[0])
	}

	switch v := val.(type) {
	case []any:
		return int64(len(v))
	default:
		panic(fmt.Sprintf("len() not supported for type %T", v))
	}
}

func (e *Evaluator) evalPrintln(args []ast.Expression) any {
	values := make([]string, 0, len(args))

	// Evaluate each argument to a string
	for _, arg := range args {
		switch args[0].(type) {
		case *ast.FunctionCallExpression:
			rets := e.Run(arg).([]any)
			for _, ret := range rets {
				values = append(values, valueToString(ret))
			}
		default:
			val := e.Run(arg)
			values = append(values, valueToString(val))
		}
	}

	// Print all values with spaces between them and newline at end
	fmt.Println(strings.Join(values, " "))
	return nil
}

func (e *Evaluator) evalMake(args []ast.Expression) any {

	typ, _ := args[0].(*ast.TypeLiteral)
	arrayType, _ := typ.T.(*types.Array)

	size := e.Run(args[1])
	sizeVal, _ := size.(int64)
	if sizeVal < 0 {
		panic("make() size cannot be negative")
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
	default:
		defaultVal = nil
	}

	for i := range arr {
		arr[i] = defaultVal
	}

	return arr
}

// Performs
func (e *Evaluator) IArith(n ast.Node) (int, bool) {
	switch n := n.(type) {
	case *ast.PrefixExpression:
		r, ok := e.IArith(n.Right)
		if !ok {
			return 0, ok
		}
		switch n.Operator {
		case "-":
			return -r, true
		}
	case *ast.InfixExpression:
		l, ok1 := e.IArith(n.Left)
		if !ok1 {
			return 0, ok1
		}
		r, ok2 := e.IArith(n.Right)
		if !ok2 {
			return 0, ok2
		}
		switch n.Operator {
		case "+":
			return l + r, true
		case "-":
			return l - r, true
		case "*":
			return l * r, true
		case "/":
			return l / r, true
		}

	case *ast.IntegerLiteral:
		return int(n.Value), true

	}

	return 0, false
}

func (e *Evaluator) addError(n ast.Node, err error) {
	pos := n.Pos()
	fmt.Printf("[ERROR] Eval failed at %d:%d - %s\n", pos.Line(), pos.Column(), err)
}

// ------- //
// Helpers //
// ------- //

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
