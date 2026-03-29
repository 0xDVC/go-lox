package main

import (
	"fmt"
	"os"

	"github.com/0xdvc/go-lox/lox"
)

func main() {
	vm := lox.NewLox()
	if len(os.Args) > 2 {
		fmt.Fprintln(os.Stderr, "Usage: golox [script]")
		os.Exit(64)
	}
	if len(os.Args) == 2 {
		if err := vm.RunFile(os.Args[1]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	if err := vm.RunPrompt(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
