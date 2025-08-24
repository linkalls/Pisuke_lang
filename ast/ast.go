package ast

import (
	"bytes"
	"pisuke/token"
	"strings"
)

// Node is the base interface for all AST nodes.
type Node interface {
	TokenLiteral() string // for debugging and testing
	String() string
}

// Statement represents a statement in the language.
type Statement interface {
	Node
	statementNode()
}

// Expression represents an expression in the language.
type Expression interface {
	Node
	expressionNode()
}

// Program is the root node of every AST our parser produces.
type Program struct {
	Statements []Statement
}

func (p *Program) TokenLiteral() string {
	if len(p.Statements) > 0 {
		return p.Statements[0].TokenLiteral()
	} else {
		return ""
	}
}

func (p *Program) String() string {
	var out bytes.Buffer
	for _, s := range p.Statements {
		out.WriteString(s.String())
	}
	return out.String()
}

// LetStatement represents a 'let' statement, e.g., `let x = 5;`
type LetStatement struct {
	Token    token.Token // the token.LET token
	Name     *Identifier
	Value    Expression
	TypeName string
}

func (ls *LetStatement) statementNode()       {}
func (ls *LetStatement) TokenLiteral() string { return ls.Token.Literal }
func (ls *LetStatement) String() string {
	var out bytes.Buffer
	out.WriteString(ls.TokenLiteral() + " ")
	out.WriteString(ls.Name.String())
	out.WriteString(" = ")
	if ls.Value != nil {
		out.WriteString(ls.Value.String())
	}
	return out.String()
}

// ConstStatement represents a 'const' statement, e.g., `const MY_CONST = 10;`
type ConstStatement struct {
	Token    token.Token // the token.CONST token
	Name     *Identifier
	Value    Expression
	TypeName string
}

func (cs *ConstStatement) statementNode()       {}
func (cs *ConstStatement) TokenLiteral() string { return cs.Token.Literal }
func (cs *ConstStatement) String() string {
	var out bytes.Buffer
	out.WriteString(cs.TokenLiteral() + " ")
	out.WriteString(cs.Name.String())
	out.WriteString(" = ")
	if cs.Value != nil {
		out.WriteString(cs.Value.String())
	}
	return out.String()
}

// ReturnStatement represents a 'return' statement, e.g., `return 5;`
type ReturnStatement struct {
	Token       token.Token // the 'return' token
	ReturnValue Expression
}

func (rs *ReturnStatement) statementNode()       {}
func (rs *ReturnStatement) TokenLiteral() string { return rs.Token.Literal }
func (rs *ReturnStatement) String() string {
	var out bytes.Buffer
	out.WriteString(rs.TokenLiteral() + " ")
	if rs.ReturnValue != nil {
		out.WriteString(rs.ReturnValue.String())
	}
	return out.String()
}

// Identifier represents an identifier (variable name).
type Identifier struct {
	Token token.Token // the token.IDENT token
	Value string
}

func (i *Identifier) expressionNode()      {}
func (i *Identifier) TokenLiteral() string { return i.Token.Literal }
func (i *Identifier) String() string       { return i.Value }

// IntegerLiteral represents an integer value.
type IntegerLiteral struct {
	Token token.Token
	Value int64
}

func (il *IntegerLiteral) expressionNode()      {}
func (il *IntegerLiteral) TokenLiteral() string { return il.Token.Literal }
func (il *IntegerLiteral) String() string       { return il.Token.Literal }

// BlockStatement represents a block of statements, e.g., `{ ... }`
type BlockStatement struct {
	Token      token.Token // the { token
	Statements []Statement
}

func (bs *BlockStatement) statementNode()       {}
func (bs *BlockStatement) TokenLiteral() string { return bs.Token.Literal }
func (bs *BlockStatement) String() string {
	var out bytes.Buffer
	out.WriteString("{ ")
	for _, s := range bs.Statements {
		out.WriteString(s.String())
	}
	out.WriteString(" }")
	return out.String()
}

// FunctionLiteral represents a function definition, e.g., `fn(x, y) { ... }`
type FunctionLiteral struct {
	Token      token.Token // The 'fn' token
	Name       *Identifier
	Parameters []*Identifier
	ParamTypes map[string]string // param name -> type (optional)
	ReturnType string
	Body       *BlockStatement
}

func (fl *FunctionLiteral) expressionNode()      {}
func (fl *FunctionLiteral) TokenLiteral() string { return fl.Token.Literal }
func (fl *FunctionLiteral) String() string {
	var out bytes.Buffer
	params := []string{}
	for _, p := range fl.Parameters {
		if fl.ParamTypes != nil {
			if t, ok := fl.ParamTypes[p.Value]; ok {
				params = append(params, p.String()+": "+t)
				continue
			}
		}
		params = append(params, p.String())
	}
	out.WriteString(fl.TokenLiteral())
	if fl.Name != nil {
		out.WriteString(" ")
		out.WriteString(fl.Name.String())
	}
	out.WriteString("(")
	out.WriteString(strings.Join(params, ", "))
	out.WriteString(") ")
	if fl.ReturnType != "" {
		out.WriteString(":" + fl.ReturnType + " ")
	}
	out.WriteString(fl.Body.String())
	return out.String()
}

// TypeDefinition represents `type Name = { ... }` style declarations
type TypeDefinition struct {
	Token token.Token // the 'type' token
	Name  *Identifier
	// Fields represent struct-like members: name and type
	Fields []*Field
}

