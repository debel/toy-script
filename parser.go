package main

import (
	"fmt"
	"strconv"
)

type (
	toyParser struct {
		tokens   []Token
		_current int
	}
)

func NewParser(tokens []Token) *toyParser {
	return &toyParser{tokens, 0}
}

func (p *toyParser) Parse() (ProgramStatement, bool) {
	return p.programStatement()
}

func (p *toyParser) programStatement() (ProgramStatement, bool) {
	program := ProgramStatement{
		[]Node{},
	}

	hasErrors := false
	for !p.done() {
		child, hasError := p.statement()
		if hasError {
			hasErrors = true
		}

		if child != nil {
			program.Body = append(program.Body, child)
		}
	}

	return program, hasErrors
}

func (p *toyParser) statement() (Node, bool) {
	if p.match(TOKEN_LEFT_PAREN) {
		if p.check(TOKEN_BUILTIN) {
			t := p.advance()
			switch t.Lexeme {
			case "@import":
				return p.importStatement()
			case "@export":
				return p.exportStatement()
			default:
				p.revert()
				p.revert()
				return p.expression()
			}
		} else {
			p.revert()
			return p.expression()
		}
	}

	failingAt := p.advance()
	return &MalformedExpression{
		failingAt,
		fmt.Errorf("unexpected token in statement at %d", failingAt.Line),
	}, true
}

func (p *toyParser) expression() (Node, bool) {
	if p.match(TOKEN_LEFT_PAREN) {
		t := p.advance()
		switch t.Type {
		case TOKEN_BUILTIN:
			switch t.Lexeme {
			case "@var":
				return p.varStatement()
			case "@list":
				return p.listLiteral()
			case "@hash":
				return p.hashLiteral()
			case "@match":
				return p.matchExpression()
			case "@func":
				return p.funcExpression()
			case "@seq":
				return p.seqExpression()
			case "@chain":
				return p.chainExpression()
			case "@async":
				return p.asyncExpression()
			// TODO: case "@stream":
			default:
				p.revert()
				return p.callExpression()
			}
		case TOKEN_IDENTIFIER, TOKEN_EQUAL, TOKEN_LESS, TOKEN_MORE:
			p.revert()
			return p.callExpression()
		default:
			return &MalformedExpression{t, fmt.Errorf("unexpected token in expression")}, true
		}
	}

	return p.simpleLiteral()
}

func (p *toyParser) simpleLiteral() (Node, bool) {
	t := p.advance()
	switch t.Type {
	case TOKEN_STRING:
		return &StringLiteral{t.Lexeme}, false
	case TOKEN_NUMBER:
		i, err := strconv.Atoi(t.Lexeme)
		if err != nil {
			return &MalformedExpression{
				t,
				fmt.Errorf("malformed number literal"),
			}, true
		}
		return &NumberLiteral{i}, false
	case TOKEN_BOOLEAN:
		return &BooleanLiteral{t.Literal.(bool)}, false
	case TOKEN_IDENTIFIER:
		p.revert()
		return p.referenceExpression()
	}

	return &MalformedExpression{
		p.peek(0),
		fmt.Errorf("unexpected token in expression %v", t),
	}, true
}

// STATEMENTS

func (p *toyParser) importStatement() (Node, bool) {
	imports := map[string]string{}

	for !p.check(TOKEN_RIGHT_PAREN) && !p.done() {
		_, err := p.consume(TOKEN_LEFT_PAREN, "expected import pair")
		if err != nil {
			return err, true
		}

		alias, err := p.consume(TOKEN_IDENTIFIER, "expected module alais")
		if err != nil {
			return err, true
		}

		path, err := p.consume(TOKEN_STRING, "expected module path")
		if err != nil {
			return err, true
		}

		_, err = p.consume(TOKEN_RIGHT_PAREN, "malformed import pair: missing closing )")
		if err != nil {
			return err, true
		}

		_, alreadyExists := imports[alias.Lexeme]
		if alreadyExists {
			return &MalformedExpression{
				Body:  alias,
				Error: fmt.Errorf("duplicated import alias"),
			}, true
		}
		imports[alias.Lexeme] = path.Lexeme
	}

	_, err := p.consume(TOKEN_RIGHT_PAREN, "expected closing ) for import statement")
	if err != nil {
		return err, true
	}

	return &ImportStatement{imports}, false
}

func (p *toyParser) varStatement() (Node, bool) {
	vars := map[string]Node{}

	for !p.check(TOKEN_RIGHT_PAREN) && !p.done() {
		_, err := p.consume(TOKEN_LEFT_PAREN, "expected variable pair")
		if err != nil {
			return err, true
		}

		name, err := p.consume(TOKEN_IDENTIFIER, "expected variable name")
		if err != nil {
			return err, true
		}

		value, hasErr := p.expression()
		if hasErr {
			return value, true
		}

		_, err = p.consume(TOKEN_RIGHT_PAREN, "malformed variable pair: missing closing )")
		if err != nil {
			return err, true
		}

		_, alreadyExists := vars[name.Lexeme]
		if alreadyExists {
			return &MalformedExpression{
				Body:  name,
				Error: fmt.Errorf("duplicated variable name"),
			}, true
		}
		vars[name.Lexeme] = value
	}

	_, err := p.consume(TOKEN_RIGHT_PAREN, "expected closing ) for var statement")
	if err != nil {
		return err, true
	}

	return &VarStatement{vars}, false
}

