package parser

import (
	"fmt"
	"strconv"
	"strings"

	"dash-lang.io/src/ast"
	"dash-lang.io/src/internal"
	"dash-lang.io/src/lexer"
	"dash-lang.io/src/token"
	"dash-lang.io/src/types"
)

const (
	_ int = iota
	LOWEST
	PIPE          // |>
	CATCH         // catch
	COLON         // :
	ASSIGN        // =
	OR            // ||
	BITWISE_OR    // |
	BITWISE_XOR   // ^
	AND           // &&
	BITWISE_AND   // &
	EQUALS        // ==
	LESSGREATER   // > or <
	LESSGREATEREQ // >= or <=
	SUM           // +
	SUBTRACT      // -
	PRODUCT       // *
	DIVIDE        // /
	SHIFT         // << >>
	NULL_COALESCE // ??
	PREFIX        // -5, !false, ?x, ~x
	POSTFIX       // x++
	CALL          // myFunction(X)
	SLICE         // a[0]
	DOT           // a.b
)

var precedences = map[token.Type]int{
	token.PIPE:          PIPE,
	token.COLON:         COLON,
	token.EQ:            EQUALS,
	token.NEQ:           EQUALS,
	token.LT:            LESSGREATER,
	token.GT:            LESSGREATER,
	token.GTE:           LESSGREATEREQ,
	token.LTE:           LESSGREATEREQ,
	token.OR:            OR,
	token.AND:           AND,
	token.BAR:           BITWISE_OR,
	token.CARET:         BITWISE_XOR,
	token.AMPERSAND:     BITWISE_AND,
	token.LSHIFT:        SHIFT,
	token.RSHIFT:        SHIFT,
	token.PLUS:          SUM,
	token.MINUS:         SUM,
	token.SLASH:         PRODUCT,
	token.ASTERISK:      PRODUCT,
	token.MOD:           PRODUCT,
	token.ASSIGN:        ASSIGN,
	token.LPAREN:        CALL,
	token.LBRACK:        SLICE,
	token.DOT:           DOT,
	token.OPTIONAL:      POSTFIX,
	token.BANG:          POSTFIX,
	token.NULL_COALESCE: NULL_COALESCE,
	token.CATCH:         CATCH,
}

type (
	prefixParseFn    func() ast.Expression
	infixParseFn     func(ast.Expression) ast.Expression
	postfixParseFn   func(ast.Expression) ast.Expression
	attributeParseFn func() ast.Attribute
	context          uint8
)

const (
	NONE context = iota
	IF_ELSE
	MATCH
)

type Parser struct {
	tknIdx            int
	tkns              []token.Token
	l                 *lexer.Lexer
	curToken          token.Token
	errors            []string
	prefixParseFns    map[token.Type]prefixParseFn
	infixParseFns     map[token.Type]infixParseFn
	postfixParseFns   map[token.Type]postfixParseFn
	attributeParseFns map[string]attributeParseFn

	attributes internal.Stack[ast.Attribute]

	context context
}

func New(l *lexer.Lexer) *Parser {
	p := &Parser{
		l: l,
	}

	p.prefixParseFns = make(map[token.Type]prefixParseFn)
	p.registerPrefix(token.COMMENT, p.parseComment)
	p.registerPrefix(token.BANG, p.parsePrefixExpression)
	p.registerPrefix(token.MINUS, p.parsePrefixExpression)
	p.registerPrefix(token.AMPERSAND, p.parsePrefixExpression)
	p.registerPrefix(token.ASTERISK, p.parsePrefixExpression)
	p.registerPrefix(token.OPTIONAL, p.parsePrefixExpression)
	p.registerPrefix(token.BNOT, p.parsePrefixExpression)
	// Literal parse functions
	p.registerPrefix(token.IDENT, p.parseIdentifierStructLiteralOrFunctionCall)
	p.registerPrefix(token.INT, p.parseIntegerLiteral)
	p.registerPrefix(token.HEX, p.parseHexLiteral)
	p.registerPrefix(token.FLOAT, p.parseFloatLiteral)
	p.registerPrefix(token.BOOL, p.parseBooleanLiteral)
	p.registerPrefix(token.NULL, p.parseNullLiteral)
	p.registerPrefix(token.FUNCTION, p.parseFunctionExpression)
	p.registerPrefix(token.STRING, p.parseStringLiteral)
	p.registerPrefix(token.WILDCARD, p.parseWildcardLiteral)
	p.registerPrefix(token.CHAR, p.parseCharacterLiteral)
	// p.registerPrefix(token.BYTE, p.parseByteLiteral)
	// p.registerPrefic(token.BIT, p.parseBitLiteral)
	p.registerPrefix(token.LPAREN, p.parseGroupedExpression)
	p.registerPrefix(token.IF, p.parseIfElseExpression)
	p.registerPrefix(token.MATCH, p.parseMatchExpression)
	p.registerPrefix(token.LBRACE, p.parseAnonymousStructLiteral)
	p.registerPrefix(token.LBRACK, p.parseArrayLiteralTypeOrCast)
	// Types
	p.registerPrefix(token.STRINGTYPE, p.parseTypeLiteral)
	p.registerPrefix(token.BOOLTYPE, p.parseTypeLiteral)
	p.registerPrefix(token.BYTETYPE, p.parseTypeLiteral)
	p.registerPrefix(token.CHARTYPE, p.parseTypeLiteral)
	p.registerPrefix(token.U8TYPE, p.parseTypeLiteral)
	p.registerPrefix(token.U16TYPE, p.parseTypeLiteral)
	p.registerPrefix(token.U32TYPE, p.parseTypeLiteral)
	p.registerPrefix(token.U64TYPE, p.parseTypeLiteral)
	// p.registerPrefix(token.U128TYPE, p.parseTypeExpression)
	// p.registerPrefix(token.U256TYPE, p.parseTypeExpression)
	p.registerPrefix(token.I8TYPE, p.parseTypeLiteral)
	p.registerPrefix(token.I16TYPE, p.parseTypeLiteral)
	p.registerPrefix(token.I32TYPE, p.parseTypeLiteral)
	p.registerPrefix(token.I64TYPE, p.parseTypeLiteral)
	// p.registerPrefix(token.I128TYPE, p.parseTypeExpression)
	// p.registerPrefix(token.I256TYPE, p.parseTypeExpression)
	// p.registerPrefix(token.FLOATTYPE, p.parseTypeLiteral)
	p.registerPrefix(token.F32TYPE, p.parseTypeLiteral)
	p.registerPrefix(token.F64TYPE, p.parseTypeLiteral)
	p.registerPrefix(token.ERROR, p.parseTypeLiteral)
	p.registerPrefix(token.ANYTYPE, p.parseTypeLiteral)
	//
	p.registerPrefix(token.TRY, p.parseTryExpression)

	p.infixParseFns = make(map[token.Type]infixParseFn)
	p.registerInfix(token.PLUS, p.parseInfixExpression)
	p.registerInfix(token.MINUS, p.parseInfixExpression)
	p.registerInfix(token.SLASH, p.parseInfixExpression)
	p.registerInfix(token.ASTERISK, p.parseInfixExpression)
	p.registerInfix(token.MOD, p.parseInfixExpression)
	p.registerInfix(token.EQ, p.parseInfixExpression)
	p.registerInfix(token.NEQ, p.parseInfixExpression)
	p.registerInfix(token.LT, p.parseInfixExpression)
	p.registerInfix(token.GT, p.parseGreaterThanOrRightShift)
	p.registerInfix(token.GTE, p.parseInfixExpression)
	p.registerInfix(token.LTE, p.parseInfixExpression)
	p.registerInfix(token.AND, p.parseInfixExpression)
	p.registerInfix(token.OR, p.parseInfixExpression)
	p.registerInfix(token.PIPE, p.parseInfixExpression)
	p.registerInfix(token.COLON, p.parseInfixExpression)
	p.registerInfix(token.DOT, p.parseDotExpression) // could be replace with parse infix
	p.registerInfix(token.ASSIGN, p.parseInfixExpression)
	p.registerInfix(token.NULL_COALESCE, p.parseInfixExpression)
	p.registerInfix(token.LSHIFT, p.parseInfixExpression)
	p.registerInfix(token.RSHIFT, p.parseInfixExpression)
	p.registerInfix(token.BAR, p.parseInfixExpression)
	p.registerInfix(token.CARET, p.parseInfixExpression)
	p.registerInfix(token.AMPERSAND, p.parseInfixExpression)
	// p.registerInfix(token.BANDNOT, p.parseInfixExpression)
	p.registerInfix(token.CATCH, p.parseCatchExpression)

	p.postfixParseFns = make(map[token.Type]postfixParseFn)
	p.registerPostfix(token.INCR, p.parsePostfixExpression)
	p.registerPostfix(token.DECR, p.parsePostfixExpression)

	p.attributeParseFns = make(map[string]attributeParseFn)
	p.registerAttribute("extern", p.parseExternAttribute)
	p.registerAttribute("inline", p.parseInlineAttribute)
	p.registerAttribute("test", p.parseBasicAttribute)

	// scan all tokens
	var tkns []token.Token
	for {
		tkn := l.NextToken()
		tkns = append(tkns, tkn)
		if tkn.Type == token.EOF {
			break
		}
	}
	p.tkns = tkns
	// tkns will never be empty as there will
	// be at least one EOF token
	p.curToken = tkns[0]

	return p
}

