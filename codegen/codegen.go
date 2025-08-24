package codegen

import (
	"bytes"
	"fmt"
	"pisuke/ast"
	"sort"
	"strings"
)

func capitalizeFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(string(s[0])) + s[1:]
}

type Generator struct {
	out         *bytes.Buffer
	indentlevel int

	requiresHttp       bool
	requiresLog        bool
	requiresFmt        bool
	requiresMiddleware bool
	variableTypes      map[string]string
	typeDefs           map[string]*ast.TypeDefinition
	requiresJson       bool
	requiresIo         bool
	requiresStrings    bool
}

func NewGenerator() *Generator {
	return &Generator{out: &bytes.Buffer{}, variableTypes: map[string]string{}, typeDefs: map[string]*ast.TypeDefinition{}}
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
		if g.requiresJson {
			finalBuf.WriteString("\t\"encoding/json\"\n")
		}
		if g.requiresIo {
			finalBuf.WriteString("\t\"io/ioutil\"\n")
		}
		if g.requiresStrings {
			finalBuf.WriteString("\t\"strings\"\n")
		}
		finalBuf.WriteString(")\n\n")
	}

	finalBuf.Write(codeBuf.Bytes())
	return finalBuf.String()
}

func (g *Generator) genProgram(program *ast.Program) {
	// Emit named functions first
	for _, stmt := range program.Statements {
		// find top-level expressions that are function literals with names
		if es, ok := stmt.(*ast.ExpressionStatement); ok {
			if fl, ok := es.Expression.(*ast.FunctionLiteral); ok && fl.Name != nil {
				// emit a top-level function
				g.writeLine(g.genFunctionLiteralTopLevel(fl))
			}
		}
		if ls, ok := stmt.(*ast.LetStatement); ok {
			if fl, ok := ls.Value.(*ast.FunctionLiteral); ok && fl.Name != nil {
				g.writeLine(g.genFunctionLiteralTopLevel(fl))
			}
		}
	}

	// If middleware groundwork requested, emit helper before main
	if g.requiresMiddleware {
		g.writeLine("var middlewares []func(http.HandlerFunc) http.HandlerFunc")
		g.writeLine("func wrapHandler(h http.HandlerFunc) http.HandlerFunc {")
		g.indentlevel++
		g.writeLine("for i := len(middlewares)-1; i >= 0; i-- {")
		g.indentlevel++
		g.writeLine("h = middlewares[i](h)")
		g.indentlevel--
		g.writeLine("}")
		g.writeLine("return h")
		g.indentlevel--
		g.writeLine("}")
	}

	g.writeLine("func main() {")
	g.indentlevel++
	for _, stmt := range program.Statements {
		g.genStatement(stmt)
	}
	g.indentlevel--
	g.writeLine("}")
}

// genFunctionLiteralTopLevel emits a named Go function declaration for a FunctionLiteral
func (g *Generator) genFunctionLiteralTopLevel(node *ast.FunctionLiteral) string {
	var b bytes.Buffer
	params := []string{}
	for _, p := range node.Parameters {
		if node.ParamTypes != nil {
			if t, ok := node.ParamTypes[p.Value]; ok {
				goType := mapTypeToGo(t)
				params = append(params, p.Value+" "+goType)
				continue
			}
		}
		params = append(params, p.Value+" interface{}")
	}
	retType := "interface{}"
	if node.ReturnType != "" {
		retType = mapTypeToGo(node.ReturnType)
	}
	b.WriteString(fmt.Sprintf("func %s(%s) %s {", node.Name.Value, strings.Join(params, ", "), retType))

	bodyGen := NewGenerator()
	bodyGen.indentlevel = 0
	for _, s := range node.Body.Statements {
		bodyGen.genStatement(s)
	}
	b.WriteString("\n")
	b.Write(bodyGen.out.Bytes())
	b.WriteString("}")
	return b.String()
}

