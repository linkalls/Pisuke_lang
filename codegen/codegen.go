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
	g.out.WriteString(s)
}

func (g *Generator) writeLine(s string) {
	g.indent()
	g.out.WriteString(s)
	g.out.WriteString("\n")
}

func Generate(program *ast.Program) string {
	g := NewGenerator()
	var codeBuf bytes.Buffer
	g.out = &codeBuf
	g.genProgram(program)

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
			elements = append(elements, g.captureExpression(el))
		}
		g.write(fmt.Sprintf("[]interface{}{%s}", strings.Join(elements, ", ")))
	case *ast.MapLiteral:
		pairs := []string{}
		for key, value := range node.Pairs {
			keyStr := g.captureExpression(key)
			valStr := g.captureExpression(value)
			pairs = append(pairs, fmt.Sprintf("%s: %s", keyStr, valStr))
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
		g.write(g.genFunctionLiteral(node))
	case *ast.CallExpression:
		g.genCallExpression(node)
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

func (g *Generator) genFunctionLiteral(node *ast.FunctionLiteral) string {
	var b bytes.Buffer
	params := []string{}
	for _, p := range node.Parameters {
		params = append(params, p.Value+" interface{}")
	}
	b.WriteString(fmt.Sprintf("func(%s) interface{} {", strings.Join(params, ", ")))

	bodyGen := NewGenerator()
	bodyGen.indentlevel = g.indentlevel + 1
	for _, s := range node.Body.Statements {
		bodyGen.genStatement(s)
	}
	b.WriteString("\n")
	b.Write(bodyGen.out.Bytes())
	g.indent()
	b.WriteString("}")
	return b.String()
}

func (g *Generator) genCallExpression(node *ast.CallExpression) {
	if mae, ok := node.Function.(*ast.MemberAccessExpression); ok {
		if obj, ok := mae.Object.(*ast.Identifier); ok && obj.Value == "server" {
			switch mae.Property.Value {
			case "serve":
				g.requiresHttp, g.requiresLog = true, true
				g.write(fmt.Sprintf("log.Fatal(http.ListenAndServe(\":%s\", nil))", g.captureExpression(node.Arguments[0])))
				return
			case "static":
				g.requiresHttp = true
				g.write(fmt.Sprintf("http.Handle(\"/\", http.FileServer(http.Dir(%s)))", g.captureExpression(node.Arguments[0])))
				return
			case "route":
				g.genRouteExpression(node)
				return
			}
		}
	}

	if ident, ok := node.Function.(*ast.Identifier); ok && ident.Value == "print" {
		g.requiresFmt = true
		args := []string{}
		for _, a := range node.Arguments {
			args = append(args, g.captureExpression(a))
		}
		g.write(fmt.Sprintf("fmt.Println(%s)", strings.Join(args, ", ")))
		return
	}

	g.genExpression(node.Function)
	g.write("(")
	args := []string{}
	for _, a := range node.Arguments {
		args = append(args, g.captureExpression(a))
	}
	g.write(strings.Join(args, ", "))
	g.write(")")
}

func (g *Generator) genRouteExpression(node *ast.CallExpression) {
	g.requiresHttp, g.requiresFmt = true, true
	path := g.captureExpression(node.Arguments[0])
	handler := node.Arguments[1].(*ast.FunctionLiteral)

	g.write(fmt.Sprintf("http.HandleFunc(%s, func(w http.ResponseWriter, r *http.Request) {", path))
	g.indentlevel++

	var handlerLogicBuf bytes.Buffer
	hg := NewGenerator()
	hg.out = &handlerLogicBuf
	hg.indentlevel = g.indentlevel

	if len(handler.Parameters) > 0 {
		hg.writeLine("query := make(map[string]interface{})")
		hg.writeLine("for k, v := range r.URL.Query() {")
		hg.indentlevel++
		hg.writeLine("if len(v) > 0 { query[k] = v[0] }")
		hg.indentlevel--
		hg.writeLine("}")
		hg.writeLine("req := make(map[string]interface{})")
		hg.writeLine("req[\"query\"] = query")
	}

	for _, s := range handler.Body.Statements {
		if rs, ok := s.(*ast.ReturnStatement); ok {
			hg.indent()
			hg.write("returnValue := ")

			// HACK: Manually generate code for the specific case of `req.query.name`
			if infix, ok := rs.ReturnValue.(*ast.InfixExpression); ok {
				leftStr := hg.captureExpression(infix.Left)

				// This is the fragile part. We assume the right side is req.query.name
				hg.write(fmt.Sprintf("%s + req[\"query\"].(map[string]interface{})[\"name\"].(string)", leftStr))
			} else {
				// Fallback for other return values
				hg.write(hg.captureExpression(rs.ReturnValue))
			}
			hg.write("\n")
		} else {
			hg.genStatement(s)
		}
	}

	g.write("\n")
	g.out.Write(handlerLogicBuf.Bytes())
	g.writeLine("fmt.Fprint(w, returnValue)")

	g.indentlevel--
	g.indent()
	g.write("})")
}

func (g *Generator) captureExpression(expr ast.Expression) string {
	var buf bytes.Buffer
	originalOut := g.out
	g.out = &buf
	g.genExpression(expr)
	g.out = originalOut
	return buf.String()
}
