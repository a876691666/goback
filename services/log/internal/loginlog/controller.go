package loginlog

import (
	"fmt"
	"strings"

	"github.com/goback/pkg/app/apis"
	"github.com/goback/pkg/app/core"
	"github.com/goback/pkg/dal"
	"github.com/goback/services/log/internal/model"
)

// List 登录日志列表
func List(e *core.RequestEvent) error {
	params, err := dal.BindQueryFromRequest(e.Request)
	if err != nil {
		return apis.Error(e, 400, err.Error())
	}
	result, err := model.LoginLogs.GetList(params)
	if err != nil {
		return apis.ErrorFromErr(e, err)
	}
	return apis.Paged(e, result.Items, result.TotalItems, result.Page, result.PerPage)
}

// Delete 删除登录日志
func Delete(e *core.RequestEvent) error {
	idsStr := e.Request.PathValue("ids")
	ids, err := parseIDs(idsStr)
	if err != nil {
		return apis.Error(e, 400, "无效的ID格式")
	}
	if err := model.LoginLogs.DeleteByIDs(ids); err != nil {
		return apis.ErrorFromErr(e, err)
	}
	return apis.Success(e, nil)
}

// Clear 清空登录日志
func Clear(e *core.RequestEvent) error {
	if err := model.LoginLogs.Truncate(); err != nil {
		return apis.ErrorFromErr(e, err)
	}
	return apis.Success(e, nil)
}

// CreateLog 创建登录日志
func CreateLog(log *model.LoginLog) error {
	return model.LoginLogs.Create(log)
}

// parseIDs 解析逗号分隔的ID字符串
func parseIDs(idsStr string) ([]int64, error) {
	parts := strings.Split(idsStr, ",")
	ids := make([]int64, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		var id int64
		if _, err := fmt.Sscanf(part, "%d", &id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}
