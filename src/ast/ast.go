package ast

import (
	"bytes"
	"strings"

	"dash-lang.io/src/token"
	"dash-lang.io/src/types"
)

type VariableEscape uint8

const (
	NO_ESCAPE VariableEscape = iota
	PASSED
	RETURNED
)

type Node interface {
	Pos() token.Pos
	TokenLiteral() string
	String() string
}

type Statement interface {
	Node
	statementNode()
}

type Literal interface {
	Node
	literalNode()
}

type Expression interface {
	Node
	expressionNode()
	Type() types.TypeSpec
	SetType(t types.TypeSpec)
}

type Library struct {
	Token           token.Token
	Name            *Identifier
	Imports         []*ImportStatement
	TypeDefinitions []*TypeDefinitionStatement
	TypeAliases     []*TypeAliasStatement
	GenericStructs  []*GenericStructStatement
	Structs         []*StructStatement
	Enums           []*EnumStatement
	Unions          []*UnionStatement
	GlobalVariables []*AssignmentStatement
	Functions       []*FunctionExpression
}

func (l *Library) statementNode()       {}
func (l *Library) TokenLiteral() string { return l.Token.Literal }
func (l *Library) String() string {
	var out bytes.Buffer
	out.WriteString(l.TokenLiteral() + " ")
	out.WriteString(l.Name.String())

	if len(l.Imports) > 0 {
		out.WriteString(" ")
	}
	for i := 0; i < len(l.Imports); i++ {
		out.WriteString(l.Imports[i].String())
	}

	if len(l.TypeDefinitions) > 0 {
		out.WriteString(" ")
	}
	for i := 0; i < len(l.TypeDefinitions); i++ {
		out.WriteString(l.TypeDefinitions[i].String())
		if i != len(l.TypeDefinitions)-1 {
			out.WriteString(" ")
		}
	}

	if len(l.TypeAliases) > 0 {
		out.WriteString(" ")
	}
	for i := 0; i < len(l.TypeAliases); i++ {
		out.WriteString(l.TypeAliases[i].String())
		if i != len(l.TypeAliases)-1 {
			out.WriteString(" ")
		}
	}

	if len(l.GenericStructs) > 0 {
		out.WriteString(" ")
	}
	for i := 0; i < len(l.GenericStructs); i++ {
		out.WriteString(l.GenericStructs[i].String())
		if i != len(l.GenericStructs)-1 {
			out.WriteString(" ")
		}
	}

	if len(l.Structs) > 0 {
		out.WriteString(" ")
	}
	for i := 0; i < len(l.Structs); i++ {
		out.WriteString(l.Structs[i].String())
		if i != len(l.Structs)-1 {
			out.WriteString(" ")
		}
	}

	if len(l.Enums) > 0 {
		out.WriteString(" ")
	}
	for i := 0; i < len(l.Enums); i++ {
		out.WriteString(l.Enums[i].String())
		if i != len(l.Enums)-1 {
			out.WriteString(" ")
		}
	}

	if len(l.Unions) > 0 {
		out.WriteString(" ")
	}
	for i := 0; i < len(l.Unions); i++ {
		out.WriteString(l.Unions[i].String())
		if i != len(l.Unions)-1 {
			out.WriteString(" ")
		}
	}

	if len(l.GlobalVariables) > 0 {
		out.WriteString(" ")
	}
	for i := 0; i < len(l.GlobalVariables); i++ {
		out.WriteString(l.GlobalVariables[i].String())
		if i != len(l.GlobalVariables)-1 {
			out.WriteString(" ")
		}
	}

	if len(l.Functions) > 0 {
		out.WriteString(" ")
	}
	for i := 0; i < len(l.Functions); i++ {
		out.WriteString(l.Functions[i].String())
		if i != len(l.Functions)-1 {
			out.WriteString(" ")
		}
	}
	return out.String()
}

type Evaluator struct {
	Unions  []*UnionStatement
	Types   []*TypeDefinitionStatement
	Structs []*StructStatement
	Enums   []*EnumStatement
	Nodes   []Node
}

func (e *Evaluator) Pos() token.Pos                { return token.Pos(0) }
func (e *Evaluator) TokenLiteral() string          { return "" }
func (e *Evaluator) String() string                { return "" }
func (e *Evaluator) visitChildren(fn func(n Node)) {}

type FileFormat struct {
	Token token.Token
	Name  *Identifier
	// Ordered list of pointer to statement or expression as they appear in file
	// Required so file can be formatted properly
	Nodes []Node
}

func (f *FileFormat) Format() string {
	var out bytes.Buffer
	out.WriteString(f.Token.Literal + " ")
	out.WriteString(f.Name.String())

	if len(f.Nodes) > 0 {
		out.WriteString(" ")
	}
	for i := 0; i < len(f.Nodes); i++ {
		out.WriteString(f.Nodes[i].String())
		if i != len(f.Nodes)-1 {
			out.WriteString(" ")
		}
	}
	return out.String()
}

// ------------//
// Statements //
// ------------//

type ImportStatement struct {
	Token   token.Token
	Package token.Token
}

func (s *ImportStatement) statementNode()       {}
func (s *ImportStatement) TokenLiteral() string { return s.Token.Literal }
func (s *ImportStatement) String() string {
	var out bytes.Buffer
	out.WriteString(s.TokenLiteral() + " ")

	out.WriteString("\"" + s.Package.Literal + "\"")

	return out.String()
}

// type <name> <type>
type TypeDefinitionStatement struct {
	Public         bool
	Token          token.Token
	Name           *Identifier
	UnderlyingType types.TypeSpec
	Guard          Expression

	// Set by semsis
	T types.TypeSpec
}

