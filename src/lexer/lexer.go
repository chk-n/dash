// This package implements the lexical analysis for the Dash language,
// converting source code to tokens. To generate tokens repeated calls
//  to 'NextToken()' are required.

package lexer

import (
	"dash-lang.io/src/token"
)

// TODO: support bytes e.g. 0b
// TODO: support hex e.g. 0x
// TODO: support octal 0o
// TODO: get tab size from config (currently 4 is assumed)

type Config struct {
	SkipComments bool
}

type Lexer struct {
	fileName string
	input    string
	// current read position
	pos     int
	prevPos int
	// current character being looked at
	ch byte
	//
	// keep track where we are in source code
	lineNumber   int
	columnNumber int
	prevToken    token.Token

	skipComments bool
}

func New(filename, input string, cfg *Config) *Lexer {
	l := &Lexer{
		fileName:     filename,
		input:        input,
		skipComments: cfg.SkipComments,
	}
	l.next()
	l.lineNumber = 1
	l.columnNumber = 1
	return l
}

func (l *Lexer) Filename() string {
	return l.fileName
}

func (l *Lexer) NextToken() token.Token {
	pos := token.NewPos(l.lineNumber, l.columnNumber)
	tok := token.Token{Type: token.EOF, Literal: "", Position: pos}

	// EOF
	if l.ch == 0 {
		return tok
	}

	l.skipWhitespace()
	col := l.columnNumber

	switch l.ch {
	case '"':
		tok.Type = token.STRING
		tok.Literal = l.readString()
	case '`':
		tok.Type = token.STRING
		tok.Literal = l.readMultilineString()
	case '\'':
		tok.Type = token.CHAR
		tok.Literal = l.readChar()
	case ':':
		tok = token.New(token.COLON, l.ch)
	case '|':
		if l.peekChar() == '|' {
			l.next()
			tok.Literal = "||"
			tok.Type = token.OR
		} else if l.peekChar() == '>' {
			l.next()
			tok.Literal = "|>"
			tok.Type = token.PIPE
		} else {
			tok = token.New(token.BAR, l.ch)
		}
	case '+':
		if l.peekChar() == '+' {
			l.next()
			tok.Literal = "++"
			tok.Type = token.INCR
		} else {
			tok = token.New(token.PLUS, l.ch)
		}
	case '-':
		if l.peekChar() == '-' {
			l.next()
			tok.Literal = "--"
			tok.Type = token.DECR
		} else {
			tok = token.New(token.MINUS, l.ch)
		}
	case '!':
		if l.peekChar() == '=' {
			l.next()
			tok.Literal = "!="
			tok.Type = token.NEQ
		} else {
			tok = token.New(token.BANG, l.ch)
		}
	case '=':
		if l.peekChar() == '=' {
			l.next()
			tok.Literal = "=="
			tok.Type = token.EQ
		} else {
			tok = token.New(token.ASSIGN, l.ch)
		}
	case '>':
		if l.peekChar() == '=' {
			l.next()
			tok.Literal = ">="
			tok.Type = token.GTE
		} else if l.peekChar() == '>' {
			// NOTE: we purposly ignore this as nested memory or array types
			// can close using '>>'
			tok = token.New(token.GT, l.ch)
		} else {
			tok = token.New(token.GT, l.ch)
		}
	case '<':
		if l.peekChar() == '=' {
			l.next()
			tok.Literal = "<="
			tok.Type = token.LTE
		} else if l.peekChar() == '<' {
			l.next()
			tok.Literal = "<<"
			tok.Type = token.LSHIFT
		} else {
			tok = token.New(token.LT, l.ch)
		}
	case '&':
		if l.peekChar() == '&' {
			l.next()
			tok.Literal = "&&"
			tok.Type = token.AND
		} else if l.peekChar() == '^' {
			l.next()
			tok.Literal = "&^"
			tok.Type = token.BANDNOT
		} else {
			tok = token.New(token.AMPERSAND, l.ch)
		}
	case '/':
		if l.peekChar() == '/' {
			tok.Literal = l.readComment()
			tok.Type = token.COMMENT
			if l.skipComments {
				l.next()
				return l.NextToken()
			}

		} else if l.peekChar() == '*' {
			l.next()
			tok.Literal = "/*"
			tok.Type = token.LMCOMMENT
		} else {
			tok = token.New(token.SLASH, l.ch)
		}
	case '*':
		if l.peekChar() == '/' {
			l.next()
			tok.Literal = "*/"
			tok.Type = token.RMCOMMENT
		} else {
			tok = token.New(token.ASTERISK, l.ch)
		}
	case '.':
		if l.peekChar() == '.' {
			l.next()
			if l.peekChar() == '.' {
				l.next()
				tok.Literal = "..."
				tok.Type = token.ELLIPSIS
			} else {
				tok.Literal = ".."
				tok.Type = token.RANGE
			}
		} else {
			tok = token.New(token.DOT, l.ch)
		}
	case '?':
		if l.peekChar() == '?' {
			l.next()
			tok.Literal = "??"
			tok.Type = token.NULL_COALESCE
		} else {
			tok = token.New(token.OPTIONAL, l.ch)
		}
	case '\\':
		tok = token.New(token.BACKSLASH, l.ch)
	case ';':
		tok = token.New(token.SEMI, l.ch)
	case '^':
		tok = token.New(token.CARET, l.ch)
	case '%':
		tok = token.New(token.MOD, l.ch)
	case '~':
		tok = token.New(token.BNOT, l.ch)
	case '{':
		tok = token.New(token.LBRACE, l.ch)
	case '}':
		tok = token.New(token.RBRACE, l.ch)
	case '(':
		tok = token.New(token.LPAREN, l.ch)
	case ')':
		tok = token.New(token.RPAREN, l.ch)
	case '[':
		tok = token.New(token.LBRACK, l.ch)
	case ']':
		tok = token.New(token.RBRACK, l.ch)
	case ',':
		tok = token.New(token.COMMA, l.ch)
	case '@':
		tok = token.New(token.AT, l.ch)
	default:
		if isLetter(l.ch) {
			if l.ch == '\'' {
				tok = token.New(token.ILLEGAL, l.ch)
				tok.Literal = l.readIdentifier()
				break
			}
			tok.Literal = l.readIdentifier()
			tok.Type = token.LookupKeyword(tok.Literal)
			if tok.Type == token.ELSE {
				l.skipWhitespace()
				// peek to see if "else if" statement
				literal := l.peekIdentifierN(2) // "if"
				typ := token.LookupKeyword(literal)
				if typ == token.IF {
					l.skipWhitespace()
					tok.Literal = tok.Literal + " " + l.readIdentifier()
					tok.Type = token.ELSEIF
				}
			}
		} else if isDigit(l.ch) {
			tok = l.readNumber()
		}
		tok.Position = token.NewPos(l.lineNumber, col)
		l.prevToken = tok
		return tok
	}
	tok.Position = token.NewPos(l.lineNumber, col)
	l.prevToken = tok
	l.next()
	return tok
}

