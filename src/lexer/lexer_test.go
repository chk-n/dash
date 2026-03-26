package lexer

import (
	"testing"

	"dash-lang.io/src/token"
)

func TestNextToken(t *testing.T) {
	type wantToken struct {
		literal   string
		tokenType token.Type
	}
	tests := []struct {
		name   string
		input  string
		tokens []wantToken
	}{
		{
			name: "legal identifiers",
			input: `
				test1
				_var
				_var_
				var_
				v'
			`,
			tokens: []wantToken{
				{"test1", token.IDENT},
				{"_var", token.IDENT},
				{"_var_", token.IDENT},
				{"var_", token.IDENT},
				{"v'", token.IDENT},
			},
		},
		{
			name: "illegal identifiers",
			input: `
				5v`,
			tokens: []wantToken{
				{"5v", token.ILLEGAL},
			},
		},
		{
			name: "literals",
			input: `
				5
				10.1
				false
				"hello, world"
				1_000
				10_000.1
				null
				'2'
				`,
			tokens: []wantToken{
				{"5", token.INT},
				{"10.1", token.FLOAT},
				{"false", token.BOOL},
				{"hello, world", token.STRING},
				{"1_000", token.INT},
				{"10_000.1", token.FLOAT},
				{"null", token.NULL},
				{"2", token.CHAR},
			},
		},
		{
			name: "operators",
			input: `
				=
				\
				:
				+
				-
				%
				/
				*
				&
				&&
				||
				!
				>=
				<=
				==
				!=
				>
				<
				|
				<<
				^
				~
				&^
				,
				(
				)
				{
				}
				[
				]
				.
				;
				?
				|>
				//
				/*
				*/
				_
				..
				...
				++
				--
				@
				?
				??
			`,
			tokens: []wantToken{
				{"=", token.ASSIGN},
				{"\\", token.BACKSLASH},
				{":", token.COLON},
				{"+", token.PLUS},
				{"-", token.MINUS},
				{"%", token.MOD},
				{"/", token.SLASH},
				{"*", token.ASTERISK},
				{"&", token.AMPERSAND},
				{"&&", token.AND},
				{"||", token.OR},
				{"!", token.BANG},
				{">=", token.GTE},
				{"<=", token.LTE},
				{"==", token.EQ},
				{"!=", token.NEQ},
				{">", token.GT},
				{"<", token.LT},
				{"|", token.BAR},
				{"<<", token.LSHIFT},
				// {">>", token.RSHIFT},
				{"^", token.CARET},
				{"~", token.BNOT},
				{"&^", token.BANDNOT},
				{",", token.COMMA},
				{"(", token.LPAREN},
				{")", token.RPAREN},
				{"{", token.LBRACE},
				{"}", token.RBRACE},
				{"[", token.LBRACK},
				{"]", token.RBRACK},
				{".", token.DOT},
				{";", token.SEMI},
				{"?", token.OPTIONAL},
				{"|>", token.PIPE},
				{"//", token.COMMENT},
				{"/*", token.LMCOMMENT},
				{"*/", token.RMCOMMENT},
				{"_", token.WILDCARD},
				{"..", token.RANGE},
				{"...", token.ELLIPSIS},
				{"++", token.INCR},
				{"--", token.DECR},
				{"@", token.AT},
				{"?", token.OPTIONAL},
				{"??", token.NULL_COALESCE},
			},
		},
		{
			name: "type keywords",
			input: `
				many
				int
				i8
				i16
				i32
				i64
				u8
				u16
				u32
				u64
				float
				f32
				f64
				string
				bool
				byte
				char
				memory
			`,
			tokens: []wantToken{
				{"many", token.MANYTYPE},
				{"int", token.INTTYPE},
				{"i8", token.I8TYPE},
				{"i16", token.I16TYPE},
				{"i32", token.I32TYPE},
				{"i64", token.I64TYPE},
				{"u8", token.U8TYPE},
				{"u16", token.U16TYPE},
				{"u32", token.U32TYPE},
				{"u64", token.U64TYPE},
				{"float", token.FLOATTYPE},
				{"f32", token.F32TYPE},
				{"f64", token.F64TYPE},
				{"string", token.STRINGTYPE},
				{"bool", token.BOOLTYPE},
				{"byte", token.BYTETYPE},
				{"char", token.CHARTYPE},
				{"memory", token.MEMORYTYPE},
			},
		},
		{
			name: "keywords",
			input: `
				lib
				pub
				struct
				enum
				type
				fn
				if
				else
				error
				try
				catch
				raise
				defer
				return
				for
				in
				break
				next
				match
				alias
				else if
				let
				var
				use
				case
			`,
			tokens: []wantToken{
				{"lib", token.LIBRARY},
				{"pub", token.PUBLIC},
				{"struct", token.STRUCT},
				{"enum", token.ENUM},
				{"type", token.TYPE},
				{"fn", token.FUNCTION},
				{"if", token.IF},
				{"else", token.ELSE},
				{"error", token.ERROR},
				{"try", token.TRY},
				{"catch", token.CATCH},
				{"raise", token.RAISE},
				{"defer", token.DEFER},
				{"return", token.RETURN},
				{"for", token.FOR},
				{"in", token.IN},
				{"break", token.BREAK},
				{"next", token.NEXT},
				{"match", token.MATCH},
				{"alias", token.ALIAS},
				// BUG: swapping else if and as in above input causes error.
				{"else if", token.ELSEIF},
				{"let", token.LET},
				{"var", token.VAR},
				{"use", token.USE},
				{"case", token.CASE},
			},
		},
		{
			name: "function ident",
			input: `
				some_func()
				`,
			tokens: []wantToken{
				{"some_func", token.IDENT},
				{"(", token.LPAREN},
				{")", token.RPAREN},
			},
		},
		{
			name:  "attributes",
			input: "@extern(c)",
			tokens: []wantToken{
				{"@", token.AT},
				{"extern", token.IDENT},
				{"(", token.LPAREN},
				{"c", token.IDENT},
				{")", token.RPAREN},
			},
		},

		{
			name: "comments",
			input: `// some comment
				
				// other comment //
				// "fn"
				`,
			tokens: []wantToken{
				{"// some comment", token.COMMENT},
				{"// other comment //", token.COMMENT},
				{"// \"fn\"", token.COMMENT},
			},
		},
		{
			name: "char",
			input: `
				'a'
				'\n'
				'\''
				'\\'
			`,
			tokens: []wantToken{
				{"a", token.CHAR},
				{`\n`, token.CHAR},
				{`\'`, token.CHAR},
				{`\\`, token.CHAR},
			},
		},
		{
			name: "string with escaped quote",
			input: `"say \"hello\""`,
			tokens: []wantToken{
				{`say \"hello\"`, token.STRING},
			},
		},
		{
			name: "string with multiple escapes",
			input: `"line1\nline2\ttab\r\nline3"`,
			tokens: []wantToken{
				{`line1\nline2\ttab\r\nline3`, token.STRING},
			},
		},
		{
			name: "string with backslash escapes",
			input: `"path\\to\\file"`,
			tokens: []wantToken{
				{`path\\to\\file`, token.STRING},
			},
		},
	}

	for _, tc := range tests {
		lcfg := &Config{}
		l := New("test.dash", tc.input, lcfg)
		t.Log(tc.name)
		i := 0
		for {
			tok := l.NextToken()
			if tok.Type == token.EOF {
				break
			}

			if tok.Literal != tc.tokens[i].literal {
				t.Errorf("\texpected token literal  %s, got %s", tc.tokens[i].literal, tok.Literal)
			}
			if tok.Type != tc.tokens[i].tokenType {
				t.Errorf("\t%s: expected type %d, got %d", tc.tokens[i].literal, tc.tokens[i].tokenType, tok.Type)
			}
			i++
		}
	}
}