func (p *Parser) Errors() []string {
	return p.errors
}

func (p *Parser) nextToken() {
	if p.tknIdx+1 >= len(p.tkns) {
		return
	}
	p.tknIdx++
	p.curToken = p.tkns[p.tknIdx]
}

func (p *Parser) peekToken() token.Token {
	if p.tknIdx+1 >= len(p.tkns) {
		return p.tkns[len(p.tkns)-1]
	}
	return p.tkns[p.tknIdx+1]
}

func (p *Parser) peekNToken(n int) token.Token {
	if p.tknIdx+n >= len(p.tkns) {
		return p.tkns[len(p.tkns)-1]
	}
	return p.tkns[p.tknIdx+n]
}

func (p *Parser) ParseLibrary() *ast.Library {
	if !p.curTokenIs(token.LIBRARY) {
		p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
		return nil
	}
	lib := &ast.Library{Token: p.curToken}
	p.nextToken()

	if !p.curTokenIs(token.IDENT) && !p.curTokenIs(token.MAIN) {
		p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
		return nil
	}
	lib.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	p.nextToken()

	for !p.curTokenIs(token.EOF) {
		switch p.curToken.Type {
		case token.PUBLIC:
			p.nextToken()
		case token.USE:
			lib.Nodes = append(lib.Nodes, p.parseUseStatement())
		case token.TYPE:
			lib.Nodes = append(lib.Nodes, p.parseTypeDefinitionStatement())
		case token.ALIAS:
			lib.Nodes = append(lib.Nodes, p.parseTypeAliasStatement())
		case token.STRUCT:
			lib.Nodes = append(lib.Nodes, p.parseStructStatement())
		case token.ENUM:
			lib.Nodes = append(lib.Nodes, p.parseEnumStatement())
		case token.UNION:
			lib.Nodes = append(lib.Nodes, p.parseUnionStatement())
		case token.LET:
			assgn := p.parseAssignmentStatement().(*ast.AssignmentStatement)
			lib.Nodes = append(lib.Nodes, assgn)
		case token.VAR:
			// TODO: throw proper error
			panic("var not allowed in global lib scope")
		case token.ERROR:
			lib.Nodes = append(lib.Nodes, p.parseErrorStatement())
		case token.COMMENT:
			// ignore comments for now
			p.nextToken()
		case token.AT:
			attr := p.parseAttribute()
			p.attributes.Push(attr)
		case token.FUNCTION:
			exp := p.parseFunctionExpression().(*ast.FunctionExpression)
			lib.Nodes = append(lib.Nodes, exp)
		default:
			p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
			return nil
		}
	}
	return lib
}

// Only parses import statements of a lib, ignoring all other tokens
func (p *Parser) ParseImports() *ast.Library {
	if !p.curTokenIs(token.LIBRARY) {
		p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
		return nil
	}
	lib := &ast.Library{Token: p.curToken}
	p.nextToken()

	if !p.curTokenIs(token.IDENT) && !p.curTokenIs(token.MAIN) {
		p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
		return nil
	}
	lib.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	p.nextToken()

	for !p.curTokenIs(token.EOF) {
		switch p.curToken.Type {
		case token.PUBLIC:
			p.nextToken()
		case token.USE:
			lib.Nodes = append(lib.Nodes, p.parseUseStatement())
		default:
			p.nextToken()
		}
	}
	return lib

}

// Parsing for REPL automatically creates a main function and adds
// statements and expressions within the function body except for
// structs, unions, enums, type defs, type aliases and functons,
// which get added to global library scope
func (p *Parser) ParseREPL() *ast.Library {
	lib := &ast.Library{
		Token: token.Token{Type: token.LIBRARY, Literal: "lib"},
		Name:  &ast.Identifier{Token: token.Token{Type: token.IDENT, Literal: "main"}, Value: "main"},
	}

	// create main function
	mainFn := &ast.FunctionExpression{
		Token: token.Token{Type: token.FUNCTION, Literal: "fn"},
		Name:  &ast.Identifier{Token: token.Token{Type: token.MAIN, Literal: "main"}, Value: "main", T: &types.Function{}},
		Body:  &ast.BlockStatement{},
	}

	for !p.curTokenIs(token.EOF) {
		switch p.curToken.Type {
		case token.PUBLIC:
			p.nextToken()
		case token.USE:
			lib.Nodes = append(lib.Nodes, p.parseUseStatement())
		case token.TYPE:
			lib.Nodes = append(lib.Nodes, p.parseTypeDefinitionStatement())
		case token.ALIAS:
			lib.Nodes = append(lib.Nodes, p.parseTypeAliasStatement())
		case token.STRUCT:
			lib.Nodes = append(lib.Nodes, p.parseStructStatement())
		case token.ENUM:
			lib.Nodes = append(lib.Nodes, p.parseEnumStatement())
		case token.UNION:
			lib.Nodes = append(lib.Nodes, p.parseUnionStatement())
		case token.ERROR:
			if p.peekTokenIs(token.LPAREN) {
				mainFn.Body.Statements = append(mainFn.Body.Statements, p.parseExpression(LOWEST))
			} else {
				lib.Nodes = append(lib.Nodes, p.parseErrorStatement())
			}
		case token.AT:
			attr := p.parseAttribute()
			p.attributes.Push(attr)
		case token.FUNCTION:
			exp := p.parseFunctionExpression().(*ast.FunctionExpression)
			lib.Nodes = append(lib.Nodes, exp)
		default:
			mainFn.Body.Statements = append(mainFn.Body.Statements, p.parseStatementInBlock())
		}
	}
	lib.Nodes = append(lib.Nodes, mainFn)
	return lib
}

func (p *Parser) ParseExpression() ast.Node {
	return p.parseStatementInBlock()
}

func (p *Parser) registerPrefix(tp token.Type, fn prefixParseFn) {
	p.prefixParseFns[tp] = fn
}
func (p *Parser) registerInfix(tp token.Type, fn infixParseFn) {
	p.infixParseFns[tp] = fn
}

func (p *Parser) registerPostfix(tp token.Type, fn postfixParseFn) {
	p.postfixParseFns[tp] = fn
}

func (p *Parser) registerAttribute(attr string, fn attributeParseFn) {
	p.attributeParseFns[attr] = fn
}

// ---------- //
// Statements //
// ---------- //

// use "path/to/lib"
// use (
//
//	"internal/code"
//
// )
func (p *Parser) parseUseStatement() *ast.UseStatement {
	stmt := &ast.UseStatement{Token: p.curToken}

	if !p.curTokenIs(token.USE) {
		p.nextToken()
		return nil
	}

	p.nextToken()

	if !p.curTokenIs(token.STRING) {
		p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
		return nil
	}

	name, ok := p.parseStringLiteral().(*ast.StringLiteral)
	if !ok {
		return nil
	}
	stmt.Name = name

	return stmt
}

func (p *Parser) parseTypeDefinitionStatement() *ast.TypeDefinitionStatement {
	stmt := &ast.TypeDefinitionStatement{Token: p.curToken}
	if p.prevTokenIs(token.PUBLIC) {
		stmt.Public = true
	}
	p.nextToken()

	if !p.curTokenIs(token.IDENT) {
		p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
		return nil
	}
	stmt.Name = p.parseIdentifier()

	if p.curTokenIsType() {
		stmt.UnderlyingType = p.parseType()
	} else {
		p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
		return nil
	}

	if p.curTokenIs(token.BAR) {
		p.nextToken()
		stmt.Guard = p.parseExpression(LOWEST)
	}

	return stmt
}

func (p *Parser) parseTypeAliasStatement() *ast.TypeAliasStatement {
	stmt := &ast.TypeAliasStatement{Token: p.curToken}
	p.nextToken()

	if !p.curTokenIs(token.IDENT) {
		p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
		return nil
	}
	stmt.Name = p.parseIdentifier()

	if p.curTokenIsType() {
		stmt.UnderlyingType = p.parseType()
	} else {
		p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
		return nil
	}

	return stmt
}

