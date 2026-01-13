package dal

import (
	"context"
	"strings"

	"github.com/goback/pkg/ssql"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// ListParams 列表查询参数（类似 PocketBase）
type ListParams struct {
	Filter    string   `query:"filter"`    // SSQL 过滤条件 例如: created >= "2022-01-01" && status = 1
	Fields    string   `query:"fields"`    // 返回字段 例如: id,name,created
	Sort      string   `query:"sort"`      // 排序 例如: -created,name (- 表示降序)
	Page      int      `query:"page"`      // 页码，从1开始
	PerPage   int      `query:"perPage"`   // 每页数量
	SkipTotal bool     `query:"skipTotal"` // 是否跳过总数统计（提升性能）
	Expand    string   `query:"expand"`    // 关联预加载 例如: user,role.permissions
}

// ListResult 列表查询结果
type ListResult[T any] struct {
	Page       int   `json:"page"`
	PerPage    int   `json:"perPage"`
	TotalItems int64 `json:"totalItems"`
	TotalPages int   `json:"totalPages"`
	Items      []T   `json:"items"`
}

// Collection 集合查询器（类似 PocketBase 的 collection API）
type Collection[T any] struct {
	db         *gorm.DB
	fieldAlias map[string]string // 字段别名映射 (前端字段名 -> 数据库字段名)
	allowedFields []string       // 允许查询的字段白名单
	defaultSort   string         // 默认排序
	maxPerPage    int            // 最大每页数量
}

// NewCollection 创建集合查询器
func NewCollection[T any](db *gorm.DB) *Collection[T] {
	return &Collection[T]{
		db:         db,
		fieldAlias: make(map[string]string),
		allowedFields: []string{},
		defaultSort:   "-id",
		maxPerPage:    500,
	}
}

// WithFieldAlias 设置字段别名映射
func (c *Collection[T]) WithFieldAlias(alias map[string]string) *Collection[T] {
	c.fieldAlias = alias
	return c
}

// WithAllowedFields 设置允许查询的字段白名单
func (c *Collection[T]) WithAllowedFields(fields []string) *Collection[T] {
	c.allowedFields = fields
	return c
}

// WithDefaultSort 设置默认排序
func (c *Collection[T]) WithDefaultSort(sort string) *Collection[T] {
	c.defaultSort = sort
	return c
}

// WithMaxPerPage 设置最大每页数量
func (c *Collection[T]) WithMaxPerPage(max int) *Collection[T] {
	c.maxPerPage = max
	return c
}

// BindQuery 从 Fiber 上下文绑定查询参数
func BindQuery(ctx *fiber.Ctx) (*ListParams, error) {
	params := &ListParams{
		Page:    1,
		PerPage: 20,
	}
	if err := ctx.QueryParser(params); err != nil {
		return nil, err
	}
	return params, nil
}

// GetList 获取分页列表
func (c *Collection[T]) GetList(ctx context.Context, params *ListParams) (*ListResult[T], error) {
	var items []T
	var totalItems int64

	// 规范化参数
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PerPage < 1 {
		params.PerPage = 20
	}
	if params.PerPage > c.maxPerPage {
		params.PerPage = c.maxPerPage
	}

	var entity T
	db := c.db.WithContext(ctx).Model(&entity)

	// 应用过滤条件
	db = c.applyFilter(db, params.Filter)

	// 应用字段选择
	db = c.applyFields(db, params.Fields)

	// 应用关联预加载
	db = c.applyExpand(db, params.Expand)

	// 统计总数（除非跳过）
	if !params.SkipTotal {
		countDB := c.db.WithContext(ctx).Model(&entity)
		countDB = c.applyFilter(countDB, params.Filter)
		if err := countDB.Count(&totalItems).Error; err != nil {
			return nil, err
		}
	}

	// 应用排序
	db = c.applySort(db, params.Sort)

	// 应用分页
	offset := (params.Page - 1) * params.PerPage
	db = db.Offset(offset).Limit(params.PerPage)

	// 执行查询
	if err := db.Find(&items).Error; err != nil {
		return nil, err
	}

	// 计算总页数
	totalPages := 0
	if !params.SkipTotal && params.PerPage > 0 {
		totalPages = int((totalItems + int64(params.PerPage) - 1) / int64(params.PerPage))
	}

	return &ListResult[T]{
		Page:       params.Page,
		PerPage:    params.PerPage,
		TotalItems: totalItems,
		TotalPages: totalPages,
		Items:      items,
	}, nil
}

// GetFullList 获取全部列表（无分页）
func (c *Collection[T]) GetFullList(ctx context.Context, params *ListParams) ([]T, error) {
	var items []T

	var entity T
	db := c.db.WithContext(ctx).Model(&entity)

	// 应用过滤条件
	db = c.applyFilter(db, params.Filter)

	// 应用字段选择
	db = c.applyFields(db, params.Fields)

	// 应用关联预加载
	db = c.applyExpand(db, params.Expand)

	// 应用排序
	db = c.applySort(db, params.Sort)

	// 执行查询
	if err := db.Find(&items).Error; err != nil {
		return nil, err
	}

	return items, nil
}

// GetFirstListItem 获取第一条匹配记录
func (c *Collection[T]) GetFirstListItem(ctx context.Context, filter string, params *ListParams) (*T, error) {
	var entity T
	db := c.db.WithContext(ctx).Model(&entity)

	// 应用过滤条件
	if filter != "" {
		db = c.applyFilter(db, filter)
	}
	if params != nil && params.Filter != "" {
		db = c.applyFilter(db, params.Filter)
	}

	// 应用字段选择
	if params != nil {
		db = c.applyFields(db, params.Fields)
		db = c.applyExpand(db, params.Expand)
		db = c.applySort(db, params.Sort)
	}

	// 查询第一条
	if err := db.First(&entity).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &entity, nil
}

// GetOne 根据ID获取单条记录
func (c *Collection[T]) GetOne(ctx context.Context, id int64, params *ListParams) (*T, error) {
	var entity T
	db := c.db.WithContext(ctx).Model(&entity)

	// 应用字段选择
	if params != nil {
		db = c.applyFields(db, params.Fields)
		db = c.applyExpand(db, params.Expand)
	}

	if err := db.Where("id = ?", id).First(&entity).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &entity, nil
}

// applyFilter 应用 SSQL 过滤条件
func (c *Collection[T]) applyFilter(db *gorm.DB, filter string) *gorm.DB {
	if filter == "" {
		return db
	}

	// 字段别名替换
	filter = c.replaceFieldAlias(filter)

	// 解析 SSQL
	expr, err := ssql.Parse(filter)
	if err != nil {
		return db // 解析失败则忽略过滤条件
	}

	// 转换为 SQL
	dialect := ssql.NewMySQLDialect()
	sql, args := expr.ToSQL(dialect)

	if sql != "" {
		db = db.Where(sql, args...)
	}

	return db
}

// applyFields 应用字段选择
func (c *Collection[T]) applyFields(db *gorm.DB, fields string) *gorm.DB {
	if fields == "" {
		return db
	}

	fieldList := strings.Split(fields, ",")
	selectFields := make([]string, 0, len(fieldList))

	for _, f := range fieldList {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}

		// 检查字段白名单
		if len(c.allowedFields) > 0 {
			allowed := false
			for _, af := range c.allowedFields {
				if f == af {
					allowed = true
					break
				}
			}
			if !allowed {
				continue
			}
		}

		// 字段别名替换
		if alias, ok := c.fieldAlias[f]; ok {
			f = alias
		}

		selectFields = append(selectFields, f)
	}

	if len(selectFields) > 0 {
		db = db.Select(selectFields)
	}

	return db
}

