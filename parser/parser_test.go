package parser

import (
	"testing"

	"github.com/0xdvc/go-lox/expr"
	"github.com/0xdvc/go-lox/scanner"
	"github.com/0xdvc/go-lox/stmt"
	"github.com/0xdvc/go-lox/token"
)

type testVM struct {
	errors []string
}

func (t *testVM) ReportError(line int, err error) {
	t.errors = append(t.errors, err.Error())
}

func (t *testVM) ReportParseError(err ParseError) {
	t.errors = append(t.errors, err.Message)
}

func parse(source string) ([]stmt.Stmt, []string) {
	vm := &testVM{}
	s := scanner.NewScanner(source, vm)
	tokens := s.ScanTokens()
	p := NewParser(vm, tokens)
	stmts := p.Parse()
	return stmts, vm.errors
}

func TestParsePrint(t *testing.T) {
	stmts, errs := parse(`print 42;`)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(stmts) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(stmts))
	}
	ps, ok := stmts[0].(stmt.Print)
	if !ok {
		t.Fatalf("expected Print statement, got %T", stmts[0])
	}
	lit, ok := ps.Expression.(expr.Literal)
	if !ok {
		t.Fatalf("expected Literal expression, got %T", ps.Expression)
	}
	if lit.Value != float64(42) {
		t.Errorf("expected 42, got %v", lit.Value)
	}
}

func TestParseVarDeclaration(t *testing.T) {
	stmts, errs := parse(`var x = 10;`)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	vs, ok := stmts[0].(stmt.Var)
	if !ok {
		t.Fatalf("expected Var statement, got %T", stmts[0])
	}
	if vs.Name.Lexeme != "x" {
		t.Errorf("expected var name 'x', got %q", vs.Name.Lexeme)
	}
}

func TestParseVarNoInit(t *testing.T) {
	stmts, errs := parse(`var x;`)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	vs := stmts[0].(stmt.Var)
	if vs.Initializer != nil {
		t.Error("expected nil initializer")
	}
}

func TestParseBinaryExpr(t *testing.T) {
	stmts, _ := parse(`print 1 + 2 * 3;`)
	ps := stmts[0].(stmt.Print)
	// Should be Binary(1, +, Binary(2, *, 3)) due to precedence
	bin, ok := ps.Expression.(expr.Binary)
	if !ok {
		t.Fatalf("expected Binary, got %T", ps.Expression)
	}
	if bin.Operator.TokenType != token.TokenType_Plus {
		t.Errorf("expected +, got %v", bin.Operator.Lexeme)
	}
	// Right side should be Binary(2, *, 3)
	right, ok := bin.Right.(expr.Binary)
	if !ok {
		t.Fatalf("expected right Binary, got %T", bin.Right)
	}
	if right.Operator.TokenType != token.TokenType_Star {
		t.Errorf("expected *, got %v", right.Operator.Lexeme)
	}
}

func TestParseGrouping(t *testing.T) {
	stmts, _ := parse(`print (1 + 2) * 3;`)
	ps := stmts[0].(stmt.Print)
	bin := ps.Expression.(expr.Binary)
	if bin.Operator.TokenType != token.TokenType_Star {
		t.Errorf("expected * at top level, got %v", bin.Operator.Lexeme)
	}
	_, ok := bin.Left.(expr.Grouping)
	if !ok {
		t.Errorf("expected Grouping on left, got %T", bin.Left)
	}
}

func TestParseIfElse(t *testing.T) {
	stmts, errs := parse(`if (true) print "yes"; else print "no";`)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	ifStmt, ok := stmts[0].(stmt.If)
	if !ok {
		t.Fatalf("expected If, got %T", stmts[0])
	}
	if ifStmt.ElseBranch == nil {
		t.Error("expected else branch")
	}
}

func TestParseWhile(t *testing.T) {
	stmts, _ := parse(`while (true) print "loop";`)
	_, ok := stmts[0].(stmt.While)
	if !ok {
		t.Fatalf("expected While, got %T", stmts[0])
	}
}

func TestParseForDesugars(t *testing.T) {
	stmts, _ := parse(`for (var i = 0; i < 10; i = i + 1) print i;`)
	// For desugars to Block containing Var + While
	block, ok := stmts[0].(stmt.Block)
	if !ok {
		t.Fatalf("expected Block (desugared for), got %T", stmts[0])
	}
	if len(block.Statements) != 2 {
		t.Fatalf("expected 2 statements in block, got %d", len(block.Statements))
	}
	_, ok = block.Statements[0].(stmt.Var)
	if !ok {
		t.Errorf("expected Var as first stmt, got %T", block.Statements[0])
	}
	_, ok = block.Statements[1].(stmt.While)
	if !ok {
		t.Errorf("expected While as second stmt, got %T", block.Statements[1])
	}
}

func TestParseFunctionDecl(t *testing.T) {
	stmts, _ := parse(`fun add(a, b) { return a + b; }`)
	fn, ok := stmts[0].(stmt.Function)
	if !ok {
		t.Fatalf("expected Function, got %T", stmts[0])
	}
	if fn.Name.Lexeme != "add" {
		t.Errorf("expected name 'add', got %q", fn.Name.Lexeme)
	}
	if len(fn.Params) != 2 {
		t.Errorf("expected 2 params, got %d", len(fn.Params))
	}
}

