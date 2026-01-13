package dicttype

import (
	"context"

	"github.com/goback/pkg/dal"
	"github.com/goback/services/dict/internal/model"
)

// Repository 字典类型仓储接口
type Repository interface {
	dal.Repository[model.DictType]
	FindByCode(ctx context.Context, code string) (*model.DictType, error)
	ExistsByCode(ctx context.Context, code string, excludeID ...int64) (bool, error)
}

// repository 字典类型仓储实现
type repository struct {
	*dal.BaseRepository[model.DictType]
}

// NewRepository 创建字典类型仓储
func NewRepository() Repository {
	return &repository{
		BaseRepository: dal.NewBaseRepository[model.DictType](),
	}
}

// FindByCode 根据编码查询
func (r *repository) FindByCode(ctx context.Context, code string) (*model.DictType, error) {
	return r.FindOne(ctx, map[string]interface{}{"code": code})
}

// ExistsByCode 检查编码是否存在
func (r *repository) ExistsByCode(ctx context.Context, code string, excludeID ...int64) (bool, error) {
	var count int64
	query := r.DB().WithContext(ctx).Model(&model.DictType{}).Where("code = ?", code)
	if len(excludeID) > 0 {
		query = query.Where("id != ?", excludeID[0])
	}
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
