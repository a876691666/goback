package ssql

import "fmt"

// Dialect SQL方言接口
type Dialect interface {
	Quote(field string) string
	Placeholder(index int) string
	OperatorSQL(op Operator) string
}

// MySQLDialect MySQL方言
type MySQLDialect struct{}

// NewMySQLDialect 创建MySQL方言
func NewMySQLDialect() *MySQLDialect {
	return &MySQLDialect{}
}

// Quote 引用字段名
func (d *MySQLDialect) Quote(field string) string {
	return "`" + field + "`"
}

// Placeholder 占位符
func (d *MySQLDialect) Placeholder(index int) string {
	return "?"
}

// OperatorSQL 操作符SQL
func (d *MySQLDialect) OperatorSQL(op Operator) string {
	switch op {
	case OpEq:
		return "="
	case OpNeq:
		return "!="
	case OpGt:
		return ">"
	case OpGte:
		return ">="
	case OpLt:
		return "<"
	case OpLte:
		return "<="
	case OpLike:
		return "LIKE"
	case OpNotLike:
		return "NOT LIKE"
	case OpIn:
		return "IN"
	case OpNotIn:
		return "NOT IN"
	case OpIsNull:
		return "IS NULL"
	case OpNotNull:
		return "IS NOT NULL"
	case OpBetween:
		return "BETWEEN"
	default:
		return "="
	}
}

// PostgreSQLDialect PostgreSQL方言
type PostgreSQLDialect struct {
	paramIndex int
}

// NewPostgreSQLDialect 创建PostgreSQL方言
func NewPostgreSQLDialect() *PostgreSQLDialect {
	return &PostgreSQLDialect{paramIndex: 0}
}

// Quote 引用字段名
func (d *PostgreSQLDialect) Quote(field string) string {
	return "\"" + field + "\""
}

// Placeholder 占位符
func (d *PostgreSQLDialect) Placeholder(index int) string {
	d.paramIndex++
	return fmt.Sprintf("$%d", d.paramIndex)
}

// OperatorSQL 操作符SQL
func (d *PostgreSQLDialect) OperatorSQL(op Operator) string {
	switch op {
	case OpLike:
		return "ILIKE" // PostgreSQL不区分大小写的LIKE
	default:
		return (&MySQLDialect{}).OperatorSQL(op)
	}
}

// SQLiteDialect SQLite方言
type SQLiteDialect struct{}

// NewSQLiteDialect 创建SQLite方言
func NewSQLiteDialect() *SQLiteDialect {
	return &SQLiteDialect{}
}

// Quote 引用字段名
func (d *SQLiteDialect) Quote(field string) string {
	return "\"" + field + "\""
}

// Placeholder 占位符
func (d *SQLiteDialect) Placeholder(index int) string {
	return "?"
}

// OperatorSQL 操作符SQL
func (d *SQLiteDialect) OperatorSQL(op Operator) string {
	return (&MySQLDialect{}).OperatorSQL(op)
}

// GetDialect 根据驱动名获取方言
func GetDialect(driver string) Dialect {
	switch driver {
	case "mysql":
		return NewMySQLDialect()
	case "postgres", "postgresql":
		return NewPostgreSQLDialect()
	case "sqlite", "sqlite3":
		return NewSQLiteDialect()
	default:
		return NewMySQLDialect()
	}
}
