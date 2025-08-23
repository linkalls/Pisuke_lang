package codegen

import (
	"bytes"
	"fmt"
	"pisuke/ast"
	"strings"
)

type Generator struct {
	out         *bytes.Buffer
	indentlevel int

	requiresHttp bool
	requiresLog  bool
	requiresFmt  bool
}

func NewGenerator() *Generator {
	return &Generator{out: &bytes.Buffer{}}
}

func (g *Generator) indent() {
	g.out.WriteString(strings.Repeat("\t", g.indentlevel))
}

func (g *Generator) write(s string) {
	// No indent, for writing parts of an expression on the same line
	g.out.WriteString(s)
}

func (g *Generator) writeLine(s string) {
	g.indent()
	g.out.WriteString(s)
	g.out.WriteString("\n")
}

func Generate(program *ast.Program) string {
	g := NewGenerator()

	// First pass to generate code and find out required imports
	var codeBuf bytes.Buffer
	g.out = &codeBuf
	g.genProgram(program)

	// Second pass to build the final output with imports
	var finalBuf bytes.Buffer
	finalBuf.WriteString("package main\n\n")

	if g.requiresHttp || g.requiresLog || g.requiresFmt {
		finalBuf.WriteString("import (\n")
		if g.requiresFmt {
			finalBuf.WriteString("\t\"fmt\"\n")
		}
		if g.requiresLog {
			finalBuf.WriteString("\t\"log\"\n")
		}
		if g.requiresHttp {
			finalBuf.WriteString("\t\"net/http\"\n")
		}
		finalBuf.WriteString(")\n\n")
	}

	finalBuf.Write(codeBuf.Bytes())
	return finalBuf.String()
}

func (g *Generator) genProgram(program *ast.Program) {
	g.writeLine("func main() {")
	g.indentlevel++
	for _, stmt := range program.Statements {
		g.genStatement(stmt)
	}
	g.indentlevel--
	g.writeLine("}")
}

func (g *Generator) genStatement(stmt ast.Statement) {
	g.indent()
	switch node := stmt.(type) {
	case *ast.LetStatement:
		g.genLetStatement(node)
	case *ast.ConstStatement:
		g.genConstStatement(node)
	case *ast.ReturnStatement:
		g.genReturnStatement(node)
	case *ast.ExpressionStatement:
		g.genExpression(node.Expression)
		g.write("\n")
	}
}