func (s *TypeDefinitionStatement) statementNode()       {}
func (s *TypeDefinitionStatement) TokenLiteral() string { return s.Token.Literal }
func (s *TypeDefinitionStatement) Type() types.TypeSpec { return s.T }
func (s *TypeDefinitionStatement) String() string {
	var out bytes.Buffer
	if s.Public {
		out.WriteString("pub ")
	}
	out.WriteString(s.TokenLiteral() + " ")
	out.WriteString(s.Name.String() + " ")
	out.WriteString(s.UnderlyingType.String())
	if s.Guard != nil {
		out.WriteString(" | " + s.Guard.String())
	}
	return out.String()
}

// alias <name> <stmt>
type TypeAliasStatement struct {
	Public         bool
	Token          token.Token
	Name           *Identifier
	UnderlyingType types.TypeSpec

	// Set by semsis
	T types.TypeSpec
}

func (s *TypeAliasStatement) statementNode()       {}
func (s *TypeAliasStatement) TokenLiteral() string { return s.Token.Literal }
func (s *TypeAliasStatement) Type() types.TypeSpec {
	return s.T
}
func (s *TypeAliasStatement) String() string {
	var out bytes.Buffer
	if s.Public {
		out.WriteString("pub ")
	}
	out.WriteString(s.TokenLiteral() + " ")
	out.WriteString(s.Name.String() + " ")
	out.WriteString(s.UnderlyingType.String())
	return out.String()
}

type GenericStructStatement struct {
	Public bool
	Token  token.Token
	Name   *Identifier
	Fields []*StructFieldStatement
	T      *types.AbstractStruct
}

func (s *GenericStructStatement) statementNode()       {}
func (s *GenericStructStatement) TokenLiteral() string { return s.Token.Literal }
func (s *GenericStructStatement) Type() types.TypeSpec {
	return s.T
}
func (s *GenericStructStatement) String() string {
	var out bytes.Buffer
	if s.Public {
		out.WriteString("pub ")
	}
	out.WriteString("gen " + s.TokenLiteral() + " ")
	out.WriteString(s.Name.String() + " {")
	for i, field := range s.Fields {
		out.WriteString(field.String())
		if i != len(s.Fields)-1 {
			out.WriteString(", ")
		}
	}
	out.WriteString("}")
	return out.String()
}

// struct <name> { }
type StructStatement struct {
	Public bool
	Token  token.Token
	Name   *Identifier
	Fields []*StructFieldStatement
	T      *types.Struct
}

func (s *StructStatement) statementNode()       {}
func (s *StructStatement) TokenLiteral() string { return s.Token.Literal }
func (s *StructStatement) Type() types.TypeSpec {
	return s.T
}
func (s *StructStatement) String() string {
	var out bytes.Buffer
	if s.Public {
		out.WriteString("pub ")
	}
	out.WriteString(s.TokenLiteral() + " ")
	out.WriteString(s.Name.String() + " {")
	for i, field := range s.Fields {
		out.WriteString(field.String())
		if i != len(s.Fields)-1 {
			out.WriteString(", ")
		}
	}
	out.WriteString("}")
	return out.String()
}

// enum <name> { }
type EnumStatement struct {
	Public bool
	Token  token.Token
	Name   *Identifier
	Fields []*Identifier
	T      *types.Enum
}

func (s *EnumStatement) statementNode()       {}
func (s *EnumStatement) TokenLiteral() string { return s.Token.Literal }
func (s *EnumStatement) String() string {
	var out bytes.Buffer
	if s.Public {
		out.WriteString("pub ")
	}
	out.WriteString("enum ")
	out.WriteString(s.Name.String() + " {")
	for i, f := range s.Fields {
		out.WriteString(f.String())
		if i != len(s.Fields)-1 {
			out.WriteString(", ")
		}
	}
	out.WriteString("}")
	return out.String()
}

// union <name> { }
type UnionStatement struct {
	Public bool
	Token  token.Token
	Name   *Identifier
	Types  []*TypeLiteral

	// Set by semsis
	T *types.Union
}

func (s *UnionStatement) statementNode()       {}
func (s *UnionStatement) TokenLiteral() string { return s.Token.Literal }
func (s *UnionStatement) String() string {
	var out bytes.Buffer
	if s.Public {
		out.WriteString("pub ")
	}
	out.WriteString("union ")
	out.WriteString(s.Name.String() + " {")
	for i, f := range s.Types {
		out.WriteString(f.String())
		if i != len(s.Types)-1 {
			out.WriteString(", ")
		}
	}
	out.WriteString("}")
	return out.String()
}

type ParameterStatement struct {
	Name *Identifier
	Type types.TypeSpec // types e.g. int, float
}

func (s *ParameterStatement) statementNode()       {}
func (s *ParameterStatement) TokenLiteral() string { return s.Name.Token.Literal }
func (s *ParameterStatement) String() string {
	var out bytes.Buffer
	out.WriteString(s.Name.String())
	if s.Type == nil {
		return out.String()
	}

	out.WriteString(" ")
	out.WriteString(s.Type.String())

	return out.String()
}

type BlockStatement struct {
	Token      token.Token // the { or : token
	Statements []Node
}

func (bs *BlockStatement) statementNode()       {}
func (bs *BlockStatement) TokenLiteral() string { return bs.Token.Literal }
func (bs *BlockStatement) String() string {
	var out bytes.Buffer

	out.WriteString("{ ")
	for _, s := range bs.Statements {
		out.WriteString(s.String() + " ")
	}
	out.WriteString("}")
	return out.String()
}

// var x
// let y
// z
type DeclarationStatement struct {
	Token    token.Token // `let`, `var` or nil
	Assignee Expression
}

func (s *DeclarationStatement) statementNode()       {}
func (s *DeclarationStatement) TokenLiteral() string { return s.Token.Literal }
func (s *DeclarationStatement) String() string {
	var out bytes.Buffer
	if s.TokenLiteral() != "" {
		out.WriteString(s.TokenLiteral() + " ")
	}
	out.WriteString(s.Assignee.String())

	if s.Assignee.Type() != nil {
		out.WriteString(" " + s.Assignee.Type().String())
	}

	return out.String()
}

