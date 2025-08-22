package codegen

import (
	"pisuke/ast"
	"testing"
)

func TestGenerateLetStatement(t *testing.T) {
	program := &ast.Program{
		Statements: []ast.Statement{
			&ast.LetStatement{
				Name:  &ast.Identifier{Value: "myVar"},
				Value: &ast.IntegerLiteral{Value: 123},
			},
		},
	}

	expected := `package main

func main() {
	var myVar = 123
	_ = myVar
}
`
	generatedCode := Generate(program)
	if generatedCode != expected {
		t.Errorf("Generated code is not correct.\nExpected:\n%s\nGot:\n%s", expected, generatedCode)
	}
}

func TestGenerateServerStatic(t *testing.T) {
	program := &ast.Program{
		Statements: []ast.Statement{
			&ast.ExpressionStatement{
				Expression: &ast.CallExpression{
					Function: &ast.MemberAccessExpression{
						Object:   &ast.Identifier{Value: "server"},
						Property: &ast.Identifier{Value: "static"},
					},
					Arguments: []ast.Expression{
						&ast.StringLiteral{Value: "./public"},
					},
				},
			},
		},
	}

	expected := `package main

import (
	"net/http"
)

func main() {
	http.Handle("/", http.FileServer(http.Dir("./public")))
}
`
	generatedCode := Generate(program)

	if generatedCode != expected {
		t.Errorf("Generated code is not correct.\nExpected:\n%s\nGot:\n%s", expected, generatedCode)
	}
}

func TestGenerateStringLiteral(t *testing.T) {
	program := &ast.Program{
		Statements: []ast.Statement{
			&ast.ExpressionStatement{
				Expression: &ast.StringLiteral{Value: "hello world"},
			},
		},
	}

	expected := `package main

func main() {
	"hello world"
}
`
	generatedCode := Generate(program)
	if generatedCode != expected {
		t.Errorf("Generated code is not correct.\nExpected:\n%s\nGot:\n%s", expected, generatedCode)
	}
}

func TestGenerateInfixExpression(t *testing.T) {
	program := &ast.Program{
		Statements: []ast.Statement{
			&ast.ExpressionStatement{
				Expression: &ast.InfixExpression{
					Left:     &ast.IntegerLiteral{Value: 5},
					Operator: "+",
					Right:    &ast.IntegerLiteral{Value: 10},
				},
			},
		},
	}

	expected := `package main

func main() {
	(5 + 10)
}
`
	generatedCode := Generate(program)
	if generatedCode != expected {
		t.Errorf("Generated code is not correct.\nExpected:\n%s\nGot:\n%s", expected, generatedCode)
	}
}

func TestGenerateFunctionAndCall(t *testing.T) {
	program := &ast.Program{
		Statements: []ast.Statement{
			&ast.LetStatement{
				Name: &ast.Identifier{Value: "add"},
				Value: &ast.FunctionLiteral{
					Parameters: []*ast.Identifier{{Value: "x"}, {Value: "y"}},
					Body: &ast.BlockStatement{
						Statements: []ast.Statement{
							&ast.ExpressionStatement{
								Expression: &ast.InfixExpression{
									Left:     &ast.Identifier{Value: "x"},
									Operator: "+",
									Right:    &ast.Identifier{Value: "y"},
								},
							},
						},
					},
				},
			},
			&ast.ExpressionStatement{
				Expression: &ast.CallExpression{
					Function: &ast.Identifier{Value: "add"},
					Arguments: []ast.Expression{
						&ast.IntegerLiteral{Value: 2},
						&ast.IntegerLiteral{Value: 3},
					},
				},
			},
		},
	}

	expected := `package main

func main() {
	var add = func(x, y) {
		(x + y)
	}
	_ = add
	add(2, 3)
}
`
	generatedCode := Generate(program)
	if generatedCode != expected {
		t.Errorf("Generated code is not correct.\nExpected:\n%s\nGot:\n%s", expected, generatedCode)
	}
}

func TestGenerateServerServe(t *testing.T) {
	program := &ast.Program{
		Statements: []ast.Statement{
			&ast.ExpressionStatement{
				Expression: &ast.CallExpression{
					Function: &ast.MemberAccessExpression{
						Object:   &ast.Identifier{Value: "server"},
						Property: &ast.Identifier{Value: "serve"},
					},
					Arguments: []ast.Expression{
						&ast.IntegerLiteral{Value: 8080},
					},
				},
			},
		},
	}

	expected := `package main

import (
	"log"
	"net/http"
)

func main() {
	log.Fatal(http.ListenAndServe(":8080", nil))
}
`
	generatedCode := Generate(program)
	if generatedCode != expected {
		t.Errorf("Generated code is not correct.\nExpected:\n%s\nGot:\n%s", expected, generatedCode)
	}
}
