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

// declaration recovers from parse errors via synchronize.
func (p *Parser) declaration() stmt.Stmt {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(ParseError); ok {
				p.synchronize()
			} else {
				panic(r)
			}
		}
	}()

	if p.match(token.TokenType_Fun) {
		return p.funDeclaration("function")
	}
	if p.match(token.TokenType_Var) {
		return p.varDeclaration()
	}
	return p.statement()
}

func (p *Parser) varDeclaration() stmt.Stmt {
	name := p.consume(token.TokenType_Identifier, "Expect variable name.")

	var initializer expr.Expr
	if p.match(token.TokenType_Equal) {
		initializer = p.expression()
	}

	p.consume(token.TokenType_Semicolon, "Expect ';' after variable declaration.")
	return stmt.Var{Name: name, Initializer: initializer}
}

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
	if p.match(token.TokenType_Return) {
		return p.returnStatement()
	}
	if p.match(token.TokenType_For) {
		return p.forStatement()
	}
	if p.match(token.TokenType_LeftBrace) {
		return stmt.Block{Statements: p.block()}
	}
	return p.expressionStatement()
}

func (p *Parser) ifStatement() stmt.Stmt {
	p.consume(token.TokenType_LeftParen, "Expect '(' after 'if'.")
	condition := p.expression()
	p.consume(token.TokenType_RightParen, "Expect ')' after if condition.")

	thenBranch := p.statement()

	var elseBranch stmt.Stmt
	if p.match(token.TokenType_Else) {
		elseBranch = p.statement()
	}

	return stmt.If{Condition: condition, ThenBranch: thenBranch, ElseBranch: elseBranch}
}

func (p *Parser) whileStatement() stmt.Stmt {
	p.consume(token.TokenType_LeftParen, "Expect '(' after 'while'.")
	condition := p.expression()
	p.consume(token.TokenType_RightParen, "Expect ')' after condition.")
	body := p.statement()

	return stmt.While{Condition: condition, Body: body}
}

// forStatement desugars `for` into a while loop wrapped in a block.
func (p *Parser) forStatement() stmt.Stmt {
	p.consume(token.TokenType_LeftParen, "Expect '(' after 'for'.")

	var initializer stmt.Stmt
	if p.match(token.TokenType_Semicolon) {
		initializer = nil
	} else if p.match(token.TokenType_Var) {
		initializer = p.varDeclaration()
	} else {
		initializer = p.expressionStatement()
	}

	var condition expr.Expr
	if !p.check(token.TokenType_Semicolon) {
		condition = p.expression()
	}
	p.consume(token.TokenType_Semicolon, "Expect ';' after loop condition.")

	var increment expr.Expr
	if !p.check(token.TokenType_RightParen) {
		increment = p.expression()
	}
	p.consume(token.TokenType_RightParen, "Expect ')' after for clauses.")

	body := p.statement()

	if increment != nil {
		body = stmt.Block{
			Statements: []stmt.Stmt{body, stmt.Expression{Expression: increment}},
		}
	}

	if condition == nil {
		condition = expr.Literal{Value: true}
	}
	body = stmt.While{Condition: condition, Body: body}

	if initializer != nil {
		body = stmt.Block{Statements: []stmt.Stmt{initializer, body}}
	}

	return body
}

// funDeclaration parses function declarations. `kind` is used in error messages
// ("function" now, "method" for classes later).
func (p *Parser) funDeclaration(kind string) stmt.Stmt {
	name := p.consume(token.TokenType_Identifier, "Expect "+kind+" name.")
	p.consume(token.TokenType_LeftParen, "Expect '(' after "+kind+" name.")

	var params []*token.Token
	if !p.check(token.TokenType_RightParen) {
		for {
			if len(params) >= 255 {
				p.error(p.peek(), "Can't have more than 255 parameters.")
			}
			params = append(params, p.consume(token.TokenType_Identifier, "Expect parameter name."))
			if !p.match(token.TokenType_Comma) {
				break
			}
		}
	}
	p.consume(token.TokenType_RightParen, "Expect ')' after parameters.")

	p.consume(token.TokenType_LeftBrace, "Expect '{' before "+kind+" body.")
	body := p.block()
	return stmt.Function{Name: name, Params: params, Body: body}
}

func (p *Parser) returnStatement() stmt.Stmt {
	keyword := p.previous()
	var value expr.Expr
	if !p.check(token.TokenType_Semicolon) {
		value = p.expression()
	}
	p.consume(token.TokenType_Semicolon, "Expect ';' after return value.")
	return stmt.Return{Keyword: keyword, Value: value}
}

func (p *Parser) printStatement() stmt.Stmt {
	value := p.expression()
	p.consume(token.TokenType_Semicolon, "Expect ';' after value.")
	return stmt.Print{Expression: value}
}

func (p *Parser) expressionStatement() stmt.Stmt {
	ex := p.expression()
	p.consume(token.TokenType_Semicolon, "Expect ';' after expression.")
	return stmt.Expression{Expression: ex}
}

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

func (p *Parser) expression() expr.Expr {
	return p.assignment()
}

// assignment parses the left side as an expression, then checks for '='.
// If found, verifies the left side was a Variable (valid l-value).
func (p *Parser) assignment() expr.Expr {
	ex := p.or()

	if p.match(token.TokenType_Equal) {
		equals := p.previous()
		value := p.assignment()

		if v, ok := ex.(expr.Variable); ok {
			return expr.Assign{Name: v.Name, Value: value}
		}

		p.error(equals, "Invalid assignment target.")
	}

	return ex
}

func (p *Parser) or() expr.Expr {
	ex := p.and()

	for p.match(token.TokenType_Or) {
		operator := p.previous()
		right := p.and()
		ex = expr.Logical{Left: ex, Operator: operator, Right: right}
	}

	return ex
}

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

	return p.call()
}

// call handles function calls: primary ( "(" args ")" )*
func (p *Parser) call() expr.Expr {
	ex := p.primary()

	for {
		if p.match(token.TokenType_LeftParen) {
			ex = p.finishCall(ex)
		} else {
			break
		}
	}

	return ex
}

func (p *Parser) finishCall(callee expr.Expr) expr.Expr {
	var arguments []expr.Expr

	if !p.check(token.TokenType_RightParen) {
		for {
			if len(arguments) >= 255 {
				p.error(p.peek(), "Can't have more than 255 arguments.")
			}
			arguments = append(arguments, p.expression())
			if !p.match(token.TokenType_Comma) {
				break
			}
		}
	}

	paren := p.consume(token.TokenType_RightParen, "Expect ')' after arguments.")
	return expr.Call{Callee: callee, Paren: paren, Arguments: arguments}
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

func (p *Parser) consume(t token.TokenType, message string) *token.Token {
	if p.check(t) {
		return p.advance()
	}
	panic(p.error(p.peek(), message))
}

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

// synchronize skips tokens until a statement boundary after a parse error.
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
