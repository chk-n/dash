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
	Type() types.Type
	SetType(t types.Type)
}

type Library struct {
	Token token.Token
	Name  *Identifier
	Nodes []Node
}

func (l *Library) statementNode()       {}
func (l *Library) TokenLiteral() string { return l.Token.Literal }
func (l *Library) String() string {
	var out bytes.Buffer
	out.WriteString(l.TokenLiteral() + " ")
	out.WriteString(l.Name.TokenLiteral())

	if len(l.Nodes) > 0 {
		out.WriteString(" ")
	}
	for i := range len(l.Nodes) {
		out.WriteString(l.Nodes[i].String())
		if i != len(l.Nodes)-1 {
			out.WriteString(" ")
		}
	}
	return out.String()
}

func (l *Library) Exports() map[string]types.Type {
	export := make(map[string]types.Type)
	for _, n := range l.Nodes {
		switch n := n.(type) {
		case *StructStatement:
			if n.Public {
				export[n.Name.Value] = n.T
			}
		case *UnionStatement:
			if n.Public {
				export[n.Name.Value] = n.T
			}
		case *EnumStatement:
			if n.Public {
				export[n.Name.Value] = n.T
			}
		case *FunctionExpression:
			if n.Public {
				export[n.Name.Value] = n.T
			}
		case *TypeDefinitionStatement:
			if n.Public {
				export[n.Name.Value] = n.T
			}
		case *TypeAliasStatement:
			if n.Public {
				export[n.Name.Value] = n.T
			}
		case *ErrorStatement:
			if n.Public {
				export[n.Name.Value] = n.T
			}
		case *AssignmentStatement:
			if n.Public {
				// for now only assignments in the form
				// pub let x, let y = 1, 2
				// allowed meaning no function calls
				for i, v := range n.Declerations {
					switch v := v.(type) {
					case *DeclarationStatement:
						export[v.Assignee.String()] = n.Values[i].Type()
					}
				}
			}

		}
	}
	return export
}

// ------------//
// Statements //
// ------------//

type UseStatement struct {
	Token token.Token
	Name  *StringLiteral
}

func (s *UseStatement) statementNode()       {}
func (s *UseStatement) TokenLiteral() string { return s.Token.Literal }
func (s *UseStatement) String() string {
	var out bytes.Buffer

	out.WriteString(s.TokenLiteral() + " ")
	out.WriteString(s.Name.String())

	return out.String()
}

// type <name> <type>
type TypeDefinitionStatement struct {
	Public         bool
	Token          token.Token
	Name           *Identifier
	UnderlyingType types.Type
	Guard          Expression

	// Set by semsis
	T types.Type
}

func (s *TypeDefinitionStatement) statementNode()       {}
func (s *TypeDefinitionStatement) TokenLiteral() string { return s.Token.Literal }
func (s *TypeDefinitionStatement) Type() types.Type     { return s.T }
func (s *TypeDefinitionStatement) String() string {
	var out bytes.Buffer
	if s.Public {
		out.WriteString("pub ")
	}
	out.WriteString(s.TokenLiteral() + " ")
	out.WriteString(s.Name.TokenLiteral() + " ")
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
	UnderlyingType types.Type

	// Set by semsis
	T types.Type
}

func (s *TypeAliasStatement) statementNode()       {}
func (s *TypeAliasStatement) TokenLiteral() string { return s.Token.Literal }
func (s *TypeAliasStatement) Type() types.Type {
	return s.T
}
func (s *TypeAliasStatement) String() string {
	var out bytes.Buffer
	if s.Public {
		out.WriteString("pub ")
	}
	out.WriteString(s.TokenLiteral() + " ")
	out.WriteString(s.Name.TokenLiteral() + " ")
	out.WriteString(s.UnderlyingType.String())
	return out.String()
}

// struct[name] { }
type StructStatement struct {
	Public            bool
	Token             token.Token
	Name              *Identifier
	GenericParameters []*GenericParameter
	Fields            []*StructFieldStatement
	T                 *types.Struct
}

