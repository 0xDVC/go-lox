package scanner

import (
	"github.com/0xdvc/go-lox/token"
)

var keywords map[string]token.TokenType = map[string]token.TokenType{
	"and":    token.TokenType_And,
	"class":  token.TokenType_Class,
	"false":  token.TokenType_False,
	"true":   token.TokenType_True,
	"while":  token.TokenType_While,
	"for":    token.TokenType_For,
	"or":     token.TokenType_Or,
	"fun":    token.TokenType_Fun,
	"if":     token.TokenType_If,
	"nil":    token.TokenType_Nil,
	"var":    token.TokenType_Var,
	"print":  token.TokenType_Print,
	"super":  token.TokenType_Super,
	"return": token.TokenType_Return,
	"else":   token.TokenType_Else,
	"this":   token.TokenType_This,
}