func (p *Parser) parseStructStatement() *ast.StructStatement {
	stmt := &ast.StructStatement{Token: p.curToken}
	if p.prevTokenIs(token.PUBLIC) {
		stmt.Public = true
	}
	p.nextToken()
	if !p.curTokenIs(token.IDENT) {
		p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
		return nil
	}
	stmt.Name = p.parseIdentifier()

	// handle optional parametrised type
	if p.curTokenIs(token.LBRACK) {
		p.nextToken()
		stmt.GenericParameters = p.parseGenericParameters()
		if !p.curTokenIs(token.RBRACK) {
			p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
			return nil
		}
		p.nextToken()
	}

	if !p.curTokenIs(token.LBRACE) {
		p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
		return nil
	}
	p.nextToken()

	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		field := &ast.StructFieldStatement{}

		// All struct fields must have names (can be keywords)
		field.Name = p.parseIdentifierOrKeyword()
		if field.Name == nil {
			return nil
		}

		if p.curTokenIsType() {
			field.Type = p.parseType()
			stmt.Fields = append(stmt.Fields, field)
		} else if p.curTokenIsIdent() {
			field.Type = p.parseUnknownNamedType()
			stmt.Fields = append(stmt.Fields, field)
		} else {
			p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
			return nil
		}

		if p.curTokenIs(token.BAR) {
			p.nextToken()
			field.Guard = p.parseExpression(LOWEST)
		}

		if p.curTokenIs(token.COMMA) {
			p.nextToken()
		}
	}
	p.nextToken()

	return stmt
}

func (p *Parser) parseEnumStatement() *ast.EnumStatement {
	stmt := &ast.EnumStatement{Token: p.curToken}
	if p.prevTokenIs(token.PUBLIC) {
		stmt.Public = true
	}
	p.nextToken()

	if !p.curTokenIs(token.IDENT) {
		p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
		return nil
	}

	stmt.Name = p.parseIdentifier()

	if !p.curTokenIs(token.LBRACE) {
		p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
		return nil
	}
	p.nextToken()

	for !p.curTokenIs(token.RBRACE) {
		if ident := p.parseIdentifierOrKeyword(); ident != nil {
			stmt.Fields = append(stmt.Fields, ident)
		} else {
			return nil
		}
		if p.curTokenIs(token.COMMA) {
			p.nextToken()
		}
	}

	fieldNames := make([]string, len(stmt.Fields))
	for i, field := range stmt.Fields {
		fieldNames[i] = field.Value
	}
	stmt.T = &types.Enum{Name: stmt.Name.String(), Size: len(stmt.Fields), Fields: fieldNames}

	p.nextToken()

	return stmt
}

func (p *Parser) parseUnionStatement() *ast.UnionStatement {
	stmt := &ast.UnionStatement{Token: p.curToken}
	if p.prevTokenIs(token.PUBLIC) {
		stmt.Public = true
	}
	p.nextToken()

	if !p.curTokenIs(token.IDENT) {
		p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
		return nil
	}

	stmt.Name = p.parseIdentifier()

	if !p.curTokenIs(token.LBRACE) {
		p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
		return nil
	}
	p.nextToken()

	for !p.curTokenIs(token.RBRACE) {
		stmt.Types = append(stmt.Types, p.parseTypeLiteral().(*ast.TypeLiteral))
		if p.curTokenIs(token.COMMA) {
			p.nextToken()
		}
	}

	stmt.T = &types.Union{Name: stmt.Name.String()}

	p.nextToken()

	return stmt
}

// TODO: add parsing function statement

// Examples
//
// * x,
// * x int
// * x []int,
// * x fn() int
// * x ...i64
func (p *Parser) parseParameterStatement(allowedOptional bool) *ast.ParameterStatement {
	stmt := &ast.ParameterStatement{}
	stmt.Name = p.parseIdentifierOrKeyword()
	if stmt.Name == nil {
		return nil
	}

	// x,
	if p.curTokenIs(token.COMMA) {
		p.nextToken()
		return stmt
	}

	// x ...int
	if p.curTokenIs(token.ELLIPSIS) {
		p.nextToken()
		stmt.Type = &types.Array{T: p.parseType()}
		return stmt
	}

	// x)  - last parameter without type (like x, but with ) instead of ,)
	if p.curTokenIs(token.RPAREN) {
		p.addError(p.curToken, errMissingArgumentType(stmt.Name.Value))
		return stmt
	}

	stmt.Type = p.parseType()
	if stmt.Type == nil {
		p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
		return nil
	}

	if !allowedOptional && p.peekTokenIs(token.COLON) {
		p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
		return nil
	}

	// x int)
	if p.curTokenIs(token.RPAREN) {
		return stmt
	}
	// x int}
	if p.curTokenIs(token.RBRACE) {
		return stmt
	}
	// x int,
	if p.curTokenIs(token.COMMA) {
		p.nextToken()
		return stmt
	}
	p.nextToken()

	return stmt
}

// let x = 3 + 3
// var x, let y = 1, 2
// x, let y = 1, 2
func (p *Parser) parseAssignmentStatement() ast.Statement {
	stmt := &ast.AssignmentStatement{}
	if p.prevTokenIs(token.PUBLIC) {
		stmt.Public = true
	}

	for !p.curTokenIs(token.ASSIGN) && !p.curTokenIs(token.EOF) {
		var tkn token.Token // let or var
		if p.curTokenIs(token.LET) || p.curTokenIs(token.VAR) {
			tkn = p.curToken
			p.nextToken()
			assignee := p.parseExpression(ASSIGN)

			stmt.Declerations = append(stmt.Declerations, &ast.DeclarationStatement{
				Token:    tkn,
				Assignee: assignee,
			})
		} else if p.curTokenIs(token.IDENT) {
			stmt.Declerations = append(stmt.Declerations, p.parseExpression(ASSIGN))
		}

		if p.curTokenIs(token.COMMA) {
			p.nextToken()
		}
	}
	p.nextToken()

	for !p.curTokenIs(token.COMMA) && !p.curTokenIs(token.EOF) {
		stmt.Values = append(stmt.Values, p.parseExpression(LOWEST))
		if !p.curTokenIs(token.COMMA) {
			break
		}
		p.nextToken()
	}

	return stmt
}

// parses partially pre-parsed assigned statement
func (p *Parser) parseAssignmentStatementPre(firstAssignee ast.Expression) ast.Statement {
	stmt := &ast.AssignmentStatement{}

	stmt.Declerations = append(stmt.Declerations, firstAssignee)

	for !p.curTokenIs(token.ASSIGN) && !p.curTokenIs(token.EOF) {
		if p.curTokenIs(token.LET) || p.curTokenIs(token.VAR) {
			tkn := p.curToken
			p.nextToken()

			assignee := p.parseExpression(ASSIGN)
			stmt.Declerations = append(stmt.Declerations, &ast.DeclarationStatement{
				Token:    tkn,
				Assignee: assignee,
			})
		} else {
			stmt.Declerations = append(stmt.Declerations, p.parseExpression(ASSIGN))
		}

		if p.curTokenIs(token.COMMA) {
			p.nextToken()
		}
	}
	p.nextToken()

	for !p.curTokenIs(token.COMMA) && !p.curTokenIs(token.EOF) {
		stmt.Values = append(stmt.Values, p.parseExpression(LOWEST))
		if !p.curTokenIs(token.COMMA) {
			break
		}
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseKeywordStatement() *ast.KeywordStatement {
	stmt := &ast.KeywordStatement{Token: p.curToken}
	p.nextToken()
	return stmt
}

func (p *Parser) parseIfElseExpression() ast.Expression {
	if !p.curTokenIs(token.IF) {
		return nil
	}

	exp := &ast.IfElseExpression{Token: p.curToken}

	if p.curTokenIs(token.LBRACE) {
		p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
		return nil
	}

	// parse if
	p.context = IF_ELSE
	cond := &ast.ConditionalExpression{Token: p.curToken}
	p.nextToken()
	cond.Condition = p.parseExpression(LOWEST)
	if !p.curTokenIs(token.LBRACE) {
		p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
		return nil
	}
	p.context = NONE

	cond.Block = p.parseBlockStatement()
	exp.Conditionals = append(exp.Conditionals, cond)

	for p.curTokenIsConditionalStatement() {
		p.context = IF_ELSE
		cond := &ast.ConditionalExpression{Token: p.curToken}

		if !p.curTokenIs(token.ELSE) {
			p.nextToken()
			cond.Condition = p.parseExpression(LOWEST)
		} else {
			p.nextToken()
		}
		p.context = NONE

		if !p.curTokenIs(token.LBRACE) {
			p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
			return nil
		}

		cond.Block = p.parseBlockStatement()
		exp.Conditionals = append(exp.Conditionals, cond)
	}

	return exp
}

// exactly the same as parseIfElseExpression except sets if its
// a statement
func (p *Parser) parseIfElseStatement() ast.Expression {
	if !p.curTokenIs(token.IF) {
		return nil
	}

	exp := &ast.IfElseExpression{Token: p.curToken, IsStatement: true}

	if p.curTokenIs(token.LBRACE) {
		p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
		return nil
	}

	// parse if
	p.context = IF_ELSE
	cond := &ast.ConditionalExpression{Token: p.curToken}
	p.nextToken()
	cond.Condition = p.parseExpression(LOWEST)
	if !p.curTokenIs(token.LBRACE) {
		p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
		return nil
	}
	p.context = NONE

	cond.Block = p.parseBlockStatement()
	exp.Conditionals = append(exp.Conditionals, cond)

	for p.curTokenIsConditionalStatement() {
		p.context = IF_ELSE
		cond := &ast.ConditionalExpression{Token: p.curToken}

		if !p.curTokenIs(token.ELSE) {
			p.nextToken()
			cond.Condition = p.parseExpression(LOWEST)
		} else {
			p.nextToken()
		}

		if !p.curTokenIs(token.LBRACE) {
			p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
			return nil
		}
		p.context = NONE

		cond.Block = p.parseBlockStatement()
		exp.Conditionals = append(exp.Conditionals, cond)
	}

	return exp
}

// return a, b
// }
// Note: does not consume } token
func (p *Parser) parseReturnStatement() *ast.ReturnStatement {
	if !p.curTokenIs(token.RETURN) {
		return nil
	}
	stmt := &ast.ReturnStatement{Token: p.curToken}
	p.nextToken()

	// NOTE: checking for case is required to ensure that return in match case does not trigger
	// parser error
	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) &&
		!p.curTokenIs(token.CASE) && !p.curTokenIs(token.ELSE) {
		if p.curTokenIs(token.COMMA) {
			p.nextToken()
		}
		stmt.Values = append(stmt.Values, p.parseExpression(LOWEST))
	}
	return stmt
}

func (p *Parser) parseBlockStatement() *ast.BlockStatement {

	if !p.curTokenIs(token.LBRACE) {
		return nil
	}

	block := &ast.BlockStatement{Token: p.curToken}
	p.nextToken()

	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		block.Statements = append(block.Statements, p.parseStatementInBlock())
	}
	p.nextToken()
	return block
}

