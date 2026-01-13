package auth

import (
	"fmt"

	"github.com/goback/pkg/ssql"
)

// DataScopeType 数据权限类型
type DataScopeType int

const (
	DataScopeAll      DataScopeType = 1 // 全部数据
	DataScopeDept     DataScopeType = 2 // 本部门数据
	DataScopeDeptSub  DataScopeType = 3 // 本部门及下级数据
	DataScopeSelf     DataScopeType = 4 // 仅本人数据
	DataScopeCustom   DataScopeType = 5 // 自定义数据权限
)

// DataScopeInfo 数据权限信息
type DataScopeInfo struct {
	Type       DataScopeType `json:"type"`
	DeptID     int64         `json:"deptId"`     // 用户所属部门ID
	DeptIDs    []int64       `json:"deptIds"`    // 自定义部门ID列表
	UserID     int64         `json:"userId"`     // 用户ID
	UserField  string        `json:"userField"`  // 用户字段名,默认 created_by
	DeptField  string        `json:"deptField"`  // 部门字段名,默认 dept_id
}

// NewDataScopeInfo 创建数据权限信息
func NewDataScopeInfo(scopeType DataScopeType, userID, deptID int64) *DataScopeInfo {
	return &DataScopeInfo{
		Type:      scopeType,
		UserID:    userID,
		DeptID:    deptID,
		UserField: "created_by",
		DeptField: "dept_id",
	}
}

// WithDeptIDs 设置自定义部门ID列表
func (d *DataScopeInfo) WithDeptIDs(deptIDs []int64) *DataScopeInfo {
	d.DeptIDs = deptIDs
	return d
}

// WithUserField 设置用户字段名
func (d *DataScopeInfo) WithUserField(field string) *DataScopeInfo {
	d.UserField = field
	return d
}

// WithDeptField 设置部门字段名
func (d *DataScopeInfo) WithDeptField(field string) *DataScopeInfo {
	d.DeptField = field
	return d
}

// ToSSQL 转换为SSQL表达式
func (d *DataScopeInfo) ToSSQL() string {
	builder := ssql.NewBuilder()

	switch d.Type {
	case DataScopeAll:
		// 全部数据,不添加任何条件
		return ""
	case DataScopeDept:
		// 本部门数据
		builder.Eq(d.DeptField, d.DeptID)
	case DataScopeDeptSub:
		// 本部门及下级数据(需要传入下级部门ID列表)
		if len(d.DeptIDs) > 0 {
			values := make([]interface{}, len(d.DeptIDs))
			for i, id := range d.DeptIDs {
				values[i] = id
			}
			builder.In(d.DeptField, values...)
		} else {
			builder.Eq(d.DeptField, d.DeptID)
		}
	case DataScopeSelf:
		// 仅本人数据
		builder.Eq(d.UserField, d.UserID)
	case DataScopeCustom:
		// 自定义数据权限
		if len(d.DeptIDs) > 0 {
			values := make([]interface{}, len(d.DeptIDs))
			for i, id := range d.DeptIDs {
				values[i] = id
			}
			builder.In(d.DeptField, values...)
		}
	}

	return builder.String()
}

// ToSQL 转换为SQL条件
func (d *DataScopeInfo) ToSQL(dialect ssql.Dialect) (string, []interface{}) {
	switch d.Type {
	case DataScopeAll:
		return "", nil
	case DataScopeDept:
		return fmt.Sprintf("%s = ?", dialect.Quote(d.DeptField)), []interface{}{d.DeptID}
	case DataScopeDeptSub:
		if len(d.DeptIDs) > 0 {
			placeholders := make([]string, len(d.DeptIDs))
			values := make([]interface{}, len(d.DeptIDs))
			for i, id := range d.DeptIDs {
				placeholders[i] = "?"
				values[i] = id
			}
			return fmt.Sprintf("%s IN (%s)", dialect.Quote(d.DeptField), joinStrings(placeholders, ", ")), values
		}
		return fmt.Sprintf("%s = ?", dialect.Quote(d.DeptField)), []interface{}{d.DeptID}
	case DataScopeSelf:
		return fmt.Sprintf("%s = ?", dialect.Quote(d.UserField)), []interface{}{d.UserID}
	case DataScopeCustom:
		if len(d.DeptIDs) > 0 {
			placeholders := make([]string, len(d.DeptIDs))
			values := make([]interface{}, len(d.DeptIDs))
			for i, id := range d.DeptIDs {
				placeholders[i] = "?"
				values[i] = id
			}
			return fmt.Sprintf("%s IN (%s)", dialect.Quote(d.DeptField), joinStrings(placeholders, ", ")), values
		}
		return "", nil
	default:
		return "", nil
	}
}

// joinStrings 连接字符串
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}

// DataScopeService 数据权限服务
type DataScopeService struct {
	// 获取部门下级ID的函数
	getDeptSubIDs func(deptID int64) ([]int64, error)
}

// NewDataScopeService 创建数据权限服务
func NewDataScopeService(getDeptSubIDs func(deptID int64) ([]int64, error)) *DataScopeService {
	return &DataScopeService{
		getDeptSubIDs: getDeptSubIDs,
	}
}

// GetDataScopeInfo 获取数据权限信息
func (s *DataScopeService) GetDataScopeInfo(scopeType DataScopeType, userID, deptID int64, customDeptIDs []int64) (*DataScopeInfo, error) {
	info := NewDataScopeInfo(scopeType, userID, deptID)

	switch scopeType {
	case DataScopeDeptSub:
		// 获取下级部门ID
		if s.getDeptSubIDs != nil {
			subIDs, err := s.getDeptSubIDs(deptID)
			if err != nil {
				return nil, err
			}
			// 包含本部门
			info.DeptIDs = append([]int64{deptID}, subIDs...)
		}
	case DataScopeCustom:
		info.DeptIDs = customDeptIDs
	}

	return info, nil
}

// BuildSQLCondition 构建SQL条件
func (s *DataScopeService) BuildSQLCondition(scopeType DataScopeType, userID, deptID int64, customDeptIDs []int64) (string, []interface{}, error) {
	info, err := s.GetDataScopeInfo(scopeType, userID, deptID, customDeptIDs)
	if err != nil {
		return "", nil, err
	}
	
	sql, args := info.ToSQL(ssql.NewMySQLDialect())
	return sql, args, nil
}
