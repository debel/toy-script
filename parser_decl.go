package main

import (
	"fmt"
	"strings"
)

type (
	ExpressionVisitor interface {
		VisitString(n *StringLiteral) any
		VisitNumber(n *NumberLiteral) any
		VisitBoolean(n *BooleanLiteral) any
		VisitList(n *ListLiteral) any
		VisitHash(n *HashLiteral) any
		VisitStream(n *StreamLiteral) any
		VisitFunc(n *FuncLiteral) any
		VisitProgram(n *ProgramStatement) any
		VisitVar(n *VarStatement) any
		VisitImport(n *ImportStatement) any
		VisitExport(n *ExportStatement) any
		VisitRef(n *ReferenceExpression) any
		VisitCall(n *CallExpression) any
		VisitMatch(n *MatchExpression) any
		VisitMalformed(n *MalformedExpression) any
		VisitSeq(n *SeqExpression) any
		VisitChain(n *ChainExpression) any
		VisitAsync(n *AsyncExpression) any
	}

	Value = any

	Node interface {
		Accept(v ExpressionVisitor) any
		Type() string
		String() string
	}

	StringLiteral struct {
		Value string
	}

	NumberLiteral struct {
		Value int
	}

	BooleanLiteral struct {
		Value bool
	}

	ListLiteral struct {
		Elements []Node
	}

	HashLiteral struct {
		Elements map[string]Node
	}

	StreamLiteral struct {
		Channal chan Node
	}

	FuncLiteral struct {
		Params []string
		Body   []Node
	}

	ProgramStatement struct {
		Body []Node
	}

	VarStatement struct {
		Vars map[string]Node
	}

	ImportStatement struct {
		Imports map[string]string
	}

	ExportStatement struct {
		Exports []Node
	}

	ReferenceType = string

	ReferenceExpression struct {
		RefName string
		RefType ReferenceType
	}

	CallExpression struct {
		Callee Node
		Args   []Node
	}

	MatchExpression struct {
		Cond  Node
		Cases map[Node]Node
	}

	MalformedExpression struct {
		Body  Value
		Error error
	}

	SeqExpression struct {
		Expressions []Node
	}

	ChainExpression struct {
		Expressions []Node
	}

	AsyncExpression struct {
		Expressions []Node
	}
)

const (
	REF_TYPE_BUILTIN  ReferenceType = "built-in"
	REF_TYPE_DECLARED ReferenceType = "declared"
	REF_TYPE_IMPORTED ReferenceType = "imported"
)

func (n *StringLiteral) Type() string {
	return "StringLiteral"
}

func (n *StringLiteral) String() string {
	return "\"" + n.Value + "\""
}

func (n *StringLiteral) Accept(v ExpressionVisitor) any {
	return v.VisitString(n)
}

func (n *NumberLiteral) Type() string {
	return "NumberLiteral"
}

func (n *NumberLiteral) String() string {
	return fmt.Sprintf("%d", n.Value)
}

func (n *NumberLiteral) Accept(v ExpressionVisitor) any {
	return v.VisitNumber(n)
}

func (n *BooleanLiteral) Type() string {
	return "BooleanLiteral"
}

func (n *BooleanLiteral) String() string {
	if n.Value {
		return "TRUE"
	}

	return "FALSE"
}

func (n *BooleanLiteral) Accept(v ExpressionVisitor) any {
	return v.VisitBoolean(n)
}

func (n *ListLiteral) Type() string {
	return "ListLiteral"
}

func (n *ListLiteral) String() string {
	str := strings.Builder{}
	str.WriteString(":LIST (\n")

	for _, el := range n.Elements {
		str.WriteString(el.String() + "\n")
	}

	str.WriteString(")")

	return str.String()
}

func (n *ListLiteral) Accept(v ExpressionVisitor) any {
	return v.VisitList(n)
}

func (n *HashLiteral) Type() string {
	return "HashLiteral"
}

func (n *HashLiteral) String() string {
	str := strings.Builder{}
	str.WriteString(":HASH (\n")

	for key, el := range n.Elements {
		str.WriteString("KEY: " + key + " VALUE: " + el.String() + "\n")
	}

	str.WriteString(")")

	return str.String()
}

func (n *HashLiteral) Accept(v ExpressionVisitor) any {
	return v.VisitHash(n)
}

func (n *StreamLiteral) Type() string {
	return "StreamLiteral"
}

func (n *StreamLiteral) String() string {
	// TODO:
	return ""
}

func (n *StreamLiteral) Accept(v ExpressionVisitor) any {
	return v.VisitStream(n)
}

func (n *FuncLiteral) Type() string {
	return "FuncLiteral"
}

func (n *FuncLiteral) String() string {
	str := strings.Builder{}
	str.WriteString(":FUNC (\n")

	str.WriteString("  PARAMS(" + strings.Join(n.Params, ", ") + ")\n")

	str.WriteString("  BODY(")
	for _, c := range n.Body {
		str.WriteString(c.String())
	}
	str.WriteString(")\n)")

	return str.String()
}

