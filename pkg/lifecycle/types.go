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
	// KeyPermissions 权限列表缓存键
	KeyPermissions = "permissions"
	// KeyRoles 角色列表缓存键
	KeyRoles = "roles"
	// KeyRolePermissions 角色权限映射缓存键
	KeyRolePermissions = "role_permissions"
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
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	Permissions []int64 `json:"permissions"` // 权限ID列表
}

// RolePermissionMap 角色权限映射
type RolePermissionMap map[int64][]Permission // roleID -> permissions

// UserRoleMap 用户角色映射
type UserRoleMap map[int64][]int64 // userID -> roleIDs
