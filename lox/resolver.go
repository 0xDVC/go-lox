package lox

import (
	"github.com/0xdvc/go-lox/expr"
	"github.com/0xdvc/go-lox/stmt"
	"github.com/0xdvc/go-lox/token"
)

type functionType int

const (
	ftNone functionType = iota
	ftFunction
	ftInitializer
	ftMethod
)

type classType int

const (
	ctNone classType = iota
	ctClass
	ctSubclass
)

type Resolver struct {
	interpreter     *Interpreter
	vm              resolverVM
	scopes          []map[string]bool
	currentFunction functionType
	currentClass    classType
}

type resolverVM interface {
	ReportResolveError(tok *token.Token, message string)
}

func NewResolver(interpreter *Interpreter, vm resolverVM) *Resolver {
	return &Resolver{
		interpreter: interpreter,
		vm:          vm,
	}
}

func (r *Resolver) Resolve(statements []stmt.Stmt) {
	for _, s := range statements {
		r.resolveStmt(s)
	}
}

func (r *Resolver) resolveStmt(s stmt.Stmt) {
	s.Accept(r)
}

func (r *Resolver) resolveExpr(e expr.Expr) {
	e.Accept(r)
}

func (r *Resolver) beginScope() {
	r.scopes = append(r.scopes, make(map[string]bool))
}

func (r *Resolver) endScope() {
	r.scopes = r.scopes[:len(r.scopes)-1]
}

func (r *Resolver) declare(name *token.Token) {
	if len(r.scopes) == 0 {
		return
	}
	scope := r.scopes[len(r.scopes)-1]
	if _, exists := scope[name.Lexeme]; exists {
		r.vm.ReportResolveError(name, "Already a variable with this name in this scope.")
	}
	scope[name.Lexeme] = false
}

func (r *Resolver) define(name *token.Token) {
	if len(r.scopes) == 0 {
		return
	}
	r.scopes[len(r.scopes)-1][name.Lexeme] = true
}

func (r *Resolver) resolveLocal(e expr.Expr, name *token.Token) {
	for i := len(r.scopes) - 1; i >= 0; i-- {
		if _, ok := r.scopes[i][name.Lexeme]; ok {
			r.interpreter.resolve(e, len(r.scopes)-1-i)
			return
		}
	}
}

func (r *Resolver) resolveFunction(f stmt.Function, ft functionType) {
	enclosingFunction := r.currentFunction
	r.currentFunction = ft

	r.beginScope()
	for _, param := range f.Params {
		r.declare(param)
		r.define(param)
	}
	r.Resolve(f.Body)
	r.endScope()

	r.currentFunction = enclosingFunction
}

// Statement visitors.

func (r *Resolver) VisitBlockStmt(s stmt.Block) {
	r.beginScope()
	r.Resolve(s.Statements)
	r.endScope()
}

func (r *Resolver) VisitVarStmt(s stmt.Var) {
	r.declare(s.Name)
	if s.Initializer != nil {
		r.resolveExpr(s.Initializer)
	}
	r.define(s.Name)
}

func (r *Resolver) VisitFunctionStmt(s stmt.Function) {
	r.declare(s.Name)
	r.define(s.Name)
	r.resolveFunction(s, ftFunction)
}

func (r *Resolver) VisitClassStmt(s stmt.Class) {
	enclosingClass := r.currentClass
	r.currentClass = ctClass

	r.declare(s.Name)
	r.define(s.Name)

	if s.Superclass != nil {
		if s.Superclass.Name.Lexeme == s.Name.Lexeme {
			r.vm.ReportResolveError(s.Superclass.Name, "A class can't inherit from itself.")
		}
		r.currentClass = ctSubclass
		r.resolveExpr(s.Superclass)

		// Create a scope for "super"
		r.beginScope()
		r.scopes[len(r.scopes)-1]["super"] = true
	}

	// Create a scope for "this"
	r.beginScope()
	r.scopes[len(r.scopes)-1]["this"] = true

	for _, method := range s.Methods {
		ft := ftMethod
		if method.Name.Lexeme == "init" {
			ft = ftInitializer
		}
		r.resolveFunction(method, ft)
	}

	r.endScope() // "this" scope

	if s.Superclass != nil {
		r.endScope() // "super" scope
	}

	r.currentClass = enclosingClass
}

