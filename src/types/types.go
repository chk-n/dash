package types

import (
	"bytes"
	"errors"
	"fmt"
	"math"

	"strings"

	"dash-lang.io/src/token"
)

var (
	errUnknownStructField = errors.New("unknown struct field")
)

type TypeSpec interface {
	String() string
	// returns name of type. Similar to String() except doesnt
	// return "?", "named_type<T>"
	// For dirty types it returns the underlying type as string
	Ident() string
	// Checks whether RHS is equal to LHS.
	// Examples:
	// []T == []i64 => TRUE
	// []i64 == []T => FALSE
	// i64 == i32   => FALSE
	Equal(other TypeSpec) bool
}

// --------- //
// Constants //
// --------- //

var (
	ConstU8   = Int{Width: 8, Signed: 0}
	ConstU16  = Int{Width: 16, Signed: 0}
	ConstU32  = Int{Width: 32, Signed: 0}
	ConstU64  = Int{Width: 64, Signed: 0}
	ConstU128 = Int{Width: 128, Signed: 0}
	ConstU256 = Int{Width: 256, Signed: 0}
	// ConstUint = Int{Width: 1, Signed: 0}

	ConstI8   = Int{Width: 8, Signed: 1}
	ConstI16  = Int{Width: 16, Signed: 1}
	ConstI32  = Int{Width: 32, Signed: 1}
	ConstI64  = Int{Width: 64, Signed: 1}
	ConstI128 = Int{Width: 128, Signed: 1}
	ConstI256 = Int{Width: 256, Signed: 1}
	// ConstInt  = Int{Width: 1, Signed: 1}

	// ConstFloat  = Basic{t: Float}
	ConstF32    = Float{Width: 32}
	ConstF64    = Float{Width: 64}
	ConstBool   = Bool{}
	ConstByte   = Byte{}
	ConstChar   = Char{}
	ConstString = String{}
	ConstNull   = Null{}
)

// ------------ //
// Scalar types //
// ------------ //

var intTypeToString = map[int]string{
	// 1:       "int",
	8:       "u8",
	16:      "u16",
	32:      "u32",
	64:      "u64",
	128:     "u128",
	256:     "u256",
	8 + 1:   "i8",
	16 + 1:  "i16",
	32 + 1:  "i32",
	64 + 1:  "i64",
	128 + 1: "i128",
	256 + 1: "i256",
}

type Int struct {
	Width  int
	Signed int
}

func (t *Int) Ident() string { return t.String() }
func (t *Int) Equal(other TypeSpec) bool {
	o, ok := other.(*Int)
	if !ok {
		return false
	}
	return t.Width == o.Width && t.Signed == o.Signed
}
func (t *Int) Type() TypeSpec {
	return t
}

func (t *Int) String() string {
	return intTypeToString[t.Width+t.Signed]
}

var floatTypeToString = map[int]string{
	// 1:       "float",
	32: "f32",
	64: "f64",
}

type Float struct {
	Width int
}

func (t *Float) Ident() string { return t.String() }
func (t *Float) Equal(other TypeSpec) bool {
	o, ok := other.(*Float)
	if !ok {
		return false
	}
	return t.Width == o.Width
}
func (t *Float) Type() TypeSpec {
	return t
}

func (t *Float) String() string {
	return floatTypeToString[t.Width]
}

type Bool struct{}

func (t *Bool) Ident() string { return t.String() }
func (t *Bool) Equal(other TypeSpec) bool {
	_, ok := other.(*Bool)
	return ok
}
func (t *Bool) Type() TypeSpec {
	return t
}

func (t *Bool) String() string {
	return "bool"
}

type Byte struct{}

func (t *Byte) Ident() string { return t.String() }
func (t *Byte) Equal(other TypeSpec) bool {
	_, ok := other.(*Byte)
	return ok
}
func (t *Byte) Type() TypeSpec {
	return t
}
func (t *Byte) String() string {
	return "byte"
}

