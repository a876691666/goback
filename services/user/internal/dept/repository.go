package dept

import (
	"context"

	"github.com/goback/pkg/dal"
	"github.com/goback/services/user/internal/model"
)

// Repository 部门仓储接口
type Repository interface {
	dal.Repository[model.Dept]
	FindByParentID(ctx context.Context, parentID int64) ([]model.Dept, error)
	FindAllEnabled(ctx context.Context) ([]model.Dept, error)
}

// repository 部门仓储实现
type repository struct {
	*dal.BaseRepository[model.Dept]
}

// NewRepository 创建部门仓储
func NewRepository() Repository {
	return &repository{
		BaseRepository: dal.NewBaseRepository[model.Dept](),
	}
}

// FindByParentID 根据父ID查找
func (r *repository) FindByParentID(ctx context.Context, parentID int64) ([]model.Dept, error) {
	return r.Find(ctx, map[string]interface{}{"parent_id": parentID})
}

// FindAllEnabled 查找所有启用的部门
func (r *repository) FindAllEnabled(ctx context.Context) ([]model.Dept, error) {
	return r.Find(ctx, map[string]interface{}{"status": 1})
}