func TestParseClass(t *testing.T) {
	stmts, _ := parse(`class Dog < Animal { speak() { print "woof"; } }`)
	cls, ok := stmts[0].(stmt.Class)
	if !ok {
		t.Fatalf("expected Class, got %T", stmts[0])
	}
	if cls.Name.Lexeme != "Dog" {
		t.Errorf("expected 'Dog', got %q", cls.Name.Lexeme)
	}
	if cls.Superclass == nil || cls.Superclass.Name.Lexeme != "Animal" {
		t.Error("expected superclass 'Animal'")
	}
	if len(cls.Methods) != 1 {
		t.Errorf("expected 1 method, got %d", len(cls.Methods))
	}
}

func TestParseUnary(t *testing.T) {
	stmts, _ := parse(`print -5;`)
	ps := stmts[0].(stmt.Print)
	u, ok := ps.Expression.(expr.Unary)
	if !ok {
		t.Fatalf("expected Unary, got %T", ps.Expression)
	}
	if u.Operator.TokenType != token.TokenType_Minus {
		t.Errorf("expected -, got %v", u.Operator.Lexeme)
	}
}

func TestParseLogical(t *testing.T) {
	stmts, _ := parse(`print true or false;`)
	ps := stmts[0].(stmt.Print)
	l, ok := ps.Expression.(expr.Logical)
	if !ok {
		t.Fatalf("expected Logical, got %T", ps.Expression)
	}
	if l.Operator.TokenType != token.TokenType_Or {
		t.Errorf("expected 'or', got %v", l.Operator.Lexeme)
	}
}

func TestParseAssignment(t *testing.T) {
	stmts, _ := parse(`var x = 1; x = 2;`)
	es := stmts[1].(stmt.Expression)
	a, ok := es.Expression.(expr.Assign)
	if !ok {
		t.Fatalf("expected Assign, got %T", es.Expression)
	}
	if a.Name.Lexeme != "x" {
		t.Errorf("expected 'x', got %q", a.Name.Lexeme)
	}
}

func TestParseErrorRecovery(t *testing.T) {
	// Missing semicolons, parser should recover and keep going
	stmts, errs := parse(`print 1; print; print 3;`)
	if len(errs) == 0 {
		t.Fatal("expected parse error")
	}
	// Should still parse some valid statements
	if len(stmts) < 1 {
		t.Fatal("expected parser to recover some statements")
	}
}

func TestParseReturn(t *testing.T) {
	stmts, _ := parse(`fun f() { return 42; }`)
	fn := stmts[0].(stmt.Function)
	ret, ok := fn.Body[0].(stmt.Return)
	if !ok {
		t.Fatalf("expected Return, got %T", fn.Body[0])
	}
	if ret.Value == nil {
		t.Error("expected return value")
	}
}

func TestParseBlock(t *testing.T) {
	stmts, _ := parse(`{ var x = 1; print x; }`)
	b, ok := stmts[0].(stmt.Block)
	if !ok {
		t.Fatalf("expected Block, got %T", stmts[0])
	}
	if len(b.Statements) != 2 {
		t.Errorf("expected 2 statements in block, got %d", len(b.Statements))
	}
}

func TestParseCallExpr(t *testing.T) {
	stmts, _ := parse(`print foo(1, 2);`)
	ps := stmts[0].(stmt.Print)
	c, ok := ps.Expression.(expr.Call)
	if !ok {
		t.Fatalf("expected Call, got %T", ps.Expression)
	}
	if len(c.Arguments) != 2 {
		t.Errorf("expected 2 args, got %d", len(c.Arguments))
	}
}

func TestParsePropertyAccess(t *testing.T) {
	stmts, _ := parse(`print obj.field;`)
	ps := stmts[0].(stmt.Print)
	g, ok := ps.Expression.(expr.Get)
	if !ok {
		t.Fatalf("expected Get, got %T", ps.Expression)
	}
	if g.Name.Lexeme != "field" {
		t.Errorf("expected 'field', got %q", g.Name.Lexeme)
	}
}

func TestParsePropertySet(t *testing.T) {
	stmts, _ := parse(`obj.x = 5;`)
	es := stmts[0].(stmt.Expression)
	s, ok := es.Expression.(expr.Set)
	if !ok {
		t.Fatalf("expected Set, got %T", es.Expression)
	}
	if s.Name.Lexeme != "x" {
		t.Errorf("expected 'x', got %q", s.Name.Lexeme)
	}
}

func TestParseSuperExpr(t *testing.T) {
	stmts, _ := parse(`class B < A { method() { super.foo(); } }`)
	cls := stmts[0].(stmt.Class)
	fn := cls.Methods[0]
	es := fn.Body[0].(stmt.Expression)
	call := es.Expression.(expr.Call)
	_, ok := call.Callee.(expr.Super)
	if !ok {
		t.Fatalf("expected Super, got %T", call.Callee)
	}
}

func TestParseThisExpr(t *testing.T) {
	stmts, _ := parse(`class A { method() { print this; } }`)
	cls := stmts[0].(stmt.Class)
	fn := cls.Methods[0]
	ps := fn.Body[0].(stmt.Print)
	_, ok := ps.Expression.(expr.This)
	if !ok {
		t.Fatalf("expected This, got %T", ps.Expression)
	}
}
