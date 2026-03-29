package expr

import "github.com/0xdvc/go-lox/token"

type Visitor interface {
	VisitLiteralExpr(expr Literal) interface{}
	VisitGroupingExpr(expr Grouping) interface{}
	VisitUnaryExpr(expr Unary) interface{}
	VisitBinaryExpr(expr Binary) interface{}
	VisitCommaExpr(expr Comma) interface{}
	VisitVariableExpr(expr Variable) interface{}
	VisitAssignExpr(expr Assign) interface{}
	VisitLogicalExpr(expr Logical) interface{}
	VisitCallExpr(expr Call) interface{}
	VisitGetExpr(expr Get) interface{}
	VisitSetExpr(expr Set) interface{}
	VisitThisExpr(expr This) interface{}
	VisitSuperExpr(expr Super) interface{}
}

type Expr interface {
	Accept(visitor Visitor) interface{}
}

type Literal struct {
	Value interface{}
}

type Grouping struct {
	Expression Expr
}

type Unary struct {
	Operator *token.Token
	Right    Expr
}

type Binary struct {
	Left     Expr
	Operator *token.Token
	Right    Expr
}

type Comma struct {
	Left  Expr
	Right Expr
}

type Variable struct {
	Name *token.Token
}

// Assign returns the assigned value so you can chain: a = b = 5
type Assign struct {
	Name  *token.Token
	Value Expr
}

// Logical short-circuits: `false and crash()` never calls crash().
// Returns the value that determined the result, not just true/false.
type Logical struct {
	Left     Expr
	Operator *token.Token
	Right    Expr
}

type Call struct {
	Callee    Expr
	Paren     *token.Token // closing ')' for error reporting
	Arguments []Expr
}

type Get struct {
	Object Expr
	Name   *token.Token
}

type Set struct {
	Object Expr
	Name   *token.Token
	Value  Expr
}

type This struct {
	Keyword *token.Token
}

type Super struct {
	Keyword *token.Token
	Method  *token.Token
}
