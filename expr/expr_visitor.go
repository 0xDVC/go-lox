package expr

func (l Literal) Accept(visitor Visitor) interface{} {
	return visitor.VisitLiteralExpr(l)
}

func (g Grouping) Accept(visitor Visitor) interface{} {
	return visitor.VisitGroupingExpr(g)
}

func (u Unary) Accept(visitor Visitor) interface{} {
	return visitor.VisitUnaryExpr(u)
}

func (b Binary) Accept(visitor Visitor) interface{} {
	return visitor.VisitBinaryExpr(b)
}

func (c Comma) Accept(visitor Visitor) interface{} {
	return visitor.VisitCommaExpr(c)
}

func (v Variable) Accept(visitor Visitor) interface{} {
	return visitor.VisitVariableExpr(v)
}

func (a Assign) Accept(visitor Visitor) interface{} {
	return visitor.VisitAssignExpr(a)
}

func (l Logical) Accept(visitor Visitor) interface{} {
	return visitor.VisitLogicalExpr(l)
}

func (c Call) Accept(visitor Visitor) interface{} {
	return visitor.VisitCallExpr(c)
}

func (g Get) Accept(visitor Visitor) interface{} {
	return visitor.VisitGetExpr(g)
}

func (s Set) Accept(visitor Visitor) interface{} {
	return visitor.VisitSetExpr(s)
}

func (t This) Accept(visitor Visitor) interface{} {
	return visitor.VisitThisExpr(t)
}

func (s Super) Accept(visitor Visitor) interface{} {
	return visitor.VisitSuperExpr(s)
}