func (n *FuncLiteral) Accept(v ExpressionVisitor) any {
	return v.VisitFunc(n)
}

func (n *ProgramStatement) Type() string {
	return "ProgramStatement"
}

func (n *ProgramStatement) String() string {
	str := strings.Builder{}
	str.WriteString(":PROGRAM (\n")

	for _, stmt := range n.Body {
		str.WriteString(stmt.String() + "\n")
	}

	str.WriteString(")")

	return str.String()
}

func (n *ProgramStatement) Accept(v ExpressionVisitor) any {
	return v.VisitProgram(n)
}

func (n *VarStatement) Type() string {
	return "VarStatement"
}

func (n *VarStatement) String() string {
	str := strings.Builder{}
	str.WriteString(":VAR (\n")

	for key, el := range n.Vars {
		str.WriteString("NAME: " + key + " VALUE: " + el.String() + "\n\n")
	}

	str.WriteString(")")

	return str.String()
}

func (n *VarStatement) Accept(v ExpressionVisitor) any {
	return v.VisitVar(n)
}

func (n *ImportStatement) Type() string {
	return "ImportStatement"
}

func (n *ImportStatement) String() string {
	str := strings.Builder{}
	str.WriteString(":IMPORTS (\n")

	for key, el := range n.Imports {
		str.WriteString("  ALIAS: " + key + " VALUE: " + el + "\n")
	}

	str.WriteString(")")

	return str.String()
}

func (n *ImportStatement) Accept(v ExpressionVisitor) any {
	return v.VisitImport(n)
}

func (n *ExportStatement) Type() string {
	return "ExportStatement"
}

func (n *ExportStatement) String() string {
	str := strings.Builder{}
	str.WriteString(":EXPORTS (\n")

	for _, el := range n.Exports {
		str.WriteString(el.String() + "\n")
	}

	str.WriteString(")")

	return str.String()
}

func (n *ExportStatement) Accept(v ExpressionVisitor) any {
	return v.VisitExport(n)
}

func (n *CallExpression) Type() string {
	return "CallExpression"
}

func (n *CallExpression) String() string {
	strArgs := []string{}

	for _, arg := range n.Args {
		strArgs = append(strArgs, arg.String())
	}

	return fmt.Sprintf("FUNC_CALL: (\n FUNC_NAME: %s\n ARGS: (%s)", n.Callee.String(), strings.Join(strArgs, ", "))
}

func (n *CallExpression) Accept(v ExpressionVisitor) any {
	return v.VisitCall(n)
}

func (n *MatchExpression) Type() string {
	return "MatchExpression"
}

func (n *MatchExpression) String() string {
	str := strings.Builder{}
	str.WriteString(" :MATCH (\n")
	str.WriteString("  :COND (" + n.Cond.String() + ")\n")

	for w, a := range n.Cases {
		str.WriteString(":WHEN ( " + w.String() + " " + a.String() + ")\n")
	}

	str.WriteString(")")
	return str.String()
}

func (n *MatchExpression) Accept(v ExpressionVisitor) any {
	return v.VisitMatch(n)
}

func (n *MalformedExpression) Type() string {
	return "MalformedExpression"
}

func (n *MalformedExpression) String() string {
	return fmt.Sprintf(":ERROR(%v, %s)", n.Body, n.Error.Error())
}

func (n *MalformedExpression) Accept(v ExpressionVisitor) any {
	return v.VisitMalformed(n)
}

func (n *ReferenceExpression) Type() string {
	return "ReferenceExpression"
}

func (n *ReferenceExpression) String() string {
	return ":REF (" + n.RefType + ", " + n.RefName + ")"
}

func (n *ReferenceExpression) Accept(v ExpressionVisitor) any {
	return v.VisitRef(n)
}

func (n *SeqExpression) Type() string {
	return "SeqExpression"
}

func (n *SeqExpression) String() string {
	str := strings.Builder{}

	str.WriteString("SEQ: (\n")

	for _, expr := range n.Expressions {
		str.WriteString("  " + expr.String() + "\n")
	}

	str.WriteString(")\n")

	return str.String()
}

func (n *SeqExpression) Accept(v ExpressionVisitor) any {
	return v.VisitSeq(n)
}

func (n *ChainExpression) Type() string {
	return "ChainExpression"
}

func (n *ChainExpression) String() string {
	str := strings.Builder{}
	str.WriteString(":CHAIN (\n")

	for _, c := range n.Expressions {
		str.WriteString(c.String() + "\n")
	}

	str.WriteString(")\n")
	return str.String()
}

func (n *ChainExpression) Accept(v ExpressionVisitor) any {
	return v.VisitChain(n)
}

func (n *AsyncExpression) Type() string {
	return "AsyncExpression"
}

func (n *AsyncExpression) String() string {
	str := strings.Builder{}

	str.WriteString(":ASYNC (\n")

	for _, expr := range n.Expressions {
		str.WriteString("  " + expr.String() + "\n")
	}

	str.WriteString(")\n")

	return str.String()
}

func (n *AsyncExpression) Accept(v ExpressionVisitor) any {
	return v.VisitAsync(n)
}
