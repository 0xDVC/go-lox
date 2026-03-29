package lox

import (
	"bytes"
	"strings"
	"testing"

	"github.com/0xdvc/go-lox/parser"
	"github.com/0xdvc/go-lox/scanner"
	"github.com/0xdvc/go-lox/token"
)

// testVM captures errors instead of printing to stderr.
type testVM struct {
	errors []string
}

func (t *testVM) ReportError(line int, err error) {
	t.errors = append(t.errors, err.Error())
}

func (t *testVM) ReportParseError(err parser.ParseError) {
	t.errors = append(t.errors, err.Message)
}

func (t *testVM) ReportResolveError(tok *token.Token, message string) {
	t.errors = append(t.errors, message)
}

// runLox executes source code and returns captured output + any errors.
func runLox(source string) (output string, errors []string, runtimeErr string) {
	vm := &testVM{}
	var buf bytes.Buffer
	interp := NewInterpreter(&buf)

	s := scanner.NewScanner(source, vm)
	tokens := s.ScanTokens()
	if len(vm.errors) > 0 {
		return "", vm.errors, ""
	}

	p := parser.NewParser(vm, tokens)
	stmts := p.Parse()
	if len(vm.errors) > 0 {
		return "", vm.errors, ""
	}

	resolver := NewResolver(interp, vm)
	resolver.Resolve(stmts)
	if len(vm.errors) > 0 {
		return "", vm.errors, ""
	}

	defer func() {
		if r := recover(); r != nil {
			if re, ok := r.(*RuntimeError); ok {
				runtimeErr = re.Message
			} else {
				panic(r)
			}
		}
	}()

	interp.Interpret(stmts)
	return buf.String(), nil, ""
}

// assertOutput runs source and checks output lines match expected.
func assertOutput(t *testing.T, name, source string, expected []string) {
	t.Helper()
	out, errs, rtErr := runLox(source)
	if len(errs) > 0 {
		t.Fatalf("%s: unexpected errors: %v", name, errs)
	}
	if rtErr != "" {
		t.Fatalf("%s: unexpected runtime error: %s", name, rtErr)
	}
	got := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(got) != len(expected) {
		t.Fatalf("%s: expected %d lines, got %d\nExpected: %v\nGot:      %v", name, len(expected), len(got), expected, got)
	}
	for i, exp := range expected {
		if got[i] != exp {
			t.Errorf("%s: line %d: expected %q, got %q", name, i+1, exp, got[i])
		}
	}
}

// Phase 1: Statements & State

func TestPrintLiteral(t *testing.T) {
	assertOutput(t, "print literal", `print "hello";`, []string{"hello"})
}

func TestVariableDeclaration(t *testing.T) {
	assertOutput(t, "var decl", `var x = 10; print x;`, []string{"10"})
}

func TestVariableAssignment(t *testing.T) {
	assertOutput(t, "var assign", `var x = 1; x = 42; print x;`, []string{"42"})
}

func TestBlockScoping(t *testing.T) {
	assertOutput(t, "block scope", `
var a = "outer";
{
  var a = "inner";
  print a;
}
print a;
`, []string{"inner", "outer"})
}

func TestNilDefault(t *testing.T) {
	assertOutput(t, "nil default", `var x; print x;`, []string{"nil"})
}

// Phase 2: Control Flow

func TestIfTrue(t *testing.T) {
	assertOutput(t, "if true", `if (true) print "yes";`, []string{"yes"})
}

func TestIfElse(t *testing.T) {
	assertOutput(t, "if else", `if (false) print "no"; else print "yes";`, []string{"yes"})
}

func TestWhileLoop(t *testing.T) {
	assertOutput(t, "while", `
var i = 0;
while (i < 3) { i = i + 1; }
print i;
`, []string{"3"})
}

func TestForLoop(t *testing.T) {
	assertOutput(t, "for", `
var sum = 0;
for (var i = 1; i <= 5; i = i + 1) { sum = sum + i; }
print sum;
`, []string{"15"})
}

func TestLogicalOr(t *testing.T) {
	assertOutput(t, "or", `print nil or "default";`, []string{"default"})
}

func TestLogicalAnd(t *testing.T) {
	assertOutput(t, "and", `print "first" and "second";`, []string{"second"})
}

func TestModulo(t *testing.T) {
	assertOutput(t, "modulo", `print 17 % 5;`, []string{"2"})
}

func TestFizzBuzz(t *testing.T) {
	assertOutput(t, "fizzbuzz", `
for (var i = 1; i <= 15; i = i + 1) {
  if (i % 15 == 0) print "FizzBuzz";
  else if (i % 3 == 0) print "Fizz";
  else if (i % 5 == 0) print "Buzz";
  else print i;
}
`, []string{
		"1", "2", "Fizz", "4", "Buzz",
		"Fizz", "7", "8", "Fizz", "Buzz",
		"11", "Fizz", "13", "14", "FizzBuzz",
	})
}

// Phase 3: Functions

func TestFunctionCall(t *testing.T) {
	assertOutput(t, "fn call", `
fun add(a, b) { return a + b; }
print add(2, 3);
`, []string{"5"})
}

