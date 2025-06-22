// This file contains core methods regarding semantical anaylsis of Dash.
// Files in this directory contain specialised tests.

package semantic

import (
	"fmt"
	"path"
	"slices"
	"strings"

	"dash-lang.io/src/ast"
	"dash-lang.io/src/internal"
	"dash-lang.io/src/token"
	"dash-lang.io/src/types"
)

type scope uint

const (
	GLOBAL scope = iota
	FOR
	ASSIGNMENT
)

type FnInfo struct {
	Type          *types.Function
	IsAnonymousFn bool // TODO: remove
}

type VarInfo struct {
	Type         types.TypeSpec
	Reassignable bool
}

type TypeInfo struct {
	Type types.TypeSpec
}

type Semantics struct {
	// Path to file being analysed
	path string
	// symbol table for externally referenced types, functions, variables etc.
	importedSt map[string]map[string]types.TypeSpec
	// TODO: replace with varSt
	// symbol table for functions
	fnSt *internal.StackedSymTab[*FnInfo]
	// symbol table for variables
	varSt *internal.StackedSymTab[*VarInfo]
	// tracks errors discovered in a library
	errors []*SemanticalError
	// TODO: remove this and replace with TODO's marked in code
	// tracks current scope e.g. if we are in a function or for loop
	scope *internal.Stack[scope]
	// current function type being analysed
	fnScope *internal.Stack[*types.Function]
	// symbol table for custom types (type defs)
	typeSt *internal.StackedSymTab[types.TypeSpec]
	// TODO: store this info in type e.g. as guarded type
	// tracks whether a given type is guarded so we
	// can mark operations as dirty e.g. for use,
	// copy update and operations with guarded type
	guardedType *internal.Cache[string, struct{}]
}

type SemanticalError struct {
	At  ast.Node
	Err error
}

func New(sourcePath string, importedSt map[string]map[string]types.TypeSpec) *Semantics {
	// convert all keys of importedSt from "../../lib_name" to "lib_name"
	// so they can be accessed in semsis using semsis scope
	normalizedImportedSt := make(map[string]map[string]types.TypeSpec)
	for key, value := range importedSt {
		normalizedKey := path.Base(key)
		normalizedImportedSt[normalizedKey] = value
	}
	return &Semantics{
		path:        sourcePath,
		fnSt:        internal.NewStackedSymbolTable[*FnInfo](),
		varSt:       internal.NewStackedSymbolTable[*VarInfo](),
		scope:       internal.NewStack[scope](),
		fnScope:     internal.NewStack[*types.Function](),
		typeSt:      internal.NewStackedSymbolTable[types.TypeSpec](),
		guardedType: internal.NewCache[string, struct{}](),
		importedSt:  normalizedImportedSt,
	}
}

func (s *Semantics) Analyse(lib *ast.Library) {
	s.scope.Push(GLOBAL)
	defer s.scope.Pop()

	s.analyseTypes(lib.Nodes)
	s.resolveAllTypeReferences(lib.Nodes)

	// Do the circular type dependency analysis across these types
	s.analyseDuplicateIdentifiers(lib.Nodes)
	s.analyseCircularTypeReferences(lib.Nodes)

	s.analyse(lib, "")
}

func (s *Semantics) Errors() []string {
	errs := make([]string, len(s.errors))
	for i, e := range s.errors {
		errs[i] = e.Err.Error()

	}
	return errs
}

func (s *Semantics) ErrorsFmt() []string {
	errs := make([]string, len(s.errors))
	for i, e := range s.errors {
		pos := e.At.Pos()
		errs[i] = fmt.Sprintf("[ERROR] Semsis failed in %s at %d:%d - %s", s.path, pos.Line(), pos.Column(), e.Err)

	}
	return errs
}