// Should only be called in block statement
func (p *Parser) parseStatementInBlock() ast.Node {
	switch p.curToken.Type {
	case token.LET, token.VAR:
		return p.parseAssignmentStatement()
	case token.IDENT:
		if p.peekTokenIs(token.LPAREN) {
			return p.parseExpression(LOWEST)
		}
		// If the next token is assign we can simply parse without any special
		// checking. We do the same if its a comma as there can be no valid
		// statement other than assignment
		if p.peekTokenIs(token.ASSIGN) || p.peekTokenIs(token.COMMA) {
			return p.parseAssignmentStatement()
		}
		// for dot, slice and index expression we need to partially parse
		// tokens before we can derermine whther it is an assignment or not
		if p.peekTokenIs(token.DOT) {
			exp := p.parseExpression(ASSIGN)
			if p.curTokenIs(token.ASSIGN) || p.curTokenIs(token.COMMA) {
				return p.parseAssignmentStatementPre(exp)
			}
			if p.curTokenIsOperator() {
				return p.parseInfixExpression(exp)
			}
			return exp
		} else if p.peekTokenIs(token.LBRACK) {
			exp := p.parseExpression(ASSIGN)
			if p.curTokenIs(token.ASSIGN) || p.curTokenIs(token.COMMA) {
				return p.parseAssignmentStatementPre(exp)
			}
			if p.curTokenIsOperator() {
				return p.parseInfixExpression(exp)
			}
			return exp
		}
		return p.parseExpression(LOWEST)
	case token.IF:
		return p.parseIfElseStatement()
	case token.FOR:
		return p.parseForStatement()
	case token.DEFER:
		return p.parseDeferExpression()
	case token.MATCH:
		return p.parseMatchStatement()
	case token.RETURN:
		return p.parseReturnStatement()
	case token.BREAK, token.NEXT:
		return p.parseKeywordStatement()
	case token.RAISE:
		return p.parseRaiseStatement()
	default:
		return p.parseExpression(LOWEST)
	}
}

// func (p *Parser) parseChannelTypeStatement() *ast.ChannelTypeStatement {
// 	stmt := &ast.ChannelTypeStatement{Token: p.curToken}
// 	p.nextToken()
// 	stmt.Type = p.parseType()

// 	return stmt
// }

// parses primitive types excluding list, map, set

func (p *Parser) parseForStatement() *ast.ForStatement {
	stmt := &ast.ForStatement{Token: p.curToken}
	p.nextToken()

	// infinite loop
	if p.curTokenIs(token.LBRACE) {
		stmt.Block = p.parseBlockStatement()
		return stmt
	}

	// tkn := p.curToken
	exp := p.parseExpression(LOWEST)
	// var assign *ast.InfixExpression
	switch a := exp.(type) {
	// We need to check if infix operation
	// is assignment as that indicates
	// we are parsing a classic for loop
	case *ast.InfixExpression:
		if a.Operator == "=" {
			// assign = a
			decl := []ast.Node{a.Left}
			val := []ast.Expression{a.Right}
			stmt.Assignment = &ast.AssignmentStatement{Declerations: decl, Values: val}
		} else {
			stmt.Condition = a
			stmt.Block = p.parseBlockStatement()
			return stmt
		}
	case *ast.FunctionCallExpression:
		stmt.Condition = a
		stmt.Block = p.parseBlockStatement()
		return stmt
	case *ast.PrefixExpression:
		stmt.Condition = a
		stmt.Block = p.parseBlockStatement()
		return stmt
	}
	// name := &ast.Identifier{Token: tkn, Value: assign.Left.String()}
	if !p.curTokenIs(token.SEMI) {
		p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
		return nil
	}
	p.nextToken()

	// BUG: check if returned type of expression is infix and only then assign
	// as user could make mistake in syntax e.g. for i = 0; len(s); i++ {}
	stmt.Condition = p.parseExpression(LOWEST).(*ast.InfixExpression)
	if !p.curTokenIs(token.SEMI) {
		p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
		return nil
	}
	p.nextToken()
	stmt.Change = p.parseExpression(LOWEST)

	stmt.Block = p.parseBlockStatement()

	return stmt
}

func (p *Parser) parseForRangeStatement() *ast.ForRangeStatement {
	stmt := &ast.ForRangeStatement{Token: p.curToken}
	p.nextToken()

	// infinite loop
	if p.curTokenIs(token.LBRACE) {
		return stmt
	}

	for !p.curTokenIs(token.IN) {
		if p.curTokenIs(token.COMMA) {
			p.nextToken()
		}
		if !p.curTokenIs(token.IDENT) {
			p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
			return nil
		}
		stmt.Variables = append(stmt.Variables, &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal})
		p.nextToken()
	}
	p.nextToken()

	if p.curTokenIs(token.IDENT) {
		stmt.Iterable = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
		p.nextToken()
	}

	if p.curTokenIs(token.SEMI) {
		p.nextToken()
		stmt.Change = p.parseExpression(LOWEST)
	}

	stmt.Block = p.parseBlockStatement()

	return stmt

}

func (p *Parser) parseMatchExpression() ast.Expression {
	es := &ast.MatchExpressionStatement{Token: p.curToken}
	p.nextToken()

	p.context = MATCH
	es.Scrutinee = p.parseExpression(LOWEST)
	p.context = NONE

	if !p.curTokenIs(token.LBRACE) {
		p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
		return nil
	}
	p.nextToken()

	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		mc := &ast.MatchCase{Token: p.curToken}
		if p.curTokenIs(token.CASE) {
			p.nextToken()
			// parse comma-separated predicates until ':'
			for {
				pred := p.parseExpression(COLON)
				mc.Predicates = append(mc.Predicates, pred)

				if p.curTokenIs(token.COMMA) {
					p.nextToken()
					continue
				}
				break
			}
		} else {
			p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
			return nil
		}

		if !p.curTokenIs(token.COLON) {
			p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
			return nil
		}
		p.nextToken()

		// parse match case block
		for !p.curTokenIs(token.CASE) && !p.curTokenIs(token.ELSE) &&
			!p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
			mc.Body = append(mc.Body, p.parseStatementInBlock())
		}

		// ensure default case is saved to proper field
		if len(mc.Predicates) == 1 && mc.Predicates[0].TokenLiteral() == "_" {
			es.Default = mc
		} else {
			es.Cases = append(es.Cases, mc)
		}

	}
	if !p.curTokenIs(token.RBRACE) {
		p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
		return nil
	}
	p.nextToken()

	return es
}

func (p *Parser) parseMatchStatement() ast.Expression {
	es := &ast.MatchExpressionStatement{Token: p.curToken, IsStatement: true}
	p.nextToken()

	p.context = MATCH
	es.Scrutinee = p.parseExpression(LOWEST)
	p.context = NONE

	if !p.curTokenIs(token.LBRACE) {
		p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
		return nil
	}
	p.nextToken()

	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		mc := &ast.MatchCase{Token: p.curToken}
		if p.curTokenIs(token.CASE) {
			p.nextToken()
			// parse comma-separated predicates until ':'
			for {
				pred := p.parseExpression(COLON)
				mc.Predicates = append(mc.Predicates, pred)

				if p.curTokenIs(token.COMMA) {
					p.nextToken()
					continue
				}
				break
			}

		} else {
			p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
			return nil
		}

		if !p.curTokenIs(token.COLON) {
			p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
			return nil
		}
		p.nextToken()

		// parse match case block
		for !p.curTokenIs(token.CASE) && !p.curTokenIs(token.ELSE) &&
			!p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
			mc.Body = append(mc.Body, p.parseStatementInBlock())
		}

		// ensure default case is saved to proper field
		if len(mc.Predicates) == 1 && mc.Predicates[0].TokenLiteral() == "_" {
			es.Default = mc
		} else {
			es.Cases = append(es.Cases, mc)
		}

	}
	if !p.curTokenIs(token.RBRACE) {
		p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
		return nil
	}
	p.nextToken()

	return es
}

