package parser

import (
	"github.com/0xdvc/go-lox/expr"
	"github.com/0xdvc/go-lox/stmt"
	"github.com/0xdvc/go-lox/token"
)

type Parser struct {
	vm      vm
	tokens  []*token.Token
	current int
}

type vm interface {
	ReportParseError(err ParseError)
}

type ParseError struct {
	Token   *token.Token
	Message string
}

func (e ParseError) Error() string {
	return e.Message
}

func NewParser(vm vm, tokens []*token.Token) *Parser {
	return &Parser{vm: vm, tokens: tokens}
}

// Parse returns a list of statements — a program is just a sequence of declarations.
func (p *Parser) Parse() []stmt.Stmt {
	var statements []stmt.Stmt
	for !p.isAtEnd() {
		s := p.declaration()
		if s != nil {
			statements = append(statements, s)
		}
	}
	return statements
}

// declaration → varDecl | statement
// This is the top-level rule. If parsing fails, we synchronize and keep going.
func (p *Parser) declaration() stmt.Stmt {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(ParseError); ok {
				p.synchronize()
			} else {
				panic(r) // re-panic if it's not a ParseError
			}
		}
	}()

	if p.match(token.TokenType_Var) {
		return p.varDeclaration()
	}
	return p.statement()
}

// varDeclaration → "var" IDENTIFIER ( "=" expression )? ";"
func (p *Parser) varDeclaration() stmt.Stmt {
	name := p.consume(token.TokenType_Identifier, "Expect variable name.")

	var initializer expr.Expr
	if p.match(token.TokenType_Equal) {
		initializer = p.expression()
	}

	p.consume(token.TokenType_Semicolon, "Expect ';' after variable declaration.")
	return stmt.Var{Name: name, Initializer: initializer}
}

// statement → printStmt | ifStmt | whileStmt | forStmt | block | exprStmt
func (p *Parser) statement() stmt.Stmt {
	if p.match(token.TokenType_If) {
		return p.ifStatement()
	}
	if p.match(token.TokenType_Print) {
		return p.printStatement()
	}
	if p.match(token.TokenType_While) {
		return p.whileStatement()
	}
	if p.match(token.TokenType_For) {
		return p.forStatement()
	}
	if p.match(token.TokenType_LeftBrace) {
		return stmt.Block{Statements: p.block()}
	}
	return p.expressionStatement()
}

// ifStatement → "if" "(" expression ")" statement ( "else" statement )?
func (p *Parser) ifStatement() stmt.Stmt {
	p.consume(token.TokenType_LeftParen, "Expect '(' after 'if'.")
	condition := p.expression()
	p.consume(token.TokenType_RightParen, "Expect ')' after if condition.")

	thenBranch := p.statement()

	// The else eagerly binds to the nearest if — no dangling else ambiguity
	var elseBranch stmt.Stmt
	if p.match(token.TokenType_Else) {
		elseBranch = p.statement()
	}

	return stmt.If{Condition: condition, ThenBranch: thenBranch, ElseBranch: elseBranch}
}

// whileStatement → "while" "(" expression ")" statement
func (p *Parser) whileStatement() stmt.Stmt {
	p.consume(token.TokenType_LeftParen, "Expect '(' after 'while'.")
	condition := p.expression()
	p.consume(token.TokenType_RightParen, "Expect ')' after condition.")
	body := p.statement()

	return stmt.While{Condition: condition, Body: body}
}

