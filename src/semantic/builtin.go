package semantic

import (
	"dash-lang.io/src/types"
)

// TODO: move these consrant types to 'types'

func isBuiltinFunction(lit string) bool {
	switch lit {
	case "len", "cap", "size", "make", "validate", "println":
		return true
	}
	return false
}

// Constant argument and return types for built-in fns
var (
	intRetT     = []types.TypeSpec{&types.ConstI64}
	genericArgT = []types.TypeSpec{&types.Generic{Name: "T"}}
	lenArgT     = []types.TypeSpec{
		&types.Generic{Name: "T", Constraints: []types.TypeSpec{
			&types.Struct{Ts: []types.StructField{{Name: "len", T: &types.ConstI64}}},
			&types.Array{T: &types.Generic{Name: "T"}},
			&types.String{},
			&types.Memory{T: &types.Array{T: &types.Generic{Name: "T"}}},
		}},
	}
	capArgT = []types.TypeSpec{
		&types.Generic{Name: "T", Constraints: []types.TypeSpec{
			&types.Struct{Ts: []types.StructField{{Name: "cap", T: &types.ConstI64}}},
			&types.Array{T: &types.Generic{Name: "T"}},
			&types.Memory{T: &types.Array{T: &types.Generic{Name: "T"}}},
		}},
	}
	printlnArgT = []types.TypeSpec{
		&types.Generic{Name: "T", Constraints: []types.TypeSpec{
			&types.ConstString,
			&types.Generic{Name: "T"},
		}},
	}
	validateArgT = []types.TypeSpec{&types.Dirty{T: &types.Generic{Name: "T"}}}
	validateRetT = []types.TypeSpec{&types.ConstBool}
)

func getBuiltinSignature(lit string, argsTypes []types.TypeSpec) (args []types.TypeSpec, rets []types.TypeSpec) {
	switch lit {
	case "len":
		return lenArgT, intRetT
	case "cap":
		return capArgT, intRetT
	case "size":
		return genericArgT, intRetT
	case "make":
		var argT []types.TypeSpec

		// if initial value ommited, dont add third argument type
		if len(argsTypes) < 3 {
			argT = []types.TypeSpec{&types.Type{T: argsTypes[0]}, &types.ConstI64}
		} else {
			argT = []types.TypeSpec{&types.Type{T: argsTypes[0]}, &types.ConstI64, argsTypes[2]}
		}
		retT := []types.TypeSpec{&types.Memory{T: argsTypes[0]}}
		return argT, retT
	case "validate":
		return validateArgT, validateRetT
	case "println":
		return printlnArgT, nil
	}
	return nil, nil
}
