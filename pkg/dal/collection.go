package dal

import (
	"strconv"
	"strings"
	"sync"

	"github.com/goback/pkg/ssql"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// 全局数据库实例
var (
	globalDB *gorm.DB
	dbMutex  sync.RWMutex
)

// SetDB 设置全局数据库（应用启动时调用）
func SetDB(db *gorm.DB) {
	dbMutex.Lock()
	defer dbMutex.Unlock()
	globalDB = db
}

// GetDB 获取全局数据库
func GetDB() *gorm.DB {
	dbMutex.RLock()
	defer dbMutex.RUnlock()
	return globalDB
}

// ListParams 列表查询参数
type ListParams struct {
	Filter    string `query:"filter"`    // SSQL 过滤条件
	Fields    string `query:"fields"`    // 返回字段
	Sort      string `query:"sort"`      // 排序 (-created 降序)
	Page      int    `query:"page"`      // 页码
	PerPage   int    `query:"perPage"`   // 每页数量
	SkipTotal bool   `query:"skipTotal"` // 跳过总数统计
	Expand    string `query:"expand"`    // 关联预加载
}

// ListResult 列表查询结果
type ListResult[T any] struct {
	Page       int   `json:"page"`
	PerPage    int   `json:"perPage"`
	TotalItems int64 `json:"totalItems"`
	TotalPages int   `json:"totalPages"`
	Items      []T   `json:"items"`
}

// Collection Model 的数据访问基类
type Collection[T any] struct {
	FieldAlias  map[string]string
	DefaultSort string
	MaxPerPage  int
}

// DB 获取数据库实例
func (c *Collection[T]) DB() *gorm.DB {
	return GetDB()
}

// ========== 工具函数 ==========

// BindQuery 绑定查询参数
func BindQuery(ctx *fiber.Ctx) (*ListParams, error) {
	params := &ListParams{Page: 1, PerPage: 20}
	if err := ctx.QueryParser(params); err != nil {
		return nil, err
	}
	return params, nil
}

// ParseInt64IDs 解析逗号分隔的 int64 ID 列表
func ParseInt64IDs(s string) ([]int64, error) {
	parts := strings.Split(s, ",")
	ids := make([]int64, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			id, err := strconv.ParseInt(p, 10, 64)
			if err != nil {
				return nil, err
			}
			ids = append(ids, id)
		}
	}
	return ids, nil
}

// ParseInt64ID 解析单个 int64 ID
func ParseInt64ID(s string) (int64, error) {
	return strconv.ParseInt(strings.TrimSpace(s), 10, 64)
}

// ========== CREATE ==========

// Create 创建单条记录
func (c *Collection[T]) Create(row *T) error {
	return c.DB().Create(row).Error
}

// CreateByFilter 根据 SSQL 过滤条件批量创建（实际上是批量插入）
func (c *Collection[T]) CreateBatch(rows []T, batchSize ...int) error {
	size := 100
	if len(batchSize) > 0 {
		size = batchSize[0]
	}
	return c.DB().CreateInBatches(rows, size).Error
}

// ========== UPDATE ==========

// UpdateByID 根据 ID 更新单条记录
func (c *Collection[T]) UpdateByID(id int64, row map[string]any) error {
	var entity T
	return c.DB().Model(&entity).Where("id = ?", id).Updates(row).Error
}

// UpdateByFilter 根据 SSQL 过滤条件批量更新
func (c *Collection[T]) UpdateByFilter(filter string, rows map[string]any) (int64, error) {
	var entity T
	db := c.DB().Model(&entity)
	if filter != "" {
		db = c.applyFilter(db, filter)
	}
	result := db.Updates(rows)
	return result.RowsAffected, result.Error
}

// ========== DELETE ==========

// DeleteByID 根据 ID 删除单条记录
func (c *Collection[T]) DeleteByID(id int64) error {
	var entity T
	return c.DB().Where("id = ?", id).Delete(&entity).Error
}

// DeleteByFilter 根据 SSQL 过滤条件批量删除
func (c *Collection[T]) DeleteByFilter(filter string) (int64, error) {
	var entity T
	db := c.DB()
	if filter != "" {
		db = c.applyFilter(db, filter)
	}
	result := db.Delete(&entity)
	return result.RowsAffected, result.Error
}

// ========== TRUNCATE ==========

// Truncate 清空表
func (c *Collection[T]) Truncate() error {
	var entity T
	stmt := &gorm.Statement{DB: c.DB()}
	if err := stmt.Parse(&entity); err != nil {
		return err
	}
	return c.DB().Exec("TRUNCATE TABLE " + stmt.Schema.Table).Error
}

