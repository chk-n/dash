package transformer

import (
	"fmt"

	"dash-lang.io/src/ast"
	"dash-lang.io/src/token"
)

// This file transforms certain statements such as `defer`
//
// Defer transformation
// - modifies return statements to ensure defer is last call made

type Transformer struct {
}

func New() *Transformer {
	return &Transformer{}
}

func (t *Transformer) Tranform(n ast.Node) {
	switch n := n.(type) {
	case *ast.Library:
		for _, fn := range n.Functions {
			t.Tranform(fn)
		}
	case *ast.FunctionExpression:
		if n.Body == nil {
			return
		}

		var deferNodes []ast.Node

		idx := len(n.Body.Statements) - 1
		for i := idx; i >= 0; i-- {
			stmt := n.Body.Statements[i]
			if d, ok := stmt.(*ast.DeferStatement); ok {
				// if block append inner statements only
				if blk, ok := d.Node.(*ast.BlockStatement); ok {
					for _, b := range blk.Statements {
						deferNodes = append(deferNodes, b)
					}
				} else {
					deferNodes = append(deferNodes, d.Node)
				}
				n.Body.Statements = append(n.Body.Statements[:i], n.Body.Statements[i+1:]...)
				idx--
			}
		}

		if len(deferNodes) == 0 {
			return
		}

		// case 1: only defer in fn block
		{
			if idx < 0 {
				for _, exp := range deferNodes {
					n.Body.Statements = append(n.Body.Statements, exp)
				}
				return
			}

		}

		last := n.Body.Statements[idx]
		ret, ok := last.(*ast.ReturnStatement)
		// case 2: no return in block
		{
			if !ok {
				for _, exp := range deferNodes {
					n.Body.Statements = append(n.Body.Statements, exp)
				}
				return
			}
		}

		// case 3: void return
		{
			if len(ret.Values) == 0 {
				// remove return
				n.Body.Statements = n.Body.Statements[:idx]
				for _, exp := range deferNodes {
					n.Body.Statements = append(n.Body.Statements, exp)
				}
				n.Body.Statements = append(n.Body.Statements, last)
				return
			}
		}

		// case 4: return with elements
		{
			n.Body.Statements = n.Body.Statements[:idx]
			var idents []ast.Expression
			// convert return elements to assignments
			for i, el := range ret.Values {
				ident := &ast.Identifier{Value: fmt.Sprintf("$%d", i)}
				idents = append(idents, ident)
				decl := []*ast.DeclarationStatement{{Token: token.NewFromLiteral(token.LET, "let"), Assignee: ident}}
				val := []ast.Expression{el}
				assgn := &ast.AssignmentStatement{Declerations: decl, Values: val}
				n.Body.Statements = append(n.Body.Statements, assgn)
			}
			for _, exp := range deferNodes {
				n.Body.Statements = append(n.Body.Statements, exp)
			}
			// create new return that references assignment variable(s)
			ret = &ast.ReturnStatement{Token: ret.Token, Values: idents}
			n.Body.Statements = append(n.Body.Statements, ret)
			return
		}
	}
}
