package scanner

import "errors"

var ErrUnterminatedString = errors.New("unterminated string")

var ErrInvalidNumberLiteral = errors.New("invalid number literal")

var ErrUnexpectedCharacter = errors.New("unexpected character")

var ErrUnterminatedComment = errors.New("multiline comment met EOF")
