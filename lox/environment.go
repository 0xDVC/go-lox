package lox

import (
	"github.com/0xdvc/go-lox/token"
)

// Environment stores variable bindings with a pointer to its enclosing scope.
type Environment struct {
	values    map[string]any
	enclosing *Environment
}

func NewEnvironment() *Environment {
	return &Environment{values: make(map[string]any)}
}

func NewEnclosedEnvironment(enclosing *Environment) *Environment {
	return &Environment{
		values:    make(map[string]any),
		enclosing: enclosing,
	}
}

func (e *Environment) Define(name string, value any) {
	e.values[name] = value
}

// Get walks up the scope chain to find a variable.
func (e *Environment) Get(name *token.Token) any {
	if value, ok := e.values[name.Lexeme]; ok {
		return value
	}
	if e.enclosing != nil {
		return e.enclosing.Get(name)
	}
	panic(&RuntimeError{
		Token:   name,
		Message: "Undefined variable '" + name.Lexeme + "'.",
	})
}

// Assign modifies an existing variable. Does not create new ones.
func (e *Environment) Assign(name *token.Token, value any) {
	if _, ok := e.values[name.Lexeme]; ok {
		e.values[name.Lexeme] = value
		return
	}
	if e.enclosing != nil {
		e.enclosing.Assign(name, value)
		return
	}
	panic(&RuntimeError{
		Token:   name,
		Message: "Undefined variable '" + name.Lexeme + "'.",
	})
}

// ancestor walks exactly `distance` parent links up the chain.
func (e *Environment) ancestor(distance int) *Environment {
	env := e
	for i := 0; i < distance; i++ {
		env = env.enclosing
	}
	return env
}

// GetAt retrieves a variable at a known scope distance (set by the resolver).
func (e *Environment) GetAt(distance int, name string) any {
	return e.ancestor(distance).values[name]
}

// AssignAt sets a variable at a known scope distance.
func (e *Environment) AssignAt(distance int, name string, value any) {
	e.ancestor(distance).values[name] = value
}
