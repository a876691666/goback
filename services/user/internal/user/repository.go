package user

import (
	"context"

	"github.com/goback/pkg/dal"
	"github.com/goback/services/user/internal/model"
)

// Repository 用户仓储接口
type Repository interface {
	dal.Repository[model.User]
	FindByUsername(ctx context.Context, username string) (*model.User, error)
	FindByEmail(ctx context.Context, email string) (*model.User, error)
	FindByPhone(ctx context.Context, phone string) (*model.User, error)
	UpdatePassword(ctx context.Context, id int64, password string) error
	UpdateStatus(ctx context.Context, id int64, status int8) error
}

// repository 用户仓储实现
type repository struct {
	*dal.BaseRepository[model.User]
}

// NewRepository 创建用户仓储
func NewRepository() Repository {
	return &repository{
		BaseRepository: dal.NewBaseRepository[model.User](),
	}
}

// FindByUsername 根据用户名查找
func (r *repository) FindByUsername(ctx context.Context, username string) (*model.User, error) {
	return r.FindOne(ctx, map[string]interface{}{"username": username})
}

// FindByEmail 根据邮箱查找
func (r *repository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	return r.FindOne(ctx, map[string]interface{}{"email": email})
}

// FindByPhone 根据手机号查找
func (r *repository) FindByPhone(ctx context.Context, phone string) (*model.User, error) {
	return r.FindOne(ctx, map[string]interface{}{"phone": phone})
}

// UpdatePassword 更新密码
func (r *repository) UpdatePassword(ctx context.Context, id int64, password string) error {
	return r.UpdateFields(ctx, id, map[string]interface{}{"password": password})
}

// UpdateStatus 更新状态
func (r *repository) UpdateStatus(ctx context.Context, id int64, status int8) error {
	return r.UpdateFields(ctx, id, map[string]interface{}{"status": status})
}
