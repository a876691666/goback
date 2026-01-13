package ssql

import (
	"fmt"
	"strings"
	"unicode"
)

// TokenType 词法单元类型
type TokenType int

const (
	TokenField      TokenType = iota // 字段名
	TokenOperator                    // 操作符
	TokenValue                       // 值
	TokenLogicAnd                    // &&
	TokenLogicOr                     // ||
	TokenLParen                      // (
	TokenRParen                      // )
	TokenLBracket                    // [
	TokenRBracket                    // ]
	TokenComma                       // ,
	TokenEOF                         // 结束
)

// Token 词法单元
type Token struct {
	Type  TokenType
	Value string
	Pos   int
}

// Lexer 词法分析器
type Lexer struct {
	input   string
	pos     int
	readPos int
	ch      byte
}

// NewLexer 创建词法分析器
func NewLexer(input string) *Lexer {
	l := &Lexer{input: input}
	l.readChar()
	return l
}

// readChar 读取下一个字符
func (l *Lexer) readChar() {
	if l.readPos >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPos]
	}
	l.pos = l.readPos
	l.readPos++
}

// peekChar 查看下一个字符
func (l *Lexer) peekChar() byte {
	if l.readPos >= len(l.input) {
		return 0
	}
	return l.input[l.readPos]
}

// skipWhitespace 跳过空白字符
func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		l.readChar()
	}
}

// Tokenize 词法分析
func (l *Lexer) Tokenize() ([]Token, error) {
	var tokens []Token

	for {
		l.skipWhitespace()

		if l.ch == 0 {
			tokens = append(tokens, Token{Type: TokenEOF, Value: "", Pos: l.pos})
			break
		}

		token, err := l.nextToken()
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, token)
	}

	return tokens, nil
}

// nextToken 获取下一个词法单元
func (l *Lexer) nextToken() (Token, error) {
	l.skipWhitespace()

	pos := l.pos

	switch l.ch {
	case '(':
		l.readChar()
		return Token{Type: TokenLParen, Value: "(", Pos: pos}, nil
	case ')':
		l.readChar()
		return Token{Type: TokenRParen, Value: ")", Pos: pos}, nil
	case '[':
		l.readChar()
		return Token{Type: TokenLBracket, Value: "[", Pos: pos}, nil
	case ']':
		l.readChar()
		return Token{Type: TokenRBracket, Value: "]", Pos: pos}, nil
	case ',':
		l.readChar()
		return Token{Type: TokenComma, Value: ",", Pos: pos}, nil
	case '&':
		if l.peekChar() == '&' {
			l.readChar()
			l.readChar()
			return Token{Type: TokenLogicAnd, Value: "&&", Pos: pos}, nil
		}
	case '|':
		if l.peekChar() == '|' {
			l.readChar()
			l.readChar()
			return Token{Type: TokenLogicOr, Value: "||", Pos: pos}, nil
		}
	case '\'', '"':
		return l.readString()
	case '=', '!', '>', '<', '~', '?':
		return l.readOperator()
	}

	// 数字
	if isDigit(l.ch) || (l.ch == '-' && isDigit(l.peekChar())) {
		return l.readNumber()
	}

	// 字段名或关键字
	if isLetter(l.ch) || l.ch == '_' {
		return l.readIdentifier()
	}

	return Token{}, fmt.Errorf("unexpected character '%c' at position %d", l.ch, l.pos)
}

// readIdentifier 读取标识符(字段名或关键字)
func (l *Lexer) readIdentifier() (Token, error) {
	pos := l.pos
	var sb strings.Builder

	for isLetter(l.ch) || isDigit(l.ch) || l.ch == '_' || l.ch == '.' {
		sb.WriteByte(l.ch)
		l.readChar()
	}

	value := sb.String()

	// 检查是否为布尔值或null
	lower := strings.ToLower(value)
	if lower == "true" || lower == "false" || lower == "null" {
		return Token{Type: TokenValue, Value: value, Pos: pos}, nil
	}

	return Token{Type: TokenField, Value: value, Pos: pos}, nil
}

