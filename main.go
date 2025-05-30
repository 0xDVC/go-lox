package main

import (
	"fmt"
	"os"

	"github.com/0xdvc/go-lox/lox"
)

func main() {
	vm := lox.Lox{}
	if len(os.Args) > 2 {
		fmt.Println("Usage: golox [script]")
		os.Exit(64)
	}
	if len(os.Args) == 2 {
		vm.RunFile(os.Args[1])
	} else {
		vm.RunPrompt()
	}
}