func (g *Generator) genStatement(stmt ast.Statement) {
	g.indent()
	switch node := stmt.(type) {
	case *ast.LetStatement:
		g.genLetStatement(node)
	case *ast.ConstStatement:
		g.genConstStatement(node)
	case *ast.TypeDefinition:
		g.genTypeDefinition(node)
	case *ast.ReturnStatement:
		g.genReturnStatement(node)
	case *ast.ExpressionStatement:
		// If this is a named top-level function literal, it has already been
		// emitted before main by genProgram; skip emitting the literal again.
		if fl, ok := node.Expression.(*ast.FunctionLiteral); ok && fl.Name != nil {
			return
		}
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
			// ensure key is a string literal in generated Go map literal
			var keyStr string
			if ks, ok := key.(*ast.StringLiteral); ok {
				keyStr = fmt.Sprintf("\"%s\"", ks.Value)
			} else if ident, ok := key.(*ast.Identifier); ok {
				keyStr = fmt.Sprintf("\"%s\"", ident.Value)
			} else {
				keyStr = fmt.Sprintf("\"%s\"", g.captureExpression(key))
			}
			valStr := g.captureExpression(value)
			pairs = append(pairs, fmt.Sprintf("%s: %s", keyStr, valStr))
		}
		g.write(fmt.Sprintf("map[string]interface{}{%s}", strings.Join(pairs, ", ")))
	case *ast.IndexExpression:
		// If left side is itself an indexed/map access (e.g. req["params"]),
		// cast it to map[string]interface{} before performing another index:
		// req["params"].(map[string]interface{})["id"]
		leftStr := g.captureExpression(node.Left)
		idxStr := g.captureExpression(node.Index)
		if strings.Contains(leftStr, "[") {
			g.write(fmt.Sprintf("%s.(map[string]interface{})[%s]", leftStr, idxStr))
		} else {
			g.write(fmt.Sprintf("%s[%s]", leftStr, idxStr))
		}
	case *ast.MemberAccessExpression:
		// Determine if the object expression is a struct (named or nested)
		if isStruct, _, _ := g.resolveStructInfo(node.Object); isStruct {
			g.genExpression(node.Object)
			g.write(".")
			g.write(capitalizeFirst(node.Property.Value))
			return
		}
		// fallback: map-style access
		leftStr := g.captureExpression(node.Object)
		if strings.Contains(leftStr, "[") {
			// e.g. req["params"] -> req["params"].(map[string]interface{})["prop"]
			g.write(fmt.Sprintf("%s.(map[string]interface{})[\"%s\"]", leftStr, node.Property.Value))
		} else {
			g.write(fmt.Sprintf("%s[\"%s\"]", leftStr, node.Property.Value))
		}
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
	// If a type annotation exists and the value is a MapLiteral,
	// emit a typed Go struct literal: TypeName{ Field: value, ... }
	if letStmt.TypeName != "" {
		if ml, ok := letStmt.Value.(*ast.MapLiteral); ok {
			// collect key -> expression map deterministically
			type pair struct {
				key     string
				valExpr ast.Expression
			}
			pairs := []pair{}
			for k, v := range ml.Pairs {
				keyStr := ""
				if ks, ok := k.(*ast.StringLiteral); ok {
					keyStr = ks.Value
				} else if ident, ok := k.(*ast.Identifier); ok {
					keyStr = ident.Value
				} else {
					keyStr = g.captureExpression(k)
				}
				pairs = append(pairs, pair{keyStr, v})
			}
			sort.Slice(pairs, func(i, j int) bool { return pairs[i].key < pairs[j].key })
			fields := []string{}
			td, hasTypeDef := g.typeDefs[letStmt.TypeName]
			// build quick lookup from pairs
			kv := map[string]ast.Expression{}
			for _, p := range pairs {
				kv[p.key] = p.valExpr
			}
			if hasTypeDef {
				for _, tf := range td.Fields {
					valExpr, ok := kv[tf.Name]
					if !ok {
						// missing value -> zero value via captureExpression of nil? fallback to zero-value literal
						fields = append(fields, fmt.Sprintf("%s: %s", capitalizeFirst(tf.Name), "nil"))
						continue
					}
					if tf.Nested != nil {
						// expect valExpr to be a MapLiteral
						if nestedMap, ok := valExpr.(*ast.MapLiteral); ok {
							// build nested struct type string
							nestedTypeParts := []string{}
							for _, nf := range tf.Nested.Fields {
								nestedTypeParts = append(nestedTypeParts, fmt.Sprintf("%s %s", capitalizeFirst(nf.Name), mapTypeToGo(nf.Type)))
							}
							nestedTypeStr := "struct{" + strings.Join(nestedTypeParts, ", ") + "}"
							// build nested literal fields
							nestedPairs := []string{}
							// build map from nestedMap pairs
							nkv := map[string]ast.Expression{}
							for k, v := range nestedMap.Pairs {
								if ks, ok := k.(*ast.StringLiteral); ok {
									nkv[ks.Value] = v
								} else if ident, ok := k.(*ast.Identifier); ok {
									nkv[ident.Value] = v
								} else {
									nkv[g.captureExpression(k)] = v
								}
							}
							for _, nf := range tf.Nested.Fields {
								nev, ok := nkv[nf.Name]
								if !ok {
									nestedPairs = append(nestedPairs, fmt.Sprintf("%s: %s", capitalizeFirst(nf.Name), "nil"))
									continue
								}
								nestedPairs = append(nestedPairs, fmt.Sprintf("%s: %s", capitalizeFirst(nf.Name), g.captureExpression(nev)))
							}
							nestedLiteral := nestedTypeStr + "{" + strings.Join(nestedPairs, ", ") + "}"
							fields = append(fields, fmt.Sprintf("%s: %s", capitalizeFirst(tf.Name), nestedLiteral))
							continue
						}
					}
					// non-nested field
					fields = append(fields, fmt.Sprintf("%s: %s", capitalizeFirst(tf.Name), g.captureExpression(valExpr)))
				}
			} else {
				// fallback: iterate pairs in deterministic order
				for _, p := range pairs {
					fields = append(fields, fmt.Sprintf("%s: %s", capitalizeFirst(p.key), g.captureExpression(p.valExpr)))
				}
			}
			g.write(fmt.Sprintf("var %s %s = %s{%s}\n", letStmt.Name.Value, letStmt.TypeName, letStmt.TypeName, strings.Join(fields, ", ")))
			// record variable's type for later member access generation
			g.variableTypes[letStmt.Name.Value] = letStmt.TypeName
			g.indent()
			g.write(fmt.Sprintf("_ = %s\n", letStmt.Name.Value))
			return
		}
	}

	// fallback: untyped or non-map values
	g.write(fmt.Sprintf("var %s = ", letStmt.Name.Value))
	g.genExpression(letStmt.Value)
	g.write("\n")
	g.indent()
	g.write(fmt.Sprintf("_ = %s\n", letStmt.Name.Value))
}