func TestRecursion(t *testing.T) {
	assertOutput(t, "fib", `
fun fib(n) {
  if (n <= 1) return n;
  return fib(n - 1) + fib(n - 2);
}
print fib(10);
`, []string{"55"})
}

func TestClosure(t *testing.T) {
	assertOutput(t, "closure", `
fun makeCounter() {
  var count = 0;
  fun increment() {
    count = count + 1;
    return count;
  }
  return increment;
}
var c = makeCounter();
print c();
print c();
print c();
`, []string{"1", "2", "3"})
}

func TestNoReturnIsNil(t *testing.T) {
	assertOutput(t, "no return", `
fun nothing() { var x = 1; }
print nothing();
`, []string{"nil"})
}

func TestFirstClassFunction(t *testing.T) {
	assertOutput(t, "first class", `
fun square(x) { return x * x; }
fun apply(f, x) { return f(x); }
print apply(square, 5);
`, []string{"25"})
}

// Phase 4: Resolving & Binding

func TestClosureScopeFix(t *testing.T) {
	assertOutput(t, "closure scope", `
var a = "global";
{
  fun showA() { print a; }
  showA();
  var a = "block";
  showA();
}
`, []string{"global", "global"})
}

// Phase 5: Classes & Inheritance

func TestBasicClass(t *testing.T) {
	assertOutput(t, "basic class", `
class Cake {
  taste() { return "delicious"; }
}
var c = Cake();
print c.taste();
`, []string{"delicious"})
}

func TestClassInit(t *testing.T) {
	assertOutput(t, "init", `
class Point {
  init(x, y) {
    this.x = x;
    this.y = y;
  }
}
var p = Point(3, 4);
print p.x;
print p.y;
`, []string{"3", "4"})
}

func TestInheritance(t *testing.T) {
	assertOutput(t, "inheritance", `
class Animal {
  init(name) { this.name = name; }
  speak() { print this.name + " speaks."; }
}
class Dog < Animal {
  speak() { print this.name + " barks!"; }
}
var d = Dog("Rex");
d.speak();
`, []string{"Rex barks!"})
}

func TestSuper(t *testing.T) {
	assertOutput(t, "super", `
class A {
  method() { print "A"; }
}
class B < A {
  method() {
    super.method();
    print "B";
  }
}
B().method();
`, []string{"A", "B"})
}

func TestFieldShadowsMethod(t *testing.T) {
	assertOutput(t, "field shadow", `
class Obj {
  method() { return "method"; }
}
var o = Obj();
print o.method();
o.method = "field";
print o.method;
`, []string{"method", "field"})
}

func TestMethodCallback(t *testing.T) {
	assertOutput(t, "callback", `
class Greeter {
  init(name) { this.name = name; }
  greet() { print "Hi, " + this.name; }
}
var g = Greeter("World");
var fn = g.greet;
fn();
`, []string{"Hi, World"})
}

// Error detection tests

func TestUndefinedVariable(t *testing.T) {
	_, _, rtErr := runLox(`print x;`)
	if rtErr == "" {
		t.Fatal("expected runtime error for undefined variable")
	}
}

func TestReturnOutsideFunction(t *testing.T) {
	_, errs, _ := runLox(`return 1;`)
	if len(errs) == 0 {
		t.Fatal("expected resolve error for return outside function")
	}
}

func TestThisOutsideClass(t *testing.T) {
	_, errs, _ := runLox(`print this;`)
	if len(errs) == 0 {
		t.Fatal("expected resolve error for this outside class")
	}
}

func TestSuperOutsideClass(t *testing.T) {
	_, errs, _ := runLox(`class A { method() { super.foo(); } }`)
	if len(errs) == 0 {
		t.Fatal("expected resolve error for super without superclass")
	}
}

func TestInheritFromSelf(t *testing.T) {
	_, errs, _ := runLox(`class A < A {}`)
	if len(errs) == 0 {
		t.Fatal("expected resolve error for inherit from self")
	}
}

func TestVarInOwnInitializer(t *testing.T) {
	_, errs, _ := runLox(`{ var a = a; }`)
	if len(errs) == 0 {
		t.Fatal("expected resolve error for var in own initializer")
	}
}

func TestDivisionByZero(t *testing.T) {
	_, _, rtErr := runLox(`print 1 / 0;`)
	if rtErr == "" {
		t.Fatal("expected runtime error for division by zero")
	}
}

func TestWrongArity(t *testing.T) {
	_, _, rtErr := runLox(`fun f(a, b) {} f(1);`)
	if rtErr == "" {
		t.Fatal("expected runtime error for wrong arity")
	}
}

func TestCallNonFunction(t *testing.T) {
	_, _, rtErr := runLox(`var x = "hi"; x();`)
	if rtErr == "" {
		t.Fatal("expected runtime error for calling non-function")
	}
}

func TestReturnValueFromInit(t *testing.T) {
	_, errs, _ := runLox(`
class Foo {
  init() { return "bad"; }
}
`)
	if len(errs) == 0 {
		t.Fatal("expected resolve error for return value from init")
	}
}
