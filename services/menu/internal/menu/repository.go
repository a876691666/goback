package menu

import (
	"context"

	"github.com/goback/pkg/dal"
	"github.com/goback/services/menu/internal/model"
)

// Repository 菜单仓储接口
type Repository interface {
	dal.Repository[model.Menu]
	FindByParentID(ctx context.Context, parentID int64) ([]model.Menu, error)
	FindByPermCode(ctx context.Context, permCode string) (*model.Menu, error)
}

// repository 菜单仓储实现
type repository struct {
	*dal.BaseRepository[model.Menu]
}

// NewRepository 创建菜单仓储
func NewRepository() Repository {
	return &repository{
		BaseRepository: dal.NewBaseRepository[model.Menu](),
	}
}

// FindByParentID 根据父ID查找
func (r *repository) FindByParentID(ctx context.Context, parentID int64) ([]model.Menu, error) {
	return r.FindAll(ctx, map[string]interface{}{"parent_id": parentID}, dal.WithOrder("sort ASC, id ASC"))
}

// FindByPermCode 根据权限编码查找
func (r *repository) FindByPermCode(ctx context.Context, permCode string) (*model.Menu, error) {
	return r.FindOne(ctx, map[string]interface{}{"perm_code": permCode})
}
