package semantic

import (
	"fmt"
	"strings"
)

// General errors
var (
	errIdentifierNotFound = func(name string) error {
		return fmt.Errorf("identifier '%s' not found", name)
	}
	errTypeMismatch = func(expectedType, gotType string) error {
		return fmt.Errorf("type mistmatch, expected type '%s' but got '%s'", expectedType, gotType)
	}
	errInvalidBooleanCondition = func(operator string) error {
		return fmt.Errorf("invalid boolean condition used '%s'", operator)
	}
	errDuplicateIdentifierFound = func(name string) error {
		return fmt.Errorf("'%s' already defined within this scope", name)
	}
	errExpressionUnused = func() error {
		return fmt.Errorf("expression is not used")
	}
	errIllegalOperationOnType = func(op, typ string) error {
		return fmt.Errorf("illegal operation '%s' on type '%s'", op, typ)
	}
	errIllegalTypeCast = func(fromType, toType string) error {
		return fmt.Errorf("illegal type cast from '%s' to '%s'", fromType, toType)
	}
)

// Assignment errors
var (
	errIllegalReassign = func() error {
		return fmt.Errorf("illegal reassignment of variable")
	}
	errAssignmentMismatch = func(assigned, variables int) error {
		return fmt.Errorf("assignment mismmatch, assigned %d values to %d variables", assigned, variables)
	}
	errCannotAssignVoidFunction = func() error {
		return fmt.Errorf("cannot assign a void function to a variable")
	}
)

// Function related semantical errors
var (
	errFunctionNotFound = func(name string) error {
		return fmt.Errorf("function '%s' not found", name)
	}
	errTooManyArguments = func(name string) error {
		return fmt.Errorf("too many arguments passed to function '%s'", name)
	}
	errTooLittleArguments = func(name string) error {
		return fmt.Errorf("too little arguments passed to function '%s'", name)
	}
	errMissingReturn = func() error {
		return fmt.Errorf("function is missing return")
	}
	errTooManyReturnValues = func(got, want string) error {
		return fmt.Errorf("too many return values, got (%s) want (%s)", got, want)
	}
	errTooLittleReturnValues = func(got, want string) error {
		return fmt.Errorf("too little return values, got (%s) want (%s)", got, want)
	}
	errInvalidTry = func() error {
		return fmt.Errorf("invalid use of 'try' with a non error-prone function")
	}
	errErrorProneNeedsTry = func(name string) error {
		return fmt.Errorf("error-prone function '%s' must be wrapped in 'try'", name)
	}
	errCannotInferTypeParameter = func(name string) error {
		return fmt.Errorf("cannot infer type parameter '%s'", name)
	}
	errTypeParameterCountMismatch = func(want, got int) error {
		return fmt.Errorf("expected %d type parameters, got %d", want, got)
	}
)

// ---------------- //
// Type declaration //
// ---------------- //

var (
	errTypeNotFound = func(name string) error {
		return fmt.Errorf("type '%s' not found", name)
	}
	errCyclicalTypeDeclarations = func(types []string) error {
		return fmt.Errorf("cyclical type declarations: %s", strings.Join(types, " -> "))
	}
)

// Unions

var (
	// This error should be used when cycle between field in union
	// and union itself is detected
	errRecursiveUnionReference = func(union string) error {
		return fmt.Errorf("type in union '%s' cannot reference itself", union)
	}
	// This error should be used when cycle detected between union
	// definitions
	errCyclicalUnions = func(types []string) error {
		return fmt.Errorf("cyclical reference between unions: %s", strings.Join(types, ", "))
	}
	errDuplicateUnionField = func(field, union string) error {
		return fmt.Errorf("duplicate field '%s' in union '%s'", field, union)
	}
)

// Enum related semantical errors
var (
	errEnumUnknownField = func(name, field string) error {
		return fmt.Errorf("enum '%s' has no field named '%s'", name, field)
	}
)

// Error related
var (
	errErrorFieldNotDefined = func(fieldName string) error {
		return fmt.Errorf("error field '%s' not defined", fieldName)
	}
	errErrorUnknownField = func(name, field string) error {
		return fmt.Errorf("error '%s' has no field named '%s'", name, field)
	}
	errErrorMissingFields = func(structName string) error {
		return fmt.Errorf("error '%s' has missing fields", structName)
	}
)

