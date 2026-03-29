package scanner

import (
	"strconv"

	"github.com/0xdvc/go-lox/token"
)

type Scanner struct {
	source []rune
	Tokens []*token.Token
	vm     vm

	start   int
	current int
	line    int
}

type vm interface {
	ReportError(line int, err error)
}

func NewScanner(source string, vm vm) *Scanner {
	return &Scanner{
		source: []rune(source),
		Tokens: make([]*token.Token, 0),
		vm:     vm,
		line:   1,
	}
}

func (s *Scanner) ScanTokens() []*token.Token {
	for !s.isAtEnd() {
		s.start = s.current
		s.scanToken()
	}

	s.Tokens = append(s.Tokens, &token.Token{
		TokenType: token.TokenType_EOF,
		Lexeme:    "",
		Literal:   nil,
		Line:      s.line,
	})
	return s.Tokens
}

func (s *Scanner) isAtEnd() bool {
	return s.current >= len(s.source)
}

func (s *Scanner) scanToken() {
	c := s.advance()

	switch c {
	case '(':
		s.addToken(token.TokenType_LeftParen, string(c))
	case ')':
		s.addToken(token.TokenType_RightParen, string(c))
	case '{':
		s.addToken(token.TokenType_LeftBrace, string(c))
	case '}':
		s.addToken(token.TokenType_RightBrace, string(c))
	case ',':
		s.addToken(token.TokenType_Comma, string(c))
	case '.':
		s.addToken(token.TokenType_Dot, string(c))
	case '-':
		s.addToken(token.TokenType_Minus, string(c))
	case '+':
		s.addToken(token.TokenType_Plus, string(c))
	case ';':
		s.addToken(token.TokenType_Semicolon, string(c))
	case '*':
		s.addToken(token.TokenType_Star, string(c))
	case '%':
		s.addToken(token.TokenType_Percent, string(c))
	case '!':
		if s.match('=') {
			s.addToken(token.TokenType_BangEqual, "!=")
		} else {
			s.addToken(token.TokenType_Bang, string(c))
		}
	case '=':
		if s.match('=') {
			s.addToken(token.TokenType_EqualEqual, "==")
		} else {
			s.addToken(token.TokenType_Equal, string(c))
		}
	case '<':
		if s.match('=') {
			s.addToken(token.TokenType_LessEqual, "<=")
		} else {
			s.addToken(token.TokenType_Less, string(c))
		}
	case '>':
		if s.match('=') {
			s.addToken(token.TokenType_GreaterEqual, ">=")
		} else {
			s.addToken(token.TokenType_Greater, string(c))
		}
	case '/':
		if s.match('/') {
			s.consumeLineComment()
			break
		}
		if s.match('*') {
			s.consumeMultiLineComment()
			break
		}
		s.addToken(token.TokenType_Slash, string(c))
	case ' ', '\r', '\t':
		break
	case '\n':
		s.line++
	case '"':
		s.scanString()
	default:
		if isDigit(c) {
			s.scanNumber()
		} else if isAlpha(c) {
			s.scanIdentifier()
		} else {
			if s.vm != nil {
				s.vm.ReportError(s.line, ErrUnexpectedCharacter)
			}
		}
	}
}

func (s *Scanner) consumeLineComment() {
	for s.peek() != '\n' && !s.isAtEnd() {
		s.advance()
	}
}

func (s *Scanner) consumeMultiLineComment() {
	for !s.isAtEnd() {
		switch c := s.peek(); c {
		case '\n':
			s.line++
			s.advance()
		case '*':
			if s.peekNext() == '/' {
				s.current += 2
				return
			}
		default:
			s.advance()
		}
	}
	if s.vm != nil {
		s.vm.ReportError(s.line, ErrUnterminatedComment)
	}
}

func (s *Scanner) scanIdentifier() {
	for isAlphaNumeric(s.peek()) {
		s.advance()
	}

	text := string(s.source[s.start:s.current])
	if t, ok := keywords[text]; ok {
		s.addToken(t, text)
		return
	}
	s.addToken(token.TokenType_Identifier, text)
}

func (s *Scanner) advance() rune {
	if s.current < len(s.source) {
		result := s.source[s.current]
		s.current++
		return result
	}
	return 0
}

func (s *Scanner) addToken(tokenType token.TokenType, lexeme string) {
	s.addTokenWithLiteral(tokenType, lexeme, nil)
}

func (s *Scanner) addTokenWithLiteral(tokenType token.TokenType, lexeme string, literal any) {
	s.Tokens = append(s.Tokens, &token.Token{
		TokenType: tokenType,
		Lexeme:    lexeme,
		Literal:   literal,
		Line:      s.line,
	})
}

func (s *Scanner) match(expected rune) bool {
	if s.isAtEnd() {
		return false
	}
	if s.source[s.current] != expected {
		return false
	}
	s.current++
	return true
}

func (s *Scanner) peek() rune {
	if s.isAtEnd() {
		return 0
	}
	return s.source[s.current]
}

func (s *Scanner) scanString() {
	for s.peek() != '"' && !s.isAtEnd() {
		if s.peek() == '\n' {
			s.line++
		}
		s.advance()
	}

	if s.isAtEnd() {
		if s.vm != nil {
			s.vm.ReportError(s.line, ErrUnterminatedString)
		}
		return
	}

	s.advance()
	value := string(s.source[s.start+1 : s.current-1])
	s.addTokenWithLiteral(token.TokenType_String, value, value)
}

func isDigit(c rune) bool {
	return c >= '0' && c <= '9'
}

func isAlpha(c rune) bool {
	return c >= 'a' && c <= 'z' ||
		c >= 'A' && c <= 'Z' ||
		c == '_'
}

func isAlphaNumeric(c rune) bool {
	return isAlpha(c) || isDigit(c)
}

func (s *Scanner) scanNumber() {
	for isDigit(s.peek()) {
		s.advance()
	}

	if s.peek() == '.' && isDigit(s.peekNext()) {
		s.advance()
		for isDigit(s.peek()) {
			s.advance()
		}
	}

	text := string(s.source[s.start:s.current])
	num, err := strconv.ParseFloat(text, 64)
	if err != nil {
		if s.vm != nil {
			s.vm.ReportError(s.line, ErrInvalidNumberLiteral)
		}
		return
	}
	s.addTokenWithLiteral(token.TokenType_Number, text, num)
}

func (s *Scanner) peekNext() rune {
	if s.current+1 >= len(s.source) {
		return 0
	}
	return s.source[s.current+1]
}
