package main

import (
	"strconv"
	"strings"
	"unicode"
)

type (
	TokenType = string
	Token     struct {
		Type    TokenType
		Lexeme  string
		Literal any
		Line    int
	}

	toyScanner struct {
		source  string
		tokens  []Token
		start   int
		current int
		line    int
	}
)

const (
	TOKEN_LEFT_PAREN  TokenType = "open-paren"
	TOKEN_RIGHT_PAREN TokenType = "close-paren"
	TOKEN_COMMA       TokenType = "comma"
	TOKEN_DOT         TokenType = "dot"
	TOKEN_MINUS       TokenType = "minus"
	TOKEN_PLUS        TokenType = "plus"
	TOKEN_SLASH       TokenType = "slash"
	TOKEN_STAR        TokenType = "star"
	TOKEN_BANG        TokenType = "bang"
	TOKEN_EQUAL       TokenType = "equal"
	TOKEN_LESS        TokenType = "less"
	TOKEN_MORE        TokenType = "more"
	TOKEN_HASH        TokenType = "hash"

	TOKEN_BUILTIN    TokenType = "built-in"
	TOKEN_IDENTIFIER TokenType = "identifier"
	TOKEN_STRING     TokenType = "string"
	TOKEN_NUMBER     TokenType = "number"
	TOKEN_BOOLEAN    TokenType = "boolean"

	TOKEN_COMMENT TokenType = "comment"

	TOKEN_ERROR TokenType = "error"
	TOKEN_EOF   TokenType = "end-of-file"
	TOKEN_NULL  TokenType = "null-token"
)

func NewScanner(source string) *toyScanner {
	return &toyScanner{source, []Token{}, 0, 0, 1}
}

func (s *toyScanner) ScanTokens() ([]Token, error) {
	for !s.done() {
		token := s.scanToken()
		if token != nil {
			s.tokens = append(s.tokens, *token)
		}
	}

	s.tokens = append(s.tokens, Token{TOKEN_EOF, "", nil, len(s.source)})
	return s.tokens, nil
}

func (s *toyScanner) scanToken() *Token {
	c := s.advance()
	switch c {
	case ' ', '\r', '\t':
		return nil
	case '\n':
		s.line += 1
		return nil
	case '(':
		return &Token{TOKEN_LEFT_PAREN, "(", nil, s.current}
	case ')':
		return &Token{TOKEN_RIGHT_PAREN, ")", nil, s.current}
	case ',':
		return &Token{TOKEN_COMMA, ",", nil, s.current}
	case '.':
		return &Token{TOKEN_DOT, ".", nil, s.current}
	case '-':
		return &Token{TOKEN_MINUS, "-", nil, s.current}
	case '+':
		return &Token{TOKEN_PLUS, "+", nil, s.current}
	case '*':
		return &Token{TOKEN_STAR, "*", nil, s.current}
	case '!':
		return &Token{TOKEN_BANG, "!", nil, s.current}
	case '=':
		return &Token{TOKEN_EQUAL, "=", nil, s.current}
	case '>':
		return &Token{TOKEN_MORE, ">", nil, s.current}
	case '<':
		return &Token{TOKEN_LESS, "<", nil, s.current}
	case '"':
		return s.stringToken()
	case '@':
		return s.builtInToken()
	case '#':
		// consume the comment but ignore it
		s.commentToken()
		return nil
	default:
		if isNumberic(c) {
			return s.numberToken()
		} else if s.match("true") {
			return &Token{TOKEN_BOOLEAN, "true", true, s.current}
		} else if s.match("false") {
			return &Token{TOKEN_BOOLEAN, "false", false, s.current}
		} else if isAlphabetic(c) {
			return s.identifierToken()
		}
	}

	return &Token{TOKEN_ERROR, string(c), nil, s.current}
}

func (s *toyScanner) done() bool {
	return s.current >= len(s.source)
}

func (s *toyScanner) advance() byte {
	char := s.source[s.current]
	s.current += 1

	return char
}

func (s *toyScanner) peek() byte {
	if s.done() {
		return byte(rune(0))
	}

	return s.source[s.current]
}

func (s *toyScanner) match(expected string) bool {
	start := s.current
	s.current -= 1
	for i := 0; i < len(expected); i += 1 {
		input := s.advance()
		if input == expected[i] {
			continue
		}

		// revert to original position
		s.current = start
		return false
	}

	return true
}

func (s *toyScanner) stringToken() *Token {
	str := strings.Builder{}
	for s.peek() != '"' && !s.done() {
		if s.peek() == '\n' {
			s.line += 1
		}
		str.WriteByte(s.advance())
	}

	if s.done() {
		return &Token{TOKEN_ERROR, "Unterminated string", nil, s.current}
	}

	// consume the closing "
	s.advance()

	return &Token{TOKEN_STRING, str.String(), nil, s.current}
}

func (s *toyScanner) commentToken() *Token {
	str := strings.Builder{}
	for s.peek() != '\n' && !s.done() {
		str.WriteByte(s.advance())
	}

	return &Token{TOKEN_COMMENT, str.String(), nil, s.current}
}

func (s *toyScanner) numberToken() *Token {
	str := strings.Builder{}
	str.WriteByte(s.source[s.current-1])

	for isNumberic(s.peek()) {
		str.WriteByte(s.advance())
	}

	i, err := strconv.Atoi(str.String())
	if err != nil {
		return &Token{TOKEN_ERROR, err.Error(), nil, s.current}
	}

	return &Token{TOKEN_NUMBER, str.String(), i, s.current}
}

func (s *toyScanner) identifierToken() *Token {
	str := strings.Builder{}
	str.WriteByte(s.source[s.current-1])

	for isAlphabetic(s.peek()) {
		if s.done() {
			return &Token{TOKEN_ERROR, "unexpected end of input", nil, s.current}
		}

		str.WriteByte(s.advance())
	}

	return &Token{TOKEN_IDENTIFIER, str.String(), nil, s.current}
}

func (s *toyScanner) builtInToken() *Token {
	str := strings.Builder{}
	str.WriteByte(s.source[s.current-1])

	for isAlphabetic(s.peek()) {
		if s.done() {
			return &Token{TOKEN_ERROR, "unexpected end of input", nil, s.current}
		}

		str.WriteByte(s.advance())
	}

	return &Token{TOKEN_BUILTIN, str.String(), nil, s.current}
}

func isNumberic(b byte) bool {
	// TODO: parse floats
	return unicode.IsDigit(rune(b))
}

func isAlphabetic(b byte) bool {
	return unicode.IsLetter(rune(b)) || b == '_'
}

func (t Token) String() string {
	switch t.Type {
	case TOKEN_IDENTIFIER:
		return ":" + t.Lexeme
	case TOKEN_STRING:
		return "'" + t.Lexeme + "'"
	case TOKEN_NUMBER:
		return t.Lexeme

	}

	return "@" + t.Type
}
