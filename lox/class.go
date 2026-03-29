package lox

import (
	"fmt"

	"github.com/0xdvc/go-lox/token"
)

// LoxClass is callable (creating instances) and holds methods.
type LoxClass struct {
	Name       string
	Superclass *LoxClass
	Methods    map[string]*LoxFunction
}

func (c *LoxClass) Arity() int {
	if init := c.FindMethod("init"); init != nil {
		return init.Arity()
	}
	return 0
}

// Call creates a new instance, runs init() if present, returns the instance.
func (c *LoxClass) Call(interpreter *Interpreter, arguments []any) any {
	instance := &LoxInstance{class: c, fields: make(map[string]any)}
	if init := c.FindMethod("init"); init != nil {
		init.Bind(instance).Call(interpreter, arguments)
	}
	return instance
}

// FindMethod checks this class, then walks the superclass chain.
func (c *LoxClass) FindMethod(name string) *LoxFunction {
	if method, ok := c.Methods[name]; ok {
		return method
	}
	if c.Superclass != nil {
		return c.Superclass.FindMethod(name)
	}
	return nil
}

func (c *LoxClass) String() string {
	return c.Name
}

// LoxInstance holds per-object fields.
type LoxInstance struct {
	class  *LoxClass
	fields map[string]any
}

// Get checks fields first, then methods. Fields shadow methods.
func (i *LoxInstance) Get(name *token.Token) any {
	if val, ok := i.fields[name.Lexeme]; ok {
		return val
	}

	if method := i.class.FindMethod(name.Lexeme); method != nil {
		return method.Bind(i)
	}

	panic(&RuntimeError{Token: name, Message: "Undefined property '" + name.Lexeme + "'."})
}

func (i *LoxInstance) Set(name *token.Token, value any) {
	i.fields[name.Lexeme] = value
}

func (i *LoxInstance) String() string {
	return fmt.Sprintf("%s instance", i.class.Name)
}