func (p *toyParser) exportStatement() (Node, bool) {
	exports := []Node{}

	for !p.check(TOKEN_RIGHT_PAREN) && !p.done() {
		if p.check(TOKEN_IDENTIFIER) {
			// NOTE: variable export
			t := p.advance()
			exports = append(exports, &ReferenceExpression{t.Lexeme, REF_TYPE_DECLARED})
		} else {
			return &MalformedExpression{
				Body:  p.peek(0),
				Error: fmt.Errorf("unexpected token in exports list"),
			}, true
		}
	}

	_, err := p.consume(TOKEN_RIGHT_PAREN, "expected closing ) for export statement")
	if err != nil {
		return err, true
	}

	return &ExportStatement{exports}, false
}

// EXPRESSIONS

func (p *toyParser) callExpression() (Node, bool) {
	hasErrors := false
	callee, hasErr := p.referenceExpression()
	if hasErr {
		hasErrors = true
	}

	args := []Node{}
	for !p.check(TOKEN_RIGHT_PAREN) && !p.done() {
		arg, hasErr := p.expression()
		if hasErr {
			hasErrors = true
		}

		args = append(args, arg)
	}

	_, err := p.consume(TOKEN_RIGHT_PAREN, "expected ) in call expression")
	if err != nil {
		return err, true
	}

	return &CallExpression{
		callee,
		args,
	}, hasErrors
}

func (p *toyParser) funcExpression() (Node, bool) {
	_, err := p.consume(TOKEN_LEFT_PAREN, "expected params list for func declaration")
	if err != nil {
		return err, true
	}

	hasErrors := false
	params := []string{}
	for !p.check(TOKEN_RIGHT_PAREN) && !p.done() {
		param, err := p.consume(TOKEN_IDENTIFIER, "expected param name")
		if err != nil {
			hasErrors = true
		}

		params = append(params, param.Lexeme)
	}

	_, err = p.consume(TOKEN_RIGHT_PAREN, "expacted ) at the end of params list")
	if err != nil {
		return err, true
	}

	_, err = p.consume(TOKEN_LEFT_PAREN, "expected body for func declaration")
	if err != nil {
		return err, true
	}

	body := []Node{}
	for !p.check(TOKEN_RIGHT_PAREN) && !p.done() {
		child, hasErr := p.expression()
		if hasErr {
			hasErrors = true
		}
		body = append(body, child)
	}

	_, err = p.consume(TOKEN_RIGHT_PAREN, "expacted ) at the end of func body")
	if err != nil {
		return err, true
	}

	_, err = p.consume(TOKEN_RIGHT_PAREN, "expected end of func declaration")
	if err != nil {
		return err, true
	}

	return &FuncLiteral{params, body}, hasErrors
}

func (p *toyParser) referenceExpression() (Node, bool) {
	part1 := p.advance()
	switch part1.Type {
	case TOKEN_LESS, TOKEN_MORE, TOKEN_EQUAL:
		return &ReferenceExpression{part1.Lexeme, REF_TYPE_BUILTIN}, false
	case TOKEN_BUILTIN:
		return &ReferenceExpression{part1.Lexeme, REF_TYPE_BUILTIN}, false
	case TOKEN_IDENTIFIER:
		if p.match(TOKEN_DOT) {
			part2, err := p.consume(TOKEN_IDENTIFIER, "malformed reference to imported value")
			if err != nil {
				return err, true
			}

			return &ReferenceExpression{
				RefName: part1.Lexeme + "." + part2.Lexeme,
				RefType: REF_TYPE_IMPORTED,
			}, false
		}

		return &ReferenceExpression{part1.Lexeme, REF_TYPE_DECLARED}, false
	}

	return &MalformedExpression{part1, fmt.Errorf("unexpected token in reference")}, true
}

func (p *toyParser) matchExpression() (Node, bool) {
	hasErrors := false
	cases := map[Node]Node{}

	cond, hasErr := p.expression()
	if hasErr {
		hasErrors = true
	}

	for !p.check(TOKEN_RIGHT_PAREN) && !p.done() {
		_, err := p.consume(TOKEN_LEFT_PAREN, "expected start of when expression")
		if err != nil {
			hasErrors = true
		}

		when, err := p.consume(TOKEN_BUILTIN, "expected when key word")
		if err != nil || when.Lexeme != "@when" {
			hasErrors = true
		}

		expected, hasErr := p.expression()
		if hasErr {
			hasErrors = true
			expected = &MalformedExpression{expected, fmt.Errorf("malformed matcher")}
		}

		action, hasErr := p.expression()
		if hasErr {
			hasErrors = true
		}

		_, err = p.consume(TOKEN_RIGHT_PAREN, "expected end of when expression")
		if err != nil {
			hasErrors = true
		}

		cases[expected] = action
	}

	_, err := p.consume(TOKEN_RIGHT_PAREN, "expected end of match expression")
	if err != nil {
		hasErrors = true
	}

	return &MatchExpression{cond, cases}, hasErrors
}

