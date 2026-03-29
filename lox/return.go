package lox

// ReturnValue is used as a panic value to unwind the call stack
// when a `return` statement executes. LoxFunction.Call() recovers it.
type ReturnValue struct {
	Value any
}