func (p *Parser) parseErrorStatement() *ast.ErrorStatement {
	stmt := &ast.ErrorStatement{Token: p.curToken}

	// Handle pub modifier if present
	if p.prevTokenIs(token.PUBLIC) {
		stmt.Public = true
	}

	// Parse error name
	p.nextToken()
	if !p.curTokenIs(token.IDENT) {
		p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
		return nil
	}
	stmt.Name = p.parseIdentifier()

	// Parse parameters if present
	if p.curTokenIs(token.LBRACE) {
		p.nextToken()

		// Parse parameters until we hit '}'
		for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
			field := &ast.ParameterStatement{}

			// All struct fields must have names (can be keywords)
			field.Name = p.parseIdentifierOrKeyword()
			if field.Name == nil {
				return nil
			}

			if p.curTokenIsType() {
				field.Type = p.parseType()
				stmt.Params = append(stmt.Params, field)
			} else if p.curTokenIsIdent() {
				field.Type = p.parseUnknownNamedType()
				stmt.Params = append(stmt.Params, field)
			} else {
				p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
				return nil
			}
		}

		// if !p.curTokenIs(token.RBRACE) {
		// p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
		// return nil
		// }

		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseRaiseStatement() ast.Statement {
	stmt := &ast.RaiseStatement{Token: p.curToken}
	p.nextToken()

	stmt.Error = p.parseExpression(LOWEST)

	return stmt
}

// ----------- //
// Expressions //
// ----------- //

func (p *Parser) parseIdentifierStructLiteralOrFunctionCall() ast.Expression {

	if p.peekTokenIs(token.LPAREN) {
		tkn := p.curToken
		p.nextToken()
		return p.parseFunctionCallExpression(tkn, nil)
	}

	ident := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	// [] is only used for generic type parameters
	// - Generic function call: identity[i32](...)
	// - Generic struct literal: vec[i64]{...}
	if p.peekTokenIs(token.LBRACK) {
		// Parse as generic function call or struct instantiation
		p.nextToken() // consume ident
		typeParams := p.parseTypeParameters()

		if p.curTokenIs(token.LPAREN) {
			return p.parseFunctionCallExpression(ident.Token, typeParams)
		} else if p.curTokenIs(token.LBRACE) {
			if p.context == IF_ELSE || p.context == MATCH {
				p.nextToken()
				return ident
			}
			return p.parseStructLiteral(ident, typeParams)
		} else {
			p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
			return nil
		}
	}

	// Handle normal struct literal, dot expression, or plain identifier
	if p.peekTokenIs(token.LBRACE) {
		if p.context == IF_ELSE || p.context == MATCH {
			p.nextToken()
			return ident
		}
		p.nextToken()
		return p.parseStructLiteral(ident, nil)

	} else if p.peekTokenIs(token.DOT) {
		p.nextToken()
		exp := p.parseDotExpression(ident)
		if p.context == IF_ELSE || p.context == MATCH {
			return exp
		}
		if p.curTokenIs(token.LBRACE) {
			return p.parseStructLiteral(exp, nil)
		}
		return exp
	}

	p.nextToken()
	return ident
}

// curTokenIsIdent checks if a token can be treated as an identifier
// in contexts where keywords should be allowed as identifiers
func (p *Parser) curTokenIsIdent() bool {
	typ := p.curToken.Type
	switch typ {
	case token.IDENT:
		return true
	// Type keywords that can be used as identifiers
	case token.INTTYPE, token.FLOATTYPE, token.STRINGTYPE, token.BOOLTYPE,
		token.BYTETYPE, token.CHARTYPE, token.I8TYPE, token.I16TYPE,
		token.I32TYPE, token.I64TYPE, token.U8TYPE, token.U16TYPE,
		token.U32TYPE, token.U64TYPE, token.F32TYPE, token.F64TYPE,
		token.ANYTYPE:
		return true
	case token.LIBRARY, token.PUBLIC,
		token.NEXT, token.BREAK,
		token.ENUM, token.TYPE:
		return true
	default:
		return false
	}
}

// parseIdentifierOrKeyword parses an identifier or a keyword that should be treated as an identifier
func (p *Parser) parseIdentifierOrKeyword() *ast.Identifier {
	if !p.curTokenIsIdent() {
		p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
		p.nextToken()
		return nil
	}
	ident := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	p.nextToken()
	return ident
}

func (p *Parser) parseIdentifier() *ast.Identifier {
	ident := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	p.nextToken()
	return ident
}

func (p *Parser) parseExpression(precedence int) ast.Expression {
	prefix := p.prefixParseFns[p.curToken.Type]
	if prefix == nil {
		p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
		p.nextToken()
		return nil
	}
	leftExp := prefix()

	postfix := p.postfixParseFns[p.curToken.Type]
	if postfix != nil {
		leftExp = postfix(leftExp)
	}

	for precedence < p.curPrecedence() {
		infix := p.infixParseFns[p.curToken.Type]
		if infix == nil {
			return leftExp
		}
		leftExp = infix(leftExp)
	}

	return leftExp
}

func (p *Parser) parseComment() ast.Expression {
	comment := &ast.Comment{Token: p.curToken}
	p.nextToken()
	return comment
}

func (p *Parser) parsePrefixExpression() ast.Expression {
	expression := &ast.PrefixExpression{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
	}
	p.nextToken()
	expression.Right = p.parseExpression(PREFIX)
	return expression
}

func (p *Parser) parseGreaterThanOrRightShift(left ast.Expression) ast.Expression {
	tkn := p.curToken
	if p.peekTokenIs(token.GT) {
		tkn.Type = token.RSHIFT
		tkn.Literal = ">>"
		p.nextToken()
	}
	expression := &ast.InfixExpression{
		Token:    tkn,
		Operator: tkn.Literal,
		Left:     left,
	}
	precedence := p.curPrecedence()
	p.nextToken()
	expression.Right = p.parseExpression(precedence)
	return expression
}

func (p *Parser) parseInfixExpression(left ast.Expression) ast.Expression {
	expression := &ast.InfixExpression{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
		Left:     left,
	}
	precedence := p.curPrecedence()
	p.nextToken()
	expression.Right = p.parseExpression(precedence)
	return expression
}

func (p *Parser) parsePostfixExpression(left ast.Expression) ast.Expression {
	exp := &ast.PostfixExpression{Token: p.curToken, Left: left}
	p.nextToken()
	return exp
}

func (p *Parser) parseDeferExpression() ast.Expression {
	exp := &ast.DeferStatement{Token: p.curToken}
	p.nextToken()

	if p.curTokenIs(token.LBRACE) {
		exp.Node = p.parseBlockStatement()
	} else {
		exp.Node = p.parseExpression(LOWEST)
	}

	// TODO: validate exp.Node is valid e.g. function call

	return exp
}

func (p *Parser) parseDotExpression(left ast.Expression) ast.Expression {
	if !p.curTokenIs(token.DOT) {
		p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
		return nil
	}
	exp := &ast.DotExpression{Token: p.curToken, Left: left}
	p.nextToken()

	if p.curTokenIsIdent() {
		// [] is only used for generic type parameters
		if p.peekTokenIs(token.LBRACK) {
			ident := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
			p.nextToken() // consume ident
			typeParams := p.parseTypeParameters()

			if p.curTokenIs(token.LPAREN) {
				exp.Right = p.parseFunctionCallExpression(ident.Token, typeParams)
			} else if p.curTokenIs(token.LBRACE) {
				exp.Right = p.parseStructLiteral(ident, typeParams)
			} else {
				p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
				return nil
			}
		} else if p.peekTokenIs(token.LPAREN) {
			tkn := p.curToken
			p.nextToken()
			exp.Right = p.parseFunctionCallExpression(tkn, nil)
		} else {
			exp.Right = p.parseIdentifierOrKeyword()
		}
	} else if p.curTokenIs(token.INT) {
		exp.Right = p.parseIntegerLiteral()
	} else {
		p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
		return nil
	}
	return exp
}

// parseFunctionCallExpression parses a generic function call
// Assumes current token is LPAREN (after type parameters have been parsed)
func (p *Parser) parseFunctionCallExpression(tkn token.Token, typeParams []types.Type) ast.Expression {
	exp := &ast.FunctionCallExpression{
		Token:          tkn,
		TypeParameters: typeParams,
	}

	if !p.curTokenIs(token.LPAREN) {
		p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
		return nil
	}
	p.nextToken()

	for !p.curTokenIs(token.RPAREN) && !p.curTokenIs(token.EOF) {
		if p.curTokenIs(token.COMMA) {
			p.nextToken()
		}
		exp.Arguments = append(exp.Arguments, p.parseExpression(LOWEST))
	}
	p.nextToken()

	return exp
}

func (p *Parser) parseGroupedExpression() ast.Expression {
	if !p.curTokenIs(token.LPAREN) {
		return nil
	}
	p.nextToken()
	exp := p.parseExpression(LOWEST)
	// if !p.curTokenIs(token.RPAREN) {
	// 	return nil
	// }
	p.nextToken()
	return exp
}

// Handles parsing:
// types as expressions (e.g. for 'make([]byte, 100)'),
// or type conversions
func (p *Parser) parseTypeLiteral() ast.Expression {
	tkn := p.curToken
	t := p.parseType()
	// handle type conversion
	if p.curTokenIs(token.LPAREN) {
		exp := &ast.TypeCastExpression{Token: tkn, Typ: t}
		p.nextToken()

		exp.Argument = p.parseExpression(LOWEST)

		if !p.curTokenIs(token.RPAREN) {
			p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
			return nil
		}
		p.nextToken()
		return exp
	} else {
		return &ast.TypeLiteral{Token: tkn, T: t}
	}
}

func (p *Parser) parseTryExpression() ast.Expression {
	exp := &ast.TryExpression{Token: p.curToken}
	p.nextToken()

	// Allow any expression after try (function calls, dot access, etc.)
	// Use LOWEST precedence to capture the entire expression
	exp.Right = p.parseExpression(LOWEST)
	return exp
}

func (p *Parser) parseCatchExpression(left ast.Expression) ast.Expression {
	exp := &ast.CatchExpression{
		Token: p.curToken,
		Left:  left,
	}
	p.nextToken()

	// Parse error identifier
	if !p.curTokenIs(token.IDENT) {
		p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
		return nil
	}
	exp.Ident = p.parseIdentifier()

	// Parse catch block
	if !p.curTokenIs(token.LBRACE) {
		p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
		return nil
	}
	exp.Block = p.parseBlockStatement()

	return exp
}

// 0..10
// func (p *Parser) parseRangeExpression() ast.Expression {
// 	stmt := &ast.RangeExpression{Token: p.curToken}

// 	if !p.curTokenIs(token.INT) {
// 		return nil
// 	}

// 	stmt.StartValue = p.parseIntegerLiteral().(*ast.IntegerLiteral)

// 	if !p.curTokenIs(token.RANGE) {
// 		return nil
// 	}
// 	p.nextToken()

// 	if !p.curTokenIs(token.INT) {
// 		return nil
// 	}
// 	stmt.EndValue = p.parseIntegerLiteral().(*ast.IntegerLiteral)

// 	return stmt
// }

// ------- //
// Literal //
// ------- //

func (p *Parser) parseIntegerLiteral() ast.Expression {
	lit := &ast.IntegerLiteral{Token: p.curToken, T: &types.ConstI64}
	i := strings.ReplaceAll(p.curToken.Literal, "_", "")
	intValue, err := strconv.ParseInt(i, 10, 64)
	if err != nil {
		p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
		p.nextToken()
		return lit
	}
	lit.Value = intValue
	p.nextToken()
	return lit
}

func (p *Parser) parseHexLiteral() ast.Expression {
	lit := &ast.HexLiteral{Token: p.curToken, T: &types.ConstI64}
	// Remove "0x" prefix and underscores
	hexStr := strings.ReplaceAll(p.curToken.Literal[2:], "_", "")
	hexValue, err := strconv.ParseUint(hexStr, 16, 64)
	if err != nil {
		p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
		p.nextToken()
		return lit
	}
	lit.Value = hexValue
	p.nextToken()
	return lit
}

func (p *Parser) parseFloatLiteral() ast.Expression {
	lit := &ast.FloatLiteral{Token: p.curToken}
	floatValue, err := strconv.ParseFloat(p.curToken.Literal, 64)
	if err != nil {
		p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
		return nil
	}
	lit.Value = floatValue
	p.nextToken()
	return lit
}

func (p *Parser) parseStringLiteral() ast.Expression {
	lit := &ast.StringLiteral{Token: p.curToken}
	p.nextToken()
	return lit
}

func (p *Parser) parseCharacterLiteral() ast.Expression {
	lit := &ast.CharacterLiteral{Token: p.curToken}
	if p.curToken.Literal[0] == '\\' {
		if len(p.curToken.Literal) == 1 {
			lit.Value = '\\'
		} else if len(p.curToken.Literal) == 2 {
			switch p.curToken.Literal[1] {
			case 'a':
				lit.Value = '\a'
			case 'b':
				lit.Value = '\b'
			case 'n':
				lit.Value = '\n'
			case 'r':
				lit.Value = '\r'
			case 't':
				lit.Value = '\t'
			case '\'':
				lit.Value = '\''
			case '\\':
				lit.Value = '\\'
			}
		} else {
			panic("\\u literals not supported yet")
		}
	} else {
		lit.Value = rune(p.curToken.Literal[0])
	}
	p.nextToken()
	return lit
}

func (p *Parser) parseBooleanLiteral() ast.Expression {
	lit := &ast.BooleanLiteral{Token: p.curToken, Value: false}
	if p.curToken.Literal == "true" {
		lit.Value = true
	}
	p.nextToken()
	return lit
}

func (p *Parser) parseNullLiteral() ast.Expression {
	lit := &ast.NullLiteral{Token: p.curToken}
	lit.Value = p.curToken.Literal
	p.nextToken()
	return lit
}

func (p *Parser) parseAnonymousStructLiteral() ast.Expression {
	return p.parseStructLiteral(nil, nil)
}

// parseStructLiteral parses a struct literal with optional type parameters
// Assumes current token is LBRACE
func (p *Parser) parseStructLiteral(exp ast.Expression, typeParams []types.Type) ast.Expression {
	lit := &ast.StructLiteral{
		Token:          p.curToken,
		Name:           exp,
		TypeParameters: typeParams,
	}
	if !p.curTokenIs(token.LBRACE) {
		p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
		return nil
	}
	p.nextToken()

	if p.curTokenIs(token.RANGE) {
		p.nextToken()
		lit.Copy = p.parseExpression(LOWEST)

		if !p.curTokenIs(token.COMMA) {
			p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
		}
		p.nextToken()
	}

	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		if p.peekTokenIs(token.COLON) {
			// case 1: struct with named field
			field := p.parseStructFieldLiteral()
			lit.Fields = append(lit.Fields, field)
		} else {
			// case 2: struct with unnamed field
			field := p.parseUnnamedStructFieldLiteral()
			lit.Fields = append(lit.Fields, field)
		}

		if p.curTokenIs(token.COMMA) {
			p.nextToken()
		}
	}

	p.nextToken()

	return lit
}

