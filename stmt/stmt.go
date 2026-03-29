package stmt

import (
	"github.com/0xdvc/go-lox/expr"
	"github.com/0xdvc/go-lox/token"
)

// StmtVisitor — anything that processes statements must implement this.
// One method per statement type, just like expr.Visitor.
type StmtVisitor interface {
	VisitExpressionStmt(stmt Expression)
	VisitPrintStmt(stmt Print)
	VisitVarStmt(stmt Var)
	VisitBlockStmt(stmt Block)
	VisitIfStmt(stmt If)
	VisitWhileStmt(stmt While)
}

// Stmt is the interface all statement AST nodes implement.
// Notice it returns nothing — statements produce side effects, not values.
type Stmt interface {
	Accept(visitor StmtVisitor)
}

// Expression wraps an expression as a statement.
// Example: `someFunction();` — the call is an expression, but the semicolon makes it a statement.
type Expression struct {
	Expression expr.Expr
}

// Print outputs the value of an expression.
// `print "hello";` — a built-in statement, not a function call in Lox.
type Print struct {
	Expression expr.Expr
}

// Var declares a variable with an optional initializer.
// `var x = 10;` or just `var x;` (initializes to nil).
type Var struct {
	Name        *token.Token
	Initializer expr.Expr // nil if no initializer
}

// Block is a list of statements wrapped in { }.
// Creates a new scope — this is where lexical scoping comes from.
type Block struct {
	Statements []Stmt
}

// If represents conditional execution.
// ElseBranch is nil when there's no else clause.
type If struct {
	Condition  expr.Expr
	ThenBranch Stmt
	ElseBranch Stmt // nil if no else
}

// While represents a loop that runs as long as the condition is truthy.
// `for` loops also become While nodes — the parser desugars them.
type While struct {
	Condition expr.Expr
	Body      Stmt
}