func (g *Generator) genExpression(expr ast.Expression) {
	switch node := expr.(type) {
	case *ast.IntegerLiteral:
		g.write(fmt.Sprintf("%d", node.Value))
	case *ast.StringLiteral:
		g.write(fmt.Sprintf("\"%s\"", node.Value))
	case *ast.Identifier:
		g.write(node.Value)
	case *ast.ListLiteral:
		elements := []string{}
		for _, el := range node.Elements {
			// This is a bit tricky, need to capture expression output
			var buf bytes.Buffer
			originalOut := g.out
			g.out = &buf
			g.genExpression(el)
			g.out = originalOut
			elements = append(elements, buf.String())
		}
		g.write(fmt.Sprintf("[]interface{}{%s}", strings.Join(elements, ", ")))
	case *ast.MapLiteral:
		pairs := []string{}
		for key, value := range node.Pairs {
			var keyBuf, valBuf bytes.Buffer
			originalOut := g.out

			g.out = &keyBuf
			g.genExpression(key)

			g.out = &valBuf
			g.genExpression(value)

			g.out = originalOut
			pairs = append(pairs, fmt.Sprintf("%s: %s", keyBuf.String(), valBuf.String()))
		}
		g.write(fmt.Sprintf("map[string]interface{}{%s}", strings.Join(pairs, ", ")))
	case *ast.IndexExpression:
		g.genExpression(node.Left)
		g.write("[")
		g.genExpression(node.Index)
		g.write("]")
	case *ast.MemberAccessExpression:
		g.genExpression(node.Object)
		g.write(fmt.Sprintf("[\"%s\"]", node.Property.Value))
	case *ast.InfixExpression:
		g.write("(")
		g.genExpression(node.Left)
		g.write(fmt.Sprintf(" %s ", node.Operator))
		g.genExpression(node.Right)
		g.write(")")
	case *ast.FunctionLiteral:
		// This case is for general function literals, not route handlers
		params := []string{}
		for _, p := range node.Parameters {
			params = append(params, p.Value)
		}
		g.write(fmt.Sprintf("func(%s) {", strings.Join(params, ", ")))
		g.write("\n")
		g.indentlevel++
		for _, s := range node.Body.Statements {
			g.genStatement(s)
		}
		g.indentlevel--
		g.indent()
		g.write("}")
	case *ast.CallExpression:
		if mae, ok := node.Function.(*ast.MemberAccessExpression); ok {
			if obj, ok := mae.Object.(*ast.Identifier); ok && obj.Value == "server" {
				switch mae.Property.Value {
				case "serve":
					g.requiresHttp = true
					g.requiresLog = true
					g.write("log.Fatal(http.ListenAndServe(\":")
					g.genExpression(node.Arguments[0])
					g.write("\", nil))")
					return
				case "static":
					g.requiresHttp = true
					g.write("http.Handle(\"/\", http.FileServer(http.Dir(")
					g.genExpression(node.Arguments[0])
					g.write(")))")
					return
				case "route":
					g.requiresHttp = true
					g.requiresFmt = true
					path := node.Arguments[0].(*ast.StringLiteral).Value
					handler := node.Arguments[1].(*ast.FunctionLiteral)

					g.write(fmt.Sprintf("http.HandleFunc(\"%s\", func(w http.ResponseWriter, r *http.Request) {", path))
					g.indentlevel++

					// Buffer for the handler's logic
					var handlerLogicBuf bytes.Buffer
					originalOut := g.out
					g.out = &handlerLogicBuf

					hasReqParam := len(handler.Parameters) > 0
					if hasReqParam {
						g.writeLine("query := make(map[string]interface{})")
						g.writeLine("for k, v := range r.URL.Query() {")
						g.indentlevel++
					g.writeLine("if len(v) > 0 {")
					g.indentlevel++
					g.writeLine("query[k] = v[0]")
					g.indentlevel--
					g.writeLine("}")
						g.indentlevel--
						g.writeLine("}")
						g.writeLine("req := make(map[string]interface{})")
						g.writeLine("req[\"query\"] = query")
					}

					// Transpile the body of the Pisuke handler
					// The return value will be captured and printed later
					for _, s := range handler.Body.Statements {
						if rs, ok := s.(*ast.ReturnStatement); ok {
							g.indent()
							g.write("returnValue := ")
							g.genExpression(rs.ReturnValue)
							g.write("\n")
						} else {
							g.genStatement(s)
						}
					}

					g.out = originalOut // Restore original buffer
					g.write("\n")
					g.out.Write(handlerLogicBuf.Bytes())
					g.writeLine("fmt.Fprint(w, returnValue)")

					g.indentlevel--
					g.indent()
					g.write("})")
					return
				}
			}
		}
		// Generic call
		args := []string{}
		originalOut := g.out
		for _, a := range node.Arguments {
			var argBuf bytes.Buffer
			g.out = &argBuf
			g.genExpression(a)
			args = append(args, argBuf.String())
		}
		g.out = originalOut

		g.genExpression(node.Function)
		g.write("(")
		g.write(strings.Join(args, ", "))
		g.write(")")
	}
}

func (g *Generator) genLetStatement(letStmt *ast.LetStatement) {
	g.write(fmt.Sprintf("var %s = ", letStmt.Name.Value))
	g.genExpression(letStmt.Value)
	g.write("\n")
	g.indent()
	g.write(fmt.Sprintf("_ = %s\n", letStmt.Name.Value))
}

func (g *Generator) genConstStatement(constStmt *ast.ConstStatement) {
	g.write(fmt.Sprintf("const %s = ", constStmt.Name.Value))
	g.genExpression(constStmt.Value)
	g.write("\n")
}

func (g *Generator) genReturnStatement(returnStmt *ast.ReturnStatement) {
	g.write("return ")
	g.genExpression(returnStmt.ReturnValue)
	g.write("\n")
}