func (p *Parser) parseStructFieldLiteral() *ast.StructFieldLiteral {
	name := p.parseIdentifier()
	if !p.curTokenIs(token.COLON) {
		p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
		return nil
	}
	p.nextToken()
	value := p.parseExpression(LOWEST)
	return &ast.StructFieldLiteral{Token: p.curToken, Name: name, Value: value}
}

func (p *Parser) parseUnnamedStructFieldLiteral() *ast.StructFieldLiteral {
	value := p.parseExpression(LOWEST)
	return &ast.StructFieldLiteral{Token: p.curToken, Value: value}
}

// - []byte
// - [3]byte
// - [][]i64()
// - [3]string()
// - []
// - [1,3,4]
func (p *Parser) parseArrayLiteralTypeOrCast() ast.Expression {
	tkn := p.curToken
	if p.peekTokenIs(token.RBRACK) {
		p.nextToken()
		p.nextToken()
		if p.curTokenIsType() {
			// case: array type cast
			arrayType := &types.Array{T: p.parseType()}
			if p.curTokenIs(token.LPAREN) {
				p.nextToken()
				exp := p.parseExpression(LOWEST)
				if !p.curTokenIs(token.RPAREN) {
					p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
					return nil
				}
				p.nextToken()
				return &ast.TypeCastExpression{Token: tkn, Argument: exp, Typ: arrayType}
			}
			// case: array type
			return &ast.TypeLiteral{Token: tkn, T: arrayType}
		}
		// case: empty array literal
		return &ast.ArrayLiteral{Token: p.curToken}
	}
	p.nextToken()
	if p.curTokenIs(token.INT) && p.peekTokenIs(token.RBRACK) {
		size := p.parseIntegerLiteral().(*ast.IntegerLiteral)
		if !p.curTokenIs(token.RBRACK) {
			p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
			return nil
		}
		p.nextToken()
		// case: one element int array
		if !p.curTokenIsType() {
			typ := &types.Array{T: size.T, Size: 1}
			return &ast.ArrayLiteral{Token: tkn, Values: []ast.Expression{size}, T: typ}
		}
		// case: sized array type cast
		sizedArrayType := &types.Array{T: p.parseType(), Size: int(size.Value)}
		if p.curTokenIs(token.LPAREN) {
			p.nextToken()
			exp := p.parseExpression(LOWEST)
			return &ast.TypeCastExpression{Token: tkn, Argument: exp, Typ: sizedArrayType}
		}
		// case: sized array type
		return &ast.TypeLiteral{Token: tkn, T: sizedArrayType}
	}

	// case: array literal with values
	lit := &ast.ArrayLiteral{Token: tkn}

	for !p.curTokenIs(token.RBRACK) && !p.curTokenIs(token.EOF) {
		if p.curTokenIs(token.COMMA) {
			p.nextToken()
		}
		// required for [a,]
		if p.curTokenIs(token.RBRACK) {
			break
		}
		if p.curTokenIs(token.COMMENT) {
			p.nextToken()
			continue
		}
		lit.Values = append(lit.Values, p.parseExpression(LOWEST))
	}

	lit.T = &types.Array{T: lit.Values[0].Type()}

	p.nextToken()
	return lit
}

