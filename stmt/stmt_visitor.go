package stmt

// Accept methods — each statement type delegates to the right visitor method.
// This is the same pattern you have in expr/expr_visitor.go.
//
// Why the visitor pattern? It lets us add new operations (interpret, print, resolve)
// without modifying the statement types themselves. Each operation is a new
// struct that implements StmtVisitor.

func (s Expression) Accept(visitor StmtVisitor) {
	visitor.VisitExpressionStmt(s)
}

func (s Print) Accept(visitor StmtVisitor) {
	visitor.VisitPrintStmt(s)
}

func (s Var) Accept(visitor StmtVisitor) {
	visitor.VisitVarStmt(s)
}

func (s Block) Accept(visitor StmtVisitor) {
	visitor.VisitBlockStmt(s)
}

func (s If) Accept(visitor StmtVisitor) {
	visitor.VisitIfStmt(s)
}

func (s While) Accept(visitor StmtVisitor) {
	visitor.VisitWhileStmt(s)
}
