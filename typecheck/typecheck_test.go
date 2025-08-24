package typecheck

import (
	"pisuke/lexer"
	"pisuke/parser"
	"testing"
)

func TestNestedTypeAndTypecheck(t *testing.T) {
	src := `type User = { id: int, name: { n: string } }
let u:User = { "id": 1, "name": { n: "Alice" } }
print(u.name.n)`
	l := lexer.New(src)
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors) != 0 {
		t.Fatalf("parser errors: %v", p.Errors)
	}
	errs := CheckProgram(program)
	if len(errs) != 0 {
		t.Fatalf("typecheck errors: %v", errs)
	}
}

func TestTypecheckDetectsMissingField(t *testing.T) {
	src := `type User = { id: int, name: { n: string } }
let u:User = { "id": 1 }`
	l := lexer.New(src)
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors) != 0 {
		t.Fatalf("parser errors: %v", p.Errors)
	}
	errs := CheckProgram(program)
	if len(errs) == 0 {
		t.Fatalf("expected missing field error, got none")
	}
}
