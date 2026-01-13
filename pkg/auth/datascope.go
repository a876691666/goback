package auth

import (
	"fmt"

	"github.com/goback/pkg/ssql"
)

// DataScopeType 数据权限类型
type DataScopeType int

// 数据权限类型常量
const (
	DataScopeAll  DataScopeType = 1 // 全部数据
	DataScopeSelf DataScopeType = 2 // 仅本人数据
)

// DataScopeInfo 数据权限信息
type DataScopeInfo struct {
	Type      DataScopeType `json:"type"`
	UserID    int64         `json:"userId"`    // 用户ID
	UserField string        `json:"userField"` // 用户字段名,默认 created_by
}

// NewDataScopeInfo 创建数据权限信息
func NewDataScopeInfo(scopeType DataScopeType, userID int64) *DataScopeInfo {
	return &DataScopeInfo{
		Type:      scopeType,
		UserID:    userID,
		UserField: "created_by",
	}
}

// WithUserField 设置用户字段名
func (d *DataScopeInfo) WithUserField(field string) *DataScopeInfo {
	d.UserField = field
	return d
}

// ToSSQL 转换为SSQL表达式
func (d *DataScopeInfo) ToSSQL() string {
	builder := ssql.NewBuilder()

	switch d.Type {
	case DataScopeAll:
		// 全部数据,不添加任何条件
		return ""
	case DataScopeSelf:
		// 仅本人数据
		builder.Eq(d.UserField, d.UserID)
	}

	return builder.String()
}

// ToSQL 转换为SQL条件
func (d *DataScopeInfo) ToSQL(dialect ssql.Dialect) (string, []interface{}) {
	switch d.Type {
	case DataScopeAll:
		return "", nil
	case DataScopeSelf:
		return fmt.Sprintf("%s = ?", dialect.Quote(d.UserField)), []interface{}{d.UserID}
	default:
		return "", nil
	}
}

// DataScopeService 数据权限服务
type DataScopeService struct{}

// NewDataScopeService 创建数据权限服务
func NewDataScopeService() *DataScopeService {
	return &DataScopeService{}
}

// GetDataScopeInfo 获取数据权限信息
func (s *DataScopeService) GetDataScopeInfo(scopeType DataScopeType, userID int64) *DataScopeInfo {
	return NewDataScopeInfo(scopeType, userID)
}

// BuildSQLCondition 构建SQL条件
func (s *DataScopeService) BuildSQLCondition(scopeType DataScopeType, userID int64) (string, []interface{}) {
	info := s.GetDataScopeInfo(scopeType, userID)
	sql, args := info.ToSQL(ssql.NewMySQLDialect())
	return sql, args
}
