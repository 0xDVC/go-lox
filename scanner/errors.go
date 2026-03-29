package scanner

import "errors"

var (
	ErrUnterminatedString   = errors.New("unterminated string")
	ErrInvalidNumberLiteral = errors.New("invalid number literal")
	ErrUnexpectedCharacter  = errors.New("unexpected character")
	ErrUnterminatedComment  = errors.New("multiline comment met EOF")
)
