// This file contains core methods regarding semantical anaylsis of Dash.
// Files in this directory contain specialised tests.

package semantic

import (
	"fmt"
	"strings"

	"dash-lang.io/src/ast"
	"dash-lang.io/src/internal"
	"dash-lang.io/src/token"
	"dash-lang.io/src/types"
)

// TODO: fix difference build vs analyse vs infer functions
// build => prepares symbol table
// infer => infers type
// analyse => perform semantic analysis

type scope uint

const (
	GLOBAL scope = iota
	FOR
	USE
	ASSIGNMENT
)

type FnInfo struct {
	Type          *types.Function
	IsAnonymousFn bool
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
	filepath string

	// symbol table for functions
	fnSt *internal.StackedSymTab[*FnInfo]
	// symbol table for variables
	varSt *internal.StackedSymTab[*VarInfo]
	// used for escape analysis
	expSt *internal.StackedSymTab[ast.Expression]
	// tracks errors discovered in a library
	errors []*SemanticalError
	// tracks current scope e.g. if we are in a function or for loop
	scope *internal.Stack[scope]
	// Should contain all functions, global variables,
	// types, aliases, enums and errors, that this librarys
	// accesses in other files
	headerCache *internal.Cache[string, types.TypeSpec]
	// current function type being analysed
	fnScope *internal.Stack[*types.Function]
	// when analysing a literal this field can be used
	// to set the type the caller expects
	expectedType *internal.Stack[types.TypeSpec]

	// symbol table for custom types (type defs)
	typeSt *internal.StackedSymTab[types.TypeSpec]
	// tracks whether a given type is guarded so we
	// can mark operations as dirty e.g. for use,
	// copy update and operations with guarded type
	guardedType *internal.Cache[string, struct{}]
}

type SemanticalError struct {
	At  ast.Node
	Err error
}

func New() *Semantics {
	return &Semantics{
		// filepath:    filepath,
		fnSt:        internal.NewStackedSymbolTable[*FnInfo](),
		varSt:       internal.NewStackedSymbolTable[*VarInfo](),
		expSt:       internal.NewStackedSymbolTable[ast.Expression](),
		scope:       internal.NewStack[scope](),
		fnScope:     internal.NewStack[*types.Function](),
		typeSt:      internal.NewStackedSymbolTable[types.TypeSpec](),
		guardedType: internal.NewCache[string, struct{}](),
	}
}

func (s *Semantics) Analyse(lib *ast.Library) {
	s.scope.Push(GLOBAL)
	defer s.scope.Pop()

	s.analyseEnumDefinitions(lib.Enums)
	s.analyseStructDefinitions(lib.Structs)
	s.analyseGenericStructDefinitions(lib.GenericStructs)
	s.analyseTypeDefinitions(lib.TypeDefinitions)
	s.analyseTypeAliases(lib.TypeAliases)
	s.analyseUnionDefinitions(lib.Unions)

	s.resolveAllTypeReferences(lib)

	for _, v := range lib.GlobalVariables {
		s.analyseAssignmentStatement(v)
	}

	for _, fn := range lib.Functions {
		// We dont need to scope the symbol table as all function literals
		// encountered here are global within library
		s.analyseFunctionExpression(fn, "")
	}

	for _, err := range lib.Errors {
		s.typeSt.Set(err.Name.TokenLiteral(), &types.Error{Name: err.Name.TokenLiteral()})
	}

	// We need to analyse type guards after type def and functions
	// analysed as guards might call functions or other types which
	// need to be analysed before hand
	for _, td := range lib.TypeDefinitions {
		if td.Guard != nil {
			// use type def name to set variabe table so that analysis
			// of guard works as guard uses type def name as 'variable'

			// s.varSt.Set(td.Name.TokenLiteral(), &VarInfo{Type: &types.Definition{Name: td.Name.TokenLiteral(), Underlying: td.T}})
			s.analyse(td.Guard, "")
			s.varSt.Clear(td.Name.TokenLiteral())
			s.guardedType.Set(td.Name.Value, struct{}{})
		}
	}
	s.analyse(lib, "")
}