// Builds symbol table for function arguments and variables, it performs some basic type inference too
func (s *Semantics) analyse(n ast.Node, name string) {
	switch n := n.(type) {
	case *ast.Library:
		for _, fn := range n.Nodes {
			s.analyse(fn, "")
		}
	case *ast.TypeDefinitionStatement:
		// We need to analyse type guards after type def and functions
		// analysed as guards might call functions or other types which
		// need to be analysed before hand
		if n.Guard != nil {
			// use type def name to set variabe table so that analysis
			// of guard works as guard uses type def name as 'variable'
			// s.varSt.Set(td.Name.TokenLiteral(), &VarInfo{Type: &types.Definition{Name: td.Name.TokenLiteral(), Underlying: td.T}})
			s.analyse(n.Guard, "")
			s.varSt.Clear(n.Name.TokenLiteral())
			s.guardedType.Set(n.Name.Value, struct{}{})
		}
	case *ast.FunctionExpression:
		// scope within function
		s.varSt.Scope()
		s.analyseFunctionExpression(n, name)
		fnType := n.Type()
		if n.ErrorProne {
			fnType := fnType.(*types.Function)
			fnType.IsErrorProne = true
			n.SetType(fnType)
		}
		// another error must have occured
		if fnType == nil {
			return
		}
		s.fnScope.Push(n.Type().(*types.Function))
		defer func() {
			s.fnScope.Pop()
			s.varSt.Unscope()
		}()

		if n.Body == nil {
			// we allow function headers to be defined
			// e.g. for 'extern(c)' functions
		} else if len(n.Body.Statements) == 0 && len(n.ReturnValues) != 0 {
			s.addError(n, errMissingReturn())
			return
		} else {
			s.analyse(n.Body, "")
		}

	case *ast.FunctionCallExpression:
		fnName := n.TokenLiteral()
		// When analysing function calls we
		// distinguish between three cases
		// 1. built in function
		// 2. type cast to type def
		// 3. function call
		for _, arg := range n.Arguments {
			s.analyse(arg, "")
		}

		if isBuiltinFunction(n.Token.Literal) {
			// Special validation for append function
			if n.Token.Literal == "append" {
				s.validateAppendFunction(n)
				return
			}

			builtintFn := getBuiltinSignature(n.Token.Literal, getTypesFromExpressions(n.Arguments))
			argTs := builtintFn.T.(*types.Function).Arg
			retTs := builtintFn.T.(*types.Function).Ret

			s.analyseCallArguments(n, argTs)
			n.ReturnTypes = retTs

			// Here we need to update variable table
			// and set the variable that was just validated
			// to not be dirty anymore. In the future with
			// error handling we an avoid relying on developer
			// to check bool for safety, or prove bool is checked.
			if n.TokenLiteral() == "validate" {
				argName := n.Arguments[0].TokenLiteral()
				varInfo, ok := s.varSt.Get(argName)
				if !ok {
					// this could occur if literal passed, so we
					// cant just panic
				} else {
					if t, ok := varInfo.Type.(*types.Dirty); ok {
						varInfo.Type = t.T
						s.varSt.Set(argName, varInfo)
					}
				}
			}
		} else if to, ok := s.typeSt.Get(fnName); ok {
			// primitive type casts excluding struct type casts
			// are handled by *ast.TypeCastExpression
			isFromLiteral := false
			switch n.Arguments[0].(type) {
			case ast.Literal:
				isFromLiteral = true
			}
			if to == nil {
				return
			}
			// if we are working with a literal we need to
			// infer the proper type and check literal
			if isFromLiteral {
				switch to.(type) {
				case *types.Union:
					// we handle to union cast below
				default:
					arg := n.Arguments[0]
					s.analyseExpressionType(arg, arg.Type(), to)
				}
			}
			from := n.Arguments[0].Type()
			if from == nil {
				return
			}
			fromUnderlying := types.GetUnderlyingType(from)
			toUnderlying := types.GetUnderlyingType(to)

			if !types.CanCoalesce(fromUnderlying, toUnderlying) {
				s.addError(n, errIllegalTypeCast(from.String(), to.String()))
				return
			}
			n.ReturnTypes = []types.TypeSpec{to}
		} else {
			rts, ok := s.fnSt.Get(fnName)
			if !ok {
				s.addError(n, errFunctionNotFound(fnName))
				return
			}

			s.analyseCallArguments(n, rts.Type.Arg)

			n.ReturnTypes = rts.Type.Ret
			n.IsAnonymousFn = rts.IsAnonymousFn
		}
		if len(n.ReturnTypes) == 0 {
			n.T = &types.Multi{}
		} else if len(n.ReturnTypes) == 1 {
			n.T = n.ReturnTypes[0]
		} else {
			n.T = &types.Multi{Ts: n.ReturnTypes}
		}

		// built in functions such as 'len' do not consume memory
		if !isBuiltinFunction(n.Token.Literal) {
			// mark mut<> as consumed
			// if identifier of type memory make sure to consume
			// memory type and store underlying type in symbol
			// table so that it cant be used later on in the
			// same scope
			for _, arg := range n.Arguments {
				if ident := s.getIdentFromPrefix(arg); ident != nil {
					mt := types.GetUnderlyingMutable(arg.Type())
					if mt == nil {
						continue
					}

					varInfo, _ := s.varSt.Get(ident.TokenLiteral())
					s.varSt.Set(ident.TokenLiteral(), &VarInfo{
						Type:         mt.T,
						Reassignable: varInfo.Reassignable,
					})
				}
			}
		}
	case *ast.TypeCastExpression:
		// TODO: validate type is defined
		// TODO: validate if conversion legal
		// TODO: validate if conversion could lead to error (e.g. if we dont know value and go from u64 to i8)
		// convert underlying value as new type
		s.analyse(n.Argument, "")
		switch n.Argument.(type) {
		case ast.Literal:
			s.analyseExpressionType(n.Argument, n.Argument.Type(), n.Typ)
		}
	case *ast.DeferStatement:
		if s.fnScope == nil {
			// This should be caught by parser as defer statements
			// should only exist within function bodies
			panic("this is a compiler error. please report")
		}
		s.analyse(n.Node, "")
	case *ast.ReturnStatement:
		if s.fnScope == nil {
			// This should be caught by parser as return statements
			// should only exist within function bodies
			panic("this is a compiler error. please report")
		}

		// infer return value types
		for _, rv := range n.Values {
			s.analyse(rv, "")
		}

		// validate number of values returned == expected
		retTypes := n.ReturnTypes()
		fnType := s.fnScope.GetLast()
		if len(fnType.Ret) < len(retTypes) {
			retTypesStr := make([]string, len(retTypes))
			for i, rt := range retTypes {
				retTypesStr[i] = rt.String()
			}
			want := strings.Join(fnType.ReturnTypesString(), ", ")
			got := strings.Join(retTypesStr, ", ")
			s.addError(n, errTooManyReturnValues(want, got))
			return
		} else if len(fnType.Ret) > len(retTypes) {
			retTypesStr := make([]string, len(retTypes))
			for i, rt := range retTypes {
				retTypesStr[i] = rt.String()

			}
			want := strings.Join(retTypesStr, ", ")
			got := strings.Join(fnType.ReturnTypesString(), ", ")
			s.addError(n, errTooLittleReturnValues(want, got))
			return
		}
		// validate return value types match with expected
		for i, rv := range n.Values {
			retT := rv.Type()
			// if nil it means another issue arised
			if retT == nil {
				continue
			}
			// we want to compare type by type and thus we need to
			// extract multi types if a function call is returned
			var typs []types.TypeSpec
			if mt, ok := retT.(*types.Multi); ok {
				typs = append(typs, mt.Ts...)
			} else {
				typs = append(typs, retT)
			}
			for j := range typs {
				expectedType := fnType.GetReturnTypeAt(i + j)
				if !s.analyseExpressionType(rv, typs[j], expectedType) {
					// s.addError(n, errTypeMismatch(expectedType.String(), rt.String()))
					continue
				}
			}
		}
	case *ast.IfElseExpression:
		for _, cond := range n.Conditionals {
			s.analyse(cond, "")
		}

		// TODO: can we remove this?
		// If assignment we want to validate two things:
		// - last value in block is an expreesion
		// - type of last expression in each block is equal
		if s.scope.GetLast() == ASSIGNMENT {
			var prevT types.TypeSpec
			hasTypeMismatch := false
			typs := make([]types.TypeSpec, len(n.Conditionals))
			for i, cond := range n.Conditionals {
				lastStmt := cond.Block.Statements[len(cond.Block.Statements)-1]
				switch exp := lastStmt.(type) {
				case ast.Expression:
					// do nothing
					if prevT == nil {
						prevT = exp.Type()
					} else if !prevT.Equal(exp.Type()) {
						hasTypeMismatch = true
					}
					typs[i] = exp.Type()
				default:
					s.addError(cond, errIfElseExpNonExp())
				}
			}
			if hasTypeMismatch {
				typesStr := make([]string, len(typs))
				for i, ts := range typs {
					typesStr[i] = ts.String()
				}
				s.addError(n, errIfElseExpTypeMismatch(strings.Join(typesStr, ", ")))
			}
			n.T = prevT
		}
	case *ast.ConditionalExpression:
		s.analyse(n.Condition, "")
		s.analyse(n.Block, "")
	case *ast.CopyExpression:
		// validate identifier defined
		s.analyse(n.Ident, "")
	case *ast.CopyUpdateExpression:
		// validate identifier defined
		s.analyse(n.Ident, "")
		s.varSt.Set(name, &VarInfo{Type: n.Type(), Reassignable: true})

		s.analyse(n.Block, "")
	case *ast.StructLiteral:
		// Infer types of anonymous structs
		if n.Name == nil {
			var typ types.Struct
			isNamed := 0
			for i, f := range n.Fields {
				s.analyse(f.Value, "")
				f.T = f.Value.Type()
				if f.Name != nil {
					typ.Ts = append(typ.Ts, types.StructField{Name: f.Name.TokenLiteral(), T: f.T})
					isNamed++
				} else {
					typ.Ts = append(typ.Ts, types.StructField{T: f.T})
				}
				n.Fields[i].Index = i
			}
			if isNamed != 0 && isNamed != len(n.Fields) {
				s.addError(n, errMixedNamedUnnamedStruct("anonymous"))
			}
			n.T = &typ
		} else {

			switch exp := n.Name.(type) {
			case *ast.Identifier:
				s.analyse(n.Name, "")
			case *ast.DotExpression:
				if lib, ok := s.importedSt[exp.Left.String()]; ok {
					typ := lib[exp.Right.String()]
					exp.Left.SetType(typ)
					exp.SetType(typ)
					n.SetType(typ)
				} else {
					// TODO: add error
					panic("this is a compiler error. please report")
				}
			}
			// validate struct exists
			typ := n.Name.Type()

			var structType *types.Struct
			switch t := typ.(type) {
			case *types.Struct:
				structType = t
			case *types.Error:
				// Handle error types with struct-like syntax
				s.analyseErrorStructLiteral(n, t)
				return
			case *types.Definition:
				// for type definitions of structs we still want to be
				// able to validate the usage. This is why we assign the
				// underlying array to 'structType' and perform the validation
				st, ok := t.Underlying.(*types.Struct)
				if !ok {
					s.addError(n, errTypeMismatch("struct", typ.String()))
					return
				}
				structType = st
			case *types.AbstractStruct:
				s.addError(n, errAliasUsedAsLiteral())
				return
			default:
				// TODO: improve error here
				if typ == nil {
					s.addError(n, errTypeMismatch("struct", "nil"))
				} else {
					s.addError(n, errTypeMismatch("struct", typ.String()))
				}
				return

			}
			namedFields := 0
			for _, f := range n.Fields {
				if f.Name != nil {
					namedFields++
				}
			}
			if namedFields != 0 && namedFields != len(n.Fields) {
				structName := n.Name.String()
				s.addError(n, errMixedNamedUnnamedStruct(structName))
				return
			}
			// Validate all required struct fields defined.
			// If we have unnamed struct we need to check
			// that all fields defined
			if namedFields == 0 {
				if len(n.Fields) != len(structType.Ts) {
					s.addError(n, errStructMissingFields(n.Name.TokenLiteral()))
					return
				}
			} else {
				for _, sft := range structType.Ts {
					// Skip any optional types
					if _, ok := sft.T.(*types.Optional); ok {
						continue
					}
					// if pointer to optional e.g. *?int we also
					// allow it to be not set
					if ptrT, ok := sft.T.(*types.Pointer); ok {
						if _, ok := ptrT.T.(*types.Optional); ok {
							continue
						}
					}
					found := false
					for _, f := range n.Fields {
						if sft.Name == f.Name.TokenLiteral() {
							found = true
							break
						}
					}
					if !found {
						s.addError(n, errStructFieldNotDefined(sft.Name))
					}
				}
			}

			// validate all fields
			for i, f := range n.Fields {

				// Set index of fields of struct literal using struct type definition,
				// as Dash allows struct fields to be defined out of order.
				// We need to handle structs with named and unnamed fields
				if f.Name != nil {
					_, index, err := structType.GetTypeByField(f.Name.TokenLiteral())
					if err != nil {
						s.addError(n, errStructUnknownField(n.Name.String(), f.Name.String()))
						continue
					}
					f.Index = index
				} else {
					f.Index = i
				}

				// infer type of value
				s.analyse(f.Value, "")

				// Get type of the field as per struct definition
				// We need to handle structs with named and unnamed fields
				var fieldType types.TypeSpec
				var err error
				if f.Name != nil {
					fieldType, _, err = structType.GetTypeByField(f.Name.TokenLiteral())
				} else {
					fieldType, err = structType.GetTypeByIndex(i)
				}
				internal.AssertTrue(err == nil, "no error expected")

				// infer type of named type
				if namedType, ok := fieldType.(*types.UnknownNamed); ok {
					t, ok := s.varSt.Get(namedType.Name)
					if !ok {
						s.addError(n, errTypeNotFound(namedType.Name))
						continue
					}

					err := structType.SetTypeByField(f.Name.TokenLiteral(), f.T)
					internal.AssertTrue(err == nil, "no error expected")

					f.T = t.Type
				} else if f.T == nil {
					f.T = fieldType
				}

				internal.AssertNotNil(f.T, "expected type of field to be defined")
				internal.AssertNotType[*types.UnknownNamed](f.T, "expected type to not be ast.Named")

				switch f.Value.(type) {
				case ast.Literal:
					s.analyseExpressionType(f.Value, f.Value.Type(), fieldType)
				default:
					// verify type of field matches type in struct definition
					if fieldType.String() != f.Type().String() {
						s.addError(n, errTypeMismatch(fieldType.String(), f.Type().String()))
						continue
					}
				}
			}

			// VarInfo
			switch t := typ.(type) {
			case *types.Struct:
				// typeDef.Type = structType
				s.varSt.Set(n.Name.TokenLiteral(), &VarInfo{Type: structType})
				n.T = structType
			// The reason this is split from *types.Struct case is because
			// we want to keep the type definition type the same. The logic
			// above just ensured that the underlying struct definition is
			// used properly by the struct literal.
			case *types.Definition:
				t.Underlying = structType
				s.varSt.Set(n.Name.TokenLiteral(), &VarInfo{Type: typ})
				n.T = typ
			}

			internal.AssertNotType[*types.UnknownNamed](structType, "expected type to not be ast.Named")

		}
	case *ast.ArrayLiteral:
		if len(n.Values) == 0 {
			return
		}

		for _, el := range n.Values {
			s.analyse(el, name)
		}

		var typ types.Array
		if n.T != nil {
			t := n.T.(*types.Array)
			typ.T = n.Values[0].Type()
			typ.Size = t.Size
		}
		n.T = &typ

	case *ast.DotExpression:
		s.analyse(n.Left, "")
		switch left := n.Left.Type().(type) {
		case *types.Array:
			s.analyse(n.Right, "")
			n.SetType(n.Right.Type())
		case *types.Enum:
			switch t := n.Right.(type) {
			case *ast.Identifier:
				// Validate that the field exists in the enum
				if !left.HasField(t.Value) {
					s.addError(n, errEnumUnknownField(left.Name, t.Value))
					return
				}
				t.T = left
			default:
				panic("todo: add semsis error")
			}
			n.SetType(n.Right.Type())
		case *types.Struct:
			switch t := n.Right.(type) {
			case *ast.Identifier:
				typ, _, err := left.GetTypeByField(n.Right.TokenLiteral())
				if err != nil {
					s.addError(n, errStructUnknownField(n.Left.TokenLiteral(), n.Right.String()))
				}
				t.T = typ
			case *ast.IntegerLiteral:
				typ, err := left.GetTypeByIndex(int(t.Value))
				if err != nil {
					s.addError(n, errStructUnknownField(n.Left.TokenLiteral(), n.Right.String()))
				}
				t.T = typ
			default:
				s.analyse(n.Right, "")
			}
			n.SetType(n.Right.Type())
		case *types.Pointer:
			switch left := left.T.(type) {
			case *types.Struct:
				switch t := n.Right.(type) {
				case *ast.Identifier:
					typ, _, err := left.GetTypeByField(n.Right.TokenLiteral())
					if err != nil {
						s.addError(n, errStructUnknownField(n.Left.TokenLiteral(), n.Right.String()))
					}
					t.T = typ
				default:
					s.analyse(n.Right, "")
				}
			}
			n.SetType(n.Right.Type())
		case *types.Definition, *types.Alias:
			u := types.GetUnderlyingType(left)
			structType, ok := u.(*types.Struct)
			if !ok {
				// TODO: error only dot for structs or enums allowed
				panic("TODO: add semsis error")
			}
			switch t := n.Right.(type) {
			case *ast.Identifier:
				typ, _, err := structType.GetTypeByField(n.Right.TokenLiteral())
				if err != nil {
					s.addError(n, errStructUnknownField(n.Left.TokenLiteral(), n.Right.String()))
				}
				t.T = typ
			default:
				s.analyse(n.Right, "")
			}
			n.SetType(n.Right.Type())
		case *types.AbstractStruct:
			switch t := n.Right.(type) {
			case *ast.Identifier:
				typ, _, err := left.GetTypeByField(n.Right.TokenLiteral())
				if err != nil {
					s.addError(n, errStructAliasUnknownField(n.Left.TokenLiteral(), n.Right.String()))
				}
				t.T = typ
			case *ast.FunctionCallExpression:
				s.analyse(n.Right, "")
			}
			n.SetType(n.Right.Type())
		case *types.ImportedNamed:
			var name string
			if fn, ok := n.Right.(*ast.FunctionCallExpression); ok {
				name = fn.TokenLiteral()
				// analyze function arguments
				for _, arg := range fn.Arguments {
					s.analyse(arg, "")
				}
			} else {
				name = n.Right.String()
			}
			// set type of expression
			typ := s.importedSt[n.Left.String()][name]
			if enumType, ok := left.Typ.(*types.Enum); ok && typ == nil {
				// special case where enum accessed from lib e.g.
				// lib_x.enum_y.field_z. When typ is nil it means
				// we are checking enum_y.field_z in map but that
				// doesnt exist as its defined under scope lib_x.
				// Validate that the field exists in the imported enum
				if !enumType.HasField(name) {
					s.addError(n, errEnumUnknownField(enumType.Name, name))
					return
				}
				n.SetType(left)
				return
			} else if t, ok := left.Typ.(*types.Struct); ok && typ == nil {
				// special case where struct accessed from lib. If it is
				// already an ImportedNamed type we leave as is and if its
				// not a builtin type we create a new ImportedNamed type
				resolvedType, _, _ := t.GetTypeByField(n.Right.String())
				_, ok := resolvedType.(*types.ImportedNamed)
				if !ok && !types.IsBuiltinType(resolvedType) {
					resolvedType = &types.ImportedNamed{
						Lib: left.Lib,
						Typ: resolvedType,
					}
				}
				n.SetType(resolvedType)
				return
			} else if typ == nil {
				s.addError(n, errIdentifierNotFound(n.Left.String()+"."+name))
				return
			}

			switch typ := typ.(type) {
			case *types.Function:
				// convert local types to be ImportNamed
				// except if they are already of type
				// ImportNamed or are a builtin type
				for i, argT := range typ.Arg {
					_, ok := argT.(*types.ImportedNamed)
					if !ok && !types.IsBuiltinType(argT) {
						typ.Arg[i] = &types.ImportedNamed{
							Lib: left.Lib,
							Typ: argT,
						}
					}
				}
				// validate function call arguments for imported functions
				if fn, ok := n.Right.(*ast.FunctionCallExpression); ok {
					s.analyseCallArguments(fn, typ.Arg)
				}

				// convert non builtin types to be
				// ImportedNamed
				for i, retT := range typ.Ret {
					_, ok := retT.(*types.ImportedNamed)
					if !ok && !types.IsBuiltinType(retT) {
						typ.Ret[i] = &types.ImportedNamed{
							Lib: left.Lib,
							Typ: retT,
						}
					}
				}
				n.SetType(&types.Multi{Ts: typ.Ret})
			default:
				left.Typ = typ
				n.SetType(left)
			}
		}
	case *ast.IndexExpression:
		s.analyse(n.Left, "")
		if n.Left.Type() == nil {
			// another error occured
			return
		}
		arrTyp := types.GetUnderlyingIndexable(n.Left.Type())
		typ := types.ArrayTypeAt(arrTyp, len(n.Indices))
		n.SetType(typ)

		s.varSt.Set(n.String(), &VarInfo{Type: n.Type()})

		for _, idx := range n.Indices {
			s.analyse(idx, "")
		}
	case *ast.SliceExpression:
		s.analyse(n.Left, "")
		for _, idx := range n.Indices {
			s.analyse(idx, "")
		}
	case *ast.ForStatement:
		s.scope.Push(FOR)
		defer s.scope.Pop()

		if n.Assignment != nil {
			assignee, ok := n.Assignment.Declerations[0].(*ast.Identifier)
			if !ok {
				s.addError(n.Assignment, errInvalidAssignment())
			} else {
				val := n.Assignment.Values[0]
				s.varSt.Set(assignee.TokenLiteral(), &VarInfo{Reassignable: true})
				s.analyse(n.Assignment, "")

				n.Assignment.SetTypeAt(0, val.Type())
				s.varSt.Set(assignee.TokenLiteral(), &VarInfo{Type: assignee.Type(), Reassignable: true})
			}
		}
		if n.Condition != nil {
			switch cond := n.Condition.(type) {
			case *ast.InfixExpression:
				// validate boolean condition used
				switch cond.Token.Type {
				case token.LT, token.LTE, token.GT, token.GTE,
					token.EQ, token.NEQ, token.OR, token.AND:
					s.analyse(n.Condition, "")
				default:
					s.addError(n, errInvalidBooleanCondition(cond.Operator))
				}

			case *ast.PrefixExpression:
				switch cond.Token.Type {
				case token.BANG:
					s.analyse(n.Condition, "")
				default:
					s.addError(n, errInvalidBooleanCondition(cond.Operator))
				}
			case *ast.FunctionCallExpression:
				s.analyse(n.Condition, "")
			default:
				panic("this is a compiler error. please report")
			}
		}
		if n.Change != nil {
			s.analyse(n.Change, "")
		}
		s.analyse(n.Block, "")
	case *ast.TryExpression:
		// TODO: check that try used in error-prone function
		s.analyse(n.Right, "")
		// TODO: check if try used with error-prone function
		n.SetType(n.Right.Type())
	case *ast.RaiseStatement:
		switch exp := n.Error.(type) {
		case *ast.StructLiteral:
			s.analyse(exp, "")
			if _, ok := s.varSt.Get(exp.Name.String()); !ok {
				s.addError(n, errIdentifierNotFound(n.Error.String()))
				return
			}
		case *ast.Identifier:
			if _, ok := s.varSt.Get(exp.TokenLiteral()); !ok {
				s.addError(n, errIdentifierNotFound(n.Error.String()))
				return
			}

		}
	case *ast.MatchExpressionStatement:

		// This is fine as its only used for
		// assignments in match when dev wants
		// to access data under new matched type
		// e.g. for unions
		if exp, ok := n.Scrutinee.(*ast.InfixExpression); ok {
			s.analyse(exp.Right, "")
			exp.SetType(exp.Right.Type())
			exp.Left.SetType(exp.Right.Type())
		} else {
			s.analyse(n.Scrutinee, "")
		}
		sT := n.Scrutinee.Type()
		// this means scrutinee was not defined or
		// another error occured
		if sT == nil {
			return
		}

		for _, c := range n.Cases {
			// Analyze each predicate in the case
			var firstMatchingType types.TypeSpec
			for _, pred := range c.Predicates {
				s.analyse(pred, "")
				cT := pred.Type()
				if cT == nil {
					continue
				}
				isLiteral := ast.IsLiteral(pred)
				if !validateMatchCaseType(sT, cT, isLiteral) {
					s.addError(c, errTypeMismatch(sT.String(), cT.String()))
					continue
				}
				// As the types match, we can replace type of predicate.
				// This is required to be able to, for example, match byte
				// with char literals
				_, isUnion := n.Scrutinee.Type().(*types.Union)
				if !isUnion {
					pred.SetType(sT)
				}
				// Keep track of the first matching type for scrutinee type assignment
				if firstMatchingType == nil {
					firstMatchingType = cT
				}
			}

			s.varSt.Scope()
			// check required as literals not stored in symbol table
			// BUG: improve check here. only store if in symbol table if
			// identifier, dot expression, index or slice expression.
			// Use the first matching predicate type for scrutinee type assignment
			if firstMatchingType != nil {
				switch exp := n.Scrutinee.(type) {
				case *ast.InfixExpression:
					if exp.Operator != "=" {
						break
					}
					_, ok := exp.Left.(*ast.Identifier)
					if !ok {
						panic("todo: add semsis error")
					}

					s.varSt.Set(exp.Left.TokenLiteral(), &VarInfo{Type: firstMatchingType})

				case *ast.Identifier:
					info, ok := s.varSt.Get(n.Scrutinee.TokenLiteral())
					if !ok {
						s.addError(n.Scrutinee, errIdentifierNotFound(n.Scrutinee.String()))
						return
					}
					s.varSt.Set(n.Scrutinee.TokenLiteral(), &VarInfo{Type: firstMatchingType, Reassignable: info.Reassignable})
				default:
					// in case of literals and expressions we dont need
					// to do anything else
				}
			}

			for _, stmt := range c.Body {
				s.analyse(stmt, "")
				switch r := stmt.(type) {
				case ast.Expression:
					n.T = r.Type()
				}
			}
			s.varSt.Unscope()
		}

		if n.Default != nil {
			s.varSt.Scope()
			for _, stmt := range n.Default.Body {
				s.analyse(stmt, "")
				switch r := stmt.(type) {
				case ast.Expression:
					n.T = r.Type()
				}
			}
			s.varSt.Unscope()
		}

	case *ast.AssignmentStatement:
		s.scope.Push(ASSIGNMENT)
		defer s.scope.Pop()

		s.analyseAssignmentStatement(n)

	case *ast.PrefixExpression:
		s.analyse(n.Right, "")
		n.T = n.Right.Type()
		switch n.Token.Type {
		case token.ASTERISK:
			if pointerT, ok := n.T.(*types.Pointer); ok {
				n.T = pointerT.T
			} else {
				s.addError(n, errIllegalValueOf("value not a pointer"))
				return
			}
		case token.AMPERSAND:
			if _, ok := n.T.(*types.Pointer); ok {
				s.addError(n, errIllegalAddressOf("unable to get address of pointer"))
				return
			}
			n.T = &types.Pointer{T: n.T}
		case token.OPTIONAL:
			t, ok := n.Right.Type().(*types.Optional)
			if !ok {
				s.addError(n, errIllegalForceUnwrap())
				return
			}
			n.SetType(t.T)

		case token.BANG:
			// TODO: check only possible on value of type bool
		}
	case *ast.InfixExpression:

		left := n.Left
		right := n.Right

		s.analyse(left, "")
		s.analyse(right, "")

		// This allows generator to only worry
		// about one case when generating '=='
		// and '!=' for NullLiterals
		switch left.(type) {
		case *ast.NullLiteral:
			n.Left, n.Right = n.Right, n.Left
		}

		// If nil it means there most likely was
		// a problem analysing LHS and/or RHS.
		// The branch can also execute if there
		// is a bug in the semantic analysis
		if left.Type() == nil || right.Type() == nil {
			return
		}

		// This check has to occur before type check
		// otherwise we cant return a helpful error
		switch n.Token.Type {
		case token.NULL_COALESCE:
			// check null not used as value in null coalesce
			if _, ok := left.(*ast.NullLiteral); ok {
				s.addError(n, errNullUsedInNullCoalesce())
				return
			}
			if _, ok := right.(*ast.NullLiteral); ok {
				s.addError(n, errNullUsedInNullCoalesce())
				return
			}
		}

		// if error occured then return. no need to set type
		// as n is InfixExpression where we compare LHS with RHS
		if !s.analyseExpressionType(n, n.Type(), nil) {
			return
		}

		leftT, rightT := left.Type(), right.Type()

		// As we know types match already we can validate infix operator
		// used correctly without rechecking RHS
		if !validateOperator(leftT, n.Token.Type) {
			s.addError(n, errIllegalOperationOnType(n.Operator, leftT.String()))
			return
		}
		// For most operations LHS and RHS type are the same
		// so we set it here. The remaining code in this block
		// handles special cases where the type might be different
		n.T = leftT

		switch n.Token.Type {
		// Check reassignment used in allowed contexts
		case token.ASSIGN:
			// This case is needed for for loop increments like 'i = i + 2'
			// which are parsed as InfixExpression, not AssignmentStatement

			if ident, ok := left.(*ast.Identifier); ok {
				ident.SetType(right.Type())
				if varInfo, exists := s.varSt.Get(ident.TokenLiteral()); exists && varInfo.Reassignable {
					s.varSt.Set(ident.TokenLiteral(), &VarInfo{Type: right.Type(), Reassignable: true})
				}
			}

		// Ensure boolean operations infered as bool
		case token.EQ, token.NEQ, token.GT, token.GTE, token.LT, token.LTE:
			n.T = &types.ConstBool
		// ensure proper use of null coalesce
		case token.NULL_COALESCE:
			// ensure null coalesche used properly with optional
			if _, ok := rightT.(*types.Optional); ok {
				s.addError(n, errNullCoalesceRHSOptional(n.Right.String()))
				return
			}
			// if _, ok := leftT.(*types.Optional); !ok {
			// 	s.addError(n, errNullCoalesceLHSNonOptional(n.Left.String()))
			// 	return
			// }
			n.T = n.Right.Type()
		}

		// If algebraic, binary or string operation done on type def
		// with guard we set the type of the value to dirty<T>. If the
		// type is already dirty<T> we ensure the infix expression type
		// remains dirty
		switch n.Token.Type {
		// TODO: add binary operations
		case token.PLUS, token.MINUS, token.ASTERISK,
			token.SLASH, token.MOD, token.NULL_COALESCE,
			token.ASSIGN:
			// if already dirty we skip
			if _, ok := rightT.(*types.Dirty); ok {
				break
			} else if _, ok := leftT.(*types.Dirty); ok {
				break
			}
			if _, ok := s.guardedType.Get(rightT.Ident()); ok {
				n.T = &types.Dirty{T: rightT}
			} else if _, ok := s.guardedType.Get(leftT.Ident()); ok {
				n.T = &types.Dirty{T: leftT}
			}
			if rt, ok := rightT.(*types.Dirty); ok {
				n.T = rt
			} else if lt, ok := leftT.(*types.Dirty); ok {
				n.T = lt
			}
		}

	case *ast.PostfixExpression:
		s.analyse(n.Left, "")
		n.SetType(n.Left.Type())
	case *ast.NullLiteral:
		n.T = &types.ConstNull
	case *ast.IntegerLiteral:
		n.T = &types.ConstI64
	case *ast.FloatLiteral:
		n.T = &types.ConstF64
	case *ast.CharacterLiteral:
		n.T = &types.ConstChar
	case *ast.Identifier:
		info, ok := s.varSt.Get(n.TokenLiteral())
		if ok && info != nil {
			if info.Type == nil {
				// NOTE: this can also signify another error occured
				return
			}
			n.T = info.Type
			return
		}
		fnInfo, ok := s.fnSt.Get(n.TokenLiteral())
		if ok && fnInfo != nil {
			n.T = fnInfo.Type
			return
		}
		if _, ok := s.importedSt[n.TokenLiteral()]; ok {
			n.T = &types.ImportedNamed{
				Lib: n.TokenLiteral(),
				// NOTE: we purposly dont set this
				// field as we dont know what the
				// unknown named value is here
				// Typ:
			}
			return
		}
		s.addError(n, errIdentifierNotFound(n.TokenLiteral()))
	case *ast.BlockStatement:
		// enter new scope
		s.varSt.Scope()
		defer s.varSt.Unscope()

		// TODO: this scope can be removed by making a new function
		// s.analyseBlockStatement(blk, TAG) e.g. USE, FOR etc.
		// case: 1 we are in a for loop
		scope := s.scope.GetLast()
		for i, st := range n.Statements {
			switch st := st.(type) {
			case *ast.KeywordStatement:
				if scope == FOR {
					if i != len(n.Statements)-1 {
						s.addError(st, errKeywordNotLastInstruction(st.TokenLiteral()))
					}
				} else {
					s.addError(st, errIllegalUseOfKeyword(st.TokenLiteral()))
				}
			default:
				s.analyse(st, "")
			}
		}

	}
}

