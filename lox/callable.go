package lox

import (
	"fmt"
	"time"

	"github.com/0xdvc/go-lox/stmt"
)

type LoxCallable interface {
	Call(interpreter *Interpreter, arguments []any) any
	Arity() int
}

// LoxFunction wraps a function declaration and its closure environment.
type LoxFunction struct {
	declaration stmt.Function
	closure     *Environment
}

func NewLoxFunction(declaration stmt.Function, closure *Environment) *LoxFunction {
	return &LoxFunction{declaration: declaration, closure: closure}
}

func (f *LoxFunction) Arity() int {
	return len(f.declaration.Params)
}

// Call executes the function body in a new scope enclosed by the closure.
// Catches ReturnValue panics to extract the return value.
func (f *LoxFunction) Call(interpreter *Interpreter, arguments []any) (result any) {
	env := NewEnclosedEnvironment(f.closure)
	for i, param := range f.declaration.Params {
		env.Define(param.Lexeme, arguments[i])
	}

	defer func() {
		if r := recover(); r != nil {
			if rv, ok := r.(ReturnValue); ok {
				result = rv.Value
			} else {
				panic(r)
			}
		}
	}()

	interpreter.executeBlock(f.declaration.Body, env)
	return nil
}

func (f *LoxFunction) String() string {
	return fmt.Sprintf("<fn %s>", f.declaration.Name.Lexeme)
}

// NativeFunction wraps a Go function for use in Lox.
type NativeFunction struct {
	arity int
	name  string
	fn    func(arguments []any) any
}

func (n *NativeFunction) Arity() int {
	return n.arity
}

func (n *NativeFunction) Call(_ *Interpreter, arguments []any) any {
	return n.fn(arguments)
}

func (n *NativeFunction) String() string {
	return fmt.Sprintf("<native fn %s>", n.name)
}

var ClockNative = &NativeFunction{
	arity: 0,
	name:  "clock",
	fn: func(_ []any) any {
		return float64(time.Now().UnixMilli()) / 1000.0
	},
}