func TestRowColumn(t *testing.T) {
	type wantToken struct {
		literal   string
		tokenType token.Type
		line      int
		column    int
	}
	tests := []struct {
		name   string
		input  string
		tokens []wantToken
	}{

		{
			name:  "enter - whitespace",
			input: "2+ 1",
			tokens: []wantToken{
				{"2", token.INT, 1, 1},
				{"+", token.PLUS, 1, 2},
				{"1", token.INT, 1, 4},
			},
		},
		{
			name: "return whitespace",
			input: `1
2
3
4`,
			tokens: []wantToken{
				{"1", token.INT, 1, 1},
				{"2", token.INT, 2, 1},
				{"3", token.INT, 3, 1},
				{"4", token.INT, 4, 1},
			},
		},
	}

	for _, tc := range tests {
		lcfg := &Config{}
		l := New("test.dash", tc.input, lcfg)
		t.Log(tc.name)
		for i := range tc.tokens {
			tok := l.NextToken()
			if tok.Literal != tc.tokens[i].literal {
				t.Errorf("expected token literal  %s, got %s", tc.tokens[i].literal, tok.Literal)
			}
			if tok.Type != tc.tokens[i].tokenType {
				t.Errorf("expected type %d, got %d", tc.tokens[i].tokenType, tok.Type)
			}
			if tok.Position.Line() != tc.tokens[i].line {
				t.Errorf("expected line %d, got %d", tc.tokens[i].line, tok.Position.Line())
			}
			if tok.Position.Column() != tc.tokens[i].column {
				t.Errorf("expected column %d, got %d", tc.tokens[i].column, tok.Position.Column())
			}
		}
	}
}
