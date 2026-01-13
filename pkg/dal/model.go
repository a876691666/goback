package dal

import (
	"time"

	"gorm.io/gorm"
)

// Model 基础模型
type Model struct {
	ID        int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deletedAt,omitempty"`
}

// ModelWithUser 带用户信息的基础模型
type ModelWithUser struct {
	Model
	CreatedBy int64 `gorm:"default:0" json:"createdBy"`
	UpdatedBy int64 `gorm:"default:0" json:"updatedBy"`
}

// Pagination 分页参数
type Pagination struct {
	Page     int   `json:"page" form:"page"`
	PageSize int   `json:"pageSize" form:"pageSize"`
	Total    int64 `json:"total"`
}

// NewPagination 创建分页参数
func NewPagination(page, pageSize int) *Pagination {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return &Pagination{
		Page:     page,
		PageSize: pageSize,
	}
}

// Offset 获取偏移量
func (p *Pagination) Offset() int {
	return (p.Page - 1) * p.PageSize
}

// PagedResult 分页结果
type PagedResult[T any] struct {
	List       []T   `json:"list"`
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	PageSize   int   `json:"pageSize"`
	TotalPages int   `json:"totalPages"`
}

// NewPagedResult 创建分页结果
func NewPagedResult[T any](list []T, total int64, pagination *Pagination) *PagedResult[T] {
	totalPages := int(total) / pagination.PageSize
	if int(total)%pagination.PageSize > 0 {
		totalPages++
	}
	return &PagedResult[T]{
		List:       list,
		Total:      total,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalPages: totalPages,
	}
}

// QueryOption 查询选项
type QueryOption func(*gorm.DB) *gorm.DB

// WithPreload 预加载关联
func WithPreload(query string, args ...interface{}) QueryOption {
	return func(db *gorm.DB) *gorm.DB {
		return db.Preload(query, args...)
	}
}

// WithSelect 选择字段
func WithSelect(fields ...string) QueryOption {
	return func(db *gorm.DB) *gorm.DB {
		return db.Select(fields)
	}
}

// WithOmit 排除字段
func WithOmit(fields ...string) QueryOption {
	return func(db *gorm.DB) *gorm.DB {
		return db.Omit(fields...)
	}
}

// WithOrder 排序
func WithOrder(order string) QueryOption {
	return func(db *gorm.DB) *gorm.DB {
		return db.Order(order)
	}
}

// WithJoin 连接查询
func WithJoin(query string, args ...interface{}) QueryOption {
	return func(db *gorm.DB) *gorm.DB {
		return db.Joins(query, args...)
	}
}

// WithGroup 分组
func WithGroup(name string) QueryOption {
	return func(db *gorm.DB) *gorm.DB {
		return db.Group(name)
	}
}

// WithHaving Having条件
func WithHaving(query interface{}, args ...interface{}) QueryOption {
	return func(db *gorm.DB) *gorm.DB {
		return db.Having(query, args...)
	}
}

// WithDistinct 去重
func WithDistinct(args ...interface{}) QueryOption {
	return func(db *gorm.DB) *gorm.DB {
		return db.Distinct(args...)
	}
}

// WithUnscoped 包含软删除数据
func WithUnscoped() QueryOption {
	return func(db *gorm.DB) *gorm.DB {
		return db.Unscoped()
	}
}

// SortOrder 排序方向
type SortOrder string

// 排序方向常量
const (
	SortAsc  SortOrder = "ASC"
	SortDesc SortOrder = "DESC"
)

// Sort 排序参数
type Sort struct {
	Field string    `json:"field" form:"field"`
	Order SortOrder `json:"order" form:"order"`
}

// ToOrderString 转换为排序字符串
func (s *Sort) ToOrderString() string {
	if s.Field == "" {
		return ""
	}
	order := s.Order
	if order == "" {
		order = SortDesc
	}
	return s.Field + " " + string(order)
}