type Char struct{}

func (t *Char) Ident() string { return t.String() }
func (t *Char) Equal(other TypeSpec) bool {
	_, ok := other.(*Char)
	return ok
}
func (t *Char) Type() TypeSpec {
	return t
}
func (t *Char) String() string {
	return "char"

}

type String struct{}

func (t *String) Ident() string { return t.String() }
func (t *String) Equal(other TypeSpec) bool {
	_, ok := other.(*String)
	return ok
}
func (t *String) Type() TypeSpec {
	return t
}

func (t *String) String() string {
	return "string"
}

type Null struct{}

func (t *Null) Ident() string { return t.String() }
func (t *Null) Equal(other TypeSpec) bool {
	_, ok := other.(*Null)
	return ok
}
func (t *Null) Type() TypeSpec {
	return t
}

func (t *Null) String() string {
	return "null"
}

// --------------- //
// Aggregate types //
// --------------- //

type Array struct {
	T TypeSpec
	// if set then array is fixed-size
	Size int
}

func (t *Array) Type() TypeSpec         { return t }
func (t *Array) InternalType() TypeSpec { return t.T }

// Returns underlying type in nested array
func (t *Array) ValueType() TypeSpec {
	vTyp, ok := t.T.(*Array)
	if !ok {
		return t.T
	}

	return vTyp.ValueType()
}
func (t *Array) Ident() string { return t.String() }
func (t *Array) String() string {
	var out bytes.Buffer
	out.WriteString("[")
	if t.Size != 0 {
		out.WriteString(fmt.Sprintf("%d", t.Size))
	}
	out.WriteString("]" + t.T.String())
	return out.String()
}
func (t *Array) Equal(other TypeSpec) bool {
	otherArr, ok := other.(*Array)
	if !ok {
		return false
	}
	if t.Size != otherArr.Size {
		return false
	}
	return t.T.Equal(otherArr.T)
}

type Struct struct {
	Name string
	Ts   []StructField
}

func (t *Struct) Type() TypeSpec { return t }
func (t *Struct) GetType(i int) TypeSpec {
	return t.Ts[i].T
}
func (t *Struct) GetTypeByIndex(idx int) (TypeSpec, error) {
	if idx >= 0 && idx <= len(t.Ts) {
		return t.Ts[idx].T, nil
	}
	return nil, errors.Join(errUnknownStructField, fmt.Errorf("%d", idx))
}
func (t *Struct) GetTypeByField(f string) (TypeSpec, int, error) {
	for i, nt := range t.Ts {
		if nt.Name == f {
			return nt.T, i, nil
		}
	}
	return nil, 0, errors.Join(errUnknownStructField, fmt.Errorf("%s", f))
}
func (t *Struct) SetTypeByField(f string, typ TypeSpec) error {
	for _, nt := range t.Ts {
		if nt.Name == f {
			nt.T = typ
			return nil
		}
	}
	return errors.Join(errUnknownStructField, fmt.Errorf("%s", f))
}
func (t *Struct) Ident() string { return t.String() }
func (t *Struct) String() string {
	// case: anonymous struct
	if t.Name == "" {
		var out bytes.Buffer
		out.WriteString("struct<")
		for i, typ := range t.Ts {
			if typ.Name != "" {
				out.WriteString(typ.Name + " " + typ.T.String())
			} else {
				out.WriteString(typ.T.String())
			}
			if i != len(t.Ts)-1 {
				out.WriteString(", ")
			}
		}
		out.WriteString(">")
		return out.String()
	}

	return t.Name
}
func (t *Struct) Equal(other TypeSpec) bool {
	o, ok := other.(*Struct)
	if !ok {
		return false
	}
	if t.Name != o.Name {
		return false
		// TODO: fix to account for optional types
	} else if len(t.Ts) != len(o.Ts) {
		return false
	}
	for i, typ := range t.Ts {
		// TODO: fix to account for assigning unnamed types
		if !typ.Equal(o.Ts[i]) {
			return false
		}
	}
	return true

}

