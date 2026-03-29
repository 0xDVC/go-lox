package lox

import (
	"fmt"

	"github.com/0xdvc/go-lox/expr"
	"github.com/0xdvc/go-lox/stmt"
	"github.com/0xdvc/go-lox/token"
)

type RuntimeError struct {
	Token   *token.Token
	Message string
}

func (e *RuntimeError) Error() string {
	return e.Message
}

type Interpreter struct {
	environment *Environment
}

func NewInterpreter() *Interpreter {
	return &Interpreter{
		environment: NewEnvironment(),
	}
}

// Interpret executes a list of statements — this is the entry point.
func (i *Interpreter) Interpret(statements []stmt.Stmt) {
	for _, s := range statements {
		i.execute(s)
	}
}

// execute runs a single statement by dispatching to the visitor.
func (i *Interpreter) execute(s stmt.Stmt) {
	s.Accept(i)
}

// --- Statement visitors ---

// VisitExpressionStmt evaluates the expression and discards the result.
// Example: `someFunction();` — we call the function for its side effect.
func (i *Interpreter) VisitExpressionStmt(s stmt.Expression) {
	i.evaluate(s.Expression)
}

// VisitPrintStmt evaluates the expression and prints the result.
func (i *Interpreter) VisitPrintStmt(s stmt.Print) {
	value := i.evaluate(s.Expression)
	fmt.Println(i.stringify(value))
}

// VisitVarStmt declares a variable. If there's an initializer, evaluate it.
// Otherwise the variable is nil (Lox's version of undefined).
func (i *Interpreter) VisitVarStmt(s stmt.Var) {
	var value any
	if s.Initializer != nil {
		value = i.evaluate(s.Initializer)
	}
	i.environment.Define(s.Name.Lexeme, value)
}

// VisitBlockStmt executes a list of statements in a new scope.
// Creates a new environment enclosed by the current one, executes the
// statements, then restores the previous environment.
func (i *Interpreter) VisitBlockStmt(s stmt.Block) {
	i.executeBlock(s.Statements, NewEnclosedEnvironment(i.environment))
}

// executeBlock runs statements in a given environment, restoring the
// previous environment when done. The `defer` ensures restoration even
// if a runtime error (panic) occurs mid-block.
func (i *Interpreter) executeBlock(statements []stmt.Stmt, env *Environment) {
	previous := i.environment
	defer func() {
		i.environment = previous
	}()

	i.environment = env
	for _, s := range statements {
		i.execute(s)
	}
}

// VisitIfStmt evaluates the condition and executes the appropriate branch.
func (i *Interpreter) VisitIfStmt(s stmt.If) {
	if i.isTruthy(i.evaluate(s.Condition)) {
		i.execute(s.ThenBranch)
	} else if s.ElseBranch != nil {
		i.execute(s.ElseBranch)
	}
}

// VisitWhileStmt loops as long as the condition is truthy.
func (i *Interpreter) VisitWhileStmt(s stmt.While) {
	for i.isTruthy(i.evaluate(s.Condition)) {
		i.execute(s.Body)
	}
}

// --- Expression visitors ---

func (i *Interpreter) VisitLiteralExpr(e expr.Literal) interface{} {
	return e.Value
}

func (i *Interpreter) VisitGroupingExpr(e expr.Grouping) interface{} {
	return i.evaluate(e.Expression)
}

func (i *Interpreter) VisitUnaryExpr(e expr.Unary) interface{} {
	right := i.evaluate(e.Right)

	switch e.Operator.TokenType {
	case token.TokenType_Minus:
		if num, ok := right.(float64); ok {
			return -num
		}
		panic(&RuntimeError{Token: e.Operator, Message: "Operand must be a number."})
	case token.TokenType_Bang:
		return !i.isTruthy(right)
	}
	return nil
}

