package role

import (
	"context"

	"github.com/goback/pkg/dal"
	"github.com/goback/services/rbac/internal/model"
)

// Repository 角色仓储接口
type Repository interface {
	dal.Repository[model.Role]
	FindByCode(ctx context.Context, code string) (*model.Role, error)
}

// repository 角色仓储实现
type repository struct {
	*dal.BaseRepository[model.Role]
}

// NewRepository 创建角色仓储
func NewRepository() Repository {
	return &repository{
		BaseRepository: dal.NewBaseRepository[model.Role](),
	}
}

// FindByCode 根据编码查找
func (r *repository) FindByCode(ctx context.Context, code string) (*model.Role, error) {
	return r.FindOne(ctx, map[string]interface{}{"code": code})
}

// DataScopeRepository 角色数据权限仓储
type DataScopeRepository interface {
	Create(ctx context.Context, roleID, deptID int64) error
	DeleteByRoleID(ctx context.Context, roleID int64) error
	FindByRoleID(ctx context.Context, roleID int64) ([]model.RoleDataScope, error)
	BatchCreate(ctx context.Context, roleID int64, deptIDs []int64) error
}

// dataScopeRepository 角色数据权限仓储实现
type dataScopeRepository struct {
	*dal.BaseRepository[model.RoleDataScope]
}

// NewDataScopeRepository 创建角色数据权限仓储
func NewDataScopeRepository() DataScopeRepository {
	return &dataScopeRepository{
		BaseRepository: dal.NewBaseRepository[model.RoleDataScope](),
	}
}

// Create 创建
func (r *dataScopeRepository) Create(ctx context.Context, roleID, deptID int64) error {
	rds := &model.RoleDataScope{
		RoleID: roleID,
		DeptID: deptID,
	}
	return r.BaseRepository.Create(ctx, rds)
}

// DeleteByRoleID 根据角色ID删除
func (r *dataScopeRepository) DeleteByRoleID(ctx context.Context, roleID int64) error {
	return r.DB().WithContext(ctx).
		Where("role_id = ?", roleID).
		Delete(&model.RoleDataScope{}).Error
}

// FindByRoleID 根据角色ID查找
func (r *dataScopeRepository) FindByRoleID(ctx context.Context, roleID int64) ([]model.RoleDataScope, error) {
	return r.FindAll(ctx, map[string]interface{}{"role_id": roleID})
}

// BatchCreate 批量创建
func (r *dataScopeRepository) BatchCreate(ctx context.Context, roleID int64, deptIDs []int64) error {
	if len(deptIDs) == 0 {
		return nil
	}

	rdss := make([]model.RoleDataScope, len(deptIDs))
	for i, deptID := range deptIDs {
		rdss[i] = model.RoleDataScope{
			RoleID: roleID,
			DeptID: deptID,
		}
	}

	return r.DB().WithContext(ctx).CreateInBatches(rdss, 100).Error
}
