package codegen

import (
	"bytes"
	"fmt"
	"pisuke/ast"
	"strings"
)

func Generate(program *ast.Program) string {
	var out bytes.Buffer

	// Go package and main function wrapper
	out.WriteString("package main\n\n")
	out.WriteString("func main() {\n")

	for _, statement := range program.Statements {
		s := genStatement(statement)
		lines := strings.Split(s, "\n")
		for _, line := range lines {
			if line != "" {
				out.WriteString("\t" + line + "\n")
			}
		}
	}

	out.WriteString("}\n")

	return out.String()
}

func genStatement(stmt ast.Statement) string {
	switch node := stmt.(type) {
	case *ast.LetStatement:
		return genLetStatement(node)
	default:
		return "" // Or some error handling
	}
}

func genLetStatement(letStmt *ast.LetStatement) string {
	// Since we don't parse expressions yet, we'll use a zero value.
	// This is a placeholder until expression parsing is implemented.
	// For `let x = ...`, we generate `var x = 0; _ = x` to avoid "declared and not used" error.
	varName := letStmt.Name.Value
	return fmt.Sprintf("var %s = 0\n_ = %s", varName, varName)
}