func (s *Semantics) analyseCallArguments(n *ast.FunctionCallExpression, expectedTs []types.TypeSpec) {
	// Check arguments passed
	if len(expectedTs) < len(n.Arguments) {
		s.addError(n, errTooManyArguments(n.TokenLiteral()))
		return
	} else if len(expectedTs) > len(n.Arguments) {
		s.addError(n, errTooLittleArguments(n.TokenLiteral()))
		return
	}

	// Check arguments of same type
	for i, arg := range n.Arguments {
		s.analyse(arg, "")
		expectedType := expectedTs[i]

		typ := s.inferUnknownNamedType(arg.Type())
		if typ == nil {
			// another error occured
			return
		}
		arg.SetType(typ)

		if isBuiltinFunction(n.TokenLiteral()) {
			// We want to treat type defs as the underlying
			// type so that they can be used with built-in
			// functions
			argT := s.coalesceTypeForBuiltIn(arg.Type(), n.TokenLiteral())
			if !expectedType.Equal(argT) {
				s.addError(arg, errTypeMismatch(expectedType.String(), argT.String()))
			}
			continue
		} else {
			switch arg.(type) {
			case ast.Literal:
				s.analyseExpressionType(arg, arg.Type(), expectedType)
			default:
				if !s.typesEqual(expectedType, arg.Type()) {
					s.addError(arg, errTypeMismatch(expectedType.String(), arg.Type().String()))
					return
				}
			}
		}

	}
}

