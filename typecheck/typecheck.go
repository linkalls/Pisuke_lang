package typecheck

import (
	"fmt"
	"pisuke/ast"
)

// CheckProgram runs simple static checks over program and returns error messages.
func CheckProgram(program *ast.Program) []string {
	errs := []string{}
	// collect type defs
	typeDefs := map[string]*ast.TypeDefinition{}
	// collect function signatures: name -> (param types, return)
	funcSigs := map[string]struct {
		ParamOrder []string
		Params     map[string]string
		Return     string
	}{}
	for _, s := range program.Statements {
		if td, ok := s.(*ast.TypeDefinition); ok {
			typeDefs[td.Name.Value] = td
		}
		if ls, ok := s.(*ast.LetStatement); ok {
			if fl, ok := ls.Value.(*ast.FunctionLiteral); ok && fl.Name != nil {
				order := []string{}
				for _, p := range fl.Parameters {
					order = append(order, p.Value)
				}
				funcSigs[fl.Name.Value] = struct {
					ParamOrder []string
					Params     map[string]string
					Return     string
				}{ParamOrder: order, Params: fl.ParamTypes, Return: fl.ReturnType}
			}
		}
		if es, ok := s.(*ast.ExpressionStatement); ok {
			if fl, ok := es.Expression.(*ast.FunctionLiteral); ok && fl.Name != nil {
				order := []string{}
				for _, p := range fl.Parameters {
					order = append(order, p.Value)
				}
				funcSigs[fl.Name.Value] = struct {
					ParamOrder []string
					Params     map[string]string
					Return     string
				}{ParamOrder: order, Params: fl.ParamTypes, Return: fl.ReturnType}
			}
		}
	}
	// collect variable types
	varTypes := map[string]string{}
	for _, s := range program.Statements {
		switch st := s.(type) {
		case *ast.LetStatement:
			if st.TypeName != "" {
				varTypes[st.Name.Value] = st.TypeName
			}
			// try to infer variable type from a map literal by matching fields
			if st.TypeName == "" {
				if ml, ok := st.Value.(*ast.MapLiteral); ok {
					// attempt to find a typeDef that matches keys
					for tname, td := range typeDefs {
						okMatch := true
						for _, f := range td.Fields {
							// must exist in map
							found := false
							for k := range ml.Pairs {
								if ks, ok := k.(*ast.StringLiteral); ok {
									if ks.Value == f.Name {
										found = true
										break
									}
								} else if id, ok := k.(*ast.Identifier); ok {
									if id.Value == f.Name {
										found = true
										break
									}
								}
							}
							if !found {
								okMatch = false
								break
							}
						}
						if okMatch {
							varTypes[st.Name.Value] = tname
							break
						}
					}
				}
			}
		case *ast.ConstStatement:
			if st.TypeName != "" {
				varTypes[st.Name.Value] = st.TypeName
			}
		}
	}

	// helper to check map literal against type definition
	var checkMapAgainstType func(m *ast.MapLiteral, td *ast.TypeDefinition, path string)
	checkMapAgainstType = func(m *ast.MapLiteral, td *ast.TypeDefinition, path string) {
		// build map of provided keys
		provided := map[string]ast.Expression{}
		for k, v := range m.Pairs {
			if ks, ok := k.(*ast.StringLiteral); ok {
				provided[ks.Value] = v
			} else if id, ok := k.(*ast.Identifier); ok {
				provided[id.Value] = v
			}
		}
		for _, f := range td.Fields {
			pv, ok := provided[f.Name]
			if !ok {
				errs = append(errs, fmt.Sprintf("%s: missing field '%s'", path, f.Name))
				continue
			}
			// check basic type
			if f.Nested != nil {
				// value must be map literal
				if mv, ok := pv.(*ast.MapLiteral); ok {
					checkMapAgainstType(mv, f.Nested, path+"."+f.Name)
				} else {
					errs = append(errs, fmt.Sprintf("%s.%s: expected nested object", path, f.Name))
				}
			} else {
				// expect simple types int/string
				switch val := pv.(type) {
				case *ast.IntegerLiteral:
					if f.Type != "int" {
						errs = append(errs, fmt.Sprintf("%s.%s: type mismatch, expected %s got int", path, f.Name, f.Type))
					}
				case *ast.StringLiteral:
					if f.Type != "string" {
						errs = append(errs, fmt.Sprintf("%s.%s: type mismatch, expected %s got string", path, f.Name, f.Type))
					}
				default:
					// other expression types not deeply checked here
					_ = val
				}
			}
		}
	}

	// validate let/const assignments
	for _, s := range program.Statements {
		switch st := s.(type) {
		case *ast.LetStatement:
			if st.TypeName != "" {
				td, ok := typeDefs[st.TypeName]
				if !ok {
					errs = append(errs, fmt.Sprintf("unknown type: %s", st.TypeName))
					continue
				}
				if ml, ok := st.Value.(*ast.MapLiteral); ok {
					checkMapAgainstType(ml, td, st.Name.Value)
				}
			}
		case *ast.ConstStatement:
			if st.TypeName != "" {
				td, ok := typeDefs[st.TypeName]
				if !ok {
					errs = append(errs, fmt.Sprintf("unknown type: %s", st.TypeName))
					continue
				}
				if ml, ok := st.Value.(*ast.MapLiteral); ok {
					checkMapAgainstType(ml, td, st.Name.Value)
				}
			}
		}
	}

	// traverse member access expressions to ensure fields exist
	var checkExpr func(expr ast.Expression, ctx string)
	checkExpr = func(expr ast.Expression, ctx string) {
		switch e := expr.(type) {
		case *ast.MemberAccessExpression:
			// resolve left side type
			if id, ok := e.Object.(*ast.Identifier); ok {
				if vt, known := varTypes[id.Value]; known {
					if td, ok := typeDefs[vt]; ok {
						found := false
						for _, f := range td.Fields {
							if f.Name == e.Property.Value {
								found = true
								break
							}
						}
						if !found {
							errs = append(errs, fmt.Sprintf("%s: unknown field '%s' on type %s", ctx, e.Property.Value, vt))
						}
					}
				}
			}
			// continue deeper
			checkExpr(e.Object, ctx)
		case *ast.CallExpression:
			// check function call against known signature if identifier
			if ident, ok := e.Function.(*ast.Identifier); ok {
				if sig, found := funcSigs[ident.Value]; found {
					// arg count check
					if len(e.Arguments) != len(sig.ParamOrder) {
						errs = append(errs, fmt.Sprintf("%s: function %s expects %d args, got %d", ctx, ident.Value, len(sig.ParamOrder), len(e.Arguments)))
					} else {
						for i, paramName := range sig.ParamOrder {
							ptyp := sig.Params[paramName]
							arg := e.Arguments[i]
							switch a := arg.(type) {
							case *ast.IntegerLiteral:
								if ptyp != "int" {
									errs = append(errs, fmt.Sprintf("%s: arg %d for %s should be %s", ctx, i, ident.Value, ptyp))
								}
							case *ast.StringLiteral:
								if ptyp != "string" {
									errs = append(errs, fmt.Sprintf("%s: arg %d for %s should be %s", ctx, i, ident.Value, ptyp))
								}
							case *ast.Identifier:
								if vt, ok := varTypes[a.Value]; ok {
									if vt != ptyp {
										errs = append(errs, fmt.Sprintf("%s: arg %d for %s: expected %s got %s", ctx, i, ident.Value, ptyp, vt))
									}
								}
							}
						}
					}
				}
			}
			// recurse into function and args
			checkExpr(e.Function, ctx)
			for _, a := range e.Arguments {
				checkExpr(a, ctx)
			}
		case *ast.IndexExpression:
			checkExpr(e.Left, ctx)
		case *ast.InfixExpression:
			checkExpr(e.Left, ctx)
			checkExpr(e.Right, ctx)
		case *ast.FunctionLiteral:
			// check body
			for _, stmt := range e.Body.Statements {
				if es, ok := stmt.(*ast.ExpressionStatement); ok {
					checkExpr(es.Expression, ctx)
				}
			}
		}
	}

	for _, s := range program.Statements {
		switch st := s.(type) {
		case *ast.ExpressionStatement:
			checkExpr(st.Expression, "<expr>")
		case *ast.LetStatement:
			checkExpr(st.Value, st.Name.Value)
		case *ast.ConstStatement:
			checkExpr(st.Value, st.Name.Value)
		}
	}

	return errs
}
