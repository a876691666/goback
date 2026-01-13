package ssql

// Builder SSQL构建器
type Builder struct {
	expressions []Expression
	logic       LogicOperator
}

// NewBuilder 创建构建器
func NewBuilder() *Builder {
	return &Builder{
		expressions: make([]Expression, 0),
		logic:       LogicAnd,
	}
}

// And 设置为AND逻辑
func (b *Builder) And() *Builder {
	b.logic = LogicAnd
	return b
}

// Or 设置为OR逻辑
func (b *Builder) Or() *Builder {
	b.logic = LogicOr
	return b
}

// Eq 等于
func (b *Builder) Eq(field string, value interface{}) *Builder {
	b.expressions = append(b.expressions, &FieldExpression{
		Field:    field,
		Operator: OpEq,
		Value:    value,
	})
	return b
}

// Neq 不等于
func (b *Builder) Neq(field string, value interface{}) *Builder {
	b.expressions = append(b.expressions, &FieldExpression{
		Field:    field,
		Operator: OpNeq,
		Value:    value,
	})
	return b
}

// Gt 大于
func (b *Builder) Gt(field string, value interface{}) *Builder {
	b.expressions = append(b.expressions, &FieldExpression{
		Field:    field,
		Operator: OpGt,
		Value:    value,
	})
	return b
}

// Gte 大于等于
func (b *Builder) Gte(field string, value interface{}) *Builder {
	b.expressions = append(b.expressions, &FieldExpression{
		Field:    field,
		Operator: OpGte,
		Value:    value,
	})
	return b
}

// Lt 小于
func (b *Builder) Lt(field string, value interface{}) *Builder {
	b.expressions = append(b.expressions, &FieldExpression{
		Field:    field,
		Operator: OpLt,
		Value:    value,
	})
	return b
}

// Lte 小于等于
func (b *Builder) Lte(field string, value interface{}) *Builder {
	b.expressions = append(b.expressions, &FieldExpression{
		Field:    field,
		Operator: OpLte,
		Value:    value,
	})
	return b
}

// Like 模糊匹配
func (b *Builder) Like(field string, value string) *Builder {
	b.expressions = append(b.expressions, &FieldExpression{
		Field:    field,
		Operator: OpLike,
		Value:    value,
	})
	return b
}

// NotLike 不匹配
func (b *Builder) NotLike(field string, value string) *Builder {
	b.expressions = append(b.expressions, &FieldExpression{
		Field:    field,
		Operator: OpNotLike,
		Value:    value,
	})
	return b
}

// In 在列表中
func (b *Builder) In(field string, values ...interface{}) *Builder {
	b.expressions = append(b.expressions, &FieldExpression{
		Field:    field,
		Operator: OpIn,
		Value:    values,
	})
	return b
}

// NotIn 不在列表中
func (b *Builder) NotIn(field string, values ...interface{}) *Builder {
	b.expressions = append(b.expressions, &FieldExpression{
		Field:    field,
		Operator: OpNotIn,
		Value:    values,
	})
	return b
}

// IsNull 为空
func (b *Builder) IsNull(field string) *Builder {
	b.expressions = append(b.expressions, &FieldExpression{
		Field:    field,
		Operator: OpIsNull,
	})
	return b
}

// NotNull 不为空
func (b *Builder) NotNull(field string) *Builder {
	b.expressions = append(b.expressions, &FieldExpression{
		Field:    field,
		Operator: OpNotNull,
	})
	return b
}

// Between 在范围内
func (b *Builder) Between(field string, start, end interface{}) *Builder {
	b.expressions = append(b.expressions, &FieldExpression{
		Field:    field,
		Operator: OpBetween,
		Value:    []interface{}{start, end},
	})
	return b
}

// Group 添加分组表达式
func (b *Builder) Group(fn func(*Builder)) *Builder {
	subBuilder := NewBuilder()
	fn(subBuilder)
	expr := subBuilder.Build()
	if expr != nil {
		b.expressions = append(b.expressions, &GroupExpression{Inner: expr})
	}
	return b
}

// Expr 添加子表达式
func (b *Builder) Expr(expr Expression) *Builder {
	if expr != nil {
		b.expressions = append(b.expressions, expr)
	}
	return b
}

// Build 构建表达式
func (b *Builder) Build() Expression {
	if len(b.expressions) == 0 {
		return nil
	}

	if len(b.expressions) == 1 {
		return b.expressions[0]
	}

	return &LogicExpression{
		Logic:       b.logic,
		Expressions: b.expressions,
	}
}

// String 转换为SSQL字符串
func (b *Builder) String() string {
	expr := b.Build()
	if expr == nil {
		return ""
	}
	return expr.String()
}

// ToSQL 转换为SQL
func (b *Builder) ToSQL(dialect Dialect) (string, []interface{}) {
	expr := b.Build()
	if expr == nil {
		return "", nil
	}
	return expr.ToSQL(dialect)
}

// ToMySQLSQL 转换为MySQL SQL
func (b *Builder) ToMySQLSQL() (string, []interface{}) {
	return b.ToSQL(NewMySQLDialect())
}

// Where 创建AND条件构建器
func Where() *Builder {
	return NewBuilder().And()
}

// WhereOr 创建OR条件构建器
func WhereOr() *Builder {
	return NewBuilder().Or()
}