// let x, v, var z = 1, 2, 3
type AssignmentStatement struct {
	Declerations []*DeclarationStatement
	Values       []Expression
}

func (s *AssignmentStatement) statementNode()       {}
func (s *AssignmentStatement) TokenLiteral() string { return "" }
func (s *AssignmentStatement) String() string {
	var out bytes.Buffer

	for i, decl := range s.Declerations {
		out.WriteString(decl.String())
		if i != len(s.Declerations)-1 {
			out.WriteString(", ")
		}
	}
	out.WriteString(" = ")
	for i, value := range s.Values {
		out.WriteString(value.String())
		if i != len(s.Values)-1 {
			out.WriteString(", ")
		}
	}
	return out.String()
}

type Identifier struct {
	Token token.Token // the token.IDENT token
	Value string
	T     types.TypeSpec
}

func (i *Identifier) expressionNode()          {}
func (i *Identifier) Type() types.TypeSpec     { return i.T }
func (i *Identifier) SetType(t types.TypeSpec) { i.T = t }
func (i *Identifier) TokenLiteral() string     { return i.Token.Literal }
func (i *Identifier) String() string           { return i.Value }

type ReturnStatement struct {
	Token  token.Token // the 'return' token
	Values []Expression
}

func (s *ReturnStatement) statementNode() {}
func (s *ReturnStatement) ReturnTypes() []types.TypeSpec {
	var typs []types.TypeSpec

	for _, val := range s.Values {
		switch mt := val.Type().(type) {
		case *types.Multi:
			typs = append(typs, mt.Ts...)
		default:
			typs = append(typs, mt)
		}
	}

	return typs
}
func (s *ReturnStatement) TokenLiteral() string { return s.Token.Literal }
func (s *ReturnStatement) String() string {
	var out bytes.Buffer
	out.WriteString(s.TokenLiteral())
	if len(s.Values) != 0 {
		out.WriteString(" ")
	}
	for i := 0; i < len(s.Values); i++ {
		out.WriteString(s.Values[i].String())
		if i != len(s.Values)-1 {
			out.WriteString(", ")
		}
	}
	return out.String()
}

// for i = 0; i < 10; i++ {}
// for g.has_more() {}
type ForStatement struct {
	Token token.Token // the "for" token
	// 'var' literal defined implicitely
	Assignment *AssignmentStatement
	Condition  Expression
	Change     Expression

	Block *BlockStatement
}

func (s *ForStatement) statementNode()       {}
func (s *ForStatement) TokenLiteral() string { return s.Token.Literal }
func (s *ForStatement) String() string {
	var out bytes.Buffer

	out.WriteString(s.Token.Literal)
	if s.Assignment != nil {
		out.WriteString(" " + s.Assignment.String() + ";")
	}
	// if we just have boolean for loop we dont print ';' at end
	if s.Assignment == nil && s.Condition != nil && s.Change == nil {
		out.WriteString(" " + s.Condition.String())
	} else if s.Condition != nil {
		out.WriteString(" " + s.Condition.String() + ";")
	}
	if s.Change != nil {
		out.WriteString(" " + s.Change.String())
	}

	if s.Block != nil {
		out.WriteString(" " + s.Block.String())
	}
	return out.String()
}

// for el in arr {}
type ForRangeStatement struct {
	Token     token.Token   // the "for" token
	Variables []*Identifier // e.g. i, v

	Start Expression
	End   Expression

	// Either range or identifier can be used
	Iterable Expression // e.g. array or list
	Change   Expression // e.g. i++, i-- or empty (then default i++)
	Block    *BlockStatement
}

func (s *ForRangeStatement) statementNode()       {}
func (s *ForRangeStatement) TokenLiteral() string { return s.Token.Literal }
func (s *ForRangeStatement) String() string {
	var out bytes.Buffer

	out.WriteString(s.Token.Literal)
	if s.Variables != nil {
		for i := 0; i < len(s.Variables); i++ {
			if i%2 != 0 {
				out.WriteString(",")
			}
			out.WriteString(" " + s.Variables[i].String())
		}
	}
	if s.Iterable != nil {
		out.WriteString(" in " + s.Iterable.String())
	}

	if s.Change != nil {
		out.WriteString("; " + s.Change.String())
	}

	if s.Block != nil {
		out.WriteString(" " + s.Block.String())
	} else {
		out.WriteString(" { }")
	}

	return out.String()
}

type WhileStatement struct {
	Token     token.Token
	Condition Expression
	Block     *BlockStatement
}

func (s *WhileStatement) statementNode()       {}
func (s *WhileStatement) TokenLiteral() string { return s.Token.Literal }
func (s *WhileStatement) String() string {
	var out bytes.Buffer

	out.WriteString(s.Token.Literal + " ")
	out.WriteString(s.Condition.String() + " ")

	if s.Block != nil {
		out.WriteString(s.Block.String())
	} else {
		out.WriteString("{ }")
	}

	return out.String()
}

type KeywordStatement struct {
	Token token.Token
}

func (s *KeywordStatement) statementNode()       {}
func (s *KeywordStatement) TokenLiteral() string { return s.Token.Literal }
func (s *KeywordStatement) String() string       { return s.Token.Literal }

type StructFieldStatement struct {
	// Can be nil if unnamed struct definition
	Name *Identifier
	Type types.TypeSpec
	// Needs to evaluate to boolean expression
	Guard Expression
}

func (s *StructFieldStatement) statementNode()       {}
func (s *StructFieldStatement) TokenLiteral() string { return s.Name.Token.Literal }
func (s *StructFieldStatement) String() string {
	var out bytes.Buffer
	if s.Name != nil {
		out.WriteString(s.Name.Value + " ")
	}
	out.WriteString(s.Type.String())
	if s.Guard != nil {
		out.WriteString(" | " + s.Guard.String())
	}
	return out.String()
}