// Struct definition related semantical errors
var (
	errStructNotFound = func(name string) error {
		return fmt.Errorf("struct '%s' not found", name)
	}
	// errDuplicateStructDefinition = func(struct1, struct2 string) error {
	// 	return fmt.Errorf("duplicate struct definition: %s and %s", struct1, struct2)
	// }
	errRecursiveStructReference = func(field, struct1 string) error {
		return fmt.Errorf("field '%s' in struct '%s' cannot reference itself", field, struct1)
	}
	// errCyclicalStructDefinitions = func() error {
	// 	return errors.New("cyclical struct definitions")
	// }
	errStructMissingFields = func(structName string) error {
		return fmt.Errorf("struct '%s' has missing fields", structName)
	}
	errStructFieldNotDefined = func(fieldName string) error {
		return fmt.Errorf("struct field '%s' not defined", fieldName)
	}
	errMixedNamedUnnamedStruct = func(name string) error {
		return fmt.Errorf("mixed named and unnamed fields in '%s' struct", name)
	}
	errStructUnknownField = func(name, field string) error {
		return fmt.Errorf("struct '%s' has no field named '%s'", name, field)
	}
)

// Struct tyoe alias related semantical errors
var (
	errAliasUsedAsLiteral = func() error {
		return fmt.Errorf("aliases can't be used as literals")
	}
	errStructAliasUnknownField = func(name, field string) error {
		return fmt.Errorf("alias '%s' has no field named '%s'", name, field)
	}
)

// ------- //
// If else //
// ------- //

var (
	errIfElseExpTypeMismatch = func(types string) error {
		return fmt.Errorf("type mismatch in if else expression got (%s)", types)
	}
	errIfElseExpNonExp = func() error {
		return fmt.Errorf("last value in if else expression not an expression")
	}
)

// -------- //
// For loop //
// -------- //

var (
	errKeywordNotLastInstruction = func(name string) error {
		return fmt.Errorf("'%s' not last instruction in block", name)
	}
	errIllegalUseOfKeyword = func(name string) error {
		switch name {
		case "break":
			return fmt.Errorf("'%s' used outside of 'use' or 'for' block", name)
		case "next":
			return fmt.Errorf("'next' used outside of for loop")
		}
		panic("unknown keyword")
	}
	errInvalidAssignment = func() error {
		return fmt.Errorf("invalid assignment: expected <identifier> = <expression>")
	}
)

var (
	errIllegalUpdate = func(name string) error {
		return fmt.Errorf("illegal update of '%s'", name)
	}
)

// ----------- //
// Pointer ops //
// ----------- //

var (
	errIllegalAddressOf = func(reason string) error {
		return fmt.Errorf("illegal 'address of' operation: %s", reason)
	}
	errIllegalValueOf = func(reason string) error {
		return fmt.Errorf("illegal 'value of' operation: %s", reason)
	}
)

// -------------- //
// Bitwise ops    //
// -------------- //

var (
	errIllegalBinaryOpOnNonInteger = func(op string) error {
		return fmt.Errorf("illegal '%s' operation: can only be used with integer types", op)
	}
)

// -------- //
// Literals //
// -------- //

var (
	errUintLiteralOverflows = func(val uint64, t string) error {
		return fmt.Errorf("unisgned integer literal '%d' overflows '%s'", val, t)
	}
	errIntLiteralOverflows = func(val int64, t string) error {
		return fmt.Errorf("integer literal '%d' overflows '%s'", val, t)
	}
	errFloatLiteralNotRepresentable = func(val float64, t string) error {
		return fmt.Errorf("float literal '%g' can't be represented as '%s'", val, t)
	}
)

// Optional type //

var (
	errNullUsedWithNonOptional = func() error {
		return fmt.Errorf("'null' used with non optional type")
	}
	errNullUsedInNullCoalesce = func() error {
		return fmt.Errorf("invalid use of '??' with 'null'")
	}
	errNullCoalesceRHSOptional = func(val string) error {
		return fmt.Errorf("invalid use of '??' with optional value '%s'", val)
	}
	errIllegalForceUnwrap = func() error {
		return fmt.Errorf("illegal force unwrap of non optional type")
	}
)
