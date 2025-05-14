package lox

import (
	"bufio"
	"fmt"
	"github.com/0xdvc/go-lox/scanner"
	"os"
)

type Lox struct {
	hadError bool
}

var VM = Lox{}

func (l *Lox) RunFile(path string) error {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	l.run(string(bytes))
	if l.hadError {
		os.Exit(65)
	}
	return nil
}

func (l *Lox) RunPrompt() {
	input := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("> ")
		if ok := input.Scan(); !ok {
			break
		}
		line := input.Text()
		l.run(line)
		l.hadError = false
	}
}

func (l *Lox) run(source string) {
	scanner := scanner.NewScanner(source)
	scanner.ScanTokens()

	for _, token := range scanner.Tokens {
		fmt.Println(token)
	}
}

func (l *Lox) ReportError(line int, err error) {
	l.report(line, "", err)
}

func (l *Lox) report(line int, where string, err error) {
	fmt.Printf("[line %d] Error %s: %s\n", line, where, err)
	l.hadError = true
}