func (s *StructStatement) statementNode()       {}
func (s *StructStatement) TokenLiteral() string { return s.Token.Literal }
func (s *StructStatement) Type() types.Type {
	return s.T
}
func (s *StructStatement) String() string {
	var out bytes.Buffer
	if s.Public {
		out.WriteString("pub ")
	}
	out.WriteString(s.TokenLiteral() + " ")
	out.WriteString(s.Name.TokenLiteral())
	// print generic parameters
	if len(s.GenericParameters) > 0 {
		out.WriteString("[")
		for i, gp := range s.GenericParameters {
			out.WriteString(gp.String())
			if i != len(s.GenericParameters)-1 {
				out.WriteString(", ")
			}
		}
		out.WriteString("]")
	}
	out.WriteString(" {")
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
	out.WriteString(s.Name.TokenLiteral() + " {")
	for i, f := range s.Fields {
		out.WriteString(f.TokenLiteral())
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
	out.WriteString(s.Name.TokenLiteral() + " {")
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
	Type types.Type // types e.g. int, float
}

func (s *ParameterStatement) statementNode()       {}
func (s *ParameterStatement) TokenLiteral() string { return s.Name.Token.Literal }
func (s *ParameterStatement) String() string {
	var out bytes.Buffer
	out.WriteString(s.Name.TokenLiteral())
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
type DeclarationStatement struct {
	Token    token.Token // `let`, `var`
	Assignee Expression
}

func (s *DeclarationStatement) statementNode()       {}
func (s *DeclarationStatement) TokenLiteral() string { return s.Token.Literal }
func (s *DeclarationStatement) String() string {
	var out bytes.Buffer
	out.WriteString(s.TokenLiteral() + " ")
	out.WriteString(s.Assignee.String())
	if s.Assignee.Type() != nil {
		out.WriteString(" " + s.Assignee.Type().String())
	}
	return out.String()
}

// let x, v, var z = 1, 2, 3
type AssignmentStatement struct {
	Public bool
	// can only be:
	// *DeclarationStatement, *Identifier,
	// *IndexExpression, *SliceExpression, or
	// *DotExpression
	Declerations []Node
	Values       []Expression
}

func (s *AssignmentStatement) statementNode()       {}
func (s *AssignmentStatement) TokenLiteral() string { return "" }
func (s *AssignmentStatement) TypeAt(i int) types.Type {
	var typ types.Type
	for j := 0; j < i; j++ {
		switch t := s.Values[j].Type().(type) {
		case *types.Multi:
			for _, t := range t.Ts {
				typ = t
				j++
			}
		default:
			typ = t
		}
	}
	return typ
}
func (s *AssignmentStatement) VarNameAt(i int) string {
	if i > len(s.Declerations)-1 {
		panic("this is a compiler error. please report")
	}
	switch decl := s.Declerations[i].(type) {
	case *Identifier:
		return decl.TokenLiteral()
	case *DeclarationStatement:
		return decl.Assignee.TokenLiteral()
	case *IndexExpression:
		return decl.TokenLiteral()
	case *SliceExpression:
		return decl.TokenLiteral()
	case *DotExpression:
		return decl.TokenLiteral()
	}
	panic("this is a compiler error. please report")
}
func (s *AssignmentStatement) IsVarAt(i int) bool {
	if i > len(s.Declerations)-1 {
		panic("this is a compiler error. please report")
	}

	decl, ok := s.Declerations[i].(*DeclarationStatement)
	if !ok {
		return false
	}
	return decl.Token.Type == token.VAR
}

func (s *AssignmentStatement) SetTypeAt(i int, t types.Type) {
	switch decl := s.Declerations[i].(type) {
	case Expression:
		decl.SetType(t)
	case *DeclarationStatement:
		decl.Assignee.SetType(t)
	default:
		panic("this is a compiler error. please report")
	}
}
func (s *AssignmentStatement) String() string {
	var out bytes.Buffer

	if s.Public {
		out.WriteString("pub ")
	}
	for i, decl := range s.Declerations {
		switch decl := decl.(type) {
		case Expression:
			out.WriteString(decl.String())
			if decl.Type() != nil {
				out.WriteString(" " + decl.Type().String())
			}
		default:
			out.WriteString(decl.String())
		}
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
	T     types.Type
}

func (i *Identifier) expressionNode()  {}
func (i *Identifier) Type() types.Type { return i.T }
func (i *Identifier) SetType(t types.Type) {
	i.T = t
}
func (i *Identifier) TokenLiteral() string { return i.Token.Literal }
func (i *Identifier) String() string {
	// if i.T != nil {
	// 	return i.Value + " " + i.T.String()
	// }
	return i.Value
}

type ReturnStatement struct {
	Token  token.Token // the 'return' token
	Values []Expression
}

func (s *ReturnStatement) statementNode() {}
func (s *ReturnStatement) ReturnTypes() []types.Type {
	var typs []types.Type

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
	for i := range len(s.Values) {
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
		for i := range len(s.Variables) {
			if i%2 != 0 {
				out.WriteString(",")
			}
			out.WriteString(" " + s.Variables[i].TokenLiteral())
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

type KeywordStatement struct {
	Token token.Token
}

func (s *KeywordStatement) statementNode()       {}
func (s *KeywordStatement) TokenLiteral() string { return s.Token.Literal }
func (s *KeywordStatement) String() string       { return s.Token.Literal }

type StructFieldStatement struct {
	// Can be nil if unnamed struct definition
	Name *Identifier
	Type types.Type
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

func (e *DeferStatement) expressionNode()      {}
func (e *DeferStatement) Type() types.Type     { return nil }
func (e *DeferStatement) SetType(t types.Type) {}
func (e *DeferStatement) TokenLiteral() string { return e.Token.Literal }
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
	T types.Type
}

func (s *MatchExpressionStatement) expressionNode()      {}
func (s *MatchExpressionStatement) statementNode()       {}
func (s *MatchExpressionStatement) Type() types.Type     { return s.T }
func (s *MatchExpressionStatement) SetType(t types.Type) { s.T = t }
func (s *MatchExpressionStatement) TokenLiteral() string { return s.Token.Literal }
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
	Token      token.Token // case or else token
	Predicates []Expression
	Body       []Node

	// Set by semsis only if used as
	// expression.
	T types.Type
}

func (mc *MatchCase) expressionNode()      {}
func (mc *MatchCase) Type() types.Type     { return mc.T }
func (mc *MatchCase) SetType(t types.Type) {}
func (mc *MatchCase) TokenLiteral() string { return mc.Token.Literal }
func (mc *MatchCase) String() string {
	var out bytes.Buffer

	out.WriteString(mc.Token.Literal)
	if len(mc.Predicates) > 0 {
		out.WriteString(" ")
		for i, pred := range mc.Predicates {
			out.WriteString(pred.String())
			if i != len(mc.Predicates)-1 {
				out.WriteString(", ")
			}
		}
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

type ErrorStatement struct {
	Token  token.Token // the 'error' token
	Name   *Identifier
	Params []*ParameterStatement
	Public bool

	T *types.Error
}

func (es *ErrorStatement) statementNode() {}
func (es *ErrorStatement) Type() types.Type {
	return es.T
}
func (es *ErrorStatement) TokenLiteral() string { return es.Token.Literal }
func (es *ErrorStatement) String() string {
	var out bytes.Buffer

	if es.Public {
		out.WriteString("pub ")
	}
	out.WriteString("error ")
	out.WriteString(es.Name.TokenLiteral())

	// Add parameters if they exist
	if len(es.Params) > 0 {
		out.WriteString("{")
		params := []string{}
		for _, p := range es.Params {
			params = append(params, p.String())
		}
		out.WriteString(strings.Join(params, ", "))
		out.WriteString("}")
	}

	return out.String()
}

type RaiseStatement struct {
	Token token.Token
	Error Expression
}

func (s *RaiseStatement) statementNode()       {}
func (s *RaiseStatement) Type() types.Type     { return &types.Error{} }
func (s *RaiseStatement) SetType(t types.Type) {}
func (s *RaiseStatement) TokenLiteral() string { return s.Token.Literal }
func (s *RaiseStatement) String() string {
	var out bytes.Buffer
	out.WriteString(s.Token.Literal)
	out.WriteString(" ")
	if s.Error != nil {
		out.WriteString(s.Error.String())
	}
	return out.String()
}

type GenericParameter struct {
	Name       *Identifier
	Constraint types.Type
}

func (g *GenericParameter) String() string {
	var out bytes.Buffer

	out.WriteString(g.Name.String())
	if g.Constraint != nil {
		out.WriteString(" ")
		out.WriteString(g.Constraint.String())
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
	Attributes        []Attribute
	Public            bool        // pub fn
	Token             token.Token // The 'fn' token
	Name              *Identifier
	GenericParameters []*GenericParameter
	Arguments         []*ParameterStatement
	ErrorProne        bool
	ReturnValues      []*TypeLiteral

	Body *BlockStatement

	// Set by semantic analysis based on arguments
	T           types.Type
	IsVariadic  bool
	IsAnonymous bool
}

func (fl *FunctionExpression) expressionNode()      {}
func (fl *FunctionExpression) SetType(t types.Type) {}
func (fl *FunctionExpression) HasAttribute(attrT AttributeType) bool {
	for _, attr := range fl.Attributes {
		if attr.Equal(attrT) {
			return true
		}
	}
	return false
}
func (fl *FunctionExpression) Type() types.Type     { return fl.T }
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
		out.WriteString(" " + fl.Name.TokenLiteral())
	}

	// print generic parameters
	if len(fl.GenericParameters) > 0 {
		out.WriteString("[")
		for i, gp := range fl.GenericParameters {
			out.WriteString(gp.String())
			if i != len(fl.GenericParameters)-1 {
				out.WriteString(", ")
			}
		}
		out.WriteString("]")
	}

	// print arguments
	args := make([]string, 0, len(fl.Arguments))
	out.WriteString("(")
	for _, arg := range fl.Arguments {
		args = append(args, arg.String())
	}
	out.WriteString(strings.Join(args, ","))
	out.WriteString(")")
	if fl.ErrorProne {
		out.WriteString("!")
	}
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
	T types.Type
}

func (s *IfElseExpression) expressionNode()      {}
func (s *IfElseExpression) Type() types.Type     { return s.T }
func (s *IfElseExpression) SetType(t types.Type) { s.T = t }
func (s *IfElseExpression) TokenLiteral() string { return s.Token.Literal }
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
	Token          token.Token // The function identifier
	TypeParameters []types.Type
	Arguments      []Expression
	ReturnTypes    []types.Type

	// Set by semantic analysis
	T             types.Type
	IsAnonymousFn bool
}

func (e *FunctionCallExpression) expressionNode() {}
func (e *FunctionCallExpression) SetType(t types.Type) {
	e.T = t
}
func (e *FunctionCallExpression) Type() types.Type     { return e.T }
func (e *FunctionCallExpression) TokenLiteral() string { return e.Token.Literal }
func (e *FunctionCallExpression) String() string {
	var out bytes.Buffer
	out.WriteString(e.Token.Literal)

	// Print type parameters if present
	if len(e.TypeParameters) > 0 {
		out.WriteString("[")
		typeParams := make([]string, 0, len(e.TypeParameters))
		for _, tp := range e.TypeParameters {
			typeParams = append(typeParams, tp.String())
		}
		out.WriteString(strings.Join(typeParams, ", "))
		out.WriteString("]")
	}

	args := make([]string, 0, len(e.Arguments))
	for _, arg := range e.Arguments {
		args = append(args, arg.String())
	}
	out.WriteString("(")
	out.WriteString(strings.Join(args, ","))
	out.WriteString(")")

	return out.String()
}

type PrefixExpression struct {
	Token    token.Token // The prefix token, e.g. ! Operator string
	Operator string
	Right    Expression

	// Set by semsis
	T types.Type
}

func (e *PrefixExpression) expressionNode()      {}
func (e *PrefixExpression) Type() types.Type     { return e.T }
func (e *PrefixExpression) SetType(t types.Type) { e.T = t }
func (e *PrefixExpression) TokenLiteral() string { return e.Token.Literal }
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
	T types.Type
}

func (e *InfixExpression) expressionNode()      {}
func (e *InfixExpression) Type() types.Type     { return e.T }
func (e *InfixExpression) SetType(t types.Type) { e.T = t }
func (e *InfixExpression) TokenLiteral() string { return e.Token.Literal }
func (e *InfixExpression) String() string {
	var out bytes.Buffer

	out.WriteString("(")

	// Special handling for assignment operations to include type annotations
	if e.Operator == "=" {
		out.WriteString(e.Left.String())
		if e.Left.Type() != nil {
			out.WriteString(" " + e.Left.Type().String())
		}
		out.WriteString(" " + e.Operator + " ")
	} else {
		out.WriteString(e.Left.String())
		out.WriteString(" " + e.Operator + " ")
	}

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
	T types.Type
}

func (e *PostfixExpression) expressionNode()  {}
func (e *PostfixExpression) Type() types.Type { return e.T }
func (e *PostfixExpression) SetType(t types.Type) {
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
	T types.Type
}

func (e *DotExpression) expressionNode()  {}
func (e *DotExpression) Type() types.Type { return e.T }
func (e *DotExpression) SetType(t types.Type) {
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

type TryExpression struct {
	Token token.Token
	Right Expression

	T types.Type
}

func (s *TryExpression) expressionNode()  {}
func (s *TryExpression) Type() types.Type { return s.Right.Type() }
func (s *TryExpression) SetType(t types.Type) {
	s.T = t
}
func (s *TryExpression) TokenLiteral() string { return s.Token.Literal }
func (s *TryExpression) String() string {
	var out bytes.Buffer
	out.WriteString(s.Token.Literal)
	out.WriteString(" ")
	out.WriteString(s.Right.String())
	return out.String()
}

type CatchExpression struct {
	Token token.Token
	Left  Expression
	Ident *Identifier
	Block *BlockStatement
}

func (s *CatchExpression) expressionNode()      {}
func (s *CatchExpression) Type() types.Type     { return &types.Function{} } // NOTE: this might need to be changed
func (s *CatchExpression) SetType(t types.Type) {}
func (s *CatchExpression) TokenLiteral() string { return s.Token.Literal }
func (s *CatchExpression) String() string {
	var out bytes.Buffer
	out.WriteString(s.Left.String() + " ")
	out.WriteString(s.Token.Literal + " ")
	out.WriteString(s.Ident.TokenLiteral() + " ")
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

	T types.Type
}

func (e *IndexExpression) expressionNode() {}
func (e *IndexExpression) SetType(t types.Type) {
	e.T = t
}

// Returns type that a variable would have when index expression executed
// v [][]i64 = a[0][0]
// Type() == i64
func (e *IndexExpression) Type() types.Type {
	return e.T
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
	// Ts []types.Type
}

func (e *SliceExpression) expressionNode()      {}
func (e *SliceExpression) Type() types.Type     { return e.Left.Type() }
func (e *SliceExpression) SetType(t types.Type) {}
func (e *SliceExpression) TokenLiteral() string { return e.Token.Literal }
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

func (e *CopyExpression) expressionNode()      {}
func (e *CopyExpression) Type() types.Type     { return e.Ident.Type() }
func (e *CopyExpression) SetType(t types.Type) {}
func (e *CopyExpression) TokenLiteral() string { return e.Token.Literal }
func (e *CopyExpression) String() string {
	return e.Ident.String() + e.TokenLiteral()
}

// x^ { x.field = 2 }
type CopyUpdateExpression struct {
	Token token.Token
	Ident Expression
	Block *BlockStatement
}

func (e *CopyUpdateExpression) expressionNode()      {}
func (e *CopyUpdateExpression) Type() types.Type     { return e.Ident.Type() }
func (e *CopyUpdateExpression) SetType(t types.Type) {}
func (e *CopyUpdateExpression) TokenLiteral() string { return e.Token.Literal }
func (e *CopyUpdateExpression) String() string {
	var out bytes.Buffer

	out.WriteString(e.Ident.String())
	out.WriteString(e.TokenLiteral())
	out.WriteString(" " + e.Block.String())

	return out.String()
}

type TypeLiteral struct {
	Token token.Token
	T     types.Type
}

func (e *TypeLiteral) expressionNode()      {}
func (e *TypeLiteral) Type() types.Type     { return e.T }
func (e *TypeLiteral) SetType(t types.Type) { e.T = t }
func (e *TypeLiteral) TokenLiteral() string { return e.Token.Literal }
func (e *TypeLiteral) String() string {
	return e.T.String()
}

type TypeCastExpression struct {
	Token    token.Token
	Argument Expression
	Typ      types.Type
}

func (e *TypeCastExpression) expressionNode()      {}
func (e *TypeCastExpression) Type() types.Type     { return e.Typ }
func (e *TypeCastExpression) SetType(t types.Type) { e.Typ = t }
func (e *TypeCastExpression) TokenLiteral() string { return e.Token.Literal }
func (e *TypeCastExpression) String() string {
	var out bytes.Buffer

	out.WriteString(e.Typ.String())
	out.WriteString("(" + e.Argument.String() + ")")

	return out.String()
}

type Comment struct {
	Token token.Token
}

func (e *Comment) expressionNode()      {}
func (e *Comment) Type() types.Type     { return nil }
func (e *Comment) SetType(t types.Type) {}
func (e *Comment) TokenLiteral() string { return e.Token.Literal }
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
	T types.Type
}

func (l *IntegerLiteral) expressionNode()      {}
func (l *IntegerLiteral) literalNode()         {}
func (l *IntegerLiteral) Type() types.Type     { return l.T }
func (l *IntegerLiteral) SetType(t types.Type) { l.T = t }
func (l *IntegerLiteral) TokenLiteral() string { return l.Token.Literal }
func (l *IntegerLiteral) String() string       { return l.Token.Literal }

type BooleanLiteral struct {
	Token token.Token
	Value bool
}

func (l *BooleanLiteral) expressionNode()      {}
func (l *BooleanLiteral) Type() types.Type     { return &types.ConstBool }
func (l *BooleanLiteral) SetType(t types.Type) {}
func (l *BooleanLiteral) TokenLiteral() string { return l.Token.Literal }
func (l *BooleanLiteral) String() string       { return l.Token.Literal }

type StringLiteral struct {
	Token token.Token
}

func (l *StringLiteral) expressionNode()      {}
func (l *StringLiteral) literalNode()         {}
func (l *StringLiteral) Type() types.Type     { return &types.ConstString }
func (l *StringLiteral) SetType(t types.Type) {}
func (l *StringLiteral) TokenLiteral() string { return l.Token.Literal }
func (l *StringLiteral) String() string {
	return `"` + l.Token.Literal + `"`
}

type CharacterLiteral struct {
	Token token.Token
	Value int32

	// Set by semsis
	T types.Type
}

func (l *CharacterLiteral) expressionNode()      {}
func (l *CharacterLiteral) literalNode()         {}
func (l *CharacterLiteral) Type() types.Type     { return l.T }
func (l *CharacterLiteral) SetType(t types.Type) { l.T = t }
func (l *CharacterLiteral) TokenLiteral() string { return l.Token.Literal }
func (l *CharacterLiteral) String() string {
	return `'` + l.Token.Literal + `'`
}

type FloatLiteral struct {
	Token token.Token
	Value float64

	// Set by semsis
	T types.Type
}

func (l *FloatLiteral) expressionNode()  {}
func (l *FloatLiteral) literalNode()     {}
func (l *FloatLiteral) Type() types.Type { return l.T }
func (l *FloatLiteral) SetType(t types.Type) {
	l.T = t
}
func (l *FloatLiteral) TokenLiteral() string { return l.Token.Literal }
func (l *FloatLiteral) String() string       { return l.Token.Literal }

type ByteLiteral struct {
	Token token.Token
	Value byte
}

func (l *ByteLiteral) expressionNode()      {}
func (l *ByteLiteral) literalNode()         {}
func (l *ByteLiteral) Type() types.Type     { return &types.ConstByte }
func (l *ByteLiteral) SetType(t types.Type) {}
func (l *ByteLiteral) TokenLiteral() string { return l.Token.Literal }
func (l *ByteLiteral) String() string       { return l.Token.Literal }

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
	T types.Type
}

func (l *NullLiteral) expressionNode() {}
func (l *NullLiteral) literalNode()    {}

// null doesnt have a type
func (l *NullLiteral) Type() types.Type     { return l.T }
func (l *NullLiteral) SetType(t types.Type) { l.T = t }
func (l *NullLiteral) TokenLiteral() string { return l.Token.Literal }
func (l *NullLiteral) String() string {
	return l.Token.Literal
}

type ArrayLiteral struct {
	Token  token.Token
	Values []Expression
	T      types.Type

	// Set by semantic analysis
	Escapes VariableEscape
}

func (l *ArrayLiteral) expressionNode()      {}
func (l *ArrayLiteral) literalNode()         {}
func (l *ArrayLiteral) Type() types.Type     { return l.T }
func (l *ArrayLiteral) SetType(t types.Type) { l.T = t }
func (l *ArrayLiteral) TokenLiteral() string { return l.Token.Literal }
func (l *ArrayLiteral) String() string {
	var out bytes.Buffer

	out.WriteString("[")
	for i := range len(l.Values) {
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
	Name           Expression // can be Identifier or DotExpression
	TypeParameters []types.Type
	Fields         []*StructFieldLiteral

	// Set by semantic analysis
	T       types.Type
	Escapes VariableEscape
}

func (s *StructLiteral) expressionNode()      {}
func (l *StructLiteral) literalNode()         {}
func (s *StructLiteral) Type() types.Type     { return s.T }
func (s *StructLiteral) SetType(t types.Type) { s.T = t }
func (s *StructLiteral) TokenLiteral() string { return s.Token.Literal }
func (s *StructLiteral) String() string {
	var out bytes.Buffer

	if s.Name != nil {
		out.WriteString(s.Name.String())
	}

	// Print type parameters if present
	if len(s.TypeParameters) > 0 {
		out.WriteString("[")
		typeParams := make([]string, 0, len(s.TypeParameters))
		for _, tp := range s.TypeParameters {
			typeParams = append(typeParams, tp.String())
		}
		out.WriteString(strings.Join(typeParams, ", "))
		out.WriteString("]")
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
	T types.Type
}

func (s *StructFieldLiteral) expressionNode()      {}
func (l *StructFieldLiteral) literalNode()         {}
func (s *StructFieldLiteral) Type() types.Type     { return s.T }
func (s *StructFieldLiteral) SetType(t types.Type) { s.T = t }
func (s *StructFieldLiteral) TokenLiteral() string { return s.Token.Literal }
func (s *StructFieldLiteral) String() string {
	var out bytes.Buffer
	if s.Name != nil {
		out.WriteString(s.Name.TokenLiteral())
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

func (s *WildcardLiteral) expressionNode()      {}
func (l *WildcardLiteral) literalNode()         {}
func (s *WildcardLiteral) Type() types.Type     { return nil }
func (s *WildcardLiteral) SetType(t types.Type) {}
func (s *WildcardLiteral) TokenLiteral() string { return s.Token.Literal }
func (s *WildcardLiteral) String() string       { return s.TokenLiteral() }

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
	Test
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
	case InlineAlways:
		return "@inline(always)"
	case Test:
		return "@test"
	default:
		panic("invalid attribute type")
	}
}

// IsLiteral recursively checks if a node is a literal
func IsLiteral(n Node) bool {
	switch n := n.(type) {
	case Literal:
		return true
	case *PrefixExpression:
		return IsLiteral(n.Right)
	default:
		return false
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

func (s *UseStatement) Pos() token.Pos {
	return s.Token.Position
}

func (s *TypeDefinitionStatement) Pos() token.Pos {
	return s.Token.Position
}

func (s *TypeAliasStatement) Pos() token.Pos {
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

func (s *KeywordStatement) Pos() token.Pos {
	return s.Token.Position
}

func (s *StructFieldStatement) Pos() token.Pos {
	if s.Name != nil {
		return s.Name.Pos()
	}
	return token.Pos(0)
}

func (s *ErrorStatement) Pos() token.Pos {
	return s.Token.Position
}

func (s *DeferStatement) Pos() token.Pos {
	return s.Token.Position
}

func (s *RaiseStatement) Pos() token.Pos {
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

func (e *TryExpression) Pos() token.Pos {
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
