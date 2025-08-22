package token

type TokenType string

type Token struct {
	Type    TokenType
	Literal string
}

const (
	ILLEGAL = "ILLEGAL"
	EOF     = "EOF"

	// Identifiers + literals
	IDENT  = "IDENT"  // add, foobar, x, y, ...
	INT    = "INT"    // 1343456
	STRING = "STRING" // "Hello World"

	// Operators
	ASSIGN = "="
	PLUS   = "+"
	MUL    = "*"

	// Delimiters
	LPAREN = "("
	RPAREN = ")"
	DOT    = "."
	LBRACE = "{"
	RBRACE = "}"
	COMMA  = ","

	// Keywords
	LET    = "LET"
	CONST  = "CONST"
	FN     = "FN"
	RETURN = "RETURN"
)