func (s *Semantics) analyseAssignmentStatement(n *ast.AssignmentStatement) {

	// validate declerations
	for _, decl := range n.Declerations {
		var ident string
		switch decl := decl.(type) {
		case *ast.DeclarationStatement:
			continue
		case *ast.Identifier:
			ident = decl.TokenLiteral()
		case *ast.DotExpression:
			ident = decl.Left.TokenLiteral()
		case *ast.IndexExpression:
			ident = decl.Left.TokenLiteral()
		case *ast.SliceExpression:
			ident = decl.Left.TokenLiteral()
		}
		ra, ok := s.varSt.Get(ident)
		if !ok && ra == nil {
			s.addError(n, errIdentifierNotFound(ident))
			continue
		}
		if !ra.Reassignable {
			s.addError(n, errIllegalUpdate(ident))
			continue
		}
	}

	declCnt := 0
	isFnCall := false
	for i := range n.Values {
		val := n.Values[i]
		s.analyse(val, n.VarNameAt(i))
		// this generally means another error happened
		if val.Type() == nil {
			declCnt++
			continue
		}

		underlyingT := types.GetUnderlyingType(val.Type())
		switch t := underlyingT.(type) {
		case *types.Multi:
			isFnCall = true
			if len(t.Ts) == 0 {
				s.addError(val, errCannotAssignVoidFunction())
				break
			}

			if len(t.Ts) > len(n.Declerations) {
				s.addError(val, errAssignmentMismatch(len(t.Ts), len(n.Declerations)))
				break
			}

			for j, rt := range t.Ts {
				n.SetTypeAt(i+j, rt)
				underlying := types.GetUnderlyingType(rt)
				if fnValType, ok := underlying.(*types.Function); ok {
					f := &FnInfo{
						Type: fnValType,
					}
					s.fnSt.Set(n.VarNameAt(i+j), f)
				} else {
					ident := n.VarNameAt(i + j)
					_, isMutable := n.TypeAt(i + j).(*types.Mutable)
					isReassignable := s.isReassignable(ident) || n.IsVarAt(i+j) || isMutable
					s.setDeclerationInSymTab(ident, rt, isReassignable)
				}
				declCnt++
			}
		case *types.Function:
			n.SetTypeAt(i, val.Type())
			f := &FnInfo{
				Type:          t,
				IsAnonymousFn: true,
			}
			s.fnSt.Set(n.VarNameAt(i), f)
			declCnt++
		default:
			decl := n.Declerations[i]
			switch decl := decl.(type) {
			case *ast.IndexExpression, *ast.SliceExpression, *ast.DotExpression:
				exp := decl.(ast.Expression)
				s.analyse(exp, "")
				if exp.Type() == nil {
					// another error occured
					return
				}
				if _, ok := val.(ast.Literal); ok {
					s.analyseExpressionType(val, val.Type(), exp.Type())
					val.SetType(exp.Type())
				} else {
					if !types.CanCoalesce(val.Type(), exp.Type()) {
						s.addError(n, errTypeMismatch(exp.Type().String(), val.Type().String()))
						return
					}
				}
			case *ast.Identifier:
				n.SetTypeAt(i, val.Type())
				s.setDeclerationInSymTab(n.VarNameAt(i), val.Type(), true)
			default:
				n.SetTypeAt(i, val.Type())
				s.setDeclerationInSymTab(n.VarNameAt(i), val.Type(), n.IsVarAt(i))
			}
			declCnt++
		}
	}

	if !isFnCall && declCnt != len(n.Declerations) {
		s.addError(n, errAssignmentMismatch(declCnt, len(n.Declerations)))
		return
	}

}

