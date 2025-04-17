package semantic

import (
	"dash-lang.io/src/ast"
	"dash-lang.io/src/token"
	"dash-lang.io/src/types"
)

// TODO: move these constant types to 'types'

func isBuiltinFunction(lit string) bool {
	switch lit {
	case "len", "cap", "size", "make", "validate", "println", "assert":
		return true
	}
	return false
}

// Constant argument and return types for built-in fns
var (
	intRetT     = []types.TypeSpec{&types.ConstI64}
	genericArgT = []types.TypeSpec{&types.Generic{Name: "T"}}
)

func getBuiltinSignature(lit string, argsTypes []types.TypeSpec) *ast.FunctionExpression {
	var args []types.TypeSpec
	var rets []types.TypeSpec
	var errorProne bool

	switch lit {
	case "len":
		args = []types.TypeSpec{
			&types.Generic{Name: "T", Constraints: []types.TypeSpec{
				&types.Struct{Ts: []types.StructField{{Name: "len", T: &types.ConstI64}}},
				&types.Array{T: &types.Generic{Name: "T"}},
				&types.String{},
				&types.Memory{T: &types.Array{T: &types.Generic{Name: "T"}}},
			}},
		}

		rets = intRetT
	case "cap":
		args = []types.TypeSpec{
			&types.Generic{Name: "T", Constraints: []types.TypeSpec{
				&types.Struct{Ts: []types.StructField{{Name: "cap", T: &types.ConstI64}}},
				&types.Array{T: &types.Generic{Name: "T"}},
				&types.Memory{T: &types.Array{T: &types.Generic{Name: "T"}}},
			}},
		}

		rets = intRetT
	case "size":
		args = genericArgT
		rets = intRetT
	case "make":
		// if initial value omitted, don't add third argument type
		if len(argsTypes) < 3 {
			args = []types.TypeSpec{&types.Type{T: argsTypes[0]}, &types.ConstI64}
		} else {
			args = []types.TypeSpec{&types.Type{T: argsTypes[0]}, &types.ConstI64, argsTypes[2]}
		}
		rets = []types.TypeSpec{&types.Memory{T: argsTypes[0]}}
	case "validate":
		args = []types.TypeSpec{&types.Dirty{T: &types.Generic{Name: "T"}}}
		rets = []types.TypeSpec{&types.ConstBool}
	case "println":
		args = []types.TypeSpec{
			&types.Generic{Name: "T", Constraints: []types.TypeSpec{
				&types.ConstString,
				&types.Generic{Name: "T"},
			}},
		}
	case "assert":
		// builtin function that with type fn(bool, string)!
		// it is error prone
		args = []types.TypeSpec{&types.ConstBool, &types.ConstString}
		errorProne = true
	default:
		return nil
	}

	// Create the function expression
	fnExpr := &ast.FunctionExpression{
		Token: token.Token{
			Type:    token.FUNCTION,
			Literal: "fn",
		},
		Name: &ast.Identifier{
			Token: token.Token{
				Type:    token.IDENT,
				Literal: lit,
			},
			Value: lit,
		},
		ErrorProne: errorProne,
		Public:     true,
	}

	params := make([]*ast.ParameterStatement, len(args))
	for i, argType := range args {
		paramName := "arg" + string(rune('0'+i))
		params[i] = &ast.ParameterStatement{
			Name: &ast.Identifier{
				Token: token.Token{
					Type:    token.IDENT,
					Literal: paramName,
				},
				Value: paramName,
			},
			Type: argType,
		}
	}
	fnExpr.Arguments = params

	if len(rets) > 0 {
		retVals := make([]*ast.TypeLiteral, len(rets))
		for i, ret := range rets {
			retVals[i] = &ast.TypeLiteral{
				Token: token.Token{
					Type:    token.TYPE,
					Literal: "type",
				},
				T: ret,
			}
		}
		fnExpr.ReturnValues = retVals
	}

	fnExpr.T = &types.Function{
		Arg:          args,
		Ret:          rets,
		IsErrorProne: errorProne,
	}

	return fnExpr
}