// ========== 列表查询 ==========

func (c *Collection[T]) GetOne(id int64) (*T, error) {
	var entity T
	result := c.DB().Where("id = ?", id).First(&entity)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, result.Error
	}
	return &entity, nil
}

func (c *Collection[T]) GetList(params *ListParams) (*ListResult[T], error) {
	var items []T
	var total int64

	c.normalizeParams(params)

	var entity T
	db := c.DB().Model(&entity)

	db = c.applyFilter(db, params.Filter)
	db = c.applyFields(db, params.Fields)
	db = c.applyExpand(db, params.Expand)

	if !params.SkipTotal {
		countDB := c.DB().Model(&entity)
		countDB = c.applyFilter(countDB, params.Filter)
		if err := countDB.Count(&total).Error; err != nil {
			return nil, err
		}
	}

	db = c.applySort(db, params.Sort)
	db = db.Offset((params.Page - 1) * params.PerPage).Limit(params.PerPage)

	if err := db.Find(&items).Error; err != nil {
		return nil, err
	}

	totalPages := 0
	if !params.SkipTotal && params.PerPage > 0 {
		totalPages = int((total + int64(params.PerPage) - 1) / int64(params.PerPage))
	}

	return &ListResult[T]{
		Page:       params.Page,
		PerPage:    params.PerPage,
		TotalItems: total,
		TotalPages: totalPages,
		Items:      items,
	}, nil
}

func (c *Collection[T]) GetFullList(params *ListParams) ([]T, error) {
	var items []T
	var entity T
	db := c.DB().Model(&entity)
	if params != nil {
		db = c.applyFilter(db, params.Filter)
		db = c.applyFields(db, params.Fields)
		db = c.applyExpand(db, params.Expand)
		db = c.applySort(db, params.Sort)
	} else {
		db = c.applySort(db, "")
	}
	return items, db.Find(&items).Error
}

// Count 根据 SSQL 过滤条件统计数量
func (c *Collection[T]) Count(params *ListParams) (int64, error) {
	var count int64
	var entity T
	db := c.DB().Model(&entity)
	if params != nil {
		db = c.applyFilter(db, params.Filter)
		db = c.applyFields(db, "")
	}
	return count, db.Count(&count).Error
}

func (c *Collection[T]) normalizeParams(p *ListParams) {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.PerPage < 1 {
		p.PerPage = 20
	}
	if p.PerPage > c.MaxPerPage {
		p.PerPage = c.MaxPerPage
	}
}

func (c *Collection[T]) applyFilter(db *gorm.DB, filter string) *gorm.DB {
	if filter == "" {
		return db
	}
	for alias, real := range c.FieldAlias {
		filter = strings.ReplaceAll(filter, alias, real)
	}
	expr, err := ssql.Parse(filter)
	if err != nil {
		return db
	}
	sql, args := expr.ToSQL(ssql.NewMySQLDialect())
	if sql != "" {
		db = db.Where(sql, args...)
	}
	return db
}

func (c *Collection[T]) applyFields(db *gorm.DB, fields string) *gorm.DB {
	if fields == "" {
		return db
	}
	var selected []string
	for _, f := range strings.Split(fields, ",") {
		if f = strings.TrimSpace(f); f != "" {
			if alias, ok := c.FieldAlias[f]; ok {
				f = alias
			}
			selected = append(selected, f)
		}
	}
	if len(selected) > 0 {
		db = db.Select(selected)
	}
	return db
}

func (c *Collection[T]) applySort(db *gorm.DB, sort string) *gorm.DB {
	if sort == "" {
		sort = c.DefaultSort
	}
	for _, s := range strings.Split(sort, ",") {
		if s = strings.TrimSpace(s); s == "" {
			continue
		}
		order := "ASC"
		if strings.HasPrefix(s, "-") {
			order, s = "DESC", s[1:]
		} else if strings.HasPrefix(s, "+") {
			s = s[1:]
		}
		if alias, ok := c.FieldAlias[s]; ok {
			s = alias
		}
		db = db.Order(s + " " + order)
	}
	return db
}

func (c *Collection[T]) applyExpand(db *gorm.DB, expand string) *gorm.DB {
	if expand == "" {
		return db
	}
	for _, rel := range strings.Split(expand, ",") {
		if rel = strings.TrimSpace(rel); rel != "" {
			db = db.Preload(rel)
		}
	}
	return db
}
