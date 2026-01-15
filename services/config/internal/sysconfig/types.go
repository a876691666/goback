package sysconfig

// PageRequest 分页查询请求
type PageRequest struct {
	Page       int    `query:"page"`       // 页码
	PageSize   int    `query:"pageSize"`   // 每页条数
	ConfigName string `query:"configName"` // 参数名称（模糊搜索）
	ConfigKey  string `query:"configKey"`  // 参数键名（模糊搜索）
	ConfigType string `query:"configType"` // 是否内置参数（Y/N）
}

// CreateRequest 创建系统参数请求
type CreateRequest struct {
	ConfigName  string `json:"configName"`  // 参数名称（必填）
	ConfigKey   string `json:"configKey"`   // 参数键名（必填）
	ConfigValue string `json:"configValue"` // 参数键值（必填）
	ConfigType  string `json:"configType"`  // 是否内置参数（Y/N）
	Remark      string `json:"remark"`      // 备注
}

// UpdateRequest 更新系统参数请求
type UpdateRequest struct {
	ID          int64  `json:"id"`          // 配置ID（必填）
	ConfigName  string `json:"configName"`  // 参数名称
	ConfigKey   string `json:"configKey"`   // 参数键名
	ConfigValue string `json:"configValue"` // 参数键值
	ConfigType  string `json:"configType"`  // 是否内置参数（Y/N）
	Remark      string `json:"remark"`      // 备注
}

// RemoveRequest 批量删除请求
type RemoveRequest struct {
	IDs []int64 `json:"ids"` // 配置ID列表
}
