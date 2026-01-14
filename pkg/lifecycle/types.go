package lifecycle

// 模块常量定义 - 用于缓存广播的模块标识
const (
	// ModuleRBAC RBAC权限模块
	ModuleRBAC = "rbac"
	// ModuleUser 用户模块
	ModuleUser = "user"
	// ModuleMenu 菜单模块
	ModuleMenu = "menu"
	// ModuleDict 字典模块
	ModuleDict = "dict"
)

// 缓存键常量定义
const (
	// KeyRBACData 完整RBAC数据缓存键（统一广播）
	// 包含: Permissions, Roles, RolePermissions, PermissionScopes
	KeyRBACData = "rbac_data"
	// KeyUserRoles 用户角色映射缓存键
	KeyUserRoles = "user_roles"
	// KeyMenuTree 菜单树缓存键
	KeyMenuTree = "menu_tree"
	// KeyDictData 字典数据缓存键前缀
	KeyDictData = "dict_data"
)

// Permission 权限信息（用于缓存传输）
type Permission struct {
	ID       int64  `json:"id"`
	Code     string `json:"code"`
	Name     string `json:"name"`
	Resource string `json:"resource"`
	Action   string `json:"action"`
}

// Role 角色信息（用于缓存传输）
type Role struct {
	ID          int64   `json:"id"`
	ParentID    int64   `json:"parentId"`
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	Status      int8    `json:"status"` // 1:正常 0:禁用
}

// RolePermissionMap 角色权限映射
type RolePermissionMap map[int64][]Permission // roleID -> permissions

// UserRoleMap 用户角色映射
type UserRoleMap map[int64][]int64 // userID -> roleIDs

// PermissionScope 权限数据范围（用于缓存传输）
type PermissionScope struct {
	ID             int64  `json:"id"`
	PermissionID   int64  `json:"permissionId"`
	Name           string `json:"name"`
	ScopeTableName string `json:"tableName"`
	SSQLRule       string `json:"ssqlRule"`
	Description    string `json:"description"`
}

// RBACData 完整的RBAC数据（用于一次性广播）
type RBACData struct {
	Permissions      []Permission      `json:"permissions"`
	Roles            []Role            `json:"roles"`
	RolePermissions  RolePermissionMap `json:"rolePermissions"`
	PermissionScopes []PermissionScope `json:"permissionScopes"`
}
