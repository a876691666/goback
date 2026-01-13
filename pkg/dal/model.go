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

// QueryOption 查询选项
type QueryOption func(*gorm.DB) *gorm.DB

func WithPreload(query string, args ...any) QueryOption {
	return func(db *gorm.DB) *gorm.DB { return db.Preload(query, args...) }
}

func WithOrder(order string) QueryOption {
	return func(db *gorm.DB) *gorm.DB { return db.Order(order) }
}

func WithSelect(fields ...string) QueryOption {
	return func(db *gorm.DB) *gorm.DB { return db.Select(fields) }
}

func WithUnscoped() QueryOption {
	return func(db *gorm.DB) *gorm.DB { return db.Unscoped() }
}