type DeferStatement struct {
	Token token.Token
	// Can only be blockstatement, fn call or fn call dot expression
	Node Node
}

func (e *DeferStatement) expressionNode()          {}
func (e *DeferStatement) Type() types.TypeSpec     { return nil }
func (e *DeferStatement) SetType(t types.TypeSpec) {}
func (e *DeferStatement) TokenLiteral() string     { return e.Token.Literal }
func (e *DeferStatement) String() string {
	var out bytes.Buffer

	out.WriteString(e.Token.Literal + " ")
	out.WriteString(e.Node.String())

	return out.String()
}

type MatchExpressionStatement struct {
	Token       token.Token
	Scrutinee   Expression
	Cases       []*MatchCase
	Default     *MatchCase
	IsStatement bool
	// Set by semsis. If nil it means match
	// functions as a statement.
	T types.TypeSpec
}

func (s *MatchExpressionStatement) expressionNode()          {}
func (s *MatchExpressionStatement) statementNode()           {}
func (s *MatchExpressionStatement) Type() types.TypeSpec     { return s.T }
func (s *MatchExpressionStatement) SetType(t types.TypeSpec) { s.T = t }
func (s *MatchExpressionStatement) TokenLiteral() string     { return s.Token.Literal }
func (s *MatchExpressionStatement) String() string {
	var out bytes.Buffer

	out.WriteString(s.Token.Literal + " ")
	out.WriteString(s.Scrutinee.String() + " { ")

	for _, c := range s.Cases {
		out.WriteString(c.String() + " ")
	}
	if s.Default != nil {
		out.WriteString(s.Default.String() + " ")
	}
	out.WriteString("}")

	return out.String()
}

type MatchCase struct {
	Token     token.Token // case or else token
	Predicate Expression
	Body      []Node

	// Set by semsis only if used as
	// expression.
	T types.TypeSpec
}

func (mc *MatchCase) expressionNode()          {}
func (mc *MatchCase) Type() types.TypeSpec     { return mc.T }
func (mc *MatchCase) SetType(t types.TypeSpec) {}
func (mc *MatchCase) TokenLiteral() string     { return mc.Token.Literal }
func (mc *MatchCase) String() string {
	var out bytes.Buffer

	out.WriteString(mc.Token.Literal)
	if mc.Predicate != nil {
		out.WriteString(" " + mc.Predicate.String())
	}
	out.WriteString(": ")
	for i, r := range mc.Body {
		out.WriteString(r.String())
		if i != len(mc.Body)-1 {
			out.WriteString(" ")
		}
	}

	return out.String()
}

//-------------//
// Expressions //
//-------------//

//	fn add(a i64, b i64) i64, i64 {
//		return a + b
//	}
type FunctionExpression struct {
	Attributes   []Attribute
	Public       bool        // pub fn
	Token        token.Token // The 'fn' token
	Name         *Identifier
	Arguments    []*ParameterStatement
	ReturnValues []*TypeLiteral

	Body *BlockStatement

	// Set by semantic analysis based on arguments
	T           types.TypeSpec
	IsVariadic  bool
	IsAnonymous bool
}

func (fl *FunctionExpression) expressionNode()          {}
func (fl *FunctionExpression) SetType(t types.TypeSpec) {}
func (fl *FunctionExpression) HasAttribute(attrT AttributeType) bool {
	for _, attr := range fl.Attributes {
		if attr.Equal(attrT) {
			return true
		}
	}
	return false
}
func (fl *FunctionExpression) Type() types.TypeSpec { return fl.T }
func (fl *FunctionExpression) TokenLiteral() string { return fl.Token.Literal }
func (fl *FunctionExpression) String() string {
	var out bytes.Buffer
	for i, attr := range fl.Attributes {
		out.WriteString(attr.String())
		if i == len(fl.Attributes)-1 {
			out.WriteString(" ")
		}
	}
	if fl.Public {
		out.WriteString("pub ")
	}
	out.WriteString(fl.TokenLiteral())
	// fl.Name check required during parsing stage
	if !fl.IsAnonymous {
		out.WriteString(" " + fl.Name.String())
	}
	// print arguments
	args := make([]string, 0, len(fl.Arguments))
	out.WriteString("(")
	for _, arg := range fl.Arguments {
		args = append(args, arg.String())
	}
	out.WriteString(strings.Join(args, ","))
	out.WriteString(")")
	// print return types
	if len(fl.ReturnValues) != 0 {
		out.WriteString(" ")
		for i, rv := range fl.ReturnValues {
			out.WriteString(rv.String())
			if i != len(fl.ReturnValues)-1 {
				out.WriteString(", ")
			}
		}
	}

	// body can be nil if function meant to be linked
	if fl.Body != nil {
		out.WriteString(" " + fl.Body.String())
	}
	return out.String()
}

type IfElseExpression struct {
	Token        token.Token // the 'if' token
	Conditionals []*ConditionalExpression
	IsStatement  bool

	// Set by semsis. If nil it
	// means if else used as statement
	T types.TypeSpec
}

func (s *IfElseExpression) expressionNode()          {}
func (s *IfElseExpression) Type() types.TypeSpec     { return s.T }
func (s *IfElseExpression) SetType(t types.TypeSpec) { s.T = t }
func (s *IfElseExpression) TokenLiteral() string     { return s.Token.Literal }
func (s *IfElseExpression) String() string {
	var out bytes.Buffer

	for _, c := range s.Conditionals {
		out.WriteString(c.String())
	}
	return out.String()
}

// ! Does not implement Expression interface
type ConditionalExpression struct {
	Token     token.Token // e.g. if, else if or else
	Condition Expression
	Block     *BlockStatement
}

func (e *ConditionalExpression) expressionNode()      {}
func (e *ConditionalExpression) TokenLiteral() string { return e.Token.Literal }
func (e *ConditionalExpression) String() string {
	var out bytes.Buffer

	if e.Token.Type != token.IF {
		out.WriteString(" ")
	}

	out.WriteString(e.Token.Literal + " ")
	if e.Condition != nil {
		out.WriteString(e.Condition.String() + " ")
	}

	if e.Block != nil {
		out.WriteString(e.Block.String())
	}

	return out.String()
}

