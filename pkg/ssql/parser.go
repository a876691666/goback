package ssql

import (
	"fmt"
	"strconv"
	"strings"
)

// Parser 语法分析器
type Parser struct {
	tokens []Token
	pos    int
}

// NewParser 创建语法分析器
func NewParser(tokens []Token) *Parser {
	return &Parser{
		tokens: tokens,
		pos:    0,
	}
}

// Parse 解析SSQL字符串
func Parse(input string) (Expression, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, nil
	}

	lexer := NewLexer(input)
	tokens, err := lexer.Tokenize()
	if err != nil {
		return nil, fmt.Errorf("lexer error: %w", err)
	}

	parser := NewParser(tokens)
	return parser.Parse()
}

// Parse 解析
func (p *Parser) Parse() (Expression, error) {
	return p.parseExpression()
}

// current 获取当前token
func (p *Parser) current() Token {
	if p.pos >= len(p.tokens) {
		return Token{Type: TokenEOF}
	}
	return p.tokens[p.pos]
}

// peek 查看下一个token
func (p *Parser) peek() Token {
	if p.pos+1 >= len(p.tokens) {
		return Token{Type: TokenEOF}
	}
	return p.tokens[p.pos+1]
}

// advance 前进到下一个token
func (p *Parser) advance() Token {
	token := p.current()
	p.pos++
	return token
}

// expect 期望指定类型的token
func (p *Parser) expect(tokenType TokenType) (Token, error) {
	token := p.current()
	if token.Type != tokenType {
		return Token{}, fmt.Errorf("expected token type %d, got %d at position %d", tokenType, token.Type, token.Pos)
	}
	p.advance()
	return token, nil
}

// parseExpression 解析表达式
func (p *Parser) parseExpression() (Expression, error) {
	return p.parseOr()
}

// parseOr 解析OR表达式
func (p *Parser) parseOr() (Expression, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}

	for p.current().Type == TokenLogicOr {
		p.advance() // 跳过 ||

		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}

		// 合并OR表达式
		if logicExpr, ok := left.(*LogicExpression); ok && logicExpr.Logic == LogicOr {
			logicExpr.Expressions = append(logicExpr.Expressions, right)
		} else {
			left = &LogicExpression{
				Logic:       LogicOr,
				Expressions: []Expression{left, right},
			}
		}
	}

	return left, nil
}

// parseAnd 解析AND表达式
func (p *Parser) parseAnd() (Expression, error) {
	left, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}

	for p.current().Type == TokenLogicAnd {
		p.advance() // 跳过 &&

		right, err := p.parsePrimary()
		if err != nil {
			return nil, err
		}

		// 合并AND表达式
		if logicExpr, ok := left.(*LogicExpression); ok && logicExpr.Logic == LogicAnd {
			logicExpr.Expressions = append(logicExpr.Expressions, right)
		} else {
			left = &LogicExpression{
				Logic:       LogicAnd,
				Expressions: []Expression{left, right},
			}
		}
	}

	return left, nil
}

// parsePrimary 解析基本表达式
func (p *Parser) parsePrimary() (Expression, error) {
	token := p.current()

	switch token.Type {
	case TokenLParen:
		return p.parseGroup()
	case TokenField:
		return p.parseField()
	case TokenEOF:
		return nil, nil
	default:
		return nil, fmt.Errorf("unexpected token '%s' at position %d", token.Value, token.Pos)
	}
}

// parseGroup 解析分组表达式
func (p *Parser) parseGroup() (Expression, error) {
	p.advance() // 跳过 (

	expr, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	if _, err := p.expect(TokenRParen); err != nil {
		return nil, fmt.Errorf("expected ')' to close group expression")
	}

	return &GroupExpression{Inner: expr}, nil
}

