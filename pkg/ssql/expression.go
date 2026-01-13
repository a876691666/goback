package ssql

import (
	"fmt"
	"strings"
)

// Operator 比较操作符
type Operator string

const (
	OpEq       Operator = "="
	OpNeq      Operator = "!="
	OpGt       Operator = ">"
	OpGte      Operator = ">="
	OpLt       Operator = "<"
	OpLte      Operator = "<="
	OpLike     Operator = "~"
	OpNotLike  Operator = "!~"
	OpIn       Operator = "?="
	OpNotIn    Operator = "?!="
	OpIsNull   Operator = "?null"
	OpNotNull  Operator = "?!null"
	OpBetween  Operator = "><"
)

// LogicOperator 逻辑操作符
type LogicOperator string

const (
	LogicAnd LogicOperator = "&&"
	LogicOr  LogicOperator = "||"
)

// Expression 表达式接口
type Expression interface {
	ToSQL(dialect Dialect) (string, []interface{})
	Validate() error
	String() string
}

// FieldExpression 字段表达式
type FieldExpression struct {
	Field    string
	Operator Operator
	Value    interface{}
}

// ToSQL 转换为SQL
func (e *FieldExpression) ToSQL(dialect Dialect) (string, []interface{}) {
	field := dialect.Quote(e.Field)
	
	switch e.Operator {
	case OpEq:
		return fmt.Sprintf("%s = %s", field, dialect.Placeholder(0)), []interface{}{e.Value}
	case OpNeq:
		return fmt.Sprintf("%s != %s", field, dialect.Placeholder(0)), []interface{}{e.Value}
	case OpGt:
		return fmt.Sprintf("%s > %s", field, dialect.Placeholder(0)), []interface{}{e.Value}
	case OpGte:
		return fmt.Sprintf("%s >= %s", field, dialect.Placeholder(0)), []interface{}{e.Value}
	case OpLt:
		return fmt.Sprintf("%s < %s", field, dialect.Placeholder(0)), []interface{}{e.Value}
	case OpLte:
		return fmt.Sprintf("%s <= %s", field, dialect.Placeholder(0)), []interface{}{e.Value}
	case OpLike:
		return fmt.Sprintf("%s LIKE %s", field, dialect.Placeholder(0)), []interface{}{"%" + fmt.Sprint(e.Value) + "%"}
	case OpNotLike:
		return fmt.Sprintf("%s NOT LIKE %s", field, dialect.Placeholder(0)), []interface{}{"%" + fmt.Sprint(e.Value) + "%"}
	case OpIn:
		values := toSlice(e.Value)
		placeholders := make([]string, len(values))
		for i := range values {
			placeholders[i] = dialect.Placeholder(i)
		}
		return fmt.Sprintf("%s IN (%s)", field, strings.Join(placeholders, ", ")), values
	case OpNotIn:
		values := toSlice(e.Value)
		placeholders := make([]string, len(values))
		for i := range values {
			placeholders[i] = dialect.Placeholder(i)
		}
		return fmt.Sprintf("%s NOT IN (%s)", field, strings.Join(placeholders, ", ")), values
	case OpIsNull:
		return fmt.Sprintf("%s IS NULL", field), nil
	case OpNotNull:
		return fmt.Sprintf("%s IS NOT NULL", field), nil
	case OpBetween:
		values := toSlice(e.Value)
		if len(values) >= 2 {
			return fmt.Sprintf("%s BETWEEN %s AND %s", field, dialect.Placeholder(0), dialect.Placeholder(1)), values[:2]
		}
		return "", nil
	default:
		return fmt.Sprintf("%s = %s", field, dialect.Placeholder(0)), []interface{}{e.Value}
	}
}

// Validate 验证表达式
func (e *FieldExpression) Validate() error {
	if e.Field == "" {
		return fmt.Errorf("field name is required")
	}
	if e.Operator == OpIn || e.Operator == OpNotIn || e.Operator == OpBetween {
		values := toSlice(e.Value)
		if len(values) == 0 {
			return fmt.Errorf("value array is required for operator %s", e.Operator)
		}
		if e.Operator == OpBetween && len(values) < 2 {
			return fmt.Errorf("between operator requires 2 values")
		}
	}
	return nil
}