type FunctionCallExpression struct {
	Token       token.Token // The function identifier
	Arguments   []Expression
	ReturnTypes []types.TypeSpec
	Catch       Expression

	// Points to underlying function definition within library
	Func *FunctionExpression

	// Set by semantic analysis
	T             types.TypeSpec
	IsAnonymousFn bool
}

func (e *FunctionCallExpression) expressionNode()          {}
func (e *FunctionCallExpression) SetType(t types.TypeSpec) {}
func (e *FunctionCallExpression) Type() types.TypeSpec     { return e.T }
func (e *FunctionCallExpression) TokenLiteral() string     { return e.Token.Literal }
func (e *FunctionCallExpression) String() string {
	var out bytes.Buffer
	out.WriteString(e.Token.Literal)

	args := make([]string, 0, len(e.Arguments))
	for _, arg := range e.Arguments {
		args = append(args, arg.String())
	}
	out.WriteString("(")
	out.WriteString(strings.Join(args, ","))
	out.WriteString(")")

	if e.Catch != nil {
		out.WriteString(" " + e.Catch.String())
	}

	return out.String()
}

type PrefixExpression struct {
	Token    token.Token // The prefix token, e.g. ! Operator string
	Operator string
	Right    Expression

	// Set by semsis
	T types.TypeSpec
}

func (e *PrefixExpression) expressionNode()          {}
func (e *PrefixExpression) Type() types.TypeSpec     { return e.T }
func (e *PrefixExpression) SetType(t types.TypeSpec) { e.T = t }
func (e *PrefixExpression) TokenLiteral() string     { return e.Token.Literal }
func (e *PrefixExpression) String() string {
	var out bytes.Buffer

	out.WriteString(e.Operator)
	out.WriteString(e.Right.String())

	return out.String()
}

type InfixExpression struct {
	Token    token.Token // The operator token, e.g. +
	Left     Expression
	Operator string
	Right    Expression

	// Set by semsis
	T types.TypeSpec
}

func (e *InfixExpression) expressionNode()          {}
func (e *InfixExpression) Type() types.TypeSpec     { return e.T }
func (e *InfixExpression) SetType(t types.TypeSpec) { e.T = t }
func (e *InfixExpression) TokenLiteral() string     { return e.Token.Literal }
func (e *InfixExpression) String() string {
	var out bytes.Buffer

	out.WriteString("(")
	out.WriteString(e.Left.String())
	out.WriteString(" " + e.Operator + " ")
	if e.Right != nil {
		out.WriteString(e.Right.String())
	}
	out.WriteString(")")

	return out.String()
}

type PostfixExpression struct {
	Token token.Token // The postfix token, e.g. ++
	Left  Expression
	// Set by semsis
	T types.TypeSpec
}

func (e *PostfixExpression) expressionNode()      {}
func (e *PostfixExpression) Type() types.TypeSpec { return e.T }
func (e *PostfixExpression) SetType(t types.TypeSpec) {
	e.T = t
}
func (e *PostfixExpression) TokenLiteral() string { return e.Token.Literal }
func (e *PostfixExpression) String() string {
	var out bytes.Buffer

	out.WriteString(e.Left.String())
	out.WriteString(e.TokenLiteral())

	return out.String()
}

type DotExpression struct {
	Token token.Token
	Left  Expression
	Right Expression

	// set by semsis
	T types.TypeSpec
}

func (e *DotExpression) expressionNode()      {}
func (e *DotExpression) Type() types.TypeSpec { return e.T }
func (e *DotExpression) SetType(t types.TypeSpec) {
	e.T = t
}
func (e *DotExpression) TokenLiteral() string { return e.Token.Literal }
func (e *DotExpression) String() string {
	var out bytes.Buffer

	out.WriteString(e.Left.String())
	out.WriteString(".")
	out.WriteString(e.Right.String())

	return out.String()
}

type CatchExpression struct {
	Token token.Token
	Block *BlockStatement
}

func (s *CatchExpression) expressionNode()          {}
func (s *CatchExpression) Type() types.TypeSpec     { return &types.Function{} } // NOTE: this might need to be changed
func (s *CatchExpression) SetType(t types.TypeSpec) {}
func (s *CatchExpression) TokenLiteral() string     { return s.Token.Literal }
func (s *CatchExpression) String() string {
	var out bytes.Buffer
	out.WriteString(s.Token.Literal + " ")
	if s.Block != nil {
		out.WriteString(s.Block.String())
	}

	return out.String()
}

// Example tree: v = a[0][0]
//
//	  Index
//	 /     \
//	a     [0,0]
type IndexExpression struct {
	Token   token.Token // [
	Left    Expression
	Indices []Expression

	// set by semsis
	T types.TypeSpec
}

func (e *IndexExpression) expressionNode() {}
func (e *IndexExpression) SetType(t types.TypeSpec) {
	e.T = t
}

// Returns type that a variable would have when index expression executed
// v [][]i64 = a[0][0]
// Type() == i64
func (e *IndexExpression) Type() types.TypeSpec {
	if e.T == nil {
		depth := len(e.Indices)
		return e.GetTypeAt(depth)
	}
	return e.T
}

