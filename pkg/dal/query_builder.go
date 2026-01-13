package dal

import (
	"context"

	"github.com/goback/pkg/ssql"
	"gorm.io/gorm"
)

// QueryBuilder 查询构建器
type QueryBuilder[T any] struct {
	db         *gorm.DB
	conditions []interface{}
	args       []interface{}
	orders     []string
	preloads   []string
	selects    []string
	omits      []string
	joins      []string
	groups     []string
	havings    []interface{}
	distinct   bool
	unscoped   bool
}

// NewQueryBuilder 创建查询构建器
func NewQueryBuilder[T any](db *gorm.DB) *QueryBuilder[T] {
	return &QueryBuilder[T]{
		db:         db,
		conditions: make([]interface{}, 0),
		args:       make([]interface{}, 0),
		orders:     make([]string, 0),
		preloads:   make([]string, 0),
		selects:    make([]string, 0),
		omits:      make([]string, 0),
		joins:      make([]string, 0),
		groups:     make([]string, 0),
		havings:    make([]interface{}, 0),
	}
}

// Where 添加条件
func (qb *QueryBuilder[T]) Where(query interface{}, args ...interface{}) *QueryBuilder[T] {
	qb.conditions = append(qb.conditions, query)
	qb.args = append(qb.args, args...)
	return qb
}

// WhereMap 添加Map条件
func (qb *QueryBuilder[T]) WhereMap(conditions map[string]interface{}) *QueryBuilder[T] {
	for k, v := range conditions {
		qb.conditions = append(qb.conditions, map[string]interface{}{k: v})
	}
	return qb
}

// WhereSSql 使用SSQL添加条件
func (qb *QueryBuilder[T]) WhereSSql(ssqlStr string) *QueryBuilder[T] {
	if ssqlStr == "" {
		return qb
	}
	
	expr, err := ssql.Parse(ssqlStr)
	if err != nil {
		return qb
	}
	
	dialect := ssql.NewMySQLDialect()
	sql, args := expr.ToSQL(dialect)
	
	if sql != "" {
		qb.conditions = append(qb.conditions, sql)
		qb.args = append(qb.args, args...)
	}
	
	return qb
}

// WhereIf 条件添加条件
func (qb *QueryBuilder[T]) WhereIf(condition bool, query interface{}, args ...interface{}) *QueryBuilder[T] {
	if condition {
		return qb.Where(query, args...)
	}
	return qb
}

// Order 添加排序
func (qb *QueryBuilder[T]) Order(order string) *QueryBuilder[T] {
	if order != "" {
		qb.orders = append(qb.orders, order)
	}
	return qb
}

// OrderBy 添加排序字段
func (qb *QueryBuilder[T]) OrderBy(field string, order SortOrder) *QueryBuilder[T] {
	if field != "" {
		qb.orders = append(qb.orders, field+" "+string(order))
	}
	return qb
}

// Preload 添加预加载
func (qb *QueryBuilder[T]) Preload(query string) *QueryBuilder[T] {
	qb.preloads = append(qb.preloads, query)
	return qb
}

// Select 选择字段
func (qb *QueryBuilder[T]) Select(fields ...string) *QueryBuilder[T] {
	qb.selects = append(qb.selects, fields...)
	return qb
}

// Omit 排除字段
func (qb *QueryBuilder[T]) Omit(fields ...string) *QueryBuilder[T] {
	qb.omits = append(qb.omits, fields...)
	return qb
}

// Join 连接查询
func (qb *QueryBuilder[T]) Join(query string) *QueryBuilder[T] {
	qb.joins = append(qb.joins, query)
	return qb
}

// Group 分组
func (qb *QueryBuilder[T]) Group(name string) *QueryBuilder[T] {
	qb.groups = append(qb.groups, name)
	return qb
}

// Having Having条件
func (qb *QueryBuilder[T]) Having(query interface{}, args ...interface{}) *QueryBuilder[T] {
	qb.havings = append(qb.havings, query)
	qb.args = append(qb.args, args...)
	return qb
}

// Distinct 去重
func (qb *QueryBuilder[T]) Distinct() *QueryBuilder[T] {
	qb.distinct = true
	return qb
}

// Unscoped 包含软删除数据
func (qb *QueryBuilder[T]) Unscoped() *QueryBuilder[T] {
	qb.unscoped = true
	return qb
}

// Build 构建查询
func (qb *QueryBuilder[T]) Build(ctx context.Context) *gorm.DB {
	var entity T
	db := qb.db.WithContext(ctx).Model(&entity)

	// 应用软删除设置
	if qb.unscoped {
		db = db.Unscoped()
	}

	// 应用去重
	if qb.distinct {
		db = db.Distinct()
	}

	// 应用选择字段
	if len(qb.selects) > 0 {
		db = db.Select(qb.selects)
	}

	// 应用排除字段
	if len(qb.omits) > 0 {
		db = db.Omit(qb.omits...)
	}

	// 应用连接
	for _, join := range qb.joins {
		db = db.Joins(join)
	}

	// 应用条件
	argIndex := 0
	for _, cond := range qb.conditions {
		switch c := cond.(type) {
		case string:
			// 计算这个条件需要的参数数量
			argsNeeded := countPlaceholders(c)
			if argIndex+argsNeeded <= len(qb.args) {
				db = db.Where(c, qb.args[argIndex:argIndex+argsNeeded]...)
				argIndex += argsNeeded
			} else {
				db = db.Where(c)
			}
		case map[string]interface{}:
			db = db.Where(c)
		default:
			db = db.Where(cond)
		}
	}

	// 应用分组
	for _, group := range qb.groups {
		db = db.Group(group)
	}

	// 应用Having
	for _, having := range qb.havings {
		db = db.Having(having)
	}

	// 应用预加载
	for _, preload := range qb.preloads {
		db = db.Preload(preload)
	}

	// 应用排序
	for _, order := range qb.orders {
		db = db.Order(order)
	}

	return db
}

// countPlaceholders 计算SQL中的占位符数量
func countPlaceholders(sql string) int {
	count := 0
	for _, c := range sql {
		if c == '?' {
			count++
		}
	}
	return count
}

// Find 查询所有
func (qb *QueryBuilder[T]) Find(ctx context.Context) ([]T, error) {
	var entities []T
	if err := qb.Build(ctx).Find(&entities).Error; err != nil {
		return nil, err
	}
	return entities, nil
}

// First 查询第一条
func (qb *QueryBuilder[T]) First(ctx context.Context) (*T, error) {
	var entity T
	if err := qb.Build(ctx).First(&entity).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &entity, nil
}

// Count 统计数量
func (qb *QueryBuilder[T]) Count(ctx context.Context) (int64, error) {
	var count int64
	if err := qb.Build(ctx).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// Paged 分页查询
func (qb *QueryBuilder[T]) Paged(ctx context.Context, pagination *Pagination) (*PagedResult[T], error) {
	var entities []T
	var total int64

	db := qb.Build(ctx)

	// 计算总数
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}

	// 分页查询
	if err := db.Offset(pagination.Offset()).Limit(pagination.PageSize).Find(&entities).Error; err != nil {
		return nil, err
	}

	return NewPagedResult(entities, total, pagination), nil
}

// Update 更新
func (qb *QueryBuilder[T]) Update(ctx context.Context, fields map[string]interface{}) error {
	return qb.Build(ctx).Updates(fields).Error
}

// Delete 删除
func (qb *QueryBuilder[T]) Delete(ctx context.Context) error {
	var entity T
	return qb.Build(ctx).Delete(&entity).Error
}