func (g *Generator) genConstStatement(constStmt *ast.ConstStatement) {
	if constStmt.TypeName != "" {
		if ml, ok := constStmt.Value.(*ast.MapLiteral); ok {
			type pair struct{ key, val string }
			pairs := []pair{}
			for k, v := range ml.Pairs {
				keyStr := ""
				if ks, ok := k.(*ast.StringLiteral); ok {
					keyStr = ks.Value
				} else {
					keyStr = g.captureExpression(k)
				}
				valStr := g.captureExpression(v)
				pairs = append(pairs, pair{keyStr, valStr})
			}
			sort.Slice(pairs, func(i, j int) bool { return pairs[i].key < pairs[j].key })
			fields := []string{}
			for _, p := range pairs {
				fields = append(fields, fmt.Sprintf("%s: %s", capitalizeFirst(p.key), p.val))
			}
			g.write(fmt.Sprintf("const %s = %s{%s}\n", constStmt.Name.Value, constStmt.TypeName, strings.Join(fields, ", ")))
			g.variableTypes[constStmt.Name.Value] = constStmt.TypeName
			return
		}
	}

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
		if node.ParamTypes != nil {
			if t, ok := node.ParamTypes[p.Value]; ok {
				// map simple Pisuke types to Go types; default to interface{}
				goType := mapTypeToGo(t)
				params = append(params, p.Value+" "+goType)
				continue
			}
		}
		params = append(params, p.Value+" interface{}")
	}
	retType := "interface{}"
	if node.ReturnType != "" {
		retType = mapTypeToGo(node.ReturnType)
	}
	b.WriteString(fmt.Sprintf("func(%s) %s {", strings.Join(params, ", "), retType))

	bodyGen := NewGenerator()
	bodyGen.indentlevel = g.indentlevel + 1
	for _, s := range node.Body.Statements {
		bodyGen.genStatement(s)
	}
	// if function body contains no return, add a default return nil to satisfy Go
	hasReturn := false
	for _, s := range node.Body.Statements {
		if _, ok := s.(*ast.ReturnStatement); ok {
			hasReturn = true
			break
		}
	}
	if !hasReturn {
		bodyGen.writeLine("return nil")
	}
	b.WriteString("\n")
	b.Write(bodyGen.out.Bytes())
	g.indent()
	b.WriteString("}")
	return b.String()
}