// a = [[1,2], [3,4]]
// GetTypeAt(0) == [][]i64
// GetTypeAt(1) == []i64
// GetTypeAt(2) == i64
// GetTypeAt(3) == panic!
func (e *IndexExpression) GetTypeAt(depth int) types.TypeSpec {
	typ := e.Left.Type()
	if typ == nil {
		return nil
	}
start:
	if depth == 0 {
		return typ
	}
	switch t := typ.(type) {
	// array type definitions are indexable like arrays
	case *types.Definition:
		typ = t.Underlying
		goto start
	case *types.Dirty:
		typ = t.T
		goto start
	// strings are also indexable like arrays
	case *types.String:
		depth--
		typ = &types.ConstByte
		goto start
	case *types.Array:
		typ = t.T
		depth--
		goto start
	}
	panic("attempted to access index expression type at an invalid index")
}
func (e *IndexExpression) TokenLiteral() string { return e.Token.Literal }
func (e *IndexExpression) String() string {
	var out bytes.Buffer

	out.WriteString(e.Left.String())
	for _, idx := range e.Indices {
		out.WriteString("[" + idx.String() + "]")
	}

	return out.String()
}

// arr[1:3][0:1]
type SliceExpression struct {
	Token   token.Token
	Left    Expression
	Indices []Expression

	// Set by semantical analysis
	// Ts []types.TypeSpec
}

func (e *SliceExpression) expressionNode()          {}
func (e *SliceExpression) Type() types.TypeSpec     { return e.Left.Type() }
func (e *SliceExpression) SetType(t types.TypeSpec) {}
func (e *SliceExpression) TokenLiteral() string     { return e.Token.Literal }
func (e *SliceExpression) String() string {
	var out bytes.Buffer

	out.WriteString(e.Left.String())
	for _, idx := range e.Indices {
		out.WriteString("[" + idx.String() + "]")
	}

	return out.String()
}

// x^
type CopyExpression struct {
	Token token.Token
	Ident Expression
}

func (e *CopyExpression) expressionNode()          {}
func (e *CopyExpression) Type() types.TypeSpec     { return e.Ident.Type() }
func (e *CopyExpression) SetType(t types.TypeSpec) {}
func (e *CopyExpression) TokenLiteral() string     { return e.Token.Literal }
func (e *CopyExpression) String() string {
	return e.Ident.String() + e.TokenLiteral()
}

// x^ { x.field = 2 }
type CopyUpdateExpression struct {
	Token token.Token
	Ident Expression
	Block *BlockStatement
}

func (e *CopyUpdateExpression) expressionNode()          {}
func (e *CopyUpdateExpression) Type() types.TypeSpec     { return e.Ident.Type() }
func (e *CopyUpdateExpression) SetType(t types.TypeSpec) {}
func (e *CopyUpdateExpression) TokenLiteral() string     { return e.Token.Literal }
func (e *CopyUpdateExpression) String() string {
	var out bytes.Buffer

	out.WriteString(e.Ident.String())
	out.WriteString(e.TokenLiteral())
	out.WriteString(" " + e.Block.String())

	return out.String()
}

type TypeLiteral struct {
	Token token.Token
	T     types.TypeSpec
}

func (e *TypeLiteral) expressionNode()          {}
func (e *TypeLiteral) Type() types.TypeSpec     { return e.T }
func (e *TypeLiteral) SetType(t types.TypeSpec) { e.T = t }
func (e *TypeLiteral) TokenLiteral() string     { return e.Token.Literal }
func (e *TypeLiteral) String() string {
	return e.T.String()
}

type TypeCastExpression struct {
	Token    token.Token
	Argument Expression
	Typ      types.TypeSpec
}

func (e *TypeCastExpression) expressionNode()          {}
func (e *TypeCastExpression) Type() types.TypeSpec     { return e.Typ }
func (e *TypeCastExpression) SetType(t types.TypeSpec) { e.Typ = t }
func (e *TypeCastExpression) TokenLiteral() string     { return e.Token.Literal }
func (e *TypeCastExpression) String() string {
	var out bytes.Buffer

	out.WriteString(e.Typ.String())
	out.WriteString("(" + e.Argument.String() + ")")

	return out.String()
}

type UseExpression struct {
	Token token.Token
	Ident *Identifier
	Block *BlockStatement

	// Set by semsis
	T types.TypeSpec
}

func (e *UseExpression) expressionNode()          {}
func (e *UseExpression) Type() types.TypeSpec     { return e.T }
func (e *UseExpression) SetType(t types.TypeSpec) { e.T = t }
func (e *UseExpression) TokenLiteral() string     { return e.Token.Literal }
func (e *UseExpression) String() string {
	var out bytes.Buffer

	out.WriteString(e.TokenLiteral() + " ")
	out.WriteString(e.Ident.String() + " ")
	out.WriteString(e.Block.String())

	return out.String()
}

type Comment struct {
	Token token.Token
}

func (e *Comment) expressionNode()          {}
func (e *Comment) Type() types.TypeSpec     { return nil }
func (e *Comment) SetType(t types.TypeSpec) {}
func (e *Comment) TokenLiteral() string     { return e.Token.Literal }
func (e *Comment) String() string {
	return e.TokenLiteral()
}

//----------//
// Literals //
//----------//

type IntegerLiteral struct {
	Token token.Token
	Value int64

	// Set by semsis
	T types.TypeSpec
}

func (l *IntegerLiteral) expressionNode()          {}
func (l *IntegerLiteral) literalNode()             {}
func (l *IntegerLiteral) Type() types.TypeSpec     { return l.T }
func (l *IntegerLiteral) SetType(t types.TypeSpec) { l.T = t }
func (l *IntegerLiteral) TokenLiteral() string     { return l.Token.Literal }
func (l *IntegerLiteral) String() string           { return l.Token.Literal }

type BooleanLiteral struct {
	Token token.Token
	Value bool
}

func (l *BooleanLiteral) expressionNode()          {}
func (l *BooleanLiteral) Type() types.TypeSpec     { return &types.ConstBool }
func (l *BooleanLiteral) SetType(t types.TypeSpec) {}
func (l *BooleanLiteral) TokenLiteral() string     { return l.Token.Literal }
func (l *BooleanLiteral) String() string           { return l.Token.Literal }

type StringLiteral struct {
	Token token.Token
	Value string
}

