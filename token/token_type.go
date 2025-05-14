//go:generate stringer -type=TokenType -trimprefix=TokenType_

package token

type TokenType int

const (
	// Single-character tokens.
	TokenType_LeftParen TokenType = iota
	TokenType_RightParen
	TokenType_LeftBrace
	TokenType_RightBrace
	TokenType_Comma
	TokenType_Dot
	TokenType_Minus
	TokenType_Plus
	TokenType_Semicolon
	TokenType_Slash
	TokenType_Star

	// One or two character tokens.
	TokenType_Bang
	TokenType_BangEqual
	TokenType_Equal
	TokenType_EqualEqual
	TokenType_Greater
	TokenType_GreaterEqual
	TokenType_Less
	TokenType_LessEqual

	// Literals.
	TokenType_Identifier
	TokenType_String
	TokenType_Number

	// Keywords.
	TokenType_And
	TokenType_Class
	TokenType_Else
	TokenType_False
	TokenType_Fun
	TokenType_For
	TokenType_If
	TokenType_Nil
	TokenType_Or
	TokenType_Print
	TokenType_Return
	TokenType_Super
	TokenType_This
	TokenType_True
	TokenType_Var
	TokenType_While

	TokenType_EOF
)

var keywords map[string]TokenType = map[string]TokenType{
	"and":    TokenType_And,
	"false":  TokenType_False,
	"or":     TokenType_Or,
	"class":  TokenType_Class,
	"else":   TokenType_Else,
	"if":     TokenType_If,
	"for":    TokenType_For,
	"nil":    TokenType_Nil,
	"print":  TokenType_Print,
	"return": TokenType_Return,
	"super":  TokenType_Super,
	"this":   TokenType_This,
	"true":   TokenType_True,
	"var":    TokenType_Var,
	"while":  TokenType_While,
	"fun":    TokenType_Fun,
}