// applySort 应用排序
func (c *Collection[T]) applySort(db *gorm.DB, sort string) *gorm.DB {
	if sort == "" {
		sort = c.defaultSort
	}

	sortFields := strings.Split(sort, ",")

	for _, s := range sortFields {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}

		order := "ASC"
		if strings.HasPrefix(s, "-") {
			order = "DESC"
			s = s[1:]
		} else if strings.HasPrefix(s, "+") {
			s = s[1:]
		}

		// 字段别名替换
		if alias, ok := c.fieldAlias[s]; ok {
			s = alias
		}

		db = db.Order(s + " " + order)
	}

	return db
}

// applyExpand 应用关联预加载
func (c *Collection[T]) applyExpand(db *gorm.DB, expand string) *gorm.DB {
	if expand == "" {
		return db
	}

	relations := strings.Split(expand, ",")
	for _, rel := range relations {
		rel = strings.TrimSpace(rel)
		if rel == "" {
			continue
		}
		// 处理嵌套关联 如 role.permissions
		db = db.Preload(rel)
	}

	return db
}

// replaceFieldAlias 替换字段别名
func (c *Collection[T]) replaceFieldAlias(filter string) string {
	for alias, realField := range c.fieldAlias {
		// 简单的字符串替换，后续可以优化为更精确的替换
		filter = strings.ReplaceAll(filter, alias, realField)
	}
	return filter
}

// Count 统计数量
func (c *Collection[T]) Count(ctx context.Context, filter string) (int64, error) {
	var count int64
	var entity T
	db := c.db.WithContext(ctx).Model(&entity)

	db = c.applyFilter(db, filter)

	if err := db.Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}