func (p *toyParser) seqExpression() (Node, bool) {
	hasErrors := false
	exprs := []Node{}

	for !p.check(TOKEN_RIGHT_PAREN) && !p.done() {
		e, hasErr := p.expression()
		if hasErr {
			hasErrors = true
		}

		exprs = append(exprs, e)
	}

	p.consume(TOKEN_RIGHT_PAREN, "exprected end of seq")

	return &SeqExpression{exprs}, hasErrors
}

func (p *toyParser) chainExpression() (Node, bool) {
	hasErrors := false

	exprs := []Node{}
	for !p.check(TOKEN_RIGHT_PAREN) && !p.done() {
		e, hasErr := p.expression()
		if hasErr {
			hasErrors = true
		}

		exprs = append(exprs, e)
	}

	_, err := p.consume(TOKEN_RIGHT_PAREN, "expected end of chain")
	if err != nil {
		return err, true
	}

	if len(exprs) == 0 {
		return &MalformedExpression{nil, fmt.Errorf("chain expectes at least one inner expression")}, true
	}

	return &ChainExpression{exprs}, hasErrors
}

func (p *toyParser) asyncExpression() (Node, bool) {
	hasErrors := false
	exprs := []Node{}

	for !p.check(TOKEN_RIGHT_PAREN) && !p.done() {
		e, hasErr := p.expression()
		if hasErr {
			hasErrors = true
		}

		exprs = append(exprs, e)
	}

	p.consume(TOKEN_RIGHT_PAREN, "exprected end of seq")

	return &AsyncExpression{exprs}, hasErrors
}

// LITERALS

func (p *toyParser) listLiteral() (Node, bool) {
	hasErrors := false
	elements := []Node{}

	for !p.check(TOKEN_RIGHT_PAREN) && !p.done() {
		el, hasErr := p.expression()
		if hasErr {
			hasErrors = true
		}
		elements = append(elements, el)
	}

	_, err := p.consume(TOKEN_RIGHT_PAREN, "expected ) at end of list")
	if err != nil {
		return err, true
	}

	return &ListLiteral{elements}, hasErrors
}

func (p *toyParser) hashLiteral() (Node, bool) {
	hasErrors := false
	store := map[string]Node{}

	for !p.check(TOKEN_RIGHT_PAREN) && !p.done() {
		// NOTE: consume pairs
		_, err := p.consume(TOKEN_LEFT_PAREN, "expected key value pair")
		if err != nil {
			return err, true
		}

		// SHIT: this means keys cannot be dymanic!
		key, err := p.consume(TOKEN_STRING, "key must be a string")
		if err != nil {
			return err, true
		}

		value, hasErr := p.expression()
		if hasErr {
			fmt.Println("value has error?!", value)
			hasErrors = true
		}

		_, alreadyExists := store[key.Lexeme]
		if alreadyExists {
			fmt.Println("WARNING: duplicate keys in hash", key)
			hasErrors = true
		}

		_, err = p.consume(TOKEN_RIGHT_PAREN, "expected end of key value pair")
		if err != nil {
			hasErrors = true
			value = err
		}

		store[key.Lexeme] = value
	}

	_, err := p.consume(TOKEN_RIGHT_PAREN, "expected end of hash literal")
	if err != nil {
		return err, true
	}

	return &HashLiteral{store}, hasErrors
}

// PARSER HELPERS

func (p *toyParser) done() bool {
	return p._current >= len(p.tokens) || p.tokens[p._current].Type == TOKEN_EOF
}

func (p *toyParser) peek(i int) Token {
	index := p._current + i
	if index >= len(p.tokens) {
		return p.tokens[len(p.tokens)-1]
	}

	return p.tokens[index]
}

func (p *toyParser) advance() Token {
	t := p.peek(0)

	p._current += 1

	return t
}

func (p *toyParser) match(_type TokenType) bool {
	if p.check(_type) {
		p.advance()
		return true
	}

	return false
}

func (p *toyParser) check(_type TokenType) bool {
	if p.done() {
		return false
	}

	return p.peek(0).Type == _type
}

func (p *toyParser) consume(_type TokenType, err string) (Token, *MalformedExpression) {
	t := p.advance()
	if t.Type == _type {
		return t, nil
	}

	return Token{}, &MalformedExpression{
		Body:  t,
		Error: fmt.Errorf("parser: %s at %d", err, t.Line),
	}
}

func (p *toyParser) revert() Token {
	if p._current-1 < 0 {
		return p.tokens[0]
	}

	p._current -= 1
	return p.peek(0)
}