func (l *StringLiteral) expressionNode()          {}
func (l *StringLiteral) literalNode()             {}
func (l *StringLiteral) Type() types.TypeSpec     { return &types.ConstString }
func (l *StringLiteral) SetType(t types.TypeSpec) {}
func (l *StringLiteral) TokenLiteral() string     { return l.Token.Literal }
func (l *StringLiteral) String() string {
	return `"` + l.Token.Literal + `"`
}

type CharacterLiteral struct {
	Token token.Token
	Value int32

	// Set by semsis
	T types.TypeSpec
}

func (l *CharacterLiteral) expressionNode()          {}
func (l *CharacterLiteral) literalNode()             {}
func (l *CharacterLiteral) Type() types.TypeSpec     { return l.T }
func (l *CharacterLiteral) SetType(t types.TypeSpec) { l.T = t }
func (l *CharacterLiteral) TokenLiteral() string     { return l.Token.Literal }
func (l *CharacterLiteral) String() string {
	return `'` + l.Token.Literal + `'`
}

type FloatLiteral struct {
	Token token.Token
	Value float64

	// Set by semsis
	T types.TypeSpec
}

func (l *FloatLiteral) expressionNode()          {}
func (l *FloatLiteral) literalNode()             {}
func (l *FloatLiteral) Type() types.TypeSpec     { return l.T }
func (l *FloatLiteral) SetType(t types.TypeSpec) {}
func (l *FloatLiteral) TokenLiteral() string     { return l.Token.Literal }
func (l *FloatLiteral) String() string           { return l.Token.Literal }

type ByteLiteral struct {
	Token token.Token
	Value byte
}

func (l *ByteLiteral) expressionNode()          {}
func (l *ByteLiteral) literalNode()             {}
func (l *ByteLiteral) Type() types.TypeSpec     { return &types.ConstByte }
func (l *ByteLiteral) SetType(t types.TypeSpec) {}
func (l *ByteLiteral) TokenLiteral() string     { return l.Token.Literal }
func (l *ByteLiteral) String() string           { return l.Token.Literal }

// NullLiteral can be used in 3 places
// as argument, as return value or within
// infix expression; '==' or '!='
type NullLiteral struct {
	Token token.Token
	Value string

	// Set by semsis. It can either be of type 'null' or
	// if literal used as argument or return value its
	// type is set to the expected type of argument or
	// return value. This is done to ensure the appropriate
	// IR can be generated.
	T types.TypeSpec
}

func (l *NullLiteral) expressionNode() {}
func (l *NullLiteral) literalNode()    {}

// null doesnt have a type
func (l *NullLiteral) Type() types.TypeSpec     { return l.T }
func (l *NullLiteral) SetType(t types.TypeSpec) { l.T = t }
func (l *NullLiteral) TokenLiteral() string     { return l.Token.Literal }
func (l *NullLiteral) String() string {
	return l.Token.Literal
}

type ArrayLiteral struct {
	Token  token.Token
	Values []Expression
	T      types.TypeSpec

	// Set by semantic analysis
	Escapes VariableEscape
}

func (l *ArrayLiteral) expressionNode()          {}
func (l *ArrayLiteral) literalNode()             {}
func (l *ArrayLiteral) Type() types.TypeSpec     { return l.T }
func (l *ArrayLiteral) SetType(t types.TypeSpec) { l.T = t }
func (l *ArrayLiteral) TokenLiteral() string     { return l.Token.Literal }
func (l *ArrayLiteral) String() string {
	var out bytes.Buffer

	out.WriteString("[")
	for i := 0; i < len(l.Values); i++ {
		out.WriteString(l.Values[i].String())
		if i != len(l.Values)-1 {
			out.WriteString(",")
		}
	}
	out.WriteString("]")
	return out.String()
}

type StructLiteral struct {
	Token token.Token
	// Same name as in struct statement.
	// Can be nil if anonymous struct.
	Name   *Identifier
	Fields []*StructFieldLiteral

	// Set by semantic analysis
	T       types.TypeSpec
	Escapes VariableEscape
}

func (s *StructLiteral) expressionNode()          {}
func (l *StructLiteral) literalNode()             {}
func (s *StructLiteral) Type() types.TypeSpec     { return s.T }
func (s *StructLiteral) SetType(t types.TypeSpec) { s.T = t }
func (s *StructLiteral) TokenLiteral() string     { return s.Token.Literal }
func (s *StructLiteral) String() string {
	var out bytes.Buffer

	if s.Name != nil {
		out.WriteString(s.Name.String())
	}
	out.WriteString("{")
	for i, f := range s.Fields {
		out.WriteString(f.String())
		if i != len(s.Fields)-1 {
			out.WriteString(", ")
		}
	}
	out.WriteString("}")
	return out.String()
}

type StructFieldLiteral struct {
	Token token.Token
	Index int
	Name  *Identifier
	Value Expression

	// Set by semsis
	T types.TypeSpec
}

func (s *StructFieldLiteral) expressionNode()          {}
func (l *StructFieldLiteral) literalNode()             {}
func (s *StructFieldLiteral) Type() types.TypeSpec     { return s.T }
func (s *StructFieldLiteral) SetType(t types.TypeSpec) { s.T = t }
func (s *StructFieldLiteral) TokenLiteral() string     { return s.Token.Literal }
func (s *StructFieldLiteral) String() string {
	var out bytes.Buffer
	if s.Name != nil {
		out.WriteString(s.Name.String())
		if s.T != nil {
			out.WriteString(" " + s.T.String())
		}
		out.WriteString(": ")
	} else if s.T != nil {
		out.WriteString(s.T.String() + ": ")
	}
	out.WriteString(s.Value.String())

	return out.String()
}

type WildcardLiteral struct {
	Token token.Token
}

