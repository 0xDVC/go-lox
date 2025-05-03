package main

import "fmt"

type Token struct {
    TokenType TokenType
    Lexeme string
    Object any
    Line int
}

func NewToken(tokenType TokenType, lexeme string, literal any, line int) {
    return Token{tokenType, lexem, literal, line}
}

funct (t *Token) String() string {
    return fmt.Sprintf("%s %s %v", t.TokenType, t.Lexeme, t.Literal)
}