// String 转换为字符串表示
func (e *FieldExpression) String() string {
	switch e.Operator {
	case OpIn, OpNotIn:
		values := toSlice(e.Value)
		strValues := make([]string, len(values))
		for i, v := range values {
			strValues[i] = formatValue(v)
		}
		return fmt.Sprintf("%s %s [%s]", e.Field, e.Operator, strings.Join(strValues, ", "))
	case OpIsNull, OpNotNull:
		return fmt.Sprintf("%s %s", e.Field, e.Operator)
	default:
		return fmt.Sprintf("%s %s %s", e.Field, e.Operator, formatValue(e.Value))
	}
}

// LogicExpression 逻辑表达式
type LogicExpression struct {
	Logic       LogicOperator
	Expressions []Expression
}

// ToSQL 转换为SQL
func (e *LogicExpression) ToSQL(dialect Dialect) (string, []interface{}) {
	if len(e.Expressions) == 0 {
		return "", nil
	}

	if len(e.Expressions) == 1 {
		return e.Expressions[0].ToSQL(dialect)
	}

	parts := make([]string, 0, len(e.Expressions))
	args := make([]interface{}, 0)

	for _, expr := range e.Expressions {
		sql, exprArgs := expr.ToSQL(dialect)
		if sql != "" {
			parts = append(parts, sql)
			args = append(args, exprArgs...)
		}
	}

	if len(parts) == 0 {
		return "", nil
	}

	if len(parts) == 1 {
		return parts[0], args
	}

	connector := " AND "
	if e.Logic == LogicOr {
		connector = " OR "
	}

	return "(" + strings.Join(parts, connector) + ")", args
}

// Validate 验证表达式
func (e *LogicExpression) Validate() error {
	for _, expr := range e.Expressions {
		if err := expr.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// String 转换为字符串表示
func (e *LogicExpression) String() string {
	if len(e.Expressions) == 0 {
		return ""
	}

	parts := make([]string, len(e.Expressions))
	for i, expr := range e.Expressions {
		parts[i] = expr.String()
	}

	connector := " && "
	if e.Logic == LogicOr {
		connector = " || "
	}

	if len(parts) == 1 {
		return parts[0]
	}

	return "(" + strings.Join(parts, connector) + ")"
}

// GroupExpression 分组表达式(括号)
type GroupExpression struct {
	Inner Expression
}

// ToSQL 转换为SQL
func (e *GroupExpression) ToSQL(dialect Dialect) (string, []interface{}) {
	if e.Inner == nil {
		return "", nil
	}
	sql, args := e.Inner.ToSQL(dialect)
	if sql == "" {
		return "", nil
	}
	return "(" + sql + ")", args
}

// Validate 验证表达式
func (e *GroupExpression) Validate() error {
	if e.Inner == nil {
		return fmt.Errorf("group expression inner is nil")
	}
	return e.Inner.Validate()
}

// String 转换为字符串表示
func (e *GroupExpression) String() string {
	if e.Inner == nil {
		return ""
	}
	return "(" + e.Inner.String() + ")"
}

// toSlice 转换为切片
func toSlice(value interface{}) []interface{} {
	switch v := value.(type) {
	case []interface{}:
		return v
	case []string:
		result := make([]interface{}, len(v))
		for i, s := range v {
			result[i] = s
		}
		return result
	case []int:
		result := make([]interface{}, len(v))
		for i, n := range v {
			result[i] = n
		}
		return result
	case []int64:
		result := make([]interface{}, len(v))
		for i, n := range v {
			result[i] = n
		}
		return result
	case []float64:
		result := make([]interface{}, len(v))
		for i, n := range v {
			result[i] = n
		}
		return result
	default:
		return []interface{}{value}
	}
}

// formatValue 格式化值为字符串
func formatValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		return fmt.Sprintf("'%s'", v)
	case nil:
		return "null"
	default:
		return fmt.Sprint(v)
	}
}