// This is a specialised TEMPORARY function to analyse a sequence of statements
// and expressions to facilitate tests in the evaluator
func (s *Semantics) AnalyseExpressions(n *ast.Evaluator) {
	s.scope.Push(GLOBAL)
	defer s.scope.Pop()

	for _, def := range n.Types {
		s.varSt.Set(def.Name.TokenLiteral(), &VarInfo{Type: def.T})
	}

	// for _, alias := range n.Aliases {
	// 	s.varSt.Set(alias.Name.TokenLiteral(), &VarInfo{Type: alias.T})
	// }

	for _, union := range n.Unions {
		s.varSt.Set(union.Name.TokenLiteral(), &VarInfo{Type: union.T})
	}

	for _, strct := range n.Structs {
		s.varSt.Set(strct.Name.TokenLiteral(), &VarInfo{Type: strct.T})
	}

	for _, enum := range n.Enums {
		s.varSt.Set(enum.Name.TokenLiteral(), &VarInfo{Type: enum.T})
	}
	for _, err := range n.Errors {
		s.typeSt.Set(err.Name.TokenLiteral(), &types.Error{Name: err.Name.TokenLiteral()})
	}

	s.analyseTypeDefinitions(n.Types)
	s.analyseStructDefinitions(n.Structs)
	s.analyseUnionDefinitions(n.Unions)

	for _, n := range n.Nodes {
		s.analyse(n, "")
	}
}

func (s *Semantics) Errors() []string {
	errs := make([]string, len(s.errors))
	for i, semErr := range s.errors {
		errs[i] = semErr.Err.Error()

	}
	return errs
}

