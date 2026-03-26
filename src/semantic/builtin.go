package semantic

import (
	"dash-lang.io/src/ast"
	"dash-lang.io/src/token"
	"dash-lang.io/src/types"
)

// TODO: move these constant types to 'types'

func isBuiltinFunction(lit string) bool {
	switch lit {
	case
		"put", "get", //"remove",
		"append", "slice",
		"len", "cap",
		"size", "make", "validate", "println", "assert":
		return true
	}
	return false
}

// Constant argument and return types for built-in fns
var (
	intRetT     = []types.Type{&types.ConstI64}
	genericArgT = []types.Type{&types.Generic{Name: "T"}}
)

func getBuiltinSignature(lit string, argsTypes []types.Type) *ast.FunctionExpression {
	var args []types.Type
	var rets []types.Type
	var errorProne bool

	switch lit {
	case "append":
		// fn append[T any](arr []T, el T) []T
		// fn append[T any](arr []T. arr2 []T) []T
		args = []types.Type{argsTypes[0], argsTypes[1]}
		rets = []types.Type{argsTypes[0]}
		errorProne = true
	case "put":
		// fn put[T any](arr []T, idx i64, T)! []T
		// fn put[T any](arr []T, idx i64, []T)! []T
		args = []types.Type{argsTypes[0], argsTypes[1], argsTypes[2]}
		rets = []types.Type{argsTypes[0]}
		errorProne = true
	case "get":
		// fn get[T any](arr []T, idx i64)! T
		var elemType types.Type
		switch argsTypes[0].(type) {
		case *types.String:
			elemType = &types.ConstByte
		default:
			arrType := types.GetUnderlyingTypeArray(argsTypes[0])
			if arrType == nil {
				return nil
			}
			elemType = arrType.T
		}
		args = []types.Type{argsTypes[0], argsTypes[1]}
		rets = []types.Type{elemType}
		errorProne = true
	case "slice":
		// fn slice[T any](arr []T, start i64, end i64)! []T
		// TODO: assert argsTypes[1] and argsTypes[2] is *types.Int
		args = []types.Type{argsTypes[0], argsTypes[1], argsTypes[2]}
		rets = []types.Type{argsTypes[0]}
		errorProne = true
	case "len":
		// fn len[T any](arr []T) i64
		args = []types.Type{
			&types.Generic{Name: "T", Constraints: []types.Type{
				&types.Struct{Ts: []types.StructField{{Name: "len", T: &types.ConstI64}}},
				&types.Array{T: &types.Generic{Name: "T"}},
				&types.String{},
				&types.Memory{T: &types.Array{T: &types.Generic{Name: "T"}}},
			}},
		}

		rets = intRetT
	case "cap":
		// fn cap[T any](arr []T) i64
		args = []types.Type{
			&types.Generic{Name: "T", Constraints: []types.Type{
				&types.Struct{Ts: []types.StructField{{Name: "cap", T: &types.ConstI64}}},
				&types.Array{T: &types.Generic{Name: "T"}},
				&types.Memory{T: &types.Array{T: &types.Generic{Name: "T"}}},
			}},
		}

		rets = intRetT
	case "size":
		// fn size[T any](x T) i64
		args = genericArgT
		rets = intRetT
	case "make":
		// if initial value omitted, don't add third argument type
		if len(argsTypes) < 3 {
			args = []types.Type{argsTypes[0], &types.ConstI64}
		} else {
			args = []types.Type{argsTypes[0], &types.ConstI64, argsTypes[2]}
		}
		rets = []types.Type{&types.Memory{T: argsTypes[0]}}
		errorProne = true
	case "validate":
		args = []types.Type{&types.Dirty{T: &types.Generic{Name: "T"}}}
		rets = []types.Type{&types.ConstBool}
	case "println":
		args = []types.Type{
			&types.Generic{Name: "T", Constraints: []types.Type{
				&types.ConstString,
				&types.ConstMany,
			}},
		}
	case "assert":
		// builtin function that with type fn(bool, string)!
		// it is error prone
		args = []types.Type{&types.ConstBool, &types.ConstString}
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