type AbstractStruct struct {
	Name string
	Ts   []StructField
}

func (t *AbstractStruct) Ident() string { return t.String() }
func (t *AbstractStruct) String() string {
	return t.Name
}
func (t *AbstractStruct) GetTypeByField(f string) (TypeSpec, int, error) {
	for i, nt := range t.Ts {
		if nt.Name == f {
			return nt.T, i, nil
		}
	}
	return nil, 0, errors.Join(errUnknownStructField, fmt.Errorf("%s", f))
}
func (t *AbstractStruct) Equal(other TypeSpec) bool {
	switch o := other.(type) {
	case *Struct:
		// For a struct to match abstract struct it needs to
		// have the same field name and type for all fields in
		// abstract struct.
		for _, asf := range t.Ts {
			sfType, _, err := o.GetTypeByField(asf.Name)
			if err != nil {
				return false
			}

			if !asf.T.Equal(sfType) {
				return false
			}
		}
	case *AbstractStruct:
		if len(t.Ts) != len(o.Ts) {
			return false
		}
		// TODO: implement rest
		panic("NOT IMPLEMENTED")
	default:
		return false
	}
	return true
}

type StructField struct {
	Name string
	T    TypeSpec
}

func (sf *StructField) Equal(other StructField) bool {
	// optimistic pointer comparison, prevents
	// infinite recursion if struct recursively
	// contains itself
	if sf.T == other.T {
		return true
	}
	return sf.Name == other.Name && sf.T.Equal(other.T)
}

// ----- //
// Other //
// ----- //

type Enum struct {
	Name string
	Size int
}

func (t *Enum) Type() TypeSpec { return t }
func (t *Enum) Ident() string  { return t.Name }
func (t *Enum) String() string {
	return t.Name
}
func (t *Enum) Equal(other TypeSpec) bool {
	otherEnum, ok := other.(*Enum)
	if !ok {
		return false
	}
	// Size does not have to match as type
	// only consists of name
	return t.Name == otherEnum.Name
}

type Union struct {
	Name string
	Ts   []TypeSpec
}

func (t *Union) Type() TypeSpec { return t }
func (t *Union) Ident() string  { return t.Name }
func (t *Union) String() string {
	return t.Name
}
func (t *Union) Equal(other TypeSpec) bool {
	o, ok := other.(*Union)
	if !ok {
		return false
	}
	return t.Name == o.Name
}

type Function struct {
	Arg          []TypeSpec
	Ret          []TypeSpec
	IsErrorProne bool
	IsVariadic   bool
}

func (t *Function) Type() TypeSpec { return t }
func (t *Function) GetArgumentTypeAt(i int) TypeSpec {
	if i < len(t.Arg) {
		return t.Arg[i]
	}
	panic("GetArgumentTypeAt out of bounds for function")
}
func (t *Function) GetReturnTypeAt(i int) TypeSpec {
	if i < len(t.Ret) {
		return t.Ret[i]
	}
	panic("GetReturnTypeAt out of bounds for function")
}
func (t *Function) ArgumentTypes() []TypeSpec { return t.Arg }
func (t *Function) ReturnTypes() []TypeSpec   { return t.Ret }
func (t *Function) ReturnTypesString() []string {
	typs := make([]string, len(t.Ret))
	for i, typ := range t.Ret {
		typs[i] = typ.String()
	}
	return typs
}
func (t *Function) IsOptional() bool { return false }
func (t *Function) Ident() string    { return t.String() }
func (t *Function) String() string {
	var out bytes.Buffer
	out.WriteString("fn(")
	args := make([]string, 0, len(t.Arg))
	for _, arg := range t.Arg {
		args = append(args, arg.String())
	}
	out.WriteString(strings.Join(args, ","))
	out.WriteString(")")

	if t.IsErrorProne {
		out.WriteString("!")
	}

	if len(t.Ret) > 0 {
		rets := make([]string, 0, len(t.Ret))
		for _, ret := range t.Ret {
			rets = append(rets, ret.String())
		}
		out.WriteString(strings.Join(rets, ","))
	}
	return out.String()
}
func (t *Function) Equal(other TypeSpec) bool {
	otherFn, ok := other.(*Function)
	if !ok {
		return false
	}
	if t.IsVariadic != otherFn.IsVariadic {
		return false
	} else if len(t.Arg) != len(otherFn.Arg) {
		return false
	} else if len(t.Ret) != len(otherFn.Ret) {
		return false
	}

	for i, typ := range t.Arg {
		if !typ.Equal(otherFn.Arg[i]) {
			return false
		}
	}
	for i, typ := range t.Ret {
		if !typ.Equal(otherFn.Ret[i]) {
			return false
		}
	}
	return true
}