// reads the next character into l.ch
// * 0 means end of file
func (l *Lexer) next() {
	// set line and column number
	if l.ch == '\n' || l.ch == '\r' {
		l.lineNumber++
		l.columnNumber = 1
	} else if l.ch == '\t' {
		// TODO: get tab size from config
		l.columnNumber += 4
	} else {
		l.columnNumber++
	}

	if l.pos >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.pos]
	}
	l.prevPos = l.pos
	l.pos++
}

func (l *Lexer) readString() string {
	pos := l.prevPos + 1
	for {
		l.next()
		if l.ch == '"' || l.ch == 0 {
			break
		}
	}
	return l.input[pos:l.prevPos]
}

func (l *Lexer) readMultilineString() string {
	pos := l.prevPos + 1
	for {
		l.next()
		if l.ch == '`' || l.ch == 0 {
			break
		}
	}
	return l.input[pos:l.prevPos]
}

func (l *Lexer) readComment() string {
	pos := l.prevPos
	for l.ch != '\n' && l.ch != 0 {
		l.next()
	}
	return l.input[pos:l.prevPos]
}

func (l *Lexer) readIdentifier() string {
	pos := l.prevPos
	for isLetter(l.ch) || isDigit(l.ch) {
		l.next()
	}
	return l.input[pos:l.prevPos]
}

func (l *Lexer) peekIdentifierN(i int) string {
	if l.pos+i >= len(l.input) {
		i = len(l.input) - l.pos
	}
	return l.input[l.prevPos : l.prevPos+i]
}

func (l *Lexer) peekChar() byte {
	if l.pos >= len(l.input) {
		return 0
	}
	return l.input[l.pos]
}

func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		l.next()
	}
}

func (l *Lexer) readNumber() token.Token {
	prevPos := l.prevPos
	var tok token.Token
	tok.Type = token.INT

	// TODO: support numbers in hex & octal notation
	for {
		if l.ch == '.' && l.peekChar() == '.' {
			break
		} else if isDigit(l.ch) {
			l.next()
		} else if l.ch == '.' {
			tok.Type = token.FLOAT
			l.next()
		} else if isLetter(l.ch) {
			tok.Type = token.ILLEGAL
			l.next()
		} else {
			break
		}
	}

	// Number can not start or end with underscore
	tok.Literal = l.input[prevPos:l.prevPos]

	if l.input[l.prevPos-1] == '_' {
		tok.Literal = l.input[prevPos : l.prevPos-1]
	}

	if tok.Literal[0] == '_' {
		tok.Literal = tok.Literal[1:]
	}

	return tok
}

// NOTE: currently we only handle
// single characters within ' '
func (l *Lexer) readChar() string {
	pos := l.prevPos + 1

	n := 0
	for {
		l.next()
		if l.ch == '\'' || l.ch == 0 {
			break
		}
		n++
	}
	if n != 1 {
		l.addError(pos, "invalid char literal")
	}

	return l.input[pos:l.prevPos]
}

func (l *Lexer) addError(offs int, rsn string) {
	// l.
}

func isLetter(ch byte) bool {
	return 'a' <= ch && ch <= 'z' ||
		'A' <= ch && ch <= 'Z' ||
		ch == '_' || ch == '\''
}

func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9' ||
		ch == '_'
}
