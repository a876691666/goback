package permission

import (
	"context"

	"github.com/goback/pkg/dal"
	"github.com/goback/services/rbac/internal/model"
)

// Repository 权限仓储接口
type Repository interface {
	dal.Repository[model.Permission]
	FindByCode(ctx context.Context, code string) (*model.Permission, error)
	FindByRoleID(ctx context.Context, roleID int64) ([]model.Permission, error)
}

// repository 权限仓储实现
type repository struct {
	*dal.BaseRepository[model.Permission]
}

// NewRepository 创建权限仓储
func NewRepository() Repository {
	return &repository{
		BaseRepository: dal.NewBaseRepository[model.Permission](),
	}
}

// FindByCode 根据编码查找
func (r *repository) FindByCode(ctx context.Context, code string) (*model.Permission, error) {
	return r.FindOne(ctx, map[string]interface{}{"code": code})
}

// FindByRoleID 根据角色ID查找权限
func (r *repository) FindByRoleID(ctx context.Context, roleID int64) ([]model.Permission, error) {
	var permissions []model.Permission
	err := r.DB().WithContext(ctx).
		Joins("JOIN sys_role_permission ON sys_role_permission.permission_id = sys_permission.id").
		Where("sys_role_permission.role_id = ?", roleID).
		Find(&permissions).Error
	return permissions, err
}

// RolePermissionRepository 角色权限关联仓储
type RolePermissionRepository interface {
	Create(ctx context.Context, roleID, permissionID int64) error
	Delete(ctx context.Context, roleID, permissionID int64) error
	DeleteByRoleID(ctx context.Context, roleID int64) error
	FindByRoleID(ctx context.Context, roleID int64) ([]model.RolePermission, error)
	BatchCreate(ctx context.Context, roleID int64, permissionIDs []int64) error
}

// rolePermissionRepository 角色权限关联仓储实现
type rolePermissionRepository struct {
	*dal.BaseRepository[model.RolePermission]
}

// NewRolePermissionRepository 创建角色权限关联仓储
func NewRolePermissionRepository() RolePermissionRepository {
	return &rolePermissionRepository{
		BaseRepository: dal.NewBaseRepository[model.RolePermission](),
	}
}

// Create 创建关联
func (r *rolePermissionRepository) Create(ctx context.Context, roleID, permissionID int64) error {
	rp := &model.RolePermission{
		RoleID:       roleID,
		PermissionID: permissionID,
	}
	return r.BaseRepository.Create(ctx, rp)
}

// Delete 删除关联
func (r *rolePermissionRepository) Delete(ctx context.Context, roleID, permissionID int64) error {
	return r.DB().WithContext(ctx).
		Where("role_id = ? AND permission_id = ?", roleID, permissionID).
		Delete(&model.RolePermission{}).Error
}

// DeleteByRoleID 根据角色ID删除所有关联
func (r *rolePermissionRepository) DeleteByRoleID(ctx context.Context, roleID int64) error {
	return r.DB().WithContext(ctx).
		Where("role_id = ?", roleID).
		Delete(&model.RolePermission{}).Error
}

// FindByRoleID 根据角色ID查找关联
func (r *rolePermissionRepository) FindByRoleID(ctx context.Context, roleID int64) ([]model.RolePermission, error) {
	return r.FindAll(ctx, map[string]interface{}{"role_id": roleID})
}

// BatchCreate 批量创建关联
func (r *rolePermissionRepository) BatchCreate(ctx context.Context, roleID int64, permissionIDs []int64) error {
	if len(permissionIDs) == 0 {
		return nil
	}

	rps := make([]model.RolePermission, len(permissionIDs))
	for i, permID := range permissionIDs {
		rps[i] = model.RolePermission{
			RoleID:       roleID,
			PermissionID: permID,
		}
	}

	return r.DB().WithContext(ctx).CreateInBatches(rps, 100).Error
}