func (s *Semantics) setDeclerationInSymTab(n string, t types.TypeSpec, isReassignable bool) {
	_, isMutable := t.(*types.Mutable)
	vi := &VarInfo{
		Type:         t,
		Reassignable: isReassignable || isMutable,
	}
	s.varSt.Set(n, vi)
}

// Recursively removes types.Definition and *types.Dirty (where applicable e.g. not for 'validate')
// to return the primitive type so it can be used with built in function.
func (s *Semantics) coalesceTypeForBuiltIn(t types.TypeSpec, fnName string) types.TypeSpec {
	switch t := t.(type) {
	case *types.Definition:
		return s.coalesceTypeForBuiltIn(t.Underlying, fnName)
	case *types.Dirty:
		if fnName != "validate" {
			return s.coalesceTypeForBuiltIn(t.T, fnName)
		}
		return t
	case *types.Mutable:
		return s.coalesceTypeForBuiltIn(t.T, fnName)
	default:
		return t
	}
}

func (s *Semantics) analyseDuplicateIdentifiers(nodes []ast.Node) {
	// check for duplicates
	m := make(map[string]uint16, len(nodes))
	for i, stmt := range nodes {
		var name string
		switch stmt := stmt.(type) {
		case *ast.StructStatement:
			name = stmt.Name.String()
		case *ast.TypeDefinitionStatement:
			name = stmt.Name.String()
		case *ast.TypeAliasStatement:
			name = stmt.Name.String()
		case *ast.UnionStatement:
			name = stmt.Name.String()
		case *ast.EnumStatement:
			name = stmt.Name.String()
		case *ast.ErrorStatement:
			if stmt == nil || stmt.Name == nil {
				continue
			}
			name = stmt.Name.String()
		case *ast.FunctionExpression:
			name = stmt.Name.String()
		default:
			continue
		}
		if _, ok := m[name]; ok {
			s.addError(stmt, errDuplicateIdentifierFound(name))
		} else {
			m[name] = uint16(i)
		}
	}

	// check for duplicate fields in union
	for _, stmt := range nodes {
		switch un := stmt.(type) {
		case *ast.UnionStatement:
			dups := make(map[string]struct{}, len(un.Types))
			for _, typ := range un.Types {
				if _, ok := dups[typ.String()]; ok {
					s.addError(typ, errDuplicateUnionField(typ.String(), un.Name.TokenLiteral()))
				} else {
					dups[typ.String()] = struct{}{}
				}
			}
		}
	}
}

