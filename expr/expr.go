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

// Variable reads a variable by name. `print x;` — the `x` is this.
type Variable struct {
	Name *token.Token
}

// Assign writes a value to a variable. `x = 42` — an expression, not a statement.
// Returns the assigned value so you can chain: `a = b = 5`
type Assign struct {
	Name  *token.Token
	Value Expr
}
