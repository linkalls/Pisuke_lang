package codegen

import (
	"pisuke/ast"
	"pisuke/token"
	"testing"
)

func TestGenerateLetStatement(t *testing.T) {
	program := &ast.Program{
		Statements: []ast.Statement{
			&ast.LetStatement{
				Token: token.Token{Type: token.LET, Literal: "let"},
				Name: &ast.Identifier{
					Token: token.Token{Type: token.IDENT, Literal: "myVar"},
					Value: "myVar",
				},
				// Value is nil because we are not parsing expressions yet
				Value: nil,
			},
			&ast.LetStatement{
				Token: token.Token{Type: token.LET, Literal: "let"},
				Name: &ast.Identifier{
					Token: token.Token{Type: token.IDENT, Literal: "anotherVar"},
					Value: "anotherVar",
				},
				Value: nil,
			},
		},
	}

	expected := `package main

func main() {
	var myVar = 0
	_ = myVar
	var anotherVar = 0
	_ = anotherVar
}
`

	generatedCode := Generate(program)

	if generatedCode != expected {
		t.Errorf("Generated code is not correct.\nExpected:\n%s\nGot:\n%s", expected, generatedCode)
	}
}