// Sets types in symbol table
func (s *Semantics) analyseTypes(nodes []ast.Node) {
	for _, stmt := range nodes {
		switch n := stmt.(type) {
		case *ast.StructStatement:
			strct := &types.Struct{Name: n.Name.TokenLiteral()}
			for _, f := range n.Fields {
				_, exists := s.varSt.Get(f.Type.Ident())

				var fieldType types.TypeSpec
				if !exists && !types.IsTypeIdent(f.Type.Ident()) {
					fieldType = &types.UnknownNamed{Name: f.Type.Ident()}
				} else {
					fieldType = f.Type
				}

				if f.Name != nil {
					field := types.StructField{Name: f.Name.TokenLiteral(), T: fieldType}
					strct.Ts = append(strct.Ts, field)
				} else {
					field := types.StructField{T: fieldType}
					strct.Ts = append(strct.Ts, field)
				}
			}
			n.T = strct
			s.varSt.Set(n.Name.String(), &VarInfo{Type: n.Type()})
		case *ast.TypeDefinitionStatement:
			// underlying type might be nil if it references
			// a type declared later
			typ := s.inferUnknownNamedType(n.UnderlyingType)
			if typ != nil {
				n.UnderlyingType = typ
			}

			// define cast function, if type has a predicate
			// cast will cause return type to be 'dirty'
			if n.Guard != nil {
				// set type as dirty type def
				n.T = &types.Dirty{T: &types.Definition{Name: n.Name.String(), Underlying: n.UnderlyingType}}
			} else {
				// set type as type def
				n.T = &types.Definition{Name: n.Name.String(), Underlying: n.UnderlyingType}
			}
			s.varSt.Set(n.Name.String(), &VarInfo{Type: n.Type()})
			s.typeSt.Set(n.Name.String(), n.Type())
		case *ast.TypeAliasStatement:
			n.T = n.UnderlyingType
			s.varSt.Set(n.Name.String(), &VarInfo{Type: n.Type()})
		case *ast.UnionStatement:
			s.varSt.Set(n.Name.String(), &VarInfo{Type: n.T})
		case *ast.EnumStatement:
			s.varSt.Set(n.Name.String(), &VarInfo{Type: n.T})
		case *ast.ErrorStatement:
			errorFields := make([]types.ErrorField, len(n.Params))
			for i, param := range n.Params {
				errorFields[i] = types.ErrorField{
					Name: param.Name.TokenLiteral(),
					T:    param.Type,
				}
			}
			err := &types.Error{
				Name:   n.Name.TokenLiteral(),
				Fields: errorFields,
			}
			n.T = err
			s.varSt.Set(n.Name.String(), &VarInfo{Type: n.Type()})
		case *ast.AssignmentStatement:
			// TODO
		}
	}
}

func (s *Semantics) analyseCircularTypeReferences(nodes []ast.Node) {
	m := make(map[string]uint16, len(nodes))
	for i, stmt := range nodes {
		var name string
		switch stmt := stmt.(type) {
		case *ast.StructStatement:
			name = stmt.Name.String()
		case *ast.UnionStatement:
			name = stmt.Name.String()
		case *ast.TypeDefinitionStatement:
			name = stmt.Name.String()
		case *ast.TypeAliasStatement:
			name = stmt.Name.String()
		default:
			continue
		}
		m[name] = uint16(i)
	}

	// build adjecency list
	adj := make([][]uint16, len(nodes))
	for i, stmt := range nodes {
		// For each field mark if it references another struct in library
		switch stmt := stmt.(type) {
		case *ast.StructStatement:
			for _, field := range stmt.Fields {
				fieldTypeIdent := field.Type.Ident()

				j, ok := m[fieldTypeIdent]
				if !ok {
					continue
				}

				switch field.Type.(type) {
				case *types.Optional:
					// empty case to capture allowed self references
				default:
					// if field references self then only allowed as optional type
					if uint16(i) == j {
						s.addError(field, errRecursiveStructReference(field.Name.TokenLiteral(), stmt.Name.TokenLiteral()))
					} else {
						adj[i] = append(adj[i], j)
					}
				}
			}
		case *ast.UnionStatement:
			for _, field := range stmt.Types {
				ident := field.String()
				j, ok := m[ident]
				if !ok {
					continue
				}
				if uint16(i) == j {
					s.addError(field, errRecursiveUnionReference(stmt.Name.TokenLiteral()))
				} else {
					adj[i] = append(adj[i], j)
				}
			}
		}
	}

	// check for cycles
	if path, hasCycle := isCyclic(adj); hasCycle {
		names := make([]string, len(path))
		for i, idx := range path {
			names[i] = getIdentifier(nodes[idx])
		}
		s.addError(nodes[path[0]], errCyclicalTypeDeclarations(names))
	}
}

func (s *Semantics) resolveAllTypeReferences(nodes []ast.Node) {
	for _, n := range nodes {
		switch n := n.(type) {
		case *ast.StructStatement:
			for i, field := range n.Fields {
				typ := s.inferUnknownNamedType(field.Type)
				if typ == nil {
					s.addError(field, errTypeNotFound(field.Type.String()))
					continue
				}
				n.T.Ts[i].T = typ
				field.Type = typ
			}

			s.varSt.Set(n.Name.String(), &VarInfo{Type: n.Type()})
			s.typeSt.Set(n.Name.String(), n.Type())
		case *ast.TypeDefinitionStatement:
			// ensure underlying type not unknown
			typ := s.inferUnknownNamedType(n.UnderlyingType)
			if typ == nil {
				s.addError(n, errTypeNotFound(n.UnderlyingType.String()))
				continue
			}
			n.UnderlyingType = typ

			// define cast function, if type has a predicate
			// cast will cause return type to be 'dirty'
			if n.Guard != nil {
				// set type as dirty type def
				n.T = &types.Dirty{T: &types.Definition{Name: n.Name.String(), Underlying: n.UnderlyingType}}
			} else {
				// set type as type def
				n.T = &types.Definition{Name: n.Name.String(), Underlying: n.UnderlyingType}
			}
			s.varSt.Set(n.Name.String(), &VarInfo{Type: n.Type()})
			s.typeSt.Set(n.Name.String(), n.Type())
		case *ast.TypeAliasStatement:
			typ := s.inferUnknownNamedType(n.UnderlyingType)
			if typ == nil {
				s.addError(n, errTypeNotFound(n.UnderlyingType.String()))
				continue
			}
			n.UnderlyingType = typ
			n.T = typ

			s.varSt.Set(n.Name.String(), &VarInfo{Type: n.Type()})
			s.typeSt.Set(n.Name.String(), n.Type())
		case *ast.UnionStatement:
			typs := make([]types.TypeSpec, len(n.Types))
			for i, typ := range n.Types {
				// if recursive def we already added error
				// and skip any further checks
				// if j, ok := m[typ.T.Ident()]; ok && uint16(i) == j {
				// 	continue
				// }
				if _, ok := typ.T.(*types.UnknownNamed); ok {
					val, ok := s.varSt.Get(typ.T.Ident())
					if !ok {
						s.addError(typ, errIdentifierNotFound(typ.String()))
						continue
					}
					internal.AssertNotNil(val.Type, "expected type to be set if stored in symbol table")

					typ.T = val.Type
					typs[i] = val.Type
				}
				typs[i] = typ.T
			}
			n.T = &types.Union{Name: n.Name.String(), Ts: typs}
			s.varSt.Set(n.Name.String(), &VarInfo{Type: n.T})
			s.typeSt.Set(n.Name.String(), n.T)
		case *ast.ErrorStatement:
			for i, field := range n.T.Fields {
				typ := s.inferUnknownNamedType(field.T)
				if typ == nil {
					s.addError(n.Params[i], errTypeNotFound(field.T.String()))
					continue
				}
				// n.T.Fields[i].T = typ
				field.T = typ
			}
			s.varSt.Set(n.Name.String(), &VarInfo{Type: n.Type()})
			s.typeSt.Set(n.Name.String(), n.Type())
		case *ast.FunctionExpression:
			// TODO: add any remaining handling here

			// We dont need to scope the symbol table as all function literals
			// encountered here are global within library
			s.analyseFunctionExpression(n, "")
		}
	}
}

// Return identifier directly or from prefix. Returning nil means
// no identifierfound
func (s *Semantics) getIdentFromPrefix(n ast.Node) *ast.Identifier {
	switch n := n.(type) {
	case *ast.PrefixExpression:
		return s.getIdentFromPrefix(n.Right)
	case *ast.Identifier:
		return n
	}
	return nil
}

func validateMatchCaseType(sT, cT types.TypeSpec, isCtLiteral bool) bool {
	if isCtLiteral {
		return types.CanCoalesce(sT, cT)
	}
	switch sT := sT.(type) {
	case *types.Union:
		for _, t := range sT.Ts {
			if validateMatchCaseType(t, cT, isCtLiteral) {
				return true
			}
		}
		return false
	default:
		return sT.Equal(cT)
	}
}