// forStatement → "for" "(" (varDecl | exprStmt | ";") expression? ";" expression? ")" statement
//
// This is the desugaring magic. We parse the three for-loop components,
// then build the equivalent while loop from existing AST nodes.
// The interpreter never sees a "for" — only blocks and while loops.
func (p *Parser) forStatement() stmt.Stmt {
	p.consume(token.TokenType_LeftParen, "Expect '(' after 'for'.")

	// 1. Initializer: var decl, expression, or nothing
	var initializer stmt.Stmt
	if p.match(token.TokenType_Semicolon) {
		initializer = nil
	} else if p.match(token.TokenType_Var) {
		initializer = p.varDeclaration()
	} else {
		initializer = p.expressionStatement()
	}

	// 2. Condition: expression or nothing (defaults to true = infinite loop)
	var condition expr.Expr
	if !p.check(token.TokenType_Semicolon) {
		condition = p.expression()
	}
	p.consume(token.TokenType_Semicolon, "Expect ';' after loop condition.")

	// 3. Increment: expression or nothing
	var increment expr.Expr
	if !p.check(token.TokenType_RightParen) {
		increment = p.expression()
	}
	p.consume(token.TokenType_RightParen, "Expect ')' after for clauses.")

	// 4. Body
	body := p.statement()

	// Now desugar: build the while loop from the bottom up.

	// If there's an increment, append it after the body
	if increment != nil {
		body = stmt.Block{
			Statements: []stmt.Stmt{body, stmt.Expression{Expression: increment}},
		}
	}

	// If no condition, default to `true` (infinite loop)
	if condition == nil {
		condition = expr.Literal{Value: true}
	}
	body = stmt.While{Condition: condition, Body: body}

	// If there's an initializer, wrap everything in a block
	if initializer != nil {
		body = stmt.Block{Statements: []stmt.Stmt{initializer, body}}
	}

	return body
}

// printStatement → "print" expression ";"
func (p *Parser) printStatement() stmt.Stmt {
	value := p.expression()
	p.consume(token.TokenType_Semicolon, "Expect ';' after value.")
	return stmt.Print{Expression: value}
}

// expressionStatement → expression ";"
func (p *Parser) expressionStatement() stmt.Stmt {
	ex := p.expression()
	p.consume(token.TokenType_Semicolon, "Expect ';' after expression.")
	return stmt.Expression{Expression: ex}
}

// block → "{" declaration* "}"
// Returns the list of statements inside the braces.
func (p *Parser) block() []stmt.Stmt {
	var statements []stmt.Stmt

	for !p.check(token.TokenType_RightBrace) && !p.isAtEnd() {
		s := p.declaration()
		if s != nil {
			statements = append(statements, s)
		}
	}

	p.consume(token.TokenType_RightBrace, "Expect '}' after block.")
	return statements
}

// expression → assignment
// The top of the expression precedence chain now starts at assignment.
func (p *Parser) expression() expr.Expr {
	return p.assignment()
}

// assignment → IDENTIFIER "=" assignment | equality
//
// This is the trickiest parsing rule. We parse the left side as an equality
// expression first. If we then see "=", we check that the left side was a
// Variable — and convert it into an Assign expression.
//
// Why? Because we can't know it's an assignment until we see the "=".
// The left side of an assignment isn't an expression we evaluate — it's
// a "target" (an l-value in compiler terminology).
func (p *Parser) assignment() expr.Expr {
	ex := p.or()

	if p.match(token.TokenType_Equal) {
		equals := p.previous()
		value := p.assignment() // right-associative: a = b = c means a = (b = c)

		if v, ok := ex.(expr.Variable); ok {
			return expr.Assign{Name: v.Name, Value: value}
		}

		p.error(equals, "Invalid assignment target.")
	}

	return ex
}

// or → and ( "or" and )*
// Lower precedence than and: `a or b and c` means `a or (b and c)`
func (p *Parser) or() expr.Expr {
	ex := p.and()

	for p.match(token.TokenType_Or) {
		operator := p.previous()
		right := p.and()
		ex = expr.Logical{Left: ex, Operator: operator, Right: right}
	}

	return ex
}

// and → equality ( "and" equality )*
func (p *Parser) and() expr.Expr {
	ex := p.equality()

	for p.match(token.TokenType_And) {
		operator := p.previous()
		right := p.equality()
		ex = expr.Logical{Left: ex, Operator: operator, Right: right}
	}

	return ex
}

func (p *Parser) equality() expr.Expr {
	ex := p.comparison()

	for p.match(token.TokenType_BangEqual, token.TokenType_EqualEqual) {
		operator := p.previous()
		right := p.comparison()
		ex = expr.Binary{Left: ex, Operator: operator, Right: right}
	}

	return ex
}

func (p *Parser) comparison() expr.Expr {
	ex := p.term()

	for p.match(token.TokenType_Greater, token.TokenType_GreaterEqual, token.TokenType_Less, token.TokenType_LessEqual) {
		operator := p.previous()
		right := p.term()
		ex = expr.Binary{Left: ex, Operator: operator, Right: right}
	}

	return ex
}

