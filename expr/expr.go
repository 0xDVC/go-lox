package expr

import "github.com/0xdvc/go-lox/token"

type Expr interface {
	String() string
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

type Grouping struct {
	Expression Expr
}

type Literal struct {
	Value any
}

type Unary struct {
	Operator *token.Token
	Right    Expr
}