func (p *Parser) parseWildcardLiteral() ast.Expression {
	lit := &ast.WildcardLiteral{Token: p.curToken}
	p.nextToken()
	return lit
}

// assumes current token is '@' when function called
func (p *Parser) parseAttribute() ast.Attribute {
	p.nextToken()

	if !p.curTokenIs(token.IDENT) {
		p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
		return nil
	}

	fn, ok := p.attributeParseFns[p.curToken.Literal]
	if !ok {
		p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
		return nil
	}

	return fn()
}

// Parser non parametrized attributed e.g. "test"
func (p *Parser) parseBasicAttribute() ast.Attribute {
	if !p.curTokenIs(token.IDENT) {
		p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
		return nil
	}

	switch p.curToken.Literal {
	case "test":
		p.nextToken()
		return &ast.BasicAttribute{Type: ast.Test}
	}

	p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
	return nil
}

// assumes current token literal is 'extern' when function called
func (p *Parser) parseExternAttribute() ast.Attribute {
	p.nextToken()
	if !p.curTokenIs(token.LPAREN) {
		p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
		return nil
	}
	p.nextToken()

	var attr *ast.BasicAttribute
	if p.curToken.Literal == "c" {
		attr = &ast.BasicAttribute{Type: ast.ExternC}
	} else {
		p.addError(p.curToken, errInvalidAttributeArgument("extern", p.curToken.Literal))
		return nil
	}
	p.nextToken()
	if !p.curTokenIs(token.RPAREN) {
		p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
		return nil
	}
	p.nextToken()

	return attr
}

// assumes current token literal is 'inline' when function called
func (p *Parser) parseInlineAttribute() ast.Attribute {
	p.nextToken()
	if !p.curTokenIs(token.LPAREN) {
		p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
		return nil
	}
	p.nextToken()

	var attr *ast.BasicAttribute
	switch p.curToken.Literal {
	case "never":
		attr = &ast.BasicAttribute{Type: ast.InlineNever}
	case "hint":
		attr = &ast.BasicAttribute{Type: ast.InlineHint}
	case "always":
		attr = &ast.BasicAttribute{Type: ast.InlineAlways}
	default:
		p.addError(p.curToken, errInvalidAttributeArgument("inline", p.curToken.Literal))
		return nil
	}
	p.nextToken()
	if !p.curTokenIs(token.RPAREN) {
		p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
		return nil
	}
	p.nextToken()

	return attr
}

func (p *Parser) parseFunctionExpression() ast.Expression {
	attrs := p.attributes.PopAll()

	lit := &ast.FunctionExpression{Attributes: attrs}

	if p.curTokenIs(token.PUBLIC) {
		lit.Public = true
		p.nextToken()
	} else if p.prevTokenIs(token.PUBLIC) {
		lit.Public = true
	}
	lit.Token = p.curToken

	if !p.curTokenIs(token.FUNCTION) {
		return nil
	}
	p.nextToken()

	if p.curTokenIs(token.IDENT) || p.curTokenIs(token.MAIN) {
		lit.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
		p.nextToken()
	} else {
		lit.IsAnonymous = true
	}

	// Parse generic parameters if present
	if p.curTokenIs(token.LBRACK) {
		p.nextToken()
		lit.GenericParameters = p.parseGenericParameters()
		if !p.curTokenIs(token.RBRACK) {
			p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
			return nil
		}
		p.nextToken()
	}

	if !p.curTokenIs(token.LPAREN) {
		p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
		return nil
	}

	p.nextToken()

	// Parse function arguments
	for !p.curTokenIs(token.RPAREN) && !p.curTokenIs(token.EOF) {
		if p.curTokenIs(token.COMMA) {
			p.nextToken()
		}
		lit.Arguments = append(lit.Arguments, p.parseParameterStatement(true))
	}
	p.nextToken()

	// TODO: parse if function is errorable

	if p.curTokenIs(token.BANG) {
		lit.ErrorProne = true
		p.nextToken()
	}

	// parse return arguments
	for p.curTokenIsType() && !p.curTokenIs(token.EOF) {
		lit.ReturnValues = append(lit.ReturnValues, p.parseTypeLiteral().(*ast.TypeLiteral))
		// skip commas
		if p.curTokenIs(token.COMMA) {
			p.nextToken()
		}
	}

	// only parse body if defined
	if p.curTokenIs(token.LBRACE) {
		lit.Body = p.parseBlockStatement()
	}

	return lit
}

// parseGenericParameters parses generic parameter lists like:
// T any
// T, E any (creates T and E with 'any' constraint)
// T any, E int
func (p *Parser) parseGenericParameters() []*ast.GenericParameter {
	var params []*ast.GenericParameter
	var pendingNames []*ast.Identifier

	for !p.curTokenIs(token.RBRACK) && !p.curTokenIs(token.EOF) {

		if !p.curTokenIs(token.IDENT) {
			p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
			return nil
		}

		name := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
		p.nextToken()

		// case 1: short hand (constraint defined later)
		if p.curTokenIs(token.COMMA) {
			p.nextToken()
			pendingNames = append(pendingNames, name)
			continue
		}

		// case 2: parse constraint if there is one
		if !p.curTokenIsType() && !p.curTokenIsIdent() {
			p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
			return nil
		}

		constraint := p.parseType()

		// fist we set all pendingNames to same constraint
		// as parsed now
		for _, pendingName := range pendingNames {
			param := &ast.GenericParameter{
				Name:       pendingName,
				Constraint: constraint,
			}
			params = append(params, param)
		}
		pendingNames = nil

		param := &ast.GenericParameter{
			Name:       name,
			Constraint: constraint,
		}
		params = append(params, param)

		if p.curTokenIs(token.COMMA) {
			p.nextToken()
			continue
		}
	}

	if len(pendingNames) > 0 {
		p.addError(p.curToken, errMissingGenericConstraint(pendingNames[0].Value))
		return nil
	}

	return params
}

// ----- //
// Types //
// ----- //

func (p *Parser) parseType() types.Type {
	var typ types.Type
	switch p.curToken.Type {
	case token.OPTIONAL:
		p.nextToken()
		return &types.Optional{T: p.parseType()}
	case token.LBRACK:
		typ = p.parseArrayType()
	case token.FUNCTION:
		typ = p.parseFunctionType()
	case token.IDENT:
		if p.peekTokenIs(token.DOT) {
			typ = p.parseImportedNamedType()
		} else {
			typ = p.parseUnknownNamedType()
		}
	case token.ASTERISK:
		typ = p.parsePointerType()
	case token.MUTABLETYPE:
		typ = p.parseMutableType()
	case token.DIRTYTYPE:
		typ = p.parseDirtyType()
	default:
		typ = p.parsePrimitiveType()
	}

	return typ
}

// []int, [10]int
func (p *Parser) parseArrayType() types.Type {
	p.nextToken()

	typ := &types.Array{}

	if p.curTokenIs(token.INT) {
		typ.Size = int(p.parseIntegerLiteral().(*ast.IntegerLiteral).Value)
	}

	if !p.curTokenIs(token.RBRACK) {
		p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
		return nil
	}
	p.nextToken()

	typ.T = p.parseType()

	return typ
}

