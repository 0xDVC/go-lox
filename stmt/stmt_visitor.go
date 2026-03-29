package stmt

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

func (s Function) Accept(visitor StmtVisitor) {
	visitor.VisitFunctionStmt(s)
}

func (s Return) Accept(visitor StmtVisitor) {
	visitor.VisitReturnStmt(s)
}

func (s Class) Accept(visitor StmtVisitor) {
	visitor.VisitClassStmt(s)
}