type Pointer struct {
	T TypeSpec
}

func (t *Pointer) Type() TypeSpec { return t }
func (t *Pointer) Ident() string  { return t.T.String() }
func (t *Pointer) String() string { return "*" + t.T.String() }
func (t *Pointer) Equal(other TypeSpec) bool {
	otherPtr, ok := other.(*Pointer)
	if !ok {
		return false
	}
	return t.T.Equal(otherPtr.T)
}

type Error struct {
	Name string
}

func (t *Error) Type() TypeSpec { return t }
func (t *Error) Ident() string  { return t.Name }
func (t *Error) String() string {
	return t.Name
}
func (t *Error) Equal(other TypeSpec) bool {
	otherErr, ok := other.(*Error)
	if !ok {
		return false
	}
	return t.Name == otherErr.Name
}

type Generic struct {
	Name string // e.g. T
	// TODO: add contraint
	Constraints []TypeSpec
}

func (t *Generic) Type() TypeSpec { return t }
func (t *Generic) Ident() string  { return t.Name }
func (t *Generic) String() string {
	var out bytes.Buffer

	out.WriteString(t.Name)
	if len(t.Constraints) != 0 {
		out.WriteString(" | ")
	}
	for i, cnstr := range t.Constraints {
		out.WriteString(cnstr.String())
		if i != len(t.Constraints)-1 {
			out.WriteString(", ")
		}
	}
	return out.String()
}
func (t *Generic) Equal(other TypeSpec) bool {
	if len(t.Constraints) == 0 {
		return true
	}
	// return true if one constraint matches
	for _, cnstr := range t.Constraints {
		if cnstr.Equal(other) {
			return true
		}
	}
	return false
}

type Optional struct {
	T TypeSpec
}

func (t *Optional) Type() TypeSpec { return t }
func (t *Optional) Ident() string {
	return t.T.Ident()
}
func (t *Optional) String() string {
	return "?" + t.T.String()
}
func (t *Optional) Equal(other TypeSpec) bool {
	// null can always be assigned to optional
	if _, ok := other.(*Null); ok {
		return true
	}
	// if both are optiona compare inner types
	if other, ok := other.(*Optional); ok {
		return t.T.Equal(other.T)
	}
	return t.T.Equal(other)
}

type Memory struct {
	T TypeSpec
}

func (t *Memory) Type() TypeSpec { return t.T }
func (t *Memory) Ident() string {
	return t.String()
}
func (t *Memory) String() string {
	return "memory<" + t.T.String() + ">"
}
func (t *Memory) Equal(other TypeSpec) bool {
	otherMem, ok := other.(*Memory)
	if !ok {
		return ok
	}

	return t.T.Equal(otherMem.Type())
}

// type Any struct{}

// func (t *Any) Ident() string {
// 	return t.String()
// }
// func (t *Any) String() string {
// 	return "any"
// }
// func (t *Any) Equal(other TypeSpec) bool {
// 	return true
// }
// func (t *Any) Type() TypeSpec {
// 	return t
// }