// Builds symbol table for function arguments and variables, it performs some basic type inference too
func (s *Semantics) analyse(n ast.Node, name string) {
	switch n := n.(type) {
	case *ast.Library:
		for _, fn := range n.Functions {
			s.analyse(fn, "")
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
					s.analyseLiteralWithType(n.Arguments[0], to)
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
			// mark memory<> as consumed
			// if identifier of type memory make sure to consume
			// memory type and store underlying type in symbol
			// table so that it cant be used later on in the
			// same scope
			for _, arg := range n.Arguments {
				if ident := s.getIdentFromPrefix(arg); ident != nil {
					mt := types.GetUnderlyingMemory(arg.Type())
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
			s.analyseLiteralWithType(n.Argument, n.Typ)
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
			for j, rt := range typs {
				expectedType := fnType.GetReturnTypeAt(i + j)
				switch lit := rv.(type) {
				case ast.Literal:
					s.analyseLiteralWithType(rv, expectedType)
				case *ast.Identifier:
					// we want to treat function values are literals
					if _, ok := s.fnSt.Get(lit.TokenLiteral()); ok {
						if !types.CanCoalesce(rt, expectedType) {
							s.addError(n, errTypeMismatch(expectedType.String(), rt.String()))
							continue
						}
					} else {
						if !expectedType.Equal(rt) {
							s.addError(n, errTypeMismatch(expectedType.String(), rt.String()))
							continue
						}
					}
				default:
					if !expectedType.Equal(rt) {
						s.addError(n, errTypeMismatch(expectedType.String(), rt.String()))
						continue
					}
				}
			}
		}
	case *ast.IfElseExpression:
		for _, cond := range n.Conditionals {
			s.analyse(cond, "")
		}

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
	case *ast.UseExpression:
		s.scope.Push(USE)
		defer s.scope.Pop()

		s.analyse(n.Ident, "")

		identType, ok := n.Ident.Type().(*types.Memory)

		if !ok {
			s.addError(n, errTypeMismatch("memory", n.Ident.T.String()))
			return
		}
		if _, ok := s.guardedType.Get(identType.Ident()); ok {
			n.T = &types.Dirty{T: identType.T}
		} else {
			n.T = identType.T
		}

		s.varSt.Set(n.Ident.TokenLiteral(), &VarInfo{Type: n.T, Reassignable: true})

		// TODO: validate only ident modified in reference (unlike copy and update)

		s.analyse(n.Block, "")

	case *ast.StructLiteral:
		// As we only need this for escape analysis we use the variable name as key
		// If literal returned directly or free floating on a block we ignore
		if name != "" {
			s.expSt.Set(name, n)
		}
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
			// validate struct exists
			structName := n.Name.TokenLiteral()
			typeDef, ok := s.varSt.Get(structName)
			if !ok {
				s.addError(n, errStructNotFound(structName))
			}
			var structType *types.Struct
			switch t := typeDef.Type.(type) {
			case *types.Struct:
				structType = t
			case *types.Definition:
				// for type definitions of structs we still want to be
				// able to validate the usage. This is why we assign the
				// underlying array to 'structType' and perform the validation
				st, ok := t.Underlying.(*types.Struct)
				if !ok {
					s.addError(n, errTypeMismatch("struct", typeDef.Type.String()))
					return
				}
				structType = st
			case *types.AbstractStruct:
				s.addError(n, errAliasUsedAsLiteral())
				return
			default:
				// TODO: improve error here
				s.addError(n, errTypeMismatch("struct", typeDef.Type.String()))
				return

			}
			namedFields := 0
			for _, f := range n.Fields {
				if f.Name != nil {
					namedFields++
				}
			}
			if namedFields != 0 && namedFields != len(n.Fields) {
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
					internal.AssertTrue(err == nil, "no error expected")
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
					s.analyseLiteralWithType(f.Value, fieldType)
				default:
					// verify type of field matches type in struct definition
					if fieldType.String() != f.Type().String() {
						s.addError(n, errTypeMismatch(fieldType.String(), f.Type().String()))
						continue
					}
				}
			}

			switch t := typeDef.Type.(type) {
			case *types.Struct:
				typeDef.Type = structType
				s.varSt.Set(n.Name.TokenLiteral(), typeDef)
				n.T = structType
			// The reason this is split from *types.Struct case is because
			// we want to keep the type definition type the same. The logic
			// above just ensured that the underlying struct definition is
			// used properly by the struct literal.
			case *types.Definition:
				t.Underlying = structType
				s.varSt.Set(n.Name.TokenLiteral(), typeDef)
				n.T = typeDef.Type
			}

			internal.AssertNotType[*types.UnknownNamed](structType, "expected type to not be ast.Named")

		}
	case *ast.ArrayLiteral:
		if len(n.Values) == 0 {
			return
		}

		// As we only store need this for escape analysis we use the variable name as key
		// If literal returned directly or free floating on a block we ignore
		if name != "" {
			s.expSt.Set(name, n)
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
		case *types.Enum:
			switch t := n.Right.(type) {
			case *ast.Identifier:
				t.T = left
			default:
				panic("todo: add semsis error")
			}
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
		}
		n.SetType(n.Right.Type())
	case *ast.IndexExpression:
		s.analyse(n.Left, "")
		switch typ := n.Left.Type().(type) {
		case nil:
			panic("this is a compiler error. please report")
		case *types.Definition:
			s.varSt.Set(n.String(), &VarInfo{Type: typ.Underlying})
		default:
			s.varSt.Set(n.String(), &VarInfo{Type: n.Type()})
		}

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
		if _, ok := s.typeSt.Get(n.Error.TokenLiteral()); !ok {
			s.addError(n, errIdentifierNotFound(n.Error.String()))
			return
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

		_, isUnion := n.Scrutinee.Type().(*types.Union)
		for _, c := range n.Cases {
			s.analyse(c.Predicate, "")
			cT := c.Predicate.Type()
			isLiteral := isLiteral(c.Predicate)
			if !validateMatchCaseType(sT, cT, isLiteral) {
				s.addError(c, errTypeMismatch(sT.String(), cT.String()))
				continue
			}
			// As the types match, we can replace type of predicate.
			// This is required to be able to, for example, match byte
			// with char literals
			if !isUnion {
				c.Predicate.SetType(sT)
			}

			s.varSt.Scope()
			// check required as literals not stored in symbol table
			// BUG: improve check here. only store if in symbol table if
			// identifier, dot expression, index or slice expression.
			switch exp := n.Scrutinee.(type) {
			case *ast.InfixExpression:
				if exp.Operator != "=" {
					break
				}
				_, ok := exp.Left.(*ast.Identifier)
				if !ok {
					panic("todo: add semsis error")
				}

				s.varSt.Set(exp.Left.TokenLiteral(), &VarInfo{Type: cT})

			case *ast.Identifier:
				info, ok := s.varSt.Get(n.Scrutinee.TokenLiteral())
				if !ok {
					s.addError(n.Scrutinee, errIdentifierNotFound(n.Scrutinee.String()))
					return
				}
				s.varSt.Set(n.Scrutinee.TokenLiteral(), &VarInfo{Type: cT, Reassignable: info.Reassignable})
			default:
				// in case of literals and expressions we dont need
				// to do anything else
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
		if left.Type() == nil {
			return
		} else if right.Type() == nil {
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

		leftT := left.Type()
		rightT := right.Type()

		// Validate both types are equal. If one side is a
		// literal we use a slightly more relaxed type checker
		switch left := left.(type) {
		case *ast.IntegerLiteral:
			rhsSign := types.GetSign(rightT)
			if rhsSign < 0 {
				s.addError(n, errTypeMismatch(leftT.String(), rightT.String()))
				return
			}
			intType := types.LowestFittingInt(left.Value, rhsSign == 1)
			if !types.CanCoalesce(intType, rightT) {
				s.addError(n, errTypeMismatch(leftT.String(), rightT.String()))
				return
			}
			leftT = types.GetUnderlyingTypeIfLiteral(rightT)
			left.SetType(leftT)
		case *ast.FloatLiteral:
			floatType := types.LowestFittingFloat(left.Value)
			if !types.CanCoalesce(floatType, rightT) {
				s.addError(n, errTypeMismatch(leftT.String(), rightT.String()))
				return
			}
			leftT = types.GetUnderlyingTypeIfLiteral(rightT)
			left.SetType(leftT)
		case *ast.CharacterLiteral:
			intType := types.LowestFittingInt(int64(left.Value), false)
			if !types.CanCoalesce(intType, rightT) {
				s.addError(n, errTypeMismatch(rightT.String(), intType.String()))
				return
			}
			leftT = types.GetUnderlyingTypeIfLiteral(rightT)
			left.SetType(leftT)
		case *ast.ArrayLiteral, *ast.StructLiteral, *ast.BooleanLiteral,
			*ast.ByteLiteral, *ast.StringLiteral, *ast.NullLiteral:
			if !types.CanCoalesce(leftT, rightT) {
				s.addError(n, errTypeMismatch(leftT.String(), rightT.String()))
				return
			}
		// This means LHS was not literal
		default:
			switch right := right.(type) {
			case *ast.IntegerLiteral:
				lhsSign := types.GetSign(leftT)
				if lhsSign < 0 {
					s.addError(n, errTypeMismatch(leftT.String(), rightT.String()))
					return
				}
				intType := types.LowestFittingInt(right.Value, lhsSign == 1)
				if !types.CanCoalesce(intType, leftT) {
					s.addError(n, errTypeMismatch(leftT.String(), intType.String()))
					return
				}
				rightT = types.GetUnderlyingTypeIfLiteral(leftT)
				right.SetType(rightT)
			case *ast.FloatLiteral:
				floatType := types.LowestFittingFloat(right.Value)
				if !types.CanCoalesce(floatType, leftT) {
					s.addError(n, errTypeMismatch(leftT.String(), rightT.String()))
					return
				}
				rightT = types.GetUnderlyingTypeIfLiteral(leftT)
				right.SetType(rightT)
			case *ast.CharacterLiteral:
				intType := types.LowestFittingInt(int64(right.Value), false)
				if !types.CanCoalesce(intType, leftT) {
					s.addError(n, errTypeMismatch(leftT.String(), intType.String()))
					return
				}
				rightT = types.GetUnderlyingTypeIfLiteral(leftT)
				right.SetType(rightT)
			case *ast.ArrayLiteral, *ast.StructLiteral, *ast.BooleanLiteral,
				*ast.ByteLiteral, *ast.StringLiteral, *ast.NullLiteral:
				if !types.CanCoalesce(rightT, leftT) {
					s.addError(n, errTypeMismatch(leftT.String(), rightT.String()))
					return
				}
				// This means neither side were a literal
			default:
				if !leftT.Equal(rightT) {
					s.addError(n, errTypeMismatch(leftT.String(), rightT.String()))
					return
				}
			}
		}

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
			// NOTE: technically this branch should not run anymore
			// with new ast.AssignmentStatement

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
		s.addError(n, errIdentifierNotFound(n.TokenLiteral()))
	case *ast.BlockStatement:
		// enter new scope
		s.varSt.Scope()
		defer s.varSt.Unscope()

		// case: 1 we are in a for loop
		scope := s.scope.GetLast()
		for i, st := range n.Statements {
			switch st := st.(type) {
			case *ast.KeywordStatement:
				if scope == USE {
					if st.Token.Type != token.BREAK {
						s.addError(st, errIllegalUseOfKeyword(st.TokenLiteral()))
						continue
					}
				} else if scope == FOR {
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
			panic("type is nil")
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
				s.analyseLiteralWithType(arg, expectedType)
			default:
				if !expectedType.Equal(arg.Type()) {
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
	for i := 0; i < len(n.Values); i++ {
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

			for j, rt := range t.Ts {
				n.SetTypeAt(i+j, rt)
				underlying := types.GetUnderlyingType(rt)
				if fnValType, ok := underlying.(*types.Function); ok {
					f := &FnInfo{
						Type: fnValType,
					}
					s.fnSt.Set(n.VarNameAt(i+j), f)
				} else {
					s.setDeclerationInSymTab(n.VarNameAt(i+j), rt, n.IsVarAt(i+j))
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
				if _, ok := val.(ast.Literal); ok {
					s.analyseLiteralWithType(val, exp.Type())
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

func (s *Semantics) setDeclerationInSymTab(n string, t types.TypeSpec, isVar bool) {
	vi := &VarInfo{
		Type:         t,
		Reassignable: isVar,
	}
	s.varSt.Set(n, vi)
}

// Checks whether an expression matches the expectedType, for literals we perform type
// coercion before checking
func (s *Semantics) analyseLiteralWithType(arg ast.Expression, expectedType types.TypeSpec) {
	switch lit := arg.(type) {
	case *ast.NullLiteral:
		switch et := expectedType.(type) {
		case *types.Optional:
			lit.SetType(et)
		case *types.Pointer:
			s.analyseLiteralWithType(arg, et.T)
		default:
			panic("semsis error, passing null to non optional type")
		}

	// For array literal we need to check all underlying values are of the same type
	// if any are a literal we coalesce the type otherwise we perform strict type check
	case *ast.ArrayLiteral:
		arrayType, ok := types.GetUnderlyingTypeIfLiteral(expectedType).(*types.Array)
		if !ok {
			s.addError(lit, errTypeMismatch(expectedType.String(), lit.T.String()))
			return
		}
		// lit.T = expectedT
		s.analyseArrayLiteral(lit, arrayType)

	// TODO: do the same for float
	// If we have integer literal we adapt type of literal so it fits
	// argument type, but only if it doesnt overflow
	case *ast.IntegerLiteral:
		expIntType := types.GetUnderlyingTypeIfLiteral(expectedType)
		switch t := expIntType.(type) {
		case *types.Byte:
			if !types.IntValueFitsIn(lit.Value, &types.ConstU8) {
				s.addError(arg, errIntLiteralOverflows(lit.Value, expIntType.String()))
				return
			}
		case *types.Int:
			if !types.IntValueFitsIn(lit.Value, t) {
				s.addError(arg, errIntLiteralOverflows(lit.Value, expIntType.String()))
				return
			}
		default:
			s.addError(arg, errTypeMismatch(expectedType.String(), lit.T.String()))
			return
		}
		lit.T = expIntType
		arg.SetType(expIntType)
	case *ast.FloatLiteral:
		expFloatType, ok := types.GetUnderlyingTypeIfLiteral(expectedType).(*types.Float)
		if !ok {
			s.addError(arg, errTypeMismatch(expectedType.String(), lit.T.String()))
			return
		}
		if !types.IsFloatRepresentableAs(lit.Value, expFloatType) {
			s.addError(arg, errFloatLiteralNotRepresentable(lit.Value, expFloatType.String()))
			return
		}
		lit.T = expFloatType
		arg.SetType(expFloatType)
		// case *ast.BooleanLiteral:
	case *ast.StructLiteral:
		structType, ok := types.GetUnderlyingTypeIfLiteral(expectedType).(*types.Struct)
		if !ok {
			s.addError(lit, errTypeMismatch(expectedType.String(), lit.T.String()))
			return
		}
		for i, field := range lit.Fields {
			var expectedFieldType types.TypeSpec
			var err error
			if field.Name.Value == "" {
				expectedFieldType, err = structType.GetTypeByIndex(i)
			} else {
				expectedFieldType, _, err = structType.GetTypeByField(field.Name.Value)
			}
			if err != nil {
				s.addError(lit, errTypeMismatch(expectedFieldType.String(), field.T.String()))
				return
			}
			s.analyseLiteralWithType(field, expectedFieldType)
		}
	}
	return
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
	default:
		return t
	}
}

func (s *Semantics) analyseEnumDefinitions(enums []*ast.EnumStatement) {
	for _, enum := range enums {
		s.varSt.Set(enum.Name.TokenLiteral(), &VarInfo{Type: enum.T})
	}
}

// Performs the following validations for struct definitions:
// - no duplicates
// - no cycles
// - no recursion in struct definition (except if field is of optional type)
// - infer any missing types (e.g. unknown_named_type)
func (s *Semantics) analyseStructDefinitions(sds []*ast.StructStatement) {
	m := make(map[string]uint16, len(sds))

	// check for duplicates
	for i, stmt := range sds {
		if _, ok := m[stmt.Name.TokenLiteral()]; ok {
			s.addError(stmt, errDuplicateIdentifierFound(stmt.Name.TokenLiteral()))
		} else {
			m[stmt.Name.TokenLiteral()] = uint16(i)
		}
	}

	// infer types, note some types will remain 'unknown_named'
	for _, sd := range sds {
		strct := &types.Struct{Name: sd.Name.TokenLiteral()}
		for _, f := range sd.Fields {
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
		sd.T = strct
		s.varSt.Set(sd.Name.TokenLiteral(), &VarInfo{Type: sd.Type()})
	}

	// build adjecency list
	adj := make([][]uint16, len(sds))
	for i, sd := range sds {
		// For each field mark if it references another struct in library
		for _, field := range sd.Fields {
			fieldTypeIdent := field.Type.Ident()

			j, ok := m[fieldTypeIdent]
			if !ok {
				continue
			}

			// if field references self then only allowed as optional type
			if uint16(i) == j {
				switch field.Type.(type) {
				case *types.Optional:
					// empty case to capture allowed self references
				default:
					s.addError(field, errRecursiveStructReference(field.Name.TokenLiteral(), sd.Name.TokenLiteral()))
				}
			} else {
				adj[i] = append(adj[i], j)
			}

		}
	}

	// check for cycles
	if path, hasCycle := isCyclic(adj); hasCycle {
		names := make([]string, len(path))
		for i, idx := range path {
			names[i] = sds[idx].Name.TokenLiteral()
		}
		s.addError(sds[path[0]], errCyclicalTypeDeclarations(names))
	}
}

func (s *Semantics) analyseGenericStructDefinitions(gss []*ast.GenericStructStatement) {
	m := make(map[string]uint16, len(gss))

	// check for duplicates
	for i, stmt := range gss {
		if _, ok := m[stmt.Name.TokenLiteral()]; ok {
			s.addError(stmt, errDuplicateIdentifierFound(stmt.Name.TokenLiteral()))
		} else {
			m[stmt.Name.TokenLiteral()] = uint16(i)
		}
	}

	// infer types, note some types will remain 'unknown_named'
	for _, gs := range gss {
		strct := &types.AbstractStruct{Name: gs.Name.TokenLiteral()}
		for _, f := range gs.Fields {
			if f.Name != nil {
				field := types.StructField{Name: f.Name.TokenLiteral(), T: f.Type}
				strct.Ts = append(strct.Ts, field)
			} else {
				field := types.StructField{T: f.Type}
				strct.Ts = append(strct.Ts, field)
			}
		}
		gs.T = strct
		s.varSt.Set(gs.Name.TokenLiteral(), &VarInfo{Type: gs.Type()})
	}

	// build adjecency list
	adj := make([][]uint16, len(gss))
	for i, gs := range gss {
		for _, field := range gs.Fields {
			fieldTypeIdent := field.Type.Ident()

			j, ok := m[fieldTypeIdent]
			if !ok {
				continue
			}

			// if field references self then only allowed as optional type
			if uint16(i) == j {
				switch field.Type.(type) {
				case *types.Optional:
					// empty case to capture allowed self references
				default:
					s.addError(field, errRecursiveStructReference(field.Name.TokenLiteral(), gs.Name.TokenLiteral()))
				}
			} else {
				adj[i] = append(adj[i], j)
			}
		}
	}

	// check for cycles
	if path, hasCycle := isCyclic(adj); hasCycle {
		names := make([]string, len(path))
		for i, idx := range path {
			names[i] = gss[idx].Name.TokenLiteral()
		}
		s.addError(gss[path[0]], errCyclicalTypeDeclarations(names))
	}

	//  set type and infer any missing unknown_name types
	for _, gs := range gss {
		strct := &types.AbstractStruct{Name: gs.Name.TokenLiteral()}
		for _, field := range gs.Fields {
			typ := s.inferUnknownNamedType(field.Type)
			if typ == nil {
				s.addError(field, errTypeNotFound(field.Type.String()))
				continue
			}
			field.Type = typ
			if field.Name != nil {
				strct.Ts = append(strct.Ts, types.StructField{Name: field.Name.TokenLiteral(), T: field.Type})
			} else {
				strct.Ts = append(strct.Ts, types.StructField{T: field.Type})
			}
		}
		gs.T = strct
		s.varSt.Set(gs.Name.TokenLiteral(), &VarInfo{Type: gs.Type()})
	}
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
	case *types.Memory:
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
	case nil:
		return nil
	}
	return typ
}

// TODO: add cycle detection
// Performs the following validations for all type definitions:
// - no duplicates
func (s *Semantics) analyseTypeDefinitions(tds []*ast.TypeDefinitionStatement) {
	m := make(map[string]uint16, len(tds))

	// check for duplicates
	for i, stmt := range tds {
		if _, ok := m[stmt.Name.TokenLiteral()]; ok {
			s.addError(stmt, errDuplicateIdentifierFound(stmt.Name.TokenLiteral()))
		} else {
			m[stmt.Name.TokenLiteral()] = uint16(i)
		}
	}

	//  set type in symbol table
	for _, td := range tds {
		// ensure underlying type not unknown
		td.UnderlyingType = s.inferUnknownNamedType(td.UnderlyingType)

		// define cast function, if type has a predicate
		// cast will cause return type to be 'dirty'
		if td.Guard != nil {
			// set type as dirty type def
			td.T = &types.Dirty{T: &types.Definition{Name: td.Name.TokenLiteral(), Underlying: td.UnderlyingType}}
		} else {
			// set type as type def
			td.T = &types.Definition{Name: td.Name.TokenLiteral(), Underlying: td.UnderlyingType}
		}
		s.varSt.Set(td.Name.TokenLiteral(), &VarInfo{Type: td.Type()})
		s.typeSt.Set(td.Name.TokenLiteral(), td.Type())
	}
}

// TODO: add cycle check
// Performs the following validations for all type aliases:
// - no duplicates
func (s *Semantics) analyseTypeAliases(tas []*ast.TypeAliasStatement) {
	m := make(map[string]uint16, len(tas))

	// check for duplicates
	for i, stmt := range tas {
		if _, ok := m[stmt.Name.TokenLiteral()]; ok {
			s.addError(stmt, errDuplicateIdentifierFound(stmt.Name.TokenLiteral()))
		} else {
			m[stmt.Name.TokenLiteral()] = uint16(i)
		}
	}

	// set symbol table for type aliases
	for _, ta := range tas {
		ta.T = ta.UnderlyingType
		s.varSt.Set(ta.Name.TokenLiteral(), &VarInfo{Type: ta.Type()})
	}

	//  set type in symbol table
	for _, ta := range tas {
		// in case we have a nested type def e.g. type abc xyz
		var typ types.TypeSpec
		if t, ok := ta.UnderlyingType.(*types.UnknownNamed); ok {
			val, ok := s.varSt.Get(t.Ident())
			if !ok {
				panic("unknown type: ")
			}

			// ensure underlying type not unknown named anymore
			ta.UnderlyingType = val.Type
			typ = val.Type
		} else {
			typ = ta.UnderlyingType
		}
		// set type as type alias
		ta.T = typ

		s.varSt.Set(ta.Name.TokenLiteral(), &VarInfo{Type: ta.Type()})
		s.typeSt.Set(ta.Name.TokenLiteral(), ta.Type())
	}
}

func (s *Semantics) analyseUnionDefinitions(uns []*ast.UnionStatement) {
	for _, u := range uns {
		s.varSt.Set(u.Name.TokenLiteral(), &VarInfo{Type: u.T})
	}

	m := make(map[string]uint16, len(uns))

	// check for duplicate union defs
	for i, stmt := range uns {
		if _, ok := m[stmt.Name.TokenLiteral()]; ok {
			s.addError(stmt, errDuplicateIdentifierFound(stmt.Name.TokenLiteral()))
		} else {
			m[stmt.Name.TokenLiteral()] = uint16(i)
		}
	}

	// Build adjencency table
	adj := make([][]uint16, len(uns))
	for i, un := range uns {
		for _, typ := range un.Types {
			fieldTypeIdent := typ.T.Ident()

			j, ok := m[fieldTypeIdent]
			if !ok {
				continue
			}

			// no recursion allowed in unions
			if uint16(i) == j {
				s.addError(typ, errCyclicalUnionField(un.Name.TokenLiteral()))
			} else {
				adj[i] = append(adj[i], j)
			}
		}
	}

	// check for cycles
	if path, hasCycle := isCyclic(adj); hasCycle {
		names := make([]string, len(path))
		for i, idx := range path {
			names[i] = uns[idx].Name.TokenLiteral()
		}
		s.addError(uns[path[0]], errCyclicalUnions(names))
	}

	// set symbol table for unions, as unions can be defined
	// out of order
	for _, un := range uns {
		s.varSt.Set(un.Name.TokenLiteral(), &VarInfo{Type: un.T})
	}

	//  set type in symbol table
	for i, un := range uns {
		typs := make([]types.TypeSpec, len(un.Types))
		for j, typ := range un.Types {
			// if recursive def we already added error
			// and skip any further checks
			if j, ok := m[typ.T.Ident()]; ok && uint16(i) == j {
				continue
			}
			if _, ok := typ.T.(*types.UnknownNamed); ok {
				val, ok := s.varSt.Get(typ.T.Ident())
				if !ok {
					s.addError(typ, errIdentifierNotFound(typ.String()))
					continue
				}
				internal.AssertNotNil(val.Type, "expected type to be set if stored in symbol table")

				typ.T = val.Type
				typs[j] = val.Type
			}
			typs[j] = typ.T
		}

		un.T = &types.Union{Name: un.Name.TokenLiteral(), Ts: typs}

		s.varSt.Set(un.Name.TokenLiteral(), &VarInfo{Type: un.T})
		s.typeSt.Set(un.Name.TokenLiteral(), un.T)
	}

	// check for duplicates within union fields
	for _, un := range uns {
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

func (s *Semantics) resolveAllTypeReferences(lib *ast.Library) {
	for _, sd := range lib.Structs {
		for i, field := range sd.Fields {
			typ := s.inferUnknownNamedType(field.Type)
			if typ == nil {
				s.addError(field, errTypeNotFound(field.Type.String()))
				continue
			}
			sd.T.Ts[i].T = typ
			field.Type = typ
		}

		s.varSt.Set(sd.Name.TokenLiteral(), &VarInfo{Type: sd.Type()})
		s.typeSt.Set(sd.Name.TokenLiteral(), sd.Type())

	}

	for _, gs := range lib.GenericStructs {
		for i, field := range gs.Fields {
			typ := s.inferUnknownNamedType(field.Type)
			if typ == nil {
				s.addError(field, errTypeNotFound(field.Type.String()))
				continue
			}
			gs.T.Ts[i].T = typ
			field.Type = typ
		}
	}

	for _, td := range lib.TypeDefinitions {
		typ := s.inferUnknownNamedType(td.UnderlyingType)
		if typ == nil {
			s.addError(td, errTypeNotFound(td.UnderlyingType.String()))
			continue
		}
		td.UnderlyingType = typ

		if td.Guard != nil {
			td.T = &types.Dirty{T: &types.Definition{Name: td.Name.TokenLiteral(), Underlying: td.UnderlyingType}}
		} else {
			td.T = &types.Definition{Name: td.Name.TokenLiteral(), Underlying: td.UnderlyingType}
		}

		// Update symbol table with resolved type
		s.varSt.Set(td.Name.TokenLiteral(), &VarInfo{Type: td.Type()})
		s.typeSt.Set(td.Name.TokenLiteral(), td.Type())
	}

	for _, ta := range lib.TypeAliases {
		typ := s.inferUnknownNamedType(ta.UnderlyingType)
		if typ == nil {
			s.addError(ta, errTypeNotFound(ta.UnderlyingType.String()))
			continue
		}
		ta.UnderlyingType = typ
		ta.T = typ

		s.varSt.Set(ta.Name.TokenLiteral(), &VarInfo{Type: ta.Type()})
		s.typeSt.Set(ta.Name.TokenLiteral(), ta.Type())
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

func isLiteral(exp ast.Expression) bool {
	if _, ok := exp.(ast.Literal); ok {
		return ok
	}
	if pExp, ok := exp.(*ast.PrefixExpression); ok {
		return isLiteral(pExp.Right)
	}
	return false
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
	for i := 0; i < len(n.Arguments); i++ {
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
		s.varSt.Set(arg.Name.TokenLiteral(), &VarInfo{Type: arg.Type})
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

func (s *Semantics) addError(n ast.Node, err error) {
	pos := n.Pos()
	fmt.Printf("[ERROR] Semsis failed in %s at %d:%d - %s\n", s.filepath, pos.Line(), pos.Column(), err)
	s.errors = append(s.errors, &SemanticalError{At: n, Err: err})
}

// ------- //
// Helpers //
// ------- //

// Very easy cycle detection alg. Returns on first cycle
func isCyclic(adj [][]uint16) ([]uint16, bool) {
	vertices := len(adj)
	paths := make([][]uint16, vertices)

	for i := 0; i < vertices; i++ {
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
		for _, pIdx := range path {
			if j == pIdx {
				return path, true
			}
		}
		if path, hasCycle := _isCyclic(j, adj, path); hasCycle {
			return path, true
		}
	}

	return nil, false

}

func getTypesFromExpressions(exps []ast.Expression) []types.TypeSpec {
	types := make([]types.TypeSpec, len(exps))
	for i, exp := range exps {
		types[i] = exp.Type()
	}
	return types
}