func (p *Parser) term() expr.Expr {
	ex := p.factor()

	for p.match(token.TokenType_Minus, token.TokenType_Plus) {
		operator := p.previous()
		right := p.factor()
		ex = expr.Binary{Left: ex, Operator: operator, Right: right}
	}

	return ex
}

func (p *Parser) factor() expr.Expr {
	ex := p.unary()

	for p.match(token.TokenType_Slash, token.TokenType_Star, token.TokenType_Percent) {
		operator := p.previous()
		right := p.unary()
		ex = expr.Binary{Left: ex, Operator: operator, Right: right}
	}

	return ex
}

func (p *Parser) unary() expr.Expr {
	if p.match(token.TokenType_Bang, token.TokenType_Minus) {
		operator := p.previous()
		right := p.unary()
		return expr.Unary{Operator: operator, Right: right}
	}

	return p.primary()
}

func (p *Parser) primary() expr.Expr {
	if p.match(token.TokenType_False) {
		return expr.Literal{Value: false}
	}

	if p.match(token.TokenType_True) {
		return expr.Literal{Value: true}
	}

	if p.match(token.TokenType_Nil) {
		return expr.Literal{Value: nil}
	}

	if p.match(token.TokenType_Number, token.TokenType_String) {
		return expr.Literal{Value: p.previous().Literal}
	}

	// Variable access — an identifier by itself is a variable read
	if p.match(token.TokenType_Identifier) {
		return expr.Variable{Name: p.previous()}
	}

	if p.match(token.TokenType_LeftParen) {
		ex := p.expression()
		p.consume(token.TokenType_RightParen, "Expect ')' after expression.")
		return expr.Grouping{Expression: ex}
	}

	panic(p.error(p.peek(), "Expect expression."))
}

// --- Helper methods ---

func (p *Parser) match(types ...token.TokenType) bool {
	for _, t := range types {
		if p.check(t) {
			p.advance()
			return true
		}
	}
	return false
}

// consume checks the next token is the expected type. If not, panics with a ParseError.
// The panic is caught by declaration()'s recover, which calls synchronize().
func (p *Parser) consume(t token.TokenType, message string) *token.Token {
	if p.check(t) {
		return p.advance()
	}
	panic(p.error(p.peek(), message))
}

// error reports a parse error to the VM and returns the error (for panicking).
func (p *Parser) error(tok *token.Token, message string) ParseError {
	err := ParseError{Token: tok, Message: message}
	p.vm.ReportParseError(err)
	return err
}

func (p *Parser) check(t token.TokenType) bool {
	if p.isAtEnd() {
		return false
	}
	return p.peek().TokenType == t
}

func (p *Parser) advance() *token.Token {
	if !p.isAtEnd() {
		p.current++
	}
	return p.previous()
}

func (p *Parser) isAtEnd() bool {
	return p.current >= len(p.tokens) || p.peek().TokenType == token.TokenType_EOF
}

func (p *Parser) peek() *token.Token {
	if p.current >= len(p.tokens) {
		return &token.Token{TokenType: token.TokenType_EOF, Lexeme: "", Line: p.tokens[len(p.tokens)-1].Line}
	}
	return p.tokens[p.current]
}

func (p *Parser) previous() *token.Token {
	if p.current <= 0 {
		return &token.Token{TokenType: token.TokenType_EOF, Lexeme: "", Line: p.peek().Line}
	}
	return p.tokens[p.current-1]
}

// synchronize skips tokens until we hit a statement boundary.
// Called after a parse error so we can keep parsing the rest of the file.
func (p *Parser) synchronize() {
	p.advance()
	for !p.isAtEnd() {
		if p.previous().TokenType == token.TokenType_Semicolon {
			return
		}
		switch p.peek().TokenType {
		case token.TokenType_Class,
			token.TokenType_Fun,
			token.TokenType_Var,
			token.TokenType_For,
			token.TokenType_If,
			token.TokenType_While,
			token.TokenType_Print,
			token.TokenType_Return:
			return
		}
		p.advance()
	}
}