// T can be left nil, meaning it accepts any type
type Type struct {
	T TypeSpec
}

func (t *Type) Ident() string {
	return t.String()
}
func (t *Type) String() string {
	if t.T == nil {
		return "type"
	}
	return "type<" + t.T.String() + ">"
}

// 'other' should never be of type 'Type'
func (t *Type) Equal(other TypeSpec) bool {
	if t.T == nil {
		return true
	}

	return t.T.Equal(other)
}

func (t *Type) Type() TypeSpec { return t.T }

type Definition struct {
	Name       string
	Underlying TypeSpec
}

func (t *Definition) Ident() string {
	return t.String()
}
func (t *Definition) String() string {
	return t.Name
}

// 'other' should never be of type 'Type'
func (t *Definition) Equal(other TypeSpec) bool {
	return t.Name == other.Ident()
}

func (t *Definition) Type() TypeSpec { return t }

type Alias struct {
	Name       string
	Underlying TypeSpec
}

func (t *Alias) Ident() string {
	return t.String()
}
func (t *Alias) String() string {
	return t.Name
}

func (t *Alias) Equal(other TypeSpec) bool {
	if otherAlias, ok := other.(*Alias); ok {
		return t.Underlying.Equal(otherAlias.Underlying)
	}
	return t.Underlying.Equal(other)
}

func (t *Alias) Type() TypeSpec { return t.Underlying }

type Dirty struct {
	T TypeSpec
}

func (t *Dirty) Type() TypeSpec { return t }

// returns type name without 'dirty<>'
func (t *Dirty) Ident() string {
	return t.T.String()
}
func (t *Dirty) String() string {
	return "dirty<" + t.T.String() + ">"
}
func (t *Dirty) Equal(other TypeSpec) bool {
	otherD, ok := other.(*Dirty)
	if !ok {
		return false
	}
	return t.T.Equal(otherD.T)
}

// ------------- //
// Special types //
// ------------- //

// These types don't formally exist within language
// they are merely here to convey additional meaning
// when parsing or analysing.

type UnknownNamed struct {
	Name string
}

func (t *UnknownNamed) Type() TypeSpec { return t }
func (t *UnknownNamed) Ident() string {
	return t.Name
}
func (t *UnknownNamed) String() string {
	return "unknown<" + t.Name + ">"
}
func (t *UnknownNamed) Equal(other TypeSpec) bool {
	otherUn, ok := other.(*UnknownNamed)
	return ok && t.Name == otherUn.Name
}

type Multi struct {
	Ts []TypeSpec
}

func (t *Multi) Type() TypeSpec { return t }
func (t *Multi) Ident() string  { return t.String() }
func (t *Multi) GetType(i int) TypeSpec {
	return t.Ts[i]
}
func (t *Multi) String() string {
	if len(t.Ts) == 1 {
		return t.Ts[0].String()
	}
	ts := make([]string, 0, len(t.Ts))
	for _, t := range t.Ts {
		ts = append(ts, t.String())
	}
	return "(" + strings.Join(ts, ",") + ")"
}
func (t *Multi) Equal(other TypeSpec) bool {
	otherMulti, ok := other.(*Multi)
	if !ok {
		return false
	}
	if len(t.Ts) != len(otherMulti.Ts) {
		return false
	}
	for i, typ := range t.Ts {
		if !typ.Equal(otherMulti.Ts[i]) {
			return false
		}
	}
	return true
}

// ------- //
// Helpers //
// ------- //

// Do not pass:
// - pointer type e.g. "*T"
// - dirty type
// - type definition
// - optional type
func IsTypeIdent(ident string) bool {
	switch ident {
	case
		"u8", "u16", "u32", "u64", "u128", "u256",
		"i8", "i16", "i32", "i64", "i128", "i256",
		"f32", "f64",
		"string",
		"bool",
		"byte",
		"array":
		return true
	default:
		return false
	}
}

