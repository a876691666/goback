package auth

import (
	"fmt"
	"sync"

	"github.com/casbin/casbin/v3"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"github.com/goback/pkg/config"
	"gorm.io/gorm"
)

var (
	enforcerOnce sync.Once
	enforcer     *casbin.Enforcer
)

// InitCasbin 初始化Casbin
func InitCasbin(db *gorm.DB, cfg *config.CasbinConfig) error {
	var err error
	enforcerOnce.Do(func() {
		// 使用GORM适配器
		adapter, adapterErr := gormadapter.NewAdapterByDB(db)
		if adapterErr != nil {
			err = fmt.Errorf("failed to create casbin adapter: %w", adapterErr)
			return
		}

		// 创建enforcer
		enforcer, err = casbin.NewEnforcer(cfg.ModelPath, adapter)
		if err != nil {
			err = fmt.Errorf("failed to create casbin enforcer: %w", err)
			return
		}

		// 加载策略
		if err = enforcer.LoadPolicy(); err != nil {
			err = fmt.Errorf("failed to load casbin policy: %w", err)
			return
		}
	})
	return err
}

// GetEnforcer 获取Enforcer
func GetEnforcer() *casbin.Enforcer {
	if enforcer == nil {
		panic("casbin enforcer not initialized, call InitCasbin first")
	}
	return enforcer
}

// CasbinService Casbin服务
type CasbinService struct {
	enforcer *casbin.Enforcer
}

// NewCasbinService 创建Casbin服务
func NewCasbinService() *CasbinService {
	return &CasbinService{
		enforcer: GetEnforcer(),
	}
}

// Enforce 权限检查
func (s *CasbinService) Enforce(sub, obj, act string) (bool, error) {
	return s.enforcer.Enforce(sub, obj, act)
}

// AddPolicy 添加策略
func (s *CasbinService) AddPolicy(sub, obj, act string) (bool, error) {
	return s.enforcer.AddPolicy(sub, obj, act)
}

// RemovePolicy 删除策略
func (s *CasbinService) RemovePolicy(sub, obj, act string) (bool, error) {
	return s.enforcer.RemovePolicy(sub, obj, act)
}

// AddPolicies 批量添加策略
func (s *CasbinService) AddPolicies(rules [][]string) (bool, error) {
	return s.enforcer.AddPolicies(rules)
}

// RemovePolicies 批量删除策略
func (s *CasbinService) RemovePolicies(rules [][]string) (bool, error) {
	return s.enforcer.RemovePolicies(rules)
}

// GetPoliciesForRole 获取角色的所有策略
func (s *CasbinService) GetPoliciesForRole(role string) [][]string {
	policies, _ := s.enforcer.GetFilteredPolicy(0, role)
	return policies
}

// AddRoleForUser 为用户添加角色
func (s *CasbinService) AddRoleForUser(user, role string) (bool, error) {
	return s.enforcer.AddGroupingPolicy(user, role)
}

// RemoveRoleForUser 移除用户的角色
func (s *CasbinService) RemoveRoleForUser(user, role string) (bool, error) {
	return s.enforcer.RemoveGroupingPolicy(user, role)
}

// GetRolesForUser 获取用户的所有角色
func (s *CasbinService) GetRolesForUser(user string) ([]string, error) {
	return s.enforcer.GetRolesForUser(user)
}

// GetUsersForRole 获取角色的所有用户
func (s *CasbinService) GetUsersForRole(role string) ([]string, error) {
	return s.enforcer.GetUsersForRole(role)
}

// DeleteRoleForUser 删除用户的指定角色
func (s *CasbinService) DeleteRoleForUser(user, role string) (bool, error) {
	return s.enforcer.DeleteRoleForUser(user, role)
}

// DeleteRolesForUser 删除用户的所有角色
func (s *CasbinService) DeleteRolesForUser(user string) (bool, error) {
	return s.enforcer.DeleteRolesForUser(user)
}

// DeleteRole 删除角色
func (s *CasbinService) DeleteRole(role string) (bool, error) {
	return s.enforcer.DeleteRole(role)
}