func (s *WildcardLiteral) expressionNode()          {}
func (l *WildcardLiteral) literalNode()             {}
func (s *WildcardLiteral) Type() types.TypeSpec     { return nil }
func (s *WildcardLiteral) SetType(t types.TypeSpec) {}
func (s *WildcardLiteral) TokenLiteral() string     { return s.Token.Literal }
func (s *WildcardLiteral) String() string           { return s.TokenLiteral() }

// ---------- //
// Attributes //
// ---------- //

type Attribute interface {
	String() string
	Equal(t AttributeType) bool
}

type AttributeType uint8

const (
	ExternC AttributeType = iota
	InlineNever
	InlineHint
	InlineAlways
)

type BasicAttribute struct {
	Type AttributeType
}

func (a *BasicAttribute) Equal(at AttributeType) bool {
	return a.Type == at
}
func (a *BasicAttribute) String() string {
	switch a.Type {
	case ExternC:
		return "@extern(c)"
	case InlineNever:
		return "@inline(never)"
	case InlineHint:
		return "@inline(hint)"
	default:
		return "@inline(always)"
	}
}

// ------- //
// Sepcial //
// ------- //

type Unreachable struct{}

func (n *Unreachable) TokenLiteral() string { return "unreachable" }
func (n *Unreachable) String() string       { return "unreachable" }

func (l *Library) Pos() token.Pos {
	return l.Token.Position
}

func (f *FileFormat) Pos() token.Pos {
	return f.Token.Position
}

func (s *ImportStatement) Pos() token.Pos {
	return s.Token.Position
}

func (s *TypeDefinitionStatement) Pos() token.Pos {
	return s.Token.Position
}

func (s *TypeAliasStatement) Pos() token.Pos {
	return s.Token.Position
}

func (s *GenericStructStatement) Pos() token.Pos {
	return s.Token.Position
}

func (s *StructStatement) Pos() token.Pos {
	return s.Token.Position
}

func (s *EnumStatement) Pos() token.Pos {
	return s.Token.Position
}

func (s *UnionStatement) Pos() token.Pos {
	return s.Token.Position
}

func (s *ParameterStatement) Pos() token.Pos {
	return s.Name.Pos()
}

func (bs *BlockStatement) Pos() token.Pos {
	return bs.Token.Position
}

func (s *DeclarationStatement) Pos() token.Pos {
	if s.Token.Type != token.ILLEGAL {
		return s.Token.Position
	}
	return s.Assignee.Pos()
}

func (s *AssignmentStatement) Pos() token.Pos {
	if len(s.Declerations) > 0 {
		return s.Declerations[0].Pos()
	}
	return token.Pos(0)
}

func (i *Identifier) Pos() token.Pos {
	return i.Token.Position
}

func (s *ReturnStatement) Pos() token.Pos {
	return s.Token.Position
}

func (s *ForStatement) Pos() token.Pos {
	return s.Token.Position
}

func (s *ForRangeStatement) Pos() token.Pos {
	return s.Token.Position
}

func (s *WhileStatement) Pos() token.Pos {
	return s.Token.Position
}

func (s *KeywordStatement) Pos() token.Pos {
	return s.Token.Position
}

func (s *StructFieldStatement) Pos() token.Pos {
	if s.Name != nil {
		return s.Name.Pos()
	}
	return token.Pos(0)
}

func (s *DeferStatement) Pos() token.Pos {
	return s.Token.Position
}

func (s *MatchExpressionStatement) Pos() token.Pos {
	return s.Token.Position
}

func (mc *MatchCase) Pos() token.Pos {
	return mc.Token.Position
}

func (fl *FunctionExpression) Pos() token.Pos {
	return fl.Token.Position
}

func (s *IfElseExpression) Pos() token.Pos {
	return s.Token.Position
}

func (e *ConditionalExpression) Pos() token.Pos {
	return e.Token.Position
}

func (e *FunctionCallExpression) Pos() token.Pos {
	return e.Token.Position
}

func (e *PrefixExpression) Pos() token.Pos {
	return e.Token.Position
}

func (e *InfixExpression) Pos() token.Pos {
	return e.Token.Position
}

func (e *PostfixExpression) Pos() token.Pos {
	return e.Token.Position
}

func (e *DotExpression) Pos() token.Pos {
	return e.Token.Position
}

func (e *CatchExpression) Pos() token.Pos {
	return e.Token.Position
}

func (e *IndexExpression) Pos() token.Pos {
	return e.Token.Position
}

func (e *SliceExpression) Pos() token.Pos {
	return e.Token.Position
}

func (e *CopyExpression) Pos() token.Pos {
	return e.Token.Position
}

func (e *CopyUpdateExpression) Pos() token.Pos {
	return e.Token.Position
}

func (e *TypeLiteral) Pos() token.Pos {
	return e.Token.Position
}

func (e *TypeCastExpression) Pos() token.Pos {
	return e.Token.Position
}

func (e *UseExpression) Pos() token.Pos {
	return e.Token.Position
}

func (e *Comment) Pos() token.Pos {
	return e.Token.Position
}

func (l *IntegerLiteral) Pos() token.Pos {
	return l.Token.Position
}

func (l *BooleanLiteral) Pos() token.Pos {
	return l.Token.Position
}

func (l *StringLiteral) Pos() token.Pos {
	return l.Token.Position
}

func (l *CharacterLiteral) Pos() token.Pos {
	return l.Token.Position
}

func (l *FloatLiteral) Pos() token.Pos {
	return l.Token.Position
}

func (l *ByteLiteral) Pos() token.Pos {
	return l.Token.Position
}

func (l *NullLiteral) Pos() token.Pos {
	return l.Token.Position
}

func (l *ArrayLiteral) Pos() token.Pos {
	return l.Token.Position
}

func (s *StructLiteral) Pos() token.Pos {
	return s.Token.Position
}

func (s *StructFieldLiteral) Pos() token.Pos {
	return s.Token.Position
}

func (s *WildcardLiteral) Pos() token.Pos {
	return s.Token.Position
}

func (n *Unreachable) Pos() token.Pos {
	return token.Pos(0)
}