func GetUnderlyingMemory(t TypeSpec) *Memory {
	switch t := t.(type) {
	case *Memory:
		return t
	case *Definition:
		return GetUnderlyingMemory(t.Underlying)
	case *Dirty:
		return GetUnderlyingMemory(t.T)
	case *Pointer:
		return GetUnderlyingMemory(t.T)
	case *Optional:
		return GetUnderlyingMemory(t.T)
	case *Alias:
		return GetUnderlyingMemory(t.Underlying)
	}
	return nil
}

// Recursively strips Definition and Dirty away
// exposing the underlying primitive type
func GetUnderlyingType(t TypeSpec) TypeSpec {
	switch t := t.(type) {
	case *Definition:
		return GetUnderlyingType(t.Underlying)
	case *Dirty:
		return GetUnderlyingType(t.T)
	}
	return t
}

// Recursively strips 'Definition', 'Dirty' and 'Optional'
// away exposing a type ripe to be checked against if
// we know we are working with a literal
func GetUnderlyingTypeIfLiteral(t TypeSpec) TypeSpec {
	switch t := t.(type) {
	case *Definition:
		return GetUnderlyingTypeIfLiteral(t.Underlying)
	case *Dirty:
		return GetUnderlyingTypeIfLiteral(t.T)
	case *Optional:
		return GetUnderlyingTypeIfLiteral(t.T)
	}
	return t
}