// parseField 解析字段表达式
func (p *Parser) parseField() (Expression, error) {
	fieldToken := p.advance()

	// 期望操作符
	opToken, err := p.expect(TokenOperator)
	if err != nil {
		return nil, fmt.Errorf("expected operator after field '%s'", fieldToken.Value)
	}

	operator := parseOperator(opToken.Value)

	// 处理特殊操作符
	if operator == OpIsNull || operator == OpNotNull {
		return &FieldExpression{
			Field:    fieldToken.Value,
			Operator: operator,
			Value:    nil,
		}, nil
	}

	// 解析值
	value, err := p.parseValue(operator)
	if err != nil {
		return nil, err
	}

	return &FieldExpression{
		Field:    fieldToken.Value,
		Operator: operator,
		Value:    value,
	}, nil
}

// parseValue 解析值
func (p *Parser) parseValue(operator Operator) (interface{}, error) {
	// 对于IN和NOT IN操作符,解析数组
	if operator == OpIn || operator == OpNotIn || operator == OpBetween {
		return p.parseArray()
	}

	token := p.current()
	if token.Type != TokenValue {
		return nil, fmt.Errorf("expected value at position %d, got '%s'", token.Pos, token.Value)
	}
	p.advance()

	return convertValue(token.Value), nil
}

// parseArray 解析数组
func (p *Parser) parseArray() ([]interface{}, error) {
	if _, err := p.expect(TokenLBracket); err != nil {
		return nil, fmt.Errorf("expected '[' for array value")
	}

	var values []interface{}

	for p.current().Type != TokenRBracket && p.current().Type != TokenEOF {
		if p.current().Type == TokenValue {
			values = append(values, convertValue(p.current().Value))
			p.advance()
		}

		if p.current().Type == TokenComma {
			p.advance()
		}
	}

	if _, err := p.expect(TokenRBracket); err != nil {
		return nil, fmt.Errorf("expected ']' to close array")
	}

	return values, nil
}

// parseOperator 解析操作符
func parseOperator(value string) Operator {
	switch value {
	case "=":
		return OpEq
	case "!=":
		return OpNeq
	case ">":
		return OpGt
	case ">=":
		return OpGte
	case "<":
		return OpLt
	case "<=":
		return OpLte
	case "~":
		return OpLike
	case "!~":
		return OpNotLike
	case "?=":
		return OpIn
	case "?!=":
		return OpNotIn
	case "?null":
		return OpIsNull
	case "?!null":
		return OpNotNull
	case "><":
		return OpBetween
	default:
		return OpEq
	}
}

// convertValue 转换值的类型
func convertValue(value string) interface{} {
	// 尝试转换为布尔值
	lower := strings.ToLower(value)
	if lower == "true" {
		return true
	}
	if lower == "false" {
		return false
	}
	if lower == "null" {
		return nil
	}

	// 尝试转换为整数
	if i, err := strconv.ParseInt(value, 10, 64); err == nil {
		return i
	}

	// 尝试转换为浮点数
	if f, err := strconv.ParseFloat(value, 64); err == nil {
		return f
	}

	// 返回字符串
	return value
}

// Build 构建SSQL字符串
func Build(expr Expression) string {
	if expr == nil {
		return ""
	}
	return expr.String()
}

// ToSQL 将SSQL转换为SQL
func ToSQL(ssqlStr string, dialect Dialect) (string, []interface{}, error) {
	expr, err := Parse(ssqlStr)
	if err != nil {
		return "", nil, err
	}

	if expr == nil {
		return "", nil, nil
	}

	sql, args := expr.ToSQL(dialect)
	return sql, args, nil
}

// ToMySQLSQL 转换为MySQL SQL
func ToMySQLSQL(ssqlStr string) (string, []interface{}, error) {
	return ToSQL(ssqlStr, NewMySQLDialect())
}

// ToPostgreSQLSQL 转换为PostgreSQL SQL
func ToPostgreSQLSQL(ssqlStr string) (string, []interface{}, error) {
	return ToSQL(ssqlStr, NewPostgreSQLDialect())
}
