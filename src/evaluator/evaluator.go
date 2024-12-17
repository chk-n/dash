package evaluator

import "dash-lang.io/src/ast"

type Evaluator struct {
}

func New() *Evaluator {
	return &Evaluator{}
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