func (i *Interpreter) VisitBinaryExpr(e expr.Binary) interface{} {
	left := i.evaluate(e.Left)
	right := i.evaluate(e.Right)

	switch e.Operator.TokenType {
	case token.TokenType_Minus:
		l, r := i.checkNumberOperands(e.Operator, left, right)
		return l - r
	case token.TokenType_Slash:
		l, r := i.checkNumberOperands(e.Operator, left, right)
		if r == 0 {
			panic(&RuntimeError{Token: e.Operator, Message: "Division by zero."})
		}
		return l / r
	case token.TokenType_Star:
		l, r := i.checkNumberOperands(e.Operator, left, right)
		return l * r
	case token.TokenType_Percent:
		l, r := i.checkNumberOperands(e.Operator, left, right)
		if r == 0 {
			panic(&RuntimeError{Token: e.Operator, Message: "Division by zero."})
		}
		return float64(int(l) % int(r))
	case token.TokenType_Plus:
		if l, ok := left.(float64); ok {
			if r, ok := right.(float64); ok {
				return l + r
			}
		}
		if l, ok := left.(string); ok {
			if r, ok := right.(string); ok {
				return l + r
			}
		}
		panic(&RuntimeError{Token: e.Operator, Message: "Operands must be two numbers or two strings."})
	case token.TokenType_Greater:
		l, r := i.checkNumberOperands(e.Operator, left, right)
		return l > r
	case token.TokenType_GreaterEqual:
		l, r := i.checkNumberOperands(e.Operator, left, right)
		return l >= r
	case token.TokenType_Less:
		l, r := i.checkNumberOperands(e.Operator, left, right)
		return l < r
	case token.TokenType_LessEqual:
		l, r := i.checkNumberOperands(e.Operator, left, right)
		return l <= r
	case token.TokenType_BangEqual:
		return !i.isEqual(left, right)
	case token.TokenType_EqualEqual:
		return i.isEqual(left, right)
	}
	return nil
}

func (i *Interpreter) VisitCommaExpr(e expr.Comma) interface{} {
	i.evaluate(e.Left)
	return i.evaluate(e.Right)
}

// VisitLogicalExpr implements short-circuit evaluation.
// Returns the value that determined the result, not true/false.
// `nil or "default"` → "default"   |   "yes" and "no" → "no"
func (i *Interpreter) VisitLogicalExpr(e expr.Logical) interface{} {
	left := i.evaluate(e.Left)

	if e.Operator.TokenType == token.TokenType_Or {
		if i.isTruthy(left) {
			return left // short-circuit: left is truthy, skip right
		}
	} else { // and
		if !i.isTruthy(left) {
			return left // short-circuit: left is falsy, skip right
		}
	}

	return i.evaluate(e.Right)
}

// VisitVariableExpr looks up a variable in the current environment chain.
func (i *Interpreter) VisitVariableExpr(e expr.Variable) interface{} {
	return i.environment.Get(e.Name)
}

// VisitAssignExpr assigns a value to an existing variable.
// Returns the assigned value — assignment is an expression in Lox.
func (i *Interpreter) VisitAssignExpr(e expr.Assign) interface{} {
	value := i.evaluate(e.Value)
	i.environment.Assign(e.Name, value)
	return value
}

// --- Helpers ---

func (i *Interpreter) evaluate(e expr.Expr) interface{} {
	return e.Accept(i)
}

func (i *Interpreter) isTruthy(obj interface{}) bool {
	if obj == nil {
		return false
	}
	if b, ok := obj.(bool); ok {
		return b
	}
	return true
}

func (i *Interpreter) isEqual(a, b interface{}) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil {
		return false
	}
	return a == b
}

// checkNumberOperands verifies both operands are float64 and returns them.
// Panics with a RuntimeError if not — this is how Lox reports type errors.
func (i *Interpreter) checkNumberOperands(operator *token.Token, left, right interface{}) (float64, float64) {
	l, lok := left.(float64)
	r, rok := right.(float64)
	if lok && rok {
		return l, r
	}
	panic(&RuntimeError{Token: operator, Message: "Operands must be numbers."})
}

func (i *Interpreter) stringify(obj interface{}) string {
	if obj == nil {
		return "nil"
	}
	if num, ok := obj.(float64); ok {
		text := fmt.Sprintf("%g", num)
		return text
	}
	return fmt.Sprintf("%v", obj)
}
