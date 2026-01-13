package dal

import (
	"context"
	"errors"

	"github.com/goback/pkg/database"
	"gorm.io/gorm"
)

// Repository 通用仓储接口
type Repository[T any] interface {
	Create(ctx context.Context, entity *T) error
	CreateBatch(ctx context.Context, entities []T) error
	Update(ctx context.Context, entity *T) error
	UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error
	Delete(ctx context.Context, id int64) error
	DeleteBatch(ctx context.Context, ids []int64) error
	FindByID(ctx context.Context, id int64, opts ...QueryOption) (*T, error)
	FindOne(ctx context.Context, conditions map[string]interface{}, opts ...QueryOption) (*T, error)
	FindAll(ctx context.Context, conditions map[string]interface{}, opts ...QueryOption) ([]T, error)
	FindPaged(ctx context.Context, conditions map[string]interface{}, pagination *Pagination, opts ...QueryOption) (*PagedResult[T], error)
	Count(ctx context.Context, conditions map[string]interface{}) (int64, error)
	Exists(ctx context.Context, conditions map[string]interface{}) (bool, error)
	Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error
	DB() *gorm.DB
}

// BaseRepository 基础仓储实现
type BaseRepository[T any] struct {
	db *gorm.DB
}

// NewBaseRepository 创建基础仓储
func NewBaseRepository[T any]() *BaseRepository[T] {
	return &BaseRepository[T]{
		db: database.Get(),
	}
}

// NewBaseRepositoryWithDB 使用指定DB创建基础仓储
func NewBaseRepositoryWithDB[T any](db *gorm.DB) *BaseRepository[T] {
	return &BaseRepository[T]{
		db: db,
	}
}

// DB 获取数据库实例
func (r *BaseRepository[T]) DB() *gorm.DB {
	return r.db
}

// Create 创建实体
func (r *BaseRepository[T]) Create(ctx context.Context, entity *T) error {
	return r.db.WithContext(ctx).Create(entity).Error
}

// CreateBatch 批量创建
func (r *BaseRepository[T]) CreateBatch(ctx context.Context, entities []T) error {
	return r.db.WithContext(ctx).CreateInBatches(entities, 100).Error
}

// Update 更新实体
func (r *BaseRepository[T]) Update(ctx context.Context, entity *T) error {
	return r.db.WithContext(ctx).Save(entity).Error
}

// UpdateFields 更新指定字段
func (r *BaseRepository[T]) UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	var entity T
	return r.db.WithContext(ctx).Model(&entity).Where("id = ?", id).Updates(fields).Error
}

// Delete 删除实体(软删除)
func (r *BaseRepository[T]) Delete(ctx context.Context, id int64) error {
	var entity T
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&entity).Error
}

// DeleteBatch 批量删除
func (r *BaseRepository[T]) DeleteBatch(ctx context.Context, ids []int64) error {
	var entity T
	return r.db.WithContext(ctx).Where("id IN ?", ids).Delete(&entity).Error
}

// HardDelete 硬删除
func (r *BaseRepository[T]) HardDelete(ctx context.Context, id int64) error {
	var entity T
	return r.db.WithContext(ctx).Unscoped().Where("id = ?", id).Delete(&entity).Error
}

// FindByID 根据ID查找
func (r *BaseRepository[T]) FindByID(ctx context.Context, id int64, opts ...QueryOption) (*T, error) {
	var entity T
	db := r.db.WithContext(ctx)

	for _, opt := range opts {
		db = opt(db)
	}

	if err := db.Where("id = ?", id).First(&entity).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &entity, nil
}

// FindOne 查找单个实体
func (r *BaseRepository[T]) FindOne(ctx context.Context, conditions map[string]interface{}, opts ...QueryOption) (*T, error) {
	var entity T
	db := r.db.WithContext(ctx)

	for _, opt := range opts {
		db = opt(db)
	}

	if err := db.Where(conditions).First(&entity).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &entity, nil
}

// FindAll 查找所有符合条件的实体
func (r *BaseRepository[T]) FindAll(ctx context.Context, conditions map[string]interface{}, opts ...QueryOption) ([]T, error) {
	var entities []T
	db := r.db.WithContext(ctx)

	for _, opt := range opts {
		db = opt(db)
	}

	if err := db.Where(conditions).Find(&entities).Error; err != nil {
		return nil, err
	}
	return entities, nil
}

// FindPaged 分页查询
func (r *BaseRepository[T]) FindPaged(ctx context.Context, conditions map[string]interface{}, pagination *Pagination, opts ...QueryOption) (*PagedResult[T], error) {
	var entities []T
	var total int64
	var entity T

	db := r.db.WithContext(ctx).Model(&entity)

	// 应用查询选项
	for _, opt := range opts {
		db = opt(db)
	}

	// 应用条件
	if len(conditions) > 0 {
		db = db.Where(conditions)
	}

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

// FindPagedWithQuery 使用自定义查询的分页查询
func (r *BaseRepository[T]) FindPagedWithQuery(ctx context.Context, query *gorm.DB, pagination *Pagination) (*PagedResult[T], error) {
	var entities []T
	var total int64

	// 计算总数
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// 分页查询
	if err := query.Offset(pagination.Offset()).Limit(pagination.PageSize).Find(&entities).Error; err != nil {
		return nil, err
	}

	return NewPagedResult(entities, total, pagination), nil
}

// Count 统计数量
func (r *BaseRepository[T]) Count(ctx context.Context, conditions map[string]interface{}) (int64, error) {
	var count int64
	var entity T

	db := r.db.WithContext(ctx).Model(&entity)
	if len(conditions) > 0 {
		db = db.Where(conditions)
	}

	if err := db.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// Exists 检查是否存在
func (r *BaseRepository[T]) Exists(ctx context.Context, conditions map[string]interface{}) (bool, error) {
	count, err := r.Count(ctx, conditions)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// Transaction 执行事务
func (r *BaseRepository[T]) Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return r.db.WithContext(ctx).Transaction(fn)
}

// Raw 执行原生SQL查询
func (r *BaseRepository[T]) Raw(ctx context.Context, sql string, values ...interface{}) *gorm.DB {
	return r.db.WithContext(ctx).Raw(sql, values...)
}

// Exec 执行原生SQL
func (r *BaseRepository[T]) Exec(ctx context.Context, sql string, values ...interface{}) error {
	return r.db.WithContext(ctx).Exec(sql, values...).Error
}