func (r *Resolver) VisitExpressionStmt(s stmt.Expression) {
	r.resolveExpr(s.Expression)
}

func (r *Resolver) VisitIfStmt(s stmt.If) {
	r.resolveExpr(s.Condition)
	r.resolveStmt(s.ThenBranch)
	if s.ElseBranch != nil {
		r.resolveStmt(s.ElseBranch)
	}
}

func (r *Resolver) VisitPrintStmt(s stmt.Print) {
	r.resolveExpr(s.Expression)
}

func (r *Resolver) VisitReturnStmt(s stmt.Return) {
	if r.currentFunction == ftNone {
		r.vm.ReportResolveError(s.Keyword, "Can't return from top-level code.")
	}
	if s.Value != nil {
		if r.currentFunction == ftInitializer {
			r.vm.ReportResolveError(s.Keyword, "Can't return a value from an initializer.")
		}
		r.resolveExpr(s.Value)
	}
}

func (r *Resolver) VisitWhileStmt(s stmt.While) {
	r.resolveExpr(s.Condition)
	r.resolveStmt(s.Body)
}

// Expression visitors.

func (r *Resolver) VisitVariableExpr(e expr.Variable) interface{} {
	if len(r.scopes) > 0 {
		if defined, ok := r.scopes[len(r.scopes)-1][e.Name.Lexeme]; ok && !defined {
			r.vm.ReportResolveError(e.Name, "Can't read local variable in its own initializer.")
		}
	}
	r.resolveLocal(e, e.Name)
	return nil
}

func (r *Resolver) VisitAssignExpr(e expr.Assign) interface{} {
	r.resolveExpr(e.Value)
	r.resolveLocal(e, e.Name)
	return nil
}

func (r *Resolver) VisitBinaryExpr(e expr.Binary) interface{} {
	r.resolveExpr(e.Left)
	r.resolveExpr(e.Right)
	return nil
}

func (r *Resolver) VisitCallExpr(e expr.Call) interface{} {
	r.resolveExpr(e.Callee)
	for _, arg := range e.Arguments {
		r.resolveExpr(arg)
	}
	return nil
}

func (r *Resolver) VisitGetExpr(e expr.Get) interface{} {
	r.resolveExpr(e.Object)
	return nil
}

func (r *Resolver) VisitSetExpr(e expr.Set) interface{} {
	r.resolveExpr(e.Value)
	r.resolveExpr(e.Object)
	return nil
}

func (r *Resolver) VisitThisExpr(e expr.This) interface{} {
	if r.currentClass == ctNone {
		r.vm.ReportResolveError(e.Keyword, "Can't use 'this' outside of a class.")
		return nil
	}
	r.resolveLocal(e, e.Keyword)
	return nil
}

func (r *Resolver) VisitSuperExpr(e expr.Super) interface{} {
	if r.currentClass == ctNone {
		r.vm.ReportResolveError(e.Keyword, "Can't use 'super' outside of a class.")
	} else if r.currentClass != ctSubclass {
		r.vm.ReportResolveError(e.Keyword, "Can't use 'super' in a class with no superclass.")
	}
	r.resolveLocal(e, e.Keyword)
	return nil
}

func (r *Resolver) VisitGroupingExpr(e expr.Grouping) interface{} {
	r.resolveExpr(e.Expression)
	return nil
}

func (r *Resolver) VisitLiteralExpr(_ expr.Literal) interface{} {
	return nil
}

func (r *Resolver) VisitLogicalExpr(e expr.Logical) interface{} {
	r.resolveExpr(e.Left)
	r.resolveExpr(e.Right)
	return nil
}

func (r *Resolver) VisitUnaryExpr(e expr.Unary) interface{} {
	r.resolveExpr(e.Right)
	return nil
}

func (r *Resolver) VisitCommaExpr(e expr.Comma) interface{} {
	r.resolveExpr(e.Left)
	r.resolveExpr(e.Right)
	return nil
}