// fn(i64) i64 or fn()
func (p *Parser) parseFunctionType() *types.Function {
	typ := &types.Function{}

	if !p.curTokenIs(token.FUNCTION) {
		p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
		return nil
	}

	p.nextToken()

	if !p.curTokenIs(token.LPAREN) {
		return nil
	}

	p.nextToken()

	for !p.curTokenIs(token.RPAREN) {
		if p.curTokenIs(token.COMMA) {
			p.nextToken()
		}
		typ.Arg = append(typ.Arg, p.parseType())
	}
	p.nextToken()

	// case: f fn(), or f fn()) or eof
	if p.curTokenIs(token.COMMA) || p.curTokenIs(token.RPAREN) || p.curTokenIs(token.EOF) {
		return typ
	}
	// case: f fn()!
	if p.curTokenIs(token.BANG) {
		typ.IsErrorProne = true
		p.nextToken()
		return typ
	}

	for !p.curTokenIs(token.EOF) {
		typ.Ret = append(typ.Ret, p.parseType())
		if p.curTokenIs(token.COMMA) {
			p.nextToken()
		}
		if !p.curTokenIsType() {
			return typ
		}
	}
	// TODO: handle this gracefully
	panic("unreachable")
}

func (p *Parser) parsePrimitiveType() types.Type {
	if !p.curTokenIsType() {
		return nil
	}

	tkn := types.TokenToType(p.curToken)
	p.nextToken()
	return tkn
}

func (p *Parser) parseImportedNamedType() types.Type {
	typ := &types.ImportedNamed{Lib: p.curToken.Literal}
	p.nextToken()
	// wat "." token
	p.nextToken()
	typ.Typ = p.parseType()

	return typ
}

func (p *Parser) parseUnknownNamedType() types.Type {
	name := p.curToken.Literal
	p.nextToken()

	// Check for generic type parameters
	if p.curTokenIs(token.LBRACK) {
		params := p.parseTypeParameters()
		// parseTypeParameters consumes the ']', so we're past it now

		typ := &types.UnknownNamed{Name: name, TypeParameters: params}
		if p.curTokenIs(token.OPTIONAL) {
			p.nextToken()
			return &types.Optional{T: typ}
		}
		return typ
	}

	typ := &types.UnknownNamed{Name: name}
	if p.curTokenIs(token.OPTIONAL) {
		p.nextToken()
		return &types.Optional{T: typ}
	}
	return typ
}

func (p *Parser) parsePointerType() types.Type {
	typ, _ := types.TokenToType(p.curToken).(*types.Pointer)
	p.nextToken()

	typ.T = p.parseType()
	return typ
}

func (p *Parser) parseMutableType() types.Type {
	typ, _ := types.TokenToType(p.curToken).(*types.Mutable)
	p.nextToken()

	if !p.curTokenIs(token.LBRACK) {
		p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
		return nil
	}
	p.nextToken()

	typ.T = p.parseType()

	if !p.curTokenIs(token.RBRACK) {
		p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
		return nil
	}
	return typ
}

func (p *Parser) parseDirtyType() types.Type {
	typ, _ := types.TokenToType(p.curToken).(*types.Dirty)
	p.nextToken()

	if !p.curTokenIs(token.LT) {
		p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
		return nil
	}
	p.nextToken()

	typ.T = p.parseType()

	if !p.curTokenIs(token.GT) {
		p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
		return nil
	}
	return typ
}

// parseTypeParameters parses comma-separated types inside brackets
func (p *Parser) parseTypeParameters() []types.Type {
	if !p.curTokenIs(token.LBRACK) {
		p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
		return nil
	}
	p.nextToken()

	var params []types.Type
	for !p.curTokenIs(token.RBRACK) && !p.curTokenIs(token.EOF) {
		if p.curTokenIs(token.COMMA) {
			p.nextToken()
		}
		param := p.parseType()
		params = append(params, param)
	}

	if !p.curTokenIs(token.RBRACK) {
		p.addError(p.curToken, errInvalidToken(p.curToken.Literal))
		return nil
	}
	p.nextToken() // consume ']'

	return params
}

func (p *Parser) addError(tkn token.Token, err error) {
	pos := tkn.Position
	msg := fmt.Sprintf("[ERROR] Parser failed in %s at %d:%d - %s", p.l.Filename(), pos.Line(), pos.Column(), err)
	p.errors = append(p.errors, msg)
}

// ------- //
// Helpers //
// ------- //

func (p *Parser) curPrecedence() int {
	if p, ok := precedences[p.curToken.Type]; ok {
		return p
	}
	return LOWEST
}

func (p *Parser) prevTokenIs(t token.Type) bool {
	if p.tknIdx-1 < 0 {
		return false
	}
	return p.tkns[p.tknIdx-1].Type == t
}

func (p *Parser) curTokenIs(t token.Type) bool {
	return p.curToken.Type == t
}

func (p *Parser) peekTokenIs(t token.Type) bool {
	return p.peekNTokenIs(1, t)
}

func (p *Parser) peekNTokenIs(n int, t token.Type) bool {
	if p.tknIdx+n >= len(p.tkns) {
		return false
	}
	return p.tkns[p.tknIdx+n].Type == t
}

func (p *Parser) curTokenIsOperator() bool {
	return p.curToken.Type == token.EQ ||
		p.curToken.Type == token.ASSIGN ||
		p.curToken.Type == token.BACKSLASH ||
		p.curToken.Type == token.COLON ||
		p.curToken.Type == token.PLUS ||
		p.curToken.Type == token.MINUS ||
		p.curToken.Type == token.MOD ||
		p.curToken.Type == token.SLASH ||
		p.curToken.Type == token.ASTERISK ||
		p.curToken.Type == token.AMPERSAND ||
		p.curToken.Type == token.AND ||
		p.curToken.Type == token.OR ||
		p.curToken.Type == token.BANG ||
		p.curToken.Type == token.GTE ||
		p.curToken.Type == token.LTE ||
		p.curToken.Type == token.NEQ ||
		p.curToken.Type == token.GT ||
		p.curToken.Type == token.LT ||
		p.curToken.Type == token.BAR ||
		p.curToken.Type == token.LSHIFT ||
		p.curToken.Type == token.RSHIFT ||
		p.curToken.Type == token.CARET ||
		p.curToken.Type == token.BNOT ||
		p.curToken.Type == token.BANDNOT ||
		p.curToken.Type == token.PIPE ||
		p.curToken.Type == token.NULL_COALESCE ||
		p.curToken.Type == token.OPTIONAL ||
		p.curToken.Type == token.INCR ||
		p.curToken.Type == token.DECR ||
		p.curToken.Type == token.ARROW
}

func (p *Parser) curTokenIsType() bool {
	return p.curToken.Type == token.BOOLTYPE ||
		p.curToken.Type == token.I8TYPE ||
		p.curToken.Type == token.U8TYPE ||
		p.curToken.Type == token.I16TYPE ||
		p.curToken.Type == token.U16TYPE ||
		p.curToken.Type == token.I32TYPE ||
		p.curToken.Type == token.U32TYPE ||
		p.curToken.Type == token.I64TYPE ||
		p.curToken.Type == token.U64TYPE ||
		p.curToken.Type == token.F32TYPE ||
		p.curToken.Type == token.F64TYPE ||
		p.curToken.Type == token.STRINGTYPE ||
		p.curToken.Type == token.BYTETYPE ||
		p.curToken.Type == token.CHARTYPE ||
		p.curToken.Type == token.LBRACK ||
		(p.curToken.Type == token.FUNCTION && p.peekToken().Type == token.LPAREN) ||
		p.curToken.Type == token.ASTERISK ||
		p.curToken.Type == token.MUTABLETYPE ||
		p.curToken.Type == token.IDENT ||
		p.curToken.Type == token.OPTIONAL ||
		p.curToken.Type == token.DIRTYTYPE ||
		p.curToken.Type == token.ERROR ||
		p.curToken.Type == token.ANYTYPE
}

func (p *Parser) peekNTokenIsType(n int) bool {
	peekTkn := p.peekNToken(n)
	return peekTkn.Type == token.BOOLTYPE ||
		peekTkn.Type == token.I8TYPE ||
		peekTkn.Type == token.U8TYPE ||
		peekTkn.Type == token.I16TYPE ||
		peekTkn.Type == token.U16TYPE ||
		peekTkn.Type == token.I32TYPE ||
		peekTkn.Type == token.U32TYPE ||
		peekTkn.Type == token.I64TYPE ||
		peekTkn.Type == token.U64TYPE ||
		peekTkn.Type == token.F32TYPE ||
		peekTkn.Type == token.F64TYPE ||
		peekTkn.Type == token.STRINGTYPE ||
		peekTkn.Type == token.BYTETYPE ||
		peekTkn.Type == token.CHARTYPE ||
		peekTkn.Type == token.LBRACK ||
		(peekTkn.Type == token.FUNCTION && p.peekNTokenIs(2, token.LPAREN)) ||
		peekTkn.Type == token.ASTERISK ||
		peekTkn.Type == token.MUTABLETYPE ||
		peekTkn.Type == token.IDENT ||
		peekTkn.Type == token.OPTIONAL ||
		peekTkn.Type == token.DIRTYTYPE ||
		peekTkn.Type == token.ERROR ||
		peekTkn.Type == token.ANYTYPE
}

func (p *Parser) curTokenIsConditionalStatement() bool {
	return p.curTokenIs(token.ELSEIF) || p.curTokenIs(token.ELSE)
}
