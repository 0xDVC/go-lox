package lox

import (
	"bufio"
	"fmt"
	"os"

	"github.com/0xdvc/go-lox/parser"
	"github.com/0xdvc/go-lox/scanner"
	"github.com/0xdvc/go-lox/token"
)

type Lox struct {
	interpreter     *Interpreter
	hadError        bool
	hadRuntimeError bool
}

// NewLox creates a new Lox VM with a persistent interpreter.
func NewLox() *Lox {
	return &Lox{
		interpreter: NewInterpreter(),
	}
}

func (l *Lox) ReportParseError(err parser.ParseError) {
	if err.Token.TokenType == token.TokenType_EOF {
		l.report(err.Token.Line, " at end", err)
	} else {
		l.report(err.Token.Line, " at '"+err.Token.Lexeme+"'", err)
	}
}

func (l *Lox) ReportError(line int, err error) {
	l.report(line, "", err)
}

func (l *Lox) RunFile(path string) error {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	l.run(string(bytes))
	if l.hadError {
		os.Exit(65)
	}
	if l.hadRuntimeError {
		os.Exit(70)
	}
	return nil
}

func (l *Lox) RunPrompt() {
	input := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("> ")
		if !input.Scan() {
			break
		}
		line := input.Text()
		l.run(line)
		// Reset error flags between REPL lines — one bad line shouldn't kill the session
		l.hadError = false
		l.hadRuntimeError = false
	}
}

func (l *Lox) run(source string) {
	s := scanner.NewScanner(source, l)
	tokens := s.ScanTokens()

	p := parser.NewParser(l, tokens)
	statements := p.Parse()

	// Don't interpret if there were parse errors
	if l.hadError {
		return
	}

	// Catch runtime errors (panics) from the interpreter
	defer func() {
		if r := recover(); r != nil {
			if re, ok := r.(*RuntimeError); ok {
				l.ReportRuntimeError(re)
				l.hadRuntimeError = true
			} else {
				fmt.Println("Unexpected panic:", r)
			}
		}
	}()

	l.interpreter.Interpret(statements)
}

func (l *Lox) report(line int, where string, err error) {
	fmt.Printf("[line %d] Error%s: %s\n", line, where, err)
	l.hadError = true
}

func (l *Lox) ReportRuntimeError(err *RuntimeError) {
	fmt.Printf("[line %d] Runtime Error: %s\n", err.Token.Line, err.Message)
	l.hadRuntimeError = true
}
