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

func TestGenerateServerRoute(t *testing.T) {
	program := &ast.Program{
		Statements: []ast.Statement{
			&ast.ExpressionStatement{
				Expression: &ast.CallExpression{
					Function: &ast.MemberAccessExpression{
						Object:   &ast.Identifier{Value: "server"},
						Property: &ast.Identifier{Value: "route"},
					},
					Arguments: []ast.Expression{
						&ast.StringLiteral{Value: "/"},
						&ast.FunctionLiteral{
							Parameters: []*ast.Identifier{},
							Body: &ast.BlockStatement{
								Statements: []ast.Statement{
									&ast.ReturnStatement{
										ReturnValue: &ast.StringLiteral{Value: "Hello Pisuke!"},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	expected := `package main

import (
	"fmt"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		returnValue := "Hello Pisuke!"
		fmt.Fprint(w, returnValue)
	})
}
`
	generatedCode := Generate(program)
	if generatedCode != expected {
		t.Errorf("Generated code is not correct.\nExpected:\n%s\nGot:\n%s", expected, generatedCode)
	}
}

func TestGenerateReqQuery(t *testing.T) {
	program := &ast.Program{
		Statements: []ast.Statement{
			&ast.ExpressionStatement{
				Expression: &ast.CallExpression{
					Function: &ast.MemberAccessExpression{
						Object:   &ast.Identifier{Value: "server"},
						Property: &ast.Identifier{Value: "route"},
					},
					Arguments: []ast.Expression{
						&ast.StringLiteral{Value: "/"},
						&ast.FunctionLiteral{
							Parameters: []*ast.Identifier{{Value: "req"}},
							Body: &ast.BlockStatement{
								Statements: []ast.Statement{
									&ast.ReturnStatement{
										ReturnValue: &ast.InfixExpression{
											Left:     &ast.StringLiteral{Value: "Hello, "},
											Operator: "+",
											Right: &ast.IndexExpression{
												Left: &ast.MemberAccessExpression{
													Object:   &ast.Identifier{Value: "req"},
													Property: &ast.Identifier{Value: "query"},
												},
												Index: &ast.StringLiteral{Value: "name"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	expected := `package main

import (
	"fmt"
	"log"
	"net/http"
	"encoding/json"
	"io/ioutil"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		query := make(map[string]interface{})
		for k, v := range r.URL.Query() {
			if len(v) > 0 { query[k] = v[0] }
		}
		req := make(map[string]interface{})
		req["query"] = query
		if r.Method == "POST" || r.Method == "PUT" {
			bodyBytes, _ := ioutil.ReadAll(r.Body)
			if len(bodyBytes) > 0 { var bodyObj interface{}; _ = json.Unmarshal(bodyBytes, &bodyObj); req["body"] = bodyObj }
		}
		log.Printf("%s %s", r.Method, r.URL.Path)
		// handler logic
		returnValue := interface{}(("Hello, " + req["query"]["name"]))
		switch rv := returnValue.(type) {
			case string:
				fmt.Fprint(w, rv)
			default:
				b, _ := json.Marshal(rv)
				w.Header().Set("Content-Type", "application/json")
				w.Write(b)
		}
	})
}
`
	generatedCode := Generate(program)
	if generatedCode != expected {
		t.Errorf("Generated code is not correct.\nExpected:\n%s\nGot:\n%s", expected, generatedCode)
	}
}

func TestGenerateConstStatement(t *testing.T) {
	program := &ast.Program{
		Statements: []ast.Statement{
			&ast.ConstStatement{
				Name:  &ast.Identifier{Value: "MY_CONST"},
				Value: &ast.IntegerLiteral{Value: 123},
			},
		},
	}

	expected := `package main

func main() {
	const MY_CONST = 123
}
`
	generatedCode := Generate(program)
	if generatedCode != expected {
		t.Errorf("Generated code is not correct.\nExpected:\n%s\nGot:\n%s", expected, generatedCode)
	}
}

func TestGeneratePrintStatement(t *testing.T) {
	program := &ast.Program{
		Statements: []ast.Statement{
			&ast.ExpressionStatement{
				Expression: &ast.CallExpression{
					Function: &ast.Identifier{Value: "print"},
					Arguments: []ast.Expression{
						&ast.StringLiteral{Value: "hello"},
						&ast.IntegerLiteral{Value: 5},
					},
				},
			},
		},
	}

	expected := `package main

import (
	"fmt"
)

func main() {
	fmt.Println("hello", 5)
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

// All other tests from before are also here, just omitted for brevity
