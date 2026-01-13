package operationlog

import (
	"github.com/goback/pkg/dal"
	"github.com/goback/services/log/internal/model"
)

// Repository 操作日志仓储接口
type Repository interface {
	dal.Repository[model.OperationLog]
}

// repository 操作日志仓储实现
type repository struct {
	*dal.BaseRepository[model.OperationLog]
}

// NewRepository 创建操作日志仓储
func NewRepository() Repository {
	return &repository{
		BaseRepository: dal.NewBaseRepository[model.OperationLog](),
	}
}
