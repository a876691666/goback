package dictdata

import (
	"context"

	"github.com/goback/pkg/dal"
	"github.com/goback/services/dict/internal/model"
)

// Repository 字典数据仓储接口
type Repository interface {
	dal.Repository[model.DictData]
	FindByTypeCode(ctx context.Context, typeCode string) ([]model.DictData, error)
	FindByTypeID(ctx context.Context, typeID int64) ([]model.DictData, error)
	DeleteByTypeID(ctx context.Context, typeID int64) error
}

// repository 字典数据仓储实现
type repository struct {
	*dal.BaseRepository[model.DictData]
}

// NewRepository 创建字典数据仓储
func NewRepository() Repository {
	return &repository{
		BaseRepository: dal.NewBaseRepository[model.DictData](),
	}
}

// FindByTypeCode 根据类型编码查询
func (r *repository) FindByTypeCode(ctx context.Context, typeCode string) ([]model.DictData, error) {
	var list []model.DictData
	err := r.DB().WithContext(ctx).
		Joins("JOIN sys_dict_type ON sys_dict_type.id = sys_dict_data.dict_type_id").
		Where("sys_dict_type.code = ? AND sys_dict_data.status = 1", typeCode).
		Order("sys_dict_data.sort ASC").
		Find(&list).Error
	return list, err
}

// FindByTypeID 根据类型ID查询
func (r *repository) FindByTypeID(ctx context.Context, typeID int64) ([]model.DictData, error) {
	var list []model.DictData
	err := r.DB().WithContext(ctx).
		Where("dict_type_id = ?", typeID).
		Order("sort ASC").
		Find(&list).Error
	return list, err
}

// DeleteByTypeID 根据类型ID删除
func (r *repository) DeleteByTypeID(ctx context.Context, typeID int64) error {
	return r.DB().WithContext(ctx).Where("dict_type_id = ?", typeID).Delete(&model.DictData{}).Error
}