func (td *TypeDefinition) statementNode()       {}
func (td *TypeDefinition) TokenLiteral() string { return td.Token.Literal }
func (td *TypeDefinition) String() string {
	var out bytes.Buffer
	out.WriteString(td.TokenLiteral() + " " + td.Name.String() + " = ")
	if td.Fields != nil {
		fields := []string{}
		for _, f := range td.Fields {
			if f.Nested != nil {
				// represent nested inline type
				nested := []string{}
				for _, nf := range f.Nested.Fields {
					nested = append(nested, nf.Name+": "+nf.Type)
				}
				fields = append(fields, f.Name+": {"+strings.Join(nested, ", ")+"}")
			} else {
				fields = append(fields, f.Name+": "+f.Type)
			}
		}
		out.WriteString("{ " + strings.Join(fields, ", ") + " }")
	}
	return out.String()
}

// Field represents a field inside a type definition: name and type
type Field struct {
	Name   string
	Type   string
	Nested *TypeDefinition
}

// CallExpression represents a function call, e.g., `myFunction(arg1, arg2)`
type CallExpression struct {
	Token     token.Token // The '(' token
	Function  Expression  // Identifier or FunctionLiteral
	Arguments []Expression
}

func (ce *CallExpression) expressionNode()      {}
func (ce *CallExpression) TokenLiteral() string { return ce.Token.Literal }
func (ce *CallExpression) String() string {
	var out bytes.Buffer
	args := []string{}
	for _, a := range ce.Arguments {
		args = append(args, a.String())
	}
	out.WriteString(ce.Function.String())
	out.WriteString("(")
	out.WriteString(strings.Join(args, ", "))
	out.WriteString(")")
	return out.String()
}

// ExpressionStatement consists of a single expression.
type ExpressionStatement struct {
	Token      token.Token // the first token of the expression
	Expression Expression
}

func (es *ExpressionStatement) statementNode()       {}
func (es *ExpressionStatement) TokenLiteral() string { return es.Token.Literal }
func (es *ExpressionStatement) String() string {
	if es.Expression != nil {
		return es.Expression.String()
	}
	return ""
}

// InfixExpression represents a binary operation, e.g., `left + right`
type InfixExpression struct {
	Token    token.Token // The operator token, e.g. +
	Left     Expression
	Operator string
	Right    Expression
}

func (ie *InfixExpression) expressionNode()      {}
func (ie *InfixExpression) TokenLiteral() string { return ie.Token.Literal }

// MemberAccessExpression represents accessing a property of an object, e.g., `my_object.property`
type MemberAccessExpression struct {
	Token    token.Token // The . token
	Object   Expression
	Property *Identifier
}

func (mae *MemberAccessExpression) expressionNode()      {}
func (mae *MemberAccessExpression) TokenLiteral() string { return mae.Token.Literal }
func (mae *MemberAccessExpression) String() string {
	return "(" + mae.Object.String() + "." + mae.Property.String() + ")"
}

// StringLiteral represents a string value, e.g., "hello world"
type StringLiteral struct {
	Token token.Token
	Value string
}

func (sl *StringLiteral) expressionNode()      {}
func (sl *StringLiteral) TokenLiteral() string { return sl.Token.Literal }
func (sl *StringLiteral) String() string       { return sl.Token.Literal }

// ListLiteral represents a list or array, e.g., `[1, 2, 3]`
type ListLiteral struct {
	Token    token.Token // the '[' token
	Elements []Expression
}

func (ll *ListLiteral) expressionNode()      {}
func (ll *ListLiteral) TokenLiteral() string { return ll.Token.Literal }
func (ll *ListLiteral) String() string {
	var out bytes.Buffer
	elements := []string{}
	for _, el := range ll.Elements {
		elements = append(elements, el.String())
	}
	out.WriteString("[")
	out.WriteString(strings.Join(elements, ", "))
	out.WriteString("]")
	return out.String()
}

// MapLiteral represents a map or hash, e.g., `{"key": "value"}`
type MapLiteral struct {
	Token token.Token // the '{' token
	Pairs map[Expression]Expression
}

func (ml *MapLiteral) expressionNode()      {}
func (ml *MapLiteral) TokenLiteral() string { return ml.Token.Literal }
func (ml *MapLiteral) String() string {
	var out bytes.Buffer
	pairs := []string{}
	for key, value := range ml.Pairs {
		pairs = append(pairs, key.String()+":"+value.String())
	}
	out.WriteString("{")
	out.WriteString(strings.Join(pairs, ", "))
	out.WriteString("}")
	return out.String()
}

// IndexExpression represents an index operation, e.g., `my_array[0]` or `my_map["key"]`
type IndexExpression struct {
	Token token.Token // The [ token
	Left  Expression
	Index Expression
}

func (ie *IndexExpression) expressionNode()      {}
func (ie *IndexExpression) TokenLiteral() string { return ie.Token.Literal }
func (ie *IndexExpression) String() string {
	var out bytes.Buffer
	out.WriteString("(")
	out.WriteString(ie.Left.String())
	out.WriteString("[")
	out.WriteString(ie.Index.String())
	out.WriteString("])")
	return out.String()
}
func (ie *InfixExpression) String() string {
	var out bytes.Buffer
	out.WriteString("(")
	out.WriteString(ie.Left.String())
	out.WriteString(" " + ie.Operator + " ")
	out.WriteString(ie.Right.String())
	out.WriteString(")")
	return out.String()
}