// Checks whether 'from' type can be coalsced to 'to'
// type. For example if 'from' is type of a literal
func CanCoalesce(from, to TypeSpec) bool {
	switch from := from.(type) {
	case *Int:
		switch to := to.(type) {
		case *Dirty:
			return CanCoalesce(from, to.T)
		case *Definition:
			return CanCoalesce(from, to.Underlying)
		case *Optional:
			return CanCoalesce(from, to.T)
		case *Int:
			return IntTypeFitsIn(from, to)
		case *Byte:
			return IntTypeFitsIn(from, &ConstU8)
		case *Char:
			return IntTypeFitsIn(from, &ConstU32)
		case *Union:
			for _, t := range to.Ts {
				if CanCoalesce(from, t) {
					return true
				}
			}
		default:
			return false
		}
	case *Float:
		switch to := to.(type) {
		case *Dirty:
			return CanCoalesce(from, to.T)
		case *Definition:
			return CanCoalesce(from, to.Underlying)
		case *Optional:
			return CanCoalesce(from, to.T)
		case *Float:
			return from.Width <= to.Width
		case *Union:
			for _, t := range to.Ts {
				if CanCoalesce(from, t) {
					return true
				}
			}
		default:
			return false
		}
	case *String:
		switch to := to.(type) {
		case *Dirty:
			return CanCoalesce(from, to.T)
		case *Definition:
			return CanCoalesce(from, to.Underlying)
		case *Optional:
			return CanCoalesce(from, to.T)
		case *String:
			return true
		case *Union:
			for _, t := range to.Ts {
				if CanCoalesce(from, t) {
					return true
				}
			}
		default:
			return false
		}
	case *Bool:
		switch to := to.(type) {
		case *Dirty:
			return CanCoalesce(from, to.T)
		case *Definition:
			return CanCoalesce(from, to.Underlying)
		case *Optional:
			return CanCoalesce(from, to.T)
		case *Bool:
			return true
		case *Union:
			for _, t := range to.Ts {
				if CanCoalesce(from, t) {
					return true
				}
			}
		default:
			return false
		}
	case *Byte:
		switch to := to.(type) {
		case *Dirty:
			return CanCoalesce(from, to.T)
		case *Definition:
			return CanCoalesce(from, to.Underlying)
		case *Optional:
			return CanCoalesce(from, to.T)
		case *Byte:
			return true
		case *Int:
			return IntTypeFitsIn(&ConstU8, to)
		case *Char:
			return true
		case *Union:
			for _, t := range to.Ts {
				if CanCoalesce(from, t) {
					return true
				}
			}
		default:
			return false
		}
	case *Char:
		switch to := to.(type) {
		case *Dirty:
			return CanCoalesce(from, to.T)
		case *Definition:
			return CanCoalesce(from, to.Underlying)
		case *Optional:
			return CanCoalesce(from, to.T)
		case *Char:
			return true
		case *Int:
			return IntTypeFitsIn(&ConstU32, to)
		case *Union:
			for _, t := range to.Ts {
				if CanCoalesce(from, t) {
					return true
				}
			}
		default:
			return false
		}
	case *Array:
		switch to := to.(type) {
		case *Dirty:
			return CanCoalesce(from, to.T)
		case *Definition:
			return CanCoalesce(from, to.Underlying)
		case *Optional:
			return CanCoalesce(from, to.T)
		case *Array:
			return CanCoalesce(from.T, to.T)
		case *Union:
			for _, t := range to.Ts {
				if CanCoalesce(from, t) {
					return true
				}
			}
		default:
			return false
		}
	case *Function:
		switch to := to.(type) {
		case *Function:
			if len(from.Arg) != len(to.Arg) {
				return false
			} else if len(from.Ret) != len(to.Ret) {
				return false
			}
			for i, at := range from.Arg {
				if !at.Equal(to.Arg[i]) {
					return false
				}
			}
			for i, rt := range from.Ret {
				if !rt.Equal(to.Ret[i]) {
					return false
				}
			}
			return true
		case *Optional:
			return CanCoalesce(from, to.T)
		case *Definition:
			return CanCoalesce(from, to.Underlying)
		}
	case *Dirty:
		switch to := to.(type) {
		case *Dirty:
			return CanCoalesce(from.T, to.T)
		}
		return CanCoalesce(from.T, to)
	case *Definition:
		switch to := to.(type) {
		case *Definition:
			return CanCoalesce(from.Underlying, to.Underlying)
		}
		return CanCoalesce(from.Underlying, to)
	case *Struct:
		switch to := to.(type) {
		case *Dirty:
			return CanCoalesce(from, to.T)
		case *Struct:
			if from.Name == to.Name {
				return true
			}

			if len(from.Ts) == 0 && len(to.Ts) == 0 {
				return true
			}
			isFromUnnamed := from.Ts[0].Name == ""
			for i, field := range to.Ts {
				var fromFieldType TypeSpec
				var err error
				if isFromUnnamed || field.Name == "" {
					fromFieldType, err = from.GetTypeByIndex(i)
				} else {
					fromFieldType, _, err = from.GetTypeByField(field.Name)
				}
				if err != nil {
					if _, ok := field.T.(*Optional); ok {
						// if from field is optional we
						// dont care its missing
						continue
					} else {
						return false
					}
				}
				if !CanCoalesce(fromFieldType, field.T) {
					return false
				}
			}
			return true
		case *Definition:
			return CanCoalesce(from, to.Underlying)
		case *Union:
			for _, t := range to.Ts {
				if CanCoalesce(from, t) {
					return true
				}
			}
		}
	case *Null:
		switch to.(type) {
		case *Optional:
			return true
		}
		return false
	}
	return false
}

// 1 == signed
// 0 == unsigned
// rest == error not int
func GetSign(t TypeSpec) int {
	if _, ok := t.(*Char); ok {
		return 0
	}
	if _, ok := t.(*Byte); ok {
		return 0
	}
	if int, ok := GetUnderlyingTypeIfLiteral(t).(*Int); ok {
		return int.Signed
	}
	return -1
}

