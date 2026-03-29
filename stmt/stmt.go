package stmt

import (
	"github.com/0xdvc/go-lox/expr"
	"github.com/0xdvc/go-lox/token"
)

type StmtVisitor interface {
	VisitExpressionStmt(stmt Expression)
	VisitPrintStmt(stmt Print)
	VisitVarStmt(stmt Var)
	VisitBlockStmt(stmt Block)
	VisitIfStmt(stmt If)
	VisitWhileStmt(stmt While)
	VisitFunctionStmt(stmt Function)
	VisitReturnStmt(stmt Return)
}

type Stmt interface {
	Accept(visitor StmtVisitor)
}

type Expression struct {
	Expression expr.Expr
}

type Print struct {
	Expression expr.Expr
}

type Var struct {
	Name        *token.Token
	Initializer expr.Expr
}

type Block struct {
	Statements []Stmt
}

type If struct {
	Condition  expr.Expr
	ThenBranch Stmt
	ElseBranch Stmt
}

// While also represents desugared `for` loops.
type While struct {
	Condition expr.Expr
	Body      Stmt
}

type Function struct {
	Name   *token.Token
	Params []*token.Token
	Body   []Stmt
}

type Return struct {
	Keyword *token.Token
	Value   expr.Expr
}