func mapTypeToGo(t string) string {
	switch t {
	case "int":
		return "int"
	case "string":
		return "string"
	default:
		return "interface{}"
	}
}

func (g *Generator) genTypeDefinition(td *ast.TypeDefinition) {
	g.writeLine("type " + td.Name.Value + " struct {")
	g.indentlevel++
	for _, f := range td.Fields {
		fieldName := capitalizeFirst(f.Name)
		if f.Nested != nil {
			// emit nested anonymous struct type
			g.writeLine(fieldName + " struct {")
			g.indentlevel++
			for _, nf := range f.Nested.Fields {
				nfName := capitalizeFirst(nf.Name)
				nfType := mapTypeToGo(nf.Type)
				g.writeLine(nfName + " " + nfType)
			}
			g.indentlevel--
			g.writeLine("}")
		} else {
			fieldType := mapTypeToGo(f.Type)
			g.writeLine(fieldName + " " + fieldType)
		}
	}
	g.indentlevel--
	g.writeLine("}")
	// record type definition for nested usage
	g.typeDefs[td.Name.Value] = td
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
	rawPath := g.captureExpression(node.Arguments[0])
	handler := node.Arguments[1].(*ast.FunctionLiteral)

	// If handler has no parameters, emit the minimal handler (preserve existing tests)
	if len(handler.Parameters) == 0 {
		g.requiresHttp = true
		g.requiresFmt = true
		g.write(fmt.Sprintf("http.HandleFunc(%s, wrapHandler(func(w http.ResponseWriter, r *http.Request) {", rawPath))
		g.indentlevel++
		g.write("\n")
		// generate simple handler body: evaluate return and print
		var handlerLogicBuf bytes.Buffer
		hg := NewGenerator()
		hg.out = &handlerLogicBuf
		hg.indentlevel = g.indentlevel

		for _, s := range handler.Body.Statements {
			if rs, ok := s.(*ast.ReturnStatement); ok {
				hg.indent()
				hg.write("returnValue := ")
				hg.write(hg.captureExpression(rs.ReturnValue))
				hg.write("\n")
			} else {
				hg.genStatement(s)
			}
		}

		// append fmt line into handler buffer so indentation matches
		hg.writeLine("fmt.Fprint(w, returnValue)")
		g.out.Write(handlerLogicBuf.Bytes())

		g.indentlevel--
		g.indent()
		g.write("})")
		return
	}

	// Rich handler generation when handler accepts a parameter (req)
	g.requiresHttp, g.requiresFmt, g.requiresLog, g.requiresJson, g.requiresIo = true, true, true, true, true

	// build path param names from rawPath (strip quotes)
	pathStr := strings.Trim(rawPath, "\"")
	paramNames := []string{}
	// normalize template path parts by trimming leading/trailing slashes so
	// that parts align with request pathParts which are produced by
	// strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	parts := strings.Split(strings.Trim(pathStr, "/"), "/")
	for _, p := range parts {
		if strings.HasPrefix(p, ":") {
			paramNames = append(paramNames, p[1:])
		}
	}

	// choose registration pattern: if path contains dynamic segments (:") use prefix up to first dynamic
	regPattern := rawPath
	if len(paramNames) > 0 {
		firstDyn := -1
		for i, p := range parts {
			if strings.HasPrefix(p, ":") {
				firstDyn = i
				break
			}
		}
		// Build a clean prefix by removing any empty parts (which happen when
		// the path starts with a leading '/') so we don't produce double
		// slashes like "//users/". Keep a single leading and trailing slash.
		prefix := "/"
		if firstDyn > 0 {
			prefixParts := parts[:firstDyn]
			cleaned := []string{}
			for _, pp := range prefixParts {
				if pp != "" {
					cleaned = append(cleaned, pp)
				}
			}
			if len(cleaned) > 0 {
				prefix = "/" + strings.Join(cleaned, "/") + "/"
			} else {
				prefix = "/"
			}
		}
		regPattern = fmt.Sprintf("\"%s\"", prefix)
	}
	g.write(fmt.Sprintf("http.HandleFunc(%s, func(w http.ResponseWriter, r *http.Request) {", regPattern))
	g.indentlevel++
	g.write("\n")

	// prepare req map
	g.writeLine("query := make(map[string]interface{})")
	g.writeLine("for k, v := range r.URL.Query() {")
	g.indentlevel++
	g.writeLine("if len(v) > 0 { query[k] = v[0] }")
	g.indentlevel--
	g.writeLine("}")
	g.writeLine("req := make(map[string]interface{})")
	g.writeLine("req[\"query\"] = query")

	// path params
	if len(paramNames) > 0 {
		g.requiresStrings = true
		g.writeLine("pathParts := strings.Split(strings.Trim(r.URL.Path, \"/\"), \"/\")")
		g.writeLine("params := make(map[string]interface{})")
		g.writeLine("// naive mapping: match positions")
		g.writeLine("for i, part := range pathParts {")
		g.indentlevel++
		g.writeLine("_ = i; _ = part")
		g.indentlevel--
		g.writeLine("}")
		// generate assignments based on param positions
		for i, p := range parts {
			if strings.HasPrefix(p, ":") {
				g.writeLine(fmt.Sprintf("if len(pathParts) > %d { params[\"%s\"] = pathParts[%d] }", i, p[1:], i))
			}
		}
		g.writeLine("req[\"params\"] = params")
	}

	// parse JSON body for POST/PUT
	// robust JSON body parsing with size guard and error handling
	g.writeLine("if r.Method == \"POST\" || r.Method == \"PUT\" {")
	g.indentlevel++
	g.writeLine("r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // limit to 1MB")
	g.writeLine("defer r.Body.Close()")
	g.writeLine("bodyBytes, err := ioutil.ReadAll(r.Body)")
	g.writeLine("if err != nil { http.Error(w, \"failed to read body\", http.StatusBadRequest); return }")
	g.writeLine("if len(bodyBytes) > 0 { var bodyObj interface{}; if err := json.Unmarshal(bodyBytes, &bodyObj); err != nil { http.Error(w, \"invalid JSON\", http.StatusBadRequest); return }; req[\"body\"] = bodyObj }")
	g.indentlevel--
	g.writeLine("}")

	// logging
	g.writeLine("log.Printf(\"%s %s\", r.Method, r.URL.Path)")

	// generate handler body
	var handlerLogicBuf bytes.Buffer
	hg := NewGenerator()
	hg.out = &handlerLogicBuf
	hg.indentlevel = g.indentlevel

	// expose req variable inside handler logic
	hg.writeLine("// handler logic")
	for _, s := range handler.Body.Statements {
		if rs, ok := s.(*ast.ReturnStatement); ok {
			hg.indent()
			hg.write("returnValue := interface{}(")
			hg.write(hg.captureExpression(rs.ReturnValue))
			hg.write(")\n")
		} else {
			hg.genStatement(s)
		}
	}

	// append serialization block into handler buffer
	hg.writeLine("switch rv := returnValue.(type) {")
	hg.indentlevel++
	hg.writeLine("case string:")
	hg.indentlevel++
	hg.writeLine("fmt.Fprint(w, rv)")
	hg.indentlevel--
	hg.writeLine("default:")
	hg.indentlevel++
	hg.writeLine("b, _ := json.Marshal(rv)")
	hg.writeLine("w.Header().Set(\"Content-Type\", \"application/json\")")
	hg.writeLine("w.Write(b)")
	hg.indentlevel--
	hg.indentlevel--
	hg.writeLine("}")

	g.out.Write(handlerLogicBuf.Bytes())

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

// resolveStructInfo attempts to determine whether the expression refers to a struct type.
// Returns (isStruct, typeName, remaining) where typeName is the Go type name if known.
func (g *Generator) resolveStructInfo(expr ast.Expression) (bool, string, []string) {
	switch e := expr.(type) {
	case *ast.Identifier:
		if t, ok := g.variableTypes[e.Value]; ok && t != "" {
			return true, t, nil
		}
		return false, "", nil
	case *ast.MemberAccessExpression:
		// resolve recursively: if left side is struct, then accessing a field may be nested struct
		if isStruct, tname, _ := g.resolveStructInfo(e.Object); isStruct {
			// try to find field info in typeDefs
			if td, ok := g.typeDefs[tname]; ok {
				for _, f := range td.Fields {
					if f.Name == e.Property.Value {
						if f.Nested != nil {
							return true, "", nil
						}
						return false, "", nil
					}
				}
			}
		}
		return false, "", nil
	default:
		return false, "", nil
	}
}