func LowestFittingInt(v int64, signed bool) *Int {
	if signed {
		if v <= math.MaxInt8 {
			return &ConstI8
		} else if v <= math.MaxInt16 {
			return &ConstI16
		} else if v <= math.MaxInt32 {
			return &ConstI32
		} else {
			return &ConstI64
		}
	} else {
		if v < 0 {
			panic("v was less than zero")
		}
		if v >= 0 && v <= math.MaxUint8 {
			return &ConstU8
		} else if v >= 0 && v <= math.MaxUint16 {
			return &ConstU16
		} else if v >= 0 && v <= math.MaxUint32 {
			return &ConstU32
		} else {
			return &ConstU64
		}
	}
}

func LowestFittingFloat(v float64) *Float {
	if v >= -math.MaxFloat32 && v <= math.MaxFloat32 {
		return &ConstF32
	}
	return &ConstF64
}

func TokenToType(tk token.Token) TypeSpec {
	switch tk.Type {
	case token.I8TYPE:
		return &ConstI8
	case token.U8TYPE:
		return &ConstU8
	case token.I16TYPE:
		return &ConstI16
	case token.U16TYPE:
		return &ConstU16
	case token.I32TYPE:
		return &ConstI32
	case token.U32TYPE:
		return &ConstU32
	case token.I64TYPE:
		return &ConstI64
	case token.U64TYPE:
		return &ConstU64
	// case token.I128TYPE:
	// 	return &ConstI128
	// case token.U128TYPE:
	// 	return &ConstU128
	// case token.I256TYPE:
	// 	return &ConstI256
	// case token.U256TYPE:
	// 	return &ConstU256
	case token.F32TYPE:
		return &ConstF32
	case token.F64TYPE:
		return &ConstF64
	case token.STRINGTYPE:
		return &ConstString
	case token.BOOLTYPE:
		return &ConstBool
	case token.BYTETYPE:
		return &ConstByte
	case token.CHARTYPE:
		return &ConstChar
	case token.ASTERISK:
		return &Pointer{}
	case token.MEMORYTYPE:
		return &Memory{}
	case token.DIRTYTYPE:
		return &Dirty{}
	}
	panic("invalid token " + tk.Literal)
}

func IntTypeFitsIn(t1 *Int, t2 *Int) bool {
	// Going from unisgned to signed is only possible
	// without having to check for error when width
	// of t2 greater than t1.
	if t1.Signed == 0 && t2.Signed == 1 {
		if t1.Width >= t2.Width {
			return false
		} else {
			return true
		}
	}

	// For all other conversions,
	// - uint to uint
	// - int to int
	// - int to uint
	// the t2 width needs to be greater
	// or equal to t1
	if t1.Width > t2.Width {
		return false
	}

	return true
}

// Checks whether an int value v can be coalesced into type t2
func IntValueFitsIn(v int64, t2 *Int) bool {
	if v < 0 && t2.Signed == 1 {
		return false
	}

	if t2.Signed == 1 {
		switch t2.Width {
		case 8:
			if v < math.MinInt8 || v > math.MaxInt8 {
				return false
			}
		case 16:
			if v < math.MinInt16 || v > math.MaxInt16 {
				return false
			}
		case 32:
			if v < math.MinInt32 || v > math.MaxInt32 {
				return false
			}
		case 64:
			if v < math.MinInt64 || v > math.MaxInt64 {
				return false
			}
		case 128, 256:
			// For widths greater than 64 bits, any int value fits
			return true
		}
	} else {
		// Unsigned integers
		if v < 0 {
			return false
		}
		switch t2.Width {
		case 8:
			if v > math.MaxUint8 {
				return false
			}
		case 16:
			if v > math.MaxUint16 {
				return false
			}
		case 32:
			if uint32(v) > math.MaxUint32 {
				return false
			}
		case 64:
			if uint64(v) > math.MaxUint64 {
				return false
			}
		case 128, 256:
			return true
		}
	}
	return true
}

func IsFloatRepresentableAs(v float64, t *Float) bool {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return false
	}
	// check float64 fits into float32
	if t.Width == 32 {
		f32 := float32(v)
		return float64(f32) == v
	}
	return true
}
