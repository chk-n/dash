package token

type Type uint8

type Pos uint64

func NewPos(l, c int) Pos {
	return Pos((uint64(l) << 32) | uint64(c))
}
func (p Pos) Line() int {
	return int(p >> 32)
}

func (p Pos) Column() int {
	return int(p & 0xFFFFFFFF)
}

type Token struct {
	Type     Type
	Literal  string
	Position Pos
}

const (
	ILLEGAL Type = iota
	EOF

	IDENT

	// main
	MAIN
	// FUNCTION_IDENT
	// FUNCTION_IDENT

	// ------------------- //
	// Literal type tokens //
	// ------------------- //

	INT
	HEX
	FLOAT
	STRING
	RAW_STRING
	BOOL
	BYTE
	NULL
	CHAR

	// ------------- //
	// Type keywords //
	// ------------- //
	//many
	MANYTYPE
	// int
	INTTYPE
	// i8
	I8TYPE
	// i16
	I16TYPE
	// i32
	I32TYPE
	// i64
	I64TYPE
	// u8
	U8TYPE
	// u16
	U16TYPE
	// u32
	U32TYPE
	// u64
	U64TYPE
	// float
	FLOATTYPE
	// f32
	F32TYPE
	// f64
	F64TYPE
	// string
	STRINGTYPE
	// bool
	BOOLTYPE
	// byte
	BYTETYPE
	// char
	CHARTYPE
	// struct
	STRUCTTYPE
	// // []T or [n]T
	// ARRAYTYPE
	// memory
	MEMORYTYPE
	// dirty
	DIRTYTYPE

	// --------- //
	// Operators //
	// --------- //

	// =
	ASSIGN
	// \
	BACKSLASH
	// :
	COLON
	// +
	PLUS
	// -
	MINUS
	// %
	MOD
	// /
	SLASH
	// *
	ASTERISK
	// &
	AMPERSAND
	// &&
	AND
	// ||
	OR
	// !
	BANG
	// >=
	GTE
	// <=
	LTE
	// ==
	EQ
	// !=
	NEQ
	// >
	GT
	// <
	LT
	// |
	BAR
	// <<
	LSHIFT
	// >>
	RSHIFT
	// ^
	CARET
	// ~
	BNOT
	// &^
	BANDNOT
	// ,
	COMMA
	// (
	LPAREN
	// )
	RPAREN
	// {
	LBRACE
	// }
	RBRACE
	// [
	LBRACK
	// ]
	RBRACK
	// .
	DOT
	// ;
	SEMI
	// ?
	OPTIONAL
	// ??
	NULL_COALESCE
	// |>
	PIPE
	// //
	COMMENT
	// /*
	LMCOMMENT
	// */
	RMCOMMENT
	// _
	WILDCARD
	// ..
	RANGE
	// ...
	ELLIPSIS
	// =>
	ARROW
	// ++
	INCR
	// --
	DECR
	// @
	AT

	// -------- //
	// Keywords //
	// -------- //

	// lib
	LIBRARY
	// pub
	PUBLIC
	// struct
	STRUCT
	// enum
	ENUM
	// type
	TYPE
	// fn
	FUNCTION
	// if
	IF
	// else
	ELSE
	// error
	ERROR
	// try
	TRY
	// catch
	CATCH
	// raise
	RAISE
	// defer
	DEFER
	// return
	RETURN
	// for
	FOR
	// in
	IN
	// break
	BREAK
	// next
	NEXT
	// match
	MATCH
	// alias
	ALIAS
	// let
	LET
	// var
	VAR
	// use
	USE
	// case
	CASE
	// union
	UNION

	// else if
	ELSEIF
)

func New(t Type, ch byte) Token {
	return Token{Type: t, Literal: string(ch)}
}

func NewFromLiteral(t Type, lit string) Token {
	return Token{Type: t, Literal: lit}
}

var keywords = map[string]Type{
	"lib":    LIBRARY,
	"main":   MAIN,
	"struct": STRUCT,
	"enum":   ENUM,
	"type":   TYPE,
	"alias":  ALIAS,
	"fn":     FUNCTION,
	"pub":    PUBLIC,
	"if":     IF,
	"else":   ELSE,
	"return": RETURN,
	"defer":  DEFER,
	"match":  MATCH,
	"error":  ERROR,
	"try":    TRY,
	"catch":  CATCH,
	"raise":  RAISE,
	"for":    FOR,
	"in":     IN,
	"break":  BREAK,
	"next":   NEXT,
	"let":    LET,
	"var":    VAR,
	"case":   CASE,
	"union":  UNION,
	"use":    USE,

	//
	"true":  BOOL,
	"false": BOOL,
	"null":  NULL,

	// Types
	"many":   MANYTYPE,
	"int":    INTTYPE,
	"i8":     I8TYPE,
	"i16":    I16TYPE,
	"i32":    I32TYPE,
	"i64":    I64TYPE,
	"u8":     U8TYPE,
	"u16":    U16TYPE,
	"u32":    U32TYPE,
	"u64":    U64TYPE,
	"float":  FLOATTYPE,
	"f32":    F32TYPE,
	"f64":    F64TYPE,
	"string": STRINGTYPE,
	"byte":   BYTETYPE,
	"char":   CHARTYPE,
	"bool":   BOOLTYPE,
	"memory": MEMORYTYPE,
	"dirty":  DIRTYTYPE,

	//
	"_": WILDCARD,
}

func LookupKeyword(k string) Type {
	if tok, ok := keywords[k]; ok {
		return tok
	}
	return IDENT
}