// NOTE: does not validate assignment!!
// int: +, -, *, /, %, <, <=, >=, >, ==, !=
// float: +, -, *, /, <, <=, >=, >, ==, !=
// optional (incl. null): ??, ==, != (BUT ?? only possible if LHS optional and RHS non optional)
// bool: ==, !=, &&, ||
// string: ==, !=, +
// byte: ==, !=, <<, >>, ...
// enum: ==, !=
// array: none
// struct: none
// dirty<T>: checks underlying type
// definition<T>: checks underlying type
func validateOperator(t types.TypeSpec, tkn token.Type) bool {
	if tkn == token.ASSIGN {
		return true
	}
	switch t := t.(type) {
	case *types.Int:
		switch tkn {
		case token.PLUS, token.MINUS, token.ASTERISK,
			token.SLASH, token.MOD, token.LT, token.LTE,
			token.GT, token.GTE, token.EQ, token.NEQ,
			token.COLON:
			return true
		default:
			return false
		}
	case *types.Float:
		switch tkn {
		case token.PLUS, token.MINUS, token.ASTERISK,
			token.SLASH, token.LT, token.LTE, token.GT,
			token.GTE, token.EQ, token.NEQ:
			return true
		default:
			return false
		}
	case *types.Optional:
		switch tkn {
		case token.NULL_COALESCE, token.EQ, token.NEQ:
			return true
		default:
			return false
		}
	case *types.Bool:
		switch tkn {
		case token.EQ, token.NEQ, token.AND, token.OR:
			return true
		default:
			return false
		}
	case *types.Byte:
		switch tkn {
		case token.PLUS, token.MINUS, token.ASTERISK,
			token.SLASH, token.MOD, token.EQ, token.NEQ,
			token.LT, token.LTE, token.GT, token.GTE:
			return true
		default:
			return false
		}
	case *types.Char:
		switch tkn {
		case token.PLUS, token.MINUS, token.ASTERISK,
			token.SLASH, token.MOD, token.EQ, token.NEQ,
			token.LT, token.LTE, token.GT, token.GTE:
			return true
		default:
			return false
		}
	case *types.String:
		switch tkn {
		case token.PLUS, token.EQ, token.NEQ:
			return true
		default:
			return false
		}
	case *types.Enum:
		switch tkn {
		case token.EQ, token.NEQ, token.GT, token.GTE, token.LT, token.LTE:
			return true
		default:
			return false
		}
	case *types.Null:
		switch tkn {
		case token.EQ, token.NEQ:
			return true
		default:
			return false
		}
	case *types.Pointer:
		switch tkn {
		case token.EQ, token.NEQ:
			return true
		default:
			return false
		}
	case *types.Dirty:
		return validateOperator(t.T, tkn)
	case *types.Definition:
		return validateOperator(t.Underlying, tkn)
	case *types.ImportedNamed:
		return validateOperator(t.Typ, tkn)
	case *types.Array:
		return false
	case *types.Struct:
		return false
	default:
		return false
	}
}

// Analyses function literl:
// - infers types for arguments e.g. (x, y i64)
// - validates named types exist in current scope
// - sets function signature type in symbol table
func (s *Semantics) analyseFunctionExpression(n *ast.FunctionExpression, name string) {
	argTypes := make([]types.TypeSpec, len(n.Arguments))
	for i := range n.Arguments {
		arg := n.Arguments[i]
		if arg.Type == nil {
			// walk forwards until not nil, if end of loop raise error
			// back track to set type e.g. sets x to i64: (x, y i64)
			for j := i + 1; j < len(n.Arguments); j++ {
				argj := n.Arguments[j]
				if argj.Type != nil {
					for z := j - 1; z >= i; z-- {
						n.Arguments[z].Type = argj.Type
						argTypes[z] = argj.Type
					}
					break
				}
			}
		}
		if importedT, ok := arg.Type.(*types.ImportedNamed); ok {
			typ := s.importedSt[importedT.Lib][importedT.Typ.Ident()]
			importedT.Typ = typ
		} else {
			typ := s.inferUnknownNamedType(arg.Type)
			if typ == nil {
				s.addError(arg, errTypeNotFound(arg.Type.Ident()))
				continue
			}
			arg.Type = typ

			// ensure if argument function is also stored in
			// function symbol table
			if ft, ok := arg.Type.(*types.Function); ok {
				s.fnSt.Set(arg.Name.TokenLiteral(), &FnInfo{Type: ft})
			}
		}
		_, isMutable := arg.Type.(*types.Mutable)
		s.varSt.Set(arg.Name.TokenLiteral(), &VarInfo{Type: arg.Type, Reassignable: isMutable})
		argTypes[i] = arg.Type
	}

	retTypes := make([]types.TypeSpec, len(n.ReturnValues))
	// set types of named return arguments
	for i, ret := range n.ReturnValues {
		typ := s.inferUnknownNamedType(ret.T)
		if typ == nil {
			s.addError(ret, errTypeNotFound(ret.T.Ident()))
			continue
		}
		ret.T = typ
		retTypes[i] = typ
	}

	// infer name for lamba functions and set fn table
	if n.Name == nil {
		n.IsAnonymous = true
		n.Name = &ast.Identifier{Token: token.Token{Type: token.IDENT, Literal: name, Position: n.Token.Position}, Value: name}
	}
	if !n.IsAnonymous && n.Name.Value == "main" {
		n.Public = true
	}
	fnType := &types.Function{Arg: argTypes, Ret: retTypes}
	n.T = fnType
	s.fnSt.Set(n.Name.TokenLiteral(), &FnInfo{Type: fnType})

}

func (s *Semantics) isReassignable(ident string) bool {
	info, ok := s.varSt.Get(ident)
	if !ok {
		return false
	}
	return info.Reassignable
}

// Validates whether expr can be coerced into the targetType in the case
// of literals or if there is an exact match for the remaining expressions
func (s *Semantics) analyseExpressionType(expr ast.Expression, exprType, targetType types.TypeSpec) bool {
	switch lit := expr.(type) {
	case *ast.PrefixExpression:
		if ast.IsLiteral(lit.Right) {
			return s.analyseExpressionType(lit.Right, lit.Right.Type(), targetType)
		}
		// not a literal, so we do exact checking
		if !exprType.Equal(targetType) {
			s.addError(expr, errTypeMismatch(targetType.String(), exprType.String()))
			return false
		}
		return true

	case *ast.InfixExpression:
		leftT := lit.Left.Type()
		rightT := lit.Right.Type()
		// try to coerce left to right if left is a literal
		if ast.IsLiteral(lit.Left) {
			if !s.analyseExpressionType(lit.Left, leftT, rightT) {
				return false
			}
			return true
		}

		// try to coerce right to left if right is a literal
		if ast.IsLiteral(lit.Right) {
			if !s.analyseExpressionType(lit.Right, rightT, leftT) {
				return false
			}
			return true
		}
		if !leftT.Equal(rightT) {
			s.addError(expr, errTypeMismatch(leftT.String(), rightT.String()))
			return false
		}

		return true

	case *ast.Identifier:
		// we want to treat function values are literals
		if _, ok := s.fnSt.Get(lit.TokenLiteral()); ok {
			if !types.CanCoalesce(exprType, targetType) {
				s.addError(expr, errTypeMismatch(targetType.String(), exprType.String()))
				return false
			}
		} else {
			if exprType == nil {
				return false
			}
			if !exprType.Equal(targetType) {
				s.addError(expr, errTypeMismatch(targetType.String(), exprType.String()))
				return false
			}
		}
		return true

	case *ast.IntegerLiteral:
		coercedType := types.GetUnderlyingTypeIfLiteral(targetType)
		switch t := coercedType.(type) {
		case *types.Byte:
			if !types.IntValueFitsIn(lit.Value, &types.ConstU8) {
				s.addError(expr, errIntLiteralOverflows(lit.Value, coercedType.String()))
				return false
			}
		case *types.Int:
			if !types.IntValueFitsIn(lit.Value, t) {
				s.addError(expr, errIntLiteralOverflows(lit.Value, coercedType.String()))
				return false
			}
		case *types.Char:
			if !types.IntValueFitsIn(lit.Value, &types.ConstU32) {
				s.addError(expr, errIntLiteralOverflows(lit.Value, coercedType.String()))
				return false
			}

		default:
			targetSign := types.GetSign(targetType)
			intType := types.LowestFittingInt(lit.Value, targetSign == 1)
			if !types.CanCoalesce(intType, targetType) {
				s.addError(expr, errTypeMismatch(targetType.String(), expr.Type().String()))
				return false
			}
			lit.SetType(intType)
			return true
		}
		lit.SetType(coercedType)
		return true

	case *ast.FloatLiteral:
		coercedType := types.GetUnderlyingTypeIfLiteral(targetType)
		floatType := types.LowestFittingFloat(lit.Value)
		switch t := coercedType.(type) {
		case *types.Float:
			if !types.IsFloatRepresentableAs(lit.Value, t) {
				s.addError(expr, errFloatLiteralNotRepresentable(lit.Value, coercedType.String()))
				return false
			}
		default:
			if !types.CanCoalesce(floatType, targetType) {
				s.addError(expr, errTypeMismatch(targetType.String(), expr.Type().String()))
				return false
			}
		}
		lit.SetType(floatType)
		return true

	case *ast.CharacterLiteral:
		intType := types.LowestFittingInt(int64(lit.Value), false)
		if !types.CanCoalesce(intType, targetType) {
			s.addError(expr, errTypeMismatch(targetType.String(), intType.String()))
			return false
		}
		coercedType := types.GetUnderlyingTypeIfLiteral(targetType)
		lit.SetType(coercedType)
		return true

	// For array literal we need to check all underlying values are of the same type
	// if any are a literal we coalesce the type otherwise we perform strict type check
	case *ast.ArrayLiteral:
		arrayType := types.GetUnderlyingTypeIfLiteral(targetType)
		if mutType, ok := arrayType.(*types.Mutable); ok {
			arrayType = mutType.T
		}
		// lit.T = expectedT
		s.analyseArrayLiteral(lit, arrayType.(*types.Array))

		// TODO:
		return true

	case *ast.StructLiteral:
		// check if coalescing possible
		structType := types.GetUnderlyingTypeIfLiteral(targetType)
		if !types.CanCoalesce(structType, targetType) {
			s.addError(lit, errTypeMismatch(targetType.String(), structType.String()))
			return false
		}
		return true
	case *ast.NullLiteral:
		switch et := targetType.(type) {
		case *types.Optional:
			if !types.CanCoalesce(exprType, targetType) {
				s.addError(expr, errTypeMismatch(targetType.String(), exprType.String()))
				return false
			}
			lit.SetType(et)
			return true
		case *types.Pointer:
			return s.analyseExpressionType(expr, expr.Type(), et.T)
		}
		return false
	case *ast.BooleanLiteral,
		*ast.ByteLiteral, *ast.StringLiteral:
		if !types.CanCoalesce(exprType, targetType) {
			s.addError(expr, errTypeMismatch(targetType.String(), exprType.String()))
			return false
		}
		return true

	default:
		if !exprType.Equal(targetType) {
			s.addError(expr, errTypeMismatch(targetType.String(), exprType.String()))
			return false
		}
		return true
	}
}

