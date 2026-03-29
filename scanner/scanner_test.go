package scanner

import (
	"testing"

	"github.com/0xdvc/go-lox/token"
)

type stubVM struct {
	errors []error
	lines  []int
}

func (s *stubVM) ReportError(line int, err error) {
	s.lines = append(s.lines, line)
	s.errors = append(s.errors, err)
}

func TestScanTokensSkipsComments(t *testing.T) {
	vm := &stubVM{}
	s := NewScanner("// line comment\nvar answer = 42; /* block comment */", vm)

	tokens := s.ScanTokens()
	got := make([]token.TokenType, 0, len(tokens))
	for _, tok := range tokens {
		got = append(got, tok.TokenType)
	}

	want := []token.TokenType{
		token.TokenType_Var,
		token.TokenType_Identifier,
		token.TokenType_Equal,
		token.TokenType_Number,
		token.TokenType_Semicolon,
		token.TokenType_EOF,
	}

	if len(vm.errors) != 0 {
		t.Fatalf("ScanTokens reported unexpected errors: %v", vm.errors)
	}
	if len(got) != len(want) {
		t.Fatalf("ScanTokens returned %d tokens, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("token %d = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestScanTokensReportsUnterminatedComment(t *testing.T) {
	vm := &stubVM{}
	s := NewScanner("/* unterminated", vm)

	_ = s.ScanTokens()

	if len(vm.errors) != 1 {
		t.Fatalf("ScanTokens reported %d errors, want 1", len(vm.errors))
	}
	if vm.errors[0] != ErrUnterminatedComment {
		t.Fatalf("ScanTokens reported %v, want %v", vm.errors[0], ErrUnterminatedComment)
	}
}