// readOperator 读取操作符
func (l *Lexer) readOperator() (Token, error) {
	pos := l.pos
	var sb strings.Builder

	// 支持的操作符: =, !=, >, >=, <, <=, ~, !~, ?=, ?!=, ?null, ?!null, ><
	switch l.ch {
	case '=':
		sb.WriteByte(l.ch)
		l.readChar()
	case '!':
		sb.WriteByte(l.ch)
		l.readChar()
		if l.ch == '=' || l.ch == '~' {
			sb.WriteByte(l.ch)
			l.readChar()
		}
	case '>':
		sb.WriteByte(l.ch)
		l.readChar()
		if l.ch == '=' || l.ch == '<' {
			sb.WriteByte(l.ch)
			l.readChar()
		}
	case '<':
		sb.WriteByte(l.ch)
		l.readChar()
		if l.ch == '=' {
			sb.WriteByte(l.ch)
			l.readChar()
		}
	case '~':
		sb.WriteByte(l.ch)
		l.readChar()
	case '?':
		sb.WriteByte(l.ch)
		l.readChar()
		if l.ch == '=' {
			sb.WriteByte(l.ch)
			l.readChar()
		} else if l.ch == '!' {
			sb.WriteByte(l.ch)
			l.readChar()
			if l.ch == '=' {
				sb.WriteByte(l.ch)
				l.readChar()
			} else if isLetter(l.ch) {
				// ?!null
				for isLetter(l.ch) {
					sb.WriteByte(l.ch)
					l.readChar()
				}
			}
		} else if isLetter(l.ch) {
			// ?null
			for isLetter(l.ch) {
				sb.WriteByte(l.ch)
				l.readChar()
			}
		}
	}

	return Token{Type: TokenOperator, Value: sb.String(), Pos: pos}, nil
}

// readString 读取字符串
func (l *Lexer) readString() (Token, error) {
	pos := l.pos
	quote := l.ch
	l.readChar() // 跳过开始引号

	var sb strings.Builder
	for l.ch != quote && l.ch != 0 {
		if l.ch == '\\' {
			l.readChar()
			switch l.ch {
			case 'n':
				sb.WriteByte('\n')
			case 't':
				sb.WriteByte('\t')
			case 'r':
				sb.WriteByte('\r')
			case '\\':
				sb.WriteByte('\\')
			case '\'':
				sb.WriteByte('\'')
			case '"':
				sb.WriteByte('"')
			default:
				sb.WriteByte(l.ch)
			}
		} else {
			sb.WriteByte(l.ch)
		}
		l.readChar()
	}

	if l.ch == 0 {
		return Token{}, fmt.Errorf("unterminated string at position %d", pos)
	}

	l.readChar() // 跳过结束引号

	return Token{Type: TokenValue, Value: sb.String(), Pos: pos}, nil
}

// readNumber 读取数字
func (l *Lexer) readNumber() (Token, error) {
	pos := l.pos
	var sb strings.Builder

	// 负号
	if l.ch == '-' {
		sb.WriteByte(l.ch)
		l.readChar()
	}

	// 整数部分
	for isDigit(l.ch) {
		sb.WriteByte(l.ch)
		l.readChar()
	}

	// 小数部分
	if l.ch == '.' && isDigit(l.peekChar()) {
		sb.WriteByte(l.ch)
		l.readChar()
		for isDigit(l.ch) {
			sb.WriteByte(l.ch)
			l.readChar()
		}
	}

	return Token{Type: TokenValue, Value: sb.String(), Pos: pos}, nil
}

// isLetter 是否为字母
func isLetter(ch byte) bool {
	return unicode.IsLetter(rune(ch))
}

// isDigit 是否为数字
func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}
