package lox

import (
	"fmt"
	"io"

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
	globals     *Environment
	environment *Environment
	locals      map[expr.Expr]int
	writer      io.Writer
}

func NewInterpreter(writer io.Writer) *Interpreter {
	globals := NewEnvironment()
	globals.Define("clock", ClockNative)

	return &Interpreter{
		globals:     globals,
		environment: globals,
		locals:      make(map[expr.Expr]int),
		writer:      writer,
	}
}

func (i *Interpreter) resolve(e expr.Expr, depth int) {
	i.locals[e] = depth
}

func (i *Interpreter) lookUpVariable(name *token.Token, e expr.Expr) any {
	if distance, ok := i.locals[e]; ok {
		return i.environment.GetAt(distance, name.Lexeme)
	}
	return i.globals.Get(name)
}

func (i *Interpreter) Interpret(statements []stmt.Stmt) {
	for _, s := range statements {
		i.execute(s)
	}
}

func (i *Interpreter) execute(s stmt.Stmt) {
	s.Accept(i)
}

// Statement visitors.

func (i *Interpreter) VisitExpressionStmt(s stmt.Expression) {
	i.evaluate(s.Expression)
}

func (i *Interpreter) VisitPrintStmt(s stmt.Print) {
	value := i.evaluate(s.Expression)
	fmt.Fprintln(i.writer, i.stringify(value))
}

func (i *Interpreter) VisitVarStmt(s stmt.Var) {
	var value any
	if s.Initializer != nil {
		value = i.evaluate(s.Initializer)
	}
	i.environment.Define(s.Name.Lexeme, value)
}

func (i *Interpreter) VisitBlockStmt(s stmt.Block) {
	i.executeBlock(s.Statements, NewEnclosedEnvironment(i.environment))
}

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

func (i *Interpreter) VisitIfStmt(s stmt.If) {
	if i.isTruthy(i.evaluate(s.Condition)) {
		i.execute(s.ThenBranch)
	} else if s.ElseBranch != nil {
		i.execute(s.ElseBranch)
	}
}

func (i *Interpreter) VisitWhileStmt(s stmt.While) {
	for i.isTruthy(i.evaluate(s.Condition)) {
		i.execute(s.Body)
	}
}

func (i *Interpreter) VisitFunctionStmt(s stmt.Function) {
	function := NewLoxFunction(s, i.environment, false)
	i.environment.Define(s.Name.Lexeme, function)
}

func (i *Interpreter) VisitClassStmt(s stmt.Class) {
	var superclass any
	if s.Superclass != nil {
		superclass = i.evaluate(s.Superclass)
		if _, ok := superclass.(*LoxClass); !ok {
			panic(&RuntimeError{Token: s.Superclass.Name, Message: "Superclass must be a class."})
		}
	}

	i.environment.Define(s.Name.Lexeme, nil)

	// If there's a superclass, create a scope with "super" bound to it.
	if s.Superclass != nil {
		i.environment = NewEnclosedEnvironment(i.environment)
		i.environment.Define("super", superclass)
	}

	methods := make(map[string]*LoxFunction)
	for _, method := range s.Methods {
		isInit := method.Name.Lexeme == "init"
		methods[method.Name.Lexeme] = NewLoxFunction(method, i.environment, isInit)
	}

	var sc *LoxClass
	if superclass != nil {
		sc = superclass.(*LoxClass)
	}
	class := &LoxClass{Name: s.Name.Lexeme, Superclass: sc, Methods: methods}

	if s.Superclass != nil {
		i.environment = i.environment.enclosing
	}

	i.environment.Assign(s.Name, class)
}

func (i *Interpreter) VisitReturnStmt(s stmt.Return) {
	var value any
	if s.Value != nil {
		value = i.evaluate(s.Value)
	}
	panic(ReturnValue{Value: value})
}

// Expression visitors.

func (i *Interpreter) VisitCallExpr(e expr.Call) interface{} {
	callee := i.evaluate(e.Callee)

	var arguments []any
	for _, arg := range e.Arguments {
		arguments = append(arguments, i.evaluate(arg))
	}

	function, ok := callee.(LoxCallable)
	if !ok {
		panic(&RuntimeError{Token: e.Paren, Message: "Can only call functions and classes."})
	}

	if len(arguments) != function.Arity() {
		panic(&RuntimeError{
			Token:   e.Paren,
			Message: fmt.Sprintf("Expected %d arguments but got %d.", function.Arity(), len(arguments)),
		})
	}

	return function.Call(i, arguments)
}

func (i *Interpreter) VisitGetExpr(e expr.Get) interface{} {
	object := i.evaluate(e.Object)
	if inst, ok := object.(*LoxInstance); ok {
		return inst.Get(e.Name)
	}
	panic(&RuntimeError{Token: e.Name, Message: "Only instances have properties."})
}

func (i *Interpreter) VisitSetExpr(e expr.Set) interface{} {
	object := i.evaluate(e.Object)
	inst, ok := object.(*LoxInstance)
	if !ok {
		panic(&RuntimeError{Token: e.Name, Message: "Only instances have fields."})
	}
	value := i.evaluate(e.Value)
	inst.Set(e.Name, value)
	return value
}

func (i *Interpreter) VisitThisExpr(e expr.This) interface{} {
	return i.lookUpVariable(e.Keyword, e)
}

func (i *Interpreter) VisitSuperExpr(e expr.Super) interface{} {
	distance := i.locals[e]
	superclass := i.environment.GetAt(distance, "super").(*LoxClass)
	// "this" is always one scope inside "super"
	object := i.environment.GetAt(distance-1, "this").(*LoxInstance)

	method := superclass.FindMethod(e.Method.Lexeme)
	if method == nil {
		panic(&RuntimeError{Token: e.Method, Message: "Undefined property '" + e.Method.Lexeme + "'."})
	}
	return method.Bind(object)
}

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

// VisitLogicalExpr short-circuits: returns the value that determined the result.
func (i *Interpreter) VisitLogicalExpr(e expr.Logical) interface{} {
	left := i.evaluate(e.Left)

	if e.Operator.TokenType == token.TokenType_Or {
		if i.isTruthy(left) {
			return left
		}
	} else {
		if !i.isTruthy(left) {
			return left
		}
	}

	return i.evaluate(e.Right)
}

func (i *Interpreter) VisitVariableExpr(e expr.Variable) interface{} {
	return i.lookUpVariable(e.Name, e)
}

func (i *Interpreter) VisitAssignExpr(e expr.Assign) interface{} {
	value := i.evaluate(e.Value)

	if distance, ok := i.locals[e]; ok {
		i.environment.AssignAt(distance, e.Name.Lexeme, value)
	} else {
		i.globals.Assign(e.Name, value)
	}
	return value
}

// Helpers.

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
		return fmt.Sprintf("%g", num)
	}
	if fn, ok := obj.(*LoxFunction); ok {
		return fn.String()
	}
	if fn, ok := obj.(*NativeFunction); ok {
		return fn.String()
	}
	if c, ok := obj.(*LoxClass); ok {
		return c.String()
	}
	if inst, ok := obj.(*LoxInstance); ok {
		return inst.String()
	}
	return fmt.Sprintf("%v", obj)
}
