package loginlog

import (
	"github.com/goback/pkg/dal"
	"github.com/goback/services/log/internal/model"
)

// Repository 登录日志仓储接口
type Repository interface {
	dal.Repository[model.LoginLog]
}

// repository 登录日志仓储实现
type repository struct {
	*dal.BaseRepository[model.LoginLog]
}

// NewRepository 创建登录日志仓储
func NewRepository() Repository {
	return &repository{
		BaseRepository: dal.NewBaseRepository[model.LoginLog](),
	}
}
