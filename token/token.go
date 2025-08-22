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

	// Delimiters
	SEMICOLON = ";"

	// Keywords
	LET   = "LET"
	CONST = "CONST"
	FN    = "FN"
)