// Recursively analyses a given array literal matches the expected type
func (s *Semantics) analyseArrayLiteral(lit *ast.ArrayLiteral, typ *types.Array) {
	switch eleT := typ.T.(type) {
	case *types.Int:
		for _, el := range lit.Values {
			elInt, ok := el.(*ast.IntegerLiteral)
			if ok {
				if !types.IntValueFitsIn(elInt.Value, eleT) {
					s.addError(el, errIntLiteralOverflows(elInt.Value, eleT.String()))
					continue
				}
				elInt.T = eleT
			} else {
				if !typ.T.Equal(el.Type()) {
					s.addError(lit, errTypeMismatch(typ.T.String(), el.Type().String()))
					continue
				}
			}
		}
	case *types.Float:
		panic("add semsis for array of floats")
	case *types.Byte:
		for _, el := range lit.Values {
			elInt, ok := el.(*ast.IntegerLiteral)
			if ok {
				if !types.IntValueFitsIn(elInt.Value, &types.ConstU8) {
					s.addError(el, errIntLiteralOverflows(elInt.Value, eleT.String()))
					continue
				}
				elInt.T = eleT
			} else {
				if !typ.T.Equal(el.Type()) {
					s.addError(lit, errTypeMismatch(typ.T.String(), el.Type().String()))
					continue
				}
			}
		}
	case *types.Array:
		for _, ele := range lit.Values {
			arrayLit, ok := ele.(*ast.ArrayLiteral)
			if !ok {
				s.addError(ele, errTypeMismatch(eleT.String(), ele.Type().String()))
				continue
			}
			s.analyseArrayLiteral(arrayLit, eleT)
		}
	}
	lit.SetType(typ)
}

func (s *Semantics) addError(n ast.Node, err error) {
	s.errors = append(s.errors, &SemanticalError{At: n, Err: err})
}

func (s *Semantics) inferUnknownNamedType(typ types.TypeSpec) types.TypeSpec {
	switch t := typ.(type) {
	case *types.Dirty:
		typ := s.inferUnknownNamedType(t.T)
		if typ == nil {
			return nil
		}
		if typ, ok := typ.(*types.Dirty); ok {
			t = typ
		}
		return t
	case *types.Array:
		typ := s.inferUnknownNamedType(t.T)
		if typ == nil {
			return nil
		}
		t.T = typ
		return t
	case *types.Mutable:
		typ := s.inferUnknownNamedType(t.T)
		if typ == nil {
			return nil
		}
		t.T = typ
		return t
	case *types.Optional:
		typ := s.inferUnknownNamedType(t.T)
		if typ == nil {
			return nil
		}
		t.T = typ
		return t
	case *types.Pointer:
		typ := s.inferUnknownNamedType(t.T)
		if typ == nil {
			return nil
		}
		t.T = typ
		return t
	case *types.Function:
		for i, at := range t.Arg {
			typ := s.inferUnknownNamedType(at)
			if typ == nil {
				return nil
			}
			t.Arg[i] = typ
		}
		for i, rt := range t.Ret {
			typ := s.inferUnknownNamedType(rt)
			if typ == nil {
				return nil
			}
			t.Ret[i] = typ
		}

	case *types.UnknownNamed:
		typeInfo, ok := s.varSt.Get(t.Ident())
		if !ok {
			return nil
		}
		return typeInfo.Type
	case *types.ImportedNamed:
		// resolve imported type
		typ, ok := s.importedSt[t.Lib][t.Typ.Ident()]
		if !ok {
			return nil
		}
		t.Typ = typ
		return t
	case nil:
		return nil
	}
	return typ
}

func (s *Semantics) analyseErrorStructLiteral(n *ast.StructLiteral, errorType *types.Error) {
	namedFields := 0
	for _, f := range n.Fields {
		if f.Name != nil {
			namedFields++
		}
	}

	if namedFields != 0 && namedFields != len(n.Fields) {
		s.addError(n, errErrorMissingFields(errorType.Name))
		return
	}

	fieldMap := make(map[string]bool)
	for _, field := range errorType.Fields {
		fieldMap[field.Name] = false
	}

	for i, f := range n.Fields {
		s.analyse(f.Value, "")
		f.T = f.Value.Type()

		fieldName := f.Name.TokenLiteral()

		var expectedType types.TypeSpec
		found := false
		for _, errorField := range errorType.Fields {
			if errorField.Name == fieldName {
				expectedType = errorField.T
				found = true
				fieldMap[fieldName] = true
				break
			}
		}

		if !found {
			s.addError(n, errErrorUnknownField(errorType.Name, fieldName))
			continue
		}

		if !types.CanCoalesce(f.T, expectedType) {
			s.addError(n, errTypeMismatch(expectedType.String(), f.T.String()))
		}

		n.Fields[i].Index = i
	}

	for fieldName, provided := range fieldMap {
		if !provided {
			s.addError(n, errErrorFieldNotDefined(fieldName))
		}
	}

	n.T = errorType
}

// ------- //
// Helpers //
// ------- //

func getIdentifier(n ast.Node) string {
	var name string
	switch n := n.(type) {
	case *ast.StructStatement:
		name = n.Name.String()
	case *ast.UnionStatement:
		name = n.Name.String()
	case *ast.TypeDefinitionStatement:
		name = n.Name.String()
	case *ast.TypeAliasStatement:
		name = n.Name.String()
	default:
		panic("unknown node")
	}
	return name
}

// Very easy cycle detection alg. Returns on first cycle
func isCyclic(adj [][]uint16) ([]uint16, bool) {
	vertices := len(adj)
	paths := make([][]uint16, vertices)

	for i := range vertices {
		if path, hasCycle := _isCyclic(uint16(i), adj, paths[i]); hasCycle {
			return path, true
		}
	}
	return nil, false
}

func _isCyclic(i uint16, adj [][]uint16, path []uint16) ([]uint16, bool) {
	path = append(path, i)
	for _, j := range adj[i] {
		// early check if cycle to path
		if slices.Contains(path, j) {
			return path, true
		}
		if path, hasCycle := _isCyclic(j, adj, path); hasCycle {
			return path, true
		}
	}

	return nil, false

}

// typesEqual compares two types for equality, handling mutable types properly
func (s *Semantics) typesEqual(expected, actual types.TypeSpec) bool {
	if expected.Equal(actual) {
		return true
	}

	// Special case: any custom error type can be assigned to generic error type
	if expectedErr, ok := expected.(*types.Error); ok && expectedErr.Name == "error" {
		if _, ok := actual.(*types.Error); ok {
			return true
		}
	}

	// as mutable types can be considered equal as their
	// underlying type we need to unwrap first
	expectedUnwrapped := s.unwrapMutable(expected)
	actualUnwrapped := s.unwrapMutable(actual)

	return expectedUnwrapped.Equal(actualUnwrapped)
}

// unwrapMutable unwraps Mutable types to their underlying types
func (s *Semantics) unwrapMutable(t types.TypeSpec) types.TypeSpec {
	if mutable, ok := t.(*types.Mutable); ok {
		return mutable.T
	}
	return t
}

func getTypesFromExpressions(exps []ast.Expression) []types.TypeSpec {
	types := make([]types.TypeSpec, len(exps))
	for i, exp := range exps {
		types[i] = exp.Type()
	}
	return types
}

// validateAppendFunction validates that append() receives correct argument types
func (s *Semantics) validateAppendFunction(n *ast.FunctionCallExpression) bool {
	if len(n.Arguments) < 2 {
		s.addError(n, errTooLittleArguments("append"))
		return false
	} else if len(n.Arguments) > 2 {
		s.addError(n, errTooManyArguments("append"))
		return false
	}

	n.SetType(n.Arguments[0].Type())
	firstArgType := s.coalesceTypeForBuiltIn(n.Arguments[0].Type(), "append")
	secondArgType := s.coalesceTypeForBuiltIn(n.Arguments[1].Type(), "append")

	// validate first argument
	arrayType, ok := firstArgType.(*types.Array)
	if !ok {
		s.addError(n.Arguments[0], errTypeMismatch("[]T", firstArgType.String()))
		return false
	}

	elementType := arrayType.T

	if secondArgArrType, ok := secondArgType.(*types.Array); ok {
		if !s.analyseExpressionType(n.Arguments[1], secondArgArrType, arrayType) {
			return false
		}
	} else {
		if !s.analyseExpressionType(n.Arguments[1], secondArgType, elementType) {
			return false
		}
	}

	n.SetType(n.Arguments[0].Type())

	return true
}
