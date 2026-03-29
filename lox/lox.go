package lox

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/0xdvc/go-lox/parser"
	"github.com/0xdvc/go-lox/scanner"
	"github.com/0xdvc/go-lox/stmt"
	"github.com/0xdvc/go-lox/token"
)

type Lox struct {
	interpreter     *Interpreter
	hadError        bool
	hadRuntimeError bool
}

func NewLox() *Lox {
	return &Lox{
		interpreter: NewInterpreter(os.Stdout),
	}
}

func (l *Lox) ReportParseError(err parser.ParseError) {
	if err.Token.TokenType == token.TokenType_EOF {
		l.report(err.Token.Line, " at end", err)
	} else {
		l.report(err.Token.Line, " at '"+err.Token.Lexeme+"'", err)
	}
}

func (l *Lox) ReportResolveError(tok *token.Token, message string) {
	if tok.TokenType == token.TokenType_EOF {
		l.report(tok.Line, " at end", errors.New(message))
	} else {
		l.report(tok.Line, " at '"+tok.Lexeme+"'", errors.New(message))
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

func (l *Lox) RunPrompt() error {
	input := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("> ")
		if !input.Scan() {
			return input.Err()
		}
		line := input.Text()
		if line == "" {
			continue
		}
		l.runRepl(line)
		l.hadError = false
		l.hadRuntimeError = false
	}

	return nil
}

// runRepl tries to evaluate bare expressions and auto-print the result.
// Falls back to normal statement execution if the input contains statements.
func (l *Lox) runRepl(source string) {
	// If the line already ends with ; or contains keywords that start
	// statements, run it as normal code.
	if strings.HasSuffix(strings.TrimSpace(source), ";") {
		l.run(source)
		return
	}

	// Try parsing as an expression by appending a semicolon.
	silent := &silentVM{}
	s := scanner.NewScanner(source+";", silent)
	tokens := s.ScanTokens()
	if len(silent.errors) > 0 {
		l.run(source)
		return
	}

	p := parser.NewParser(silent, tokens)
	stmts := p.Parse()
	if len(silent.errors) > 0 || len(stmts) != 1 {
		l.run(source)
		return
	}

	// If it parsed as a single expression statement, auto-print the result.
	if es, ok := stmts[0].(stmt.Expression); ok {
		resolver := NewResolver(l.interpreter, l)
		resolver.Resolve(stmts)
		if l.hadError {
			return
		}

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

		result := l.interpreter.evaluate(es.Expression)
		fmt.Fprintln(l.interpreter.writer, l.interpreter.stringify(result))
		return
	}

	// Not a bare expression, run normally.
	l.run(source)
}

// silentVM swallows errors for speculative parsing.
type silentVM struct {
	errors []string
}

func (s *silentVM) ReportError(_ int, err error) {
	s.errors = append(s.errors, err.Error())
}

func (s *silentVM) ReportParseError(err parser.ParseError) {
	s.errors = append(s.errors, err.Message)
}

func (l *Lox) run(source string) {
	s := scanner.NewScanner(source, l)
	tokens := s.ScanTokens()

	p := parser.NewParser(l, tokens)
	statements := p.Parse()

	if l.hadError {
		return
	}

	resolver := NewResolver(l.interpreter, l)
	resolver.Resolve(statements)

	if l.hadError {
		return
	}

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