// DeletePermissionsForRole 删除角色的所有权限
func (s *CasbinService) DeletePermissionsForRole(role string) (bool, error) {
	return s.enforcer.DeletePermissionsForUser(role)
}

// HasPermission 检查是否有权限
func (s *CasbinService) HasPermission(sub, obj, act string) bool {
	ok, _ := s.enforcer.Enforce(sub, obj, act)
	return ok
}

// GetAllRoles 获取所有角色
func (s *CasbinService) GetAllRoles() []string {
	roles, _ := s.enforcer.GetAllRoles()
	return roles
}

// GetAllSubjects 获取所有主体
func (s *CasbinService) GetAllSubjects() []string {
	subjects, _ := s.enforcer.GetAllSubjects()
	return subjects
}

// GetAllObjects 获取所有对象
func (s *CasbinService) GetAllObjects() []string {
	objects, _ := s.enforcer.GetAllObjects()
	return objects
}

// GetAllActions 获取所有动作
func (s *CasbinService) GetAllActions() []string {
	actions, _ := s.enforcer.GetAllActions()
	return actions
}

// GetPermissionsForUser 获取用户的所有权限
func (s *CasbinService) GetPermissionsForUser(user string) [][]string {
	perms, _ := s.enforcer.GetPermissionsForUser(user)
	return perms
}

// SavePolicy 保存策略
func (s *CasbinService) SavePolicy() error {
	return s.enforcer.SavePolicy()
}

// LoadPolicy 重新加载策略
func (s *CasbinService) LoadPolicy() error {
	return s.enforcer.LoadPolicy()
}

// ClearPolicy 清除所有策略
func (s *CasbinService) ClearPolicy() {
	s.enforcer.ClearPolicy()
}

// UpdatePolicy 更新策略
func (s *CasbinService) UpdatePolicy(oldRule, newRule []string) (bool, error) {
	return s.enforcer.UpdatePolicy(oldRule, newRule)
}

// SyncRolePermissions 同步角色权限
func (s *CasbinService) SyncRolePermissions(role string, permissions [][]string) error {
	// 删除角色现有权限
	s.enforcer.DeletePermissionsForUser(role)

	// 添加新权限
	for _, perm := range permissions {
		if len(perm) >= 2 {
			s.enforcer.AddPolicy(role, perm[0], perm[1])
		}
	}

	return nil
}

// CheckUserPermission 检查用户权限(包含角色继承)
func (s *CasbinService) CheckUserPermission(userID int64, resource, action string) bool {
	user := fmt.Sprintf("user:%d", userID)
	ok, _ := s.enforcer.Enforce(user, resource, action)
	return ok
}

// CheckRolePermission 检查角色权限
func (s *CasbinService) CheckRolePermission(roleCode, resource, action string) bool {
	role := fmt.Sprintf("role:%s", roleCode)
	ok, _ := s.enforcer.Enforce(role, resource, action)
	return ok
}

// SetUserRole 设置用户角色(1对1)
func (s *CasbinService) SetUserRole(userID int64, roleCode string) error {
	user := fmt.Sprintf("user:%d", userID)
	role := fmt.Sprintf("role:%s", roleCode)

	// 先删除用户现有角色
	s.enforcer.DeleteRolesForUser(user)

	// 添加新角色
	_, err := s.enforcer.AddGroupingPolicy(user, role)
	return err
}

// SetRolePermissions 设置角色权限
func (s *CasbinService) SetRolePermissions(roleCode string, permissions []Permission) error {
	role := fmt.Sprintf("role:%s", roleCode)

	// 删除角色现有权限
	s.enforcer.DeletePermissionsForUser(role)

	// 添加新权限
	for _, perm := range permissions {
		s.enforcer.AddPolicy(role, perm.Resource, perm.Action)
	}

	return nil
}

// Permission 权限定义
type Permission struct {
	Resource string `json:"resource"` // 资源路径,如 /api/users/*
	Action   string `json:"action"`   // 动作,如 GET, POST, PUT, DELETE, *
}
