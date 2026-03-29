package lox

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunFileReturnsReadError(t *testing.T) {
	vm := NewLox()

	err := vm.RunFile(filepath.Join(t.TempDir(), "missing.lox"))
	if err == nil {
		t.Fatal("RunFile should return an error when the source file does not exist")
	}
	if !os.IsNotExist(err) {
		t.Fatalf("RunFile returned %v, want a not-exist error", err)
	}
}
