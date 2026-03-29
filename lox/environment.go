package lox

import (
	"github.com/0xdvc/go-lox/token"
)

// Environment stores variable bindings. Each environment has an optional
// pointer to its enclosing (parent) environment, forming a chain.
// Variable lookup walks up this chain — that's lexical scoping.
type Environment struct {
	values    map[string]any
	enclosing *Environment
}

// NewEnvironment creates a global environment (no parent).
func NewEnvironment() *Environment {
	return &Environment{values: make(map[string]any)}
}

// NewEnclosedEnvironment creates a child environment.
// Used when entering a block { } — the new scope can see the outer scope.
func NewEnclosedEnvironment(enclosing *Environment) *Environment {
	return &Environment{
		values:    make(map[string]any),
		enclosing: enclosing,
	}
}

// Define creates a new variable in the current scope.
// If it already exists in this scope, it overwrites it.
// This is intentional — Lox allows `var x = 1; var x = 2;` in the REPL.
func (e *Environment) Define(name string, value any) {
	e.values[name] = value
}

// Get looks up a variable by name. If not found in this scope,
// walks up the enclosing chain. If not found anywhere, runtime error.
func (e *Environment) Get(name *token.Token) any {
	// Check current scope first
	if value, ok := e.values[name.Lexeme]; ok {
		return value
	}

	// Walk up to the parent scope
	if e.enclosing != nil {
		return e.enclosing.Get(name)
	}

	// Nowhere left to look — undefined variable
	panic(&RuntimeError{
		Token:   name,
		Message: "Undefined variable '" + name.Lexeme + "'.",
	})
}

// Assign modifies an existing variable. Unlike Define, it does NOT create
// a new variable — if the name doesn't exist anywhere in the chain, that's an error.
// This prevents typos from silently creating new variables.
func (e *Environment) Assign(name *token.Token, value any) {
	// Check current scope
	if _, ok := e.values[name.Lexeme]; ok {
		e.values[name.Lexeme] = value
		return
	}

	// Walk up to the parent scope
	if e.enclosing != nil {
		e.enclosing.Assign(name, value)
		return
	}

	// Undefined — can't assign to something that doesn't exist
	panic(&RuntimeError{
		Token:   name,
		Message: "Undefined variable '" + name.Lexeme + "'.",
	})
}
