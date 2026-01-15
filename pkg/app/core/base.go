package core

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"go-micro.dev/v5/registry"

	"github.com/goback/pkg/app/tools/cron"
	"github.com/goback/pkg/app/tools/filesystem"
	"github.com/goback/pkg/app/tools/hook"
	"github.com/goback/pkg/app/tools/router"
	"github.com/goback/pkg/app/tools/store"
	"github.com/goback/pkg/app/tools/subscriptions"
)

const (
	LocalStorageDirName       string = "storage"
	LocalBackupsDirName       string = "backups"
	LocalTempDirName          string = ".temp_to_delete"
	LocalAutocertCacheDirName string = ".autocert_cache"
)

// -------------------------------------------------------------------
// Lifecycle Event Types
// -------------------------------------------------------------------

const lifecycleTopic = "service:lifecycle"

// LifecycleMessage 生命周期广播消息
type LifecycleMessage struct {
	Service   string    `json:"service"`
	NodeID    string    `json:"node_id"`
	Event     string    `json:"event"` // "started", "ready", "stopping", "stopped"
	Timestamp time.Time `json:"timestamp"`
	Metadata  any       `json:"metadata,omitempty"`
}

// LifecycleEvent 生命周期钩子事件
type LifecycleEvent struct {
	hook.Event
	App     App
	Message *LifecycleMessage
}

// -------------------------------------------------------------------
// Cache Module Constants
// -------------------------------------------------------------------

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
	// KeyRBACData 完整RBAC数据缓存键
	KeyRBACData = "rbac_data"
	// KeyUserRoles 用户角色映射缓存键
	KeyUserRoles = "user_roles"
	// KeyMenuTree 菜单树缓存键
	KeyMenuTree = "menu_tree"
	// KeyDictData 字典数据缓存键前缀
	KeyDictData = "dict_data"
)

// -------------------------------------------------------------------
// RBAC Types
// -------------------------------------------------------------------

// Permission 权限信息
type Permission struct {
	ID       int64  `json:"id"`
	Code     string `json:"code"`
	Name     string `json:"name"`
	Resource string `json:"resource"`
	Action   string `json:"action"`
}

// Role 角色信息
type Role struct {
	ID       int64  `json:"id"`
	ParentID int64  `json:"parentId"`
	Code     string `json:"code"`
	Name     string `json:"name"`
	Status   int8   `json:"status"` // 1:正常 0:禁用
}

// RolePermissionMap 角色权限映射
type RolePermissionMap map[int64][]Permission // roleID -> permissions

// UserRoleMap 用户角色映射
type UserRoleMap map[int64][]int64 // userID -> roleIDs

// PermissionScope 权限数据范围
type PermissionScope struct {
	ID             int64  `json:"id"`
	PermissionID   int64  `json:"permissionId"`
	Name           string `json:"name"`
	ScopeTableName string `json:"tableName"`
	SSQLRule       string `json:"ssqlRule"`
	Description    string `json:"description"`
}

// RBACData 完整的RBAC数据
type RBACData struct {
	Permissions      []Permission      `json:"permissions"`
	Roles            []Role            `json:"roles"`
	RolePermissions  RolePermissionMap `json:"rolePermissions"`
	PermissionScopes []PermissionScope `json:"permissionScopes"`
}

// -------------------------------------------------------------------
// RBAC Cache
// -------------------------------------------------------------------

// RBACCache RBAC缓存管理器
type RBACCache struct {
	data RBACData
	mu   sync.RWMutex

	// 预计算的索引
	roleMap    map[int64]*Role       // roleID -> Role
	permMap    map[int64]*Permission // permID -> Permission
	childRoles map[int64][]int64     // parentID -> childIDs
}

// NewRBACCache 创建RBAC缓存管理器
func NewRBACCache() *RBACCache {
	return &RBACCache{
		roleMap:    make(map[int64]*Role),
		permMap:    make(map[int64]*Permission),
		childRoles: make(map[int64][]int64),
	}
}

// Update 更新RBAC数据
func (rc *RBACCache) Update(data RBACData) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	rc.data = data

	// 重建索引
	rc.roleMap = make(map[int64]*Role, len(data.Roles))
	for i := range data.Roles {
		rc.roleMap[data.Roles[i].ID] = &data.Roles[i]
	}

	rc.permMap = make(map[int64]*Permission, len(data.Permissions))
	for i := range data.Permissions {
		rc.permMap[data.Permissions[i].ID] = &data.Permissions[i]
	}

	rc.childRoles = make(map[int64][]int64)
	for _, role := range data.Roles {
		rc.childRoles[role.ParentID] = append(rc.childRoles[role.ParentID], role.ID)
	}
}

// GetRole 获取角色
func (rc *RBACCache) GetRole(roleID int64) (*Role, bool) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	role, ok := rc.roleMap[roleID]
	return role, ok
}

// GetPermission 获取权限
func (rc *RBACCache) GetPermission(permID int64) (*Permission, bool) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	perm, ok := rc.permMap[permID]
	return perm, ok
}

// GetRoleAndDescendantIDs 获取角色及其所有启用的后代角色ID
func (rc *RBACCache) GetRoleAndDescendantIDs(roleID int64) ([]int64, error) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	role, ok := rc.roleMap[roleID]
	if !ok {
		return nil, fmt.Errorf("角色不存在: %d", roleID)
	}
	if role.Status != 1 {
		return nil, fmt.Errorf("角色已被禁用: %d", roleID)
	}

	result := []int64{roleID}
	collected := make(map[int64]bool)
	collected[roleID] = true

	// BFS遍历所有后代
	queue := []int64{roleID}
	for len(queue) > 0 {
		currentID := queue[0]
		queue = queue[1:]

		for _, childID := range rc.childRoles[currentID] {
			if collected[childID] {
				continue
			}
			collected[childID] = true

			child, ok := rc.roleMap[childID]
			if ok && child.Status == 1 {
				result = append(result, childID)
				queue = append(queue, childID)
			}
		}
	}

	return result, nil
}

// GetAggregatedPermissions 聚合指定角色列表的所有权限（去重）
func (rc *RBACCache) GetAggregatedPermissions(roleIDs []int64) map[int64]Permission {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	permissionMap := make(map[int64]Permission)
	for _, rid := range roleIDs {
		if perms, ok := rc.data.RolePermissions[rid]; ok {
			for _, perm := range perms {
				permissionMap[perm.ID] = perm
			}
		}
	}
	return permissionMap
}

// GetPermissionScopes 获取指定权限ID列表和表名的数据范围规则
func (rc *RBACCache) GetPermissionScopes(permissionIDs []int64, tableName string) []PermissionScope {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	permIDSet := make(map[int64]bool, len(permissionIDs))
	for _, id := range permissionIDs {
		permIDSet[id] = true
	}

	var result []PermissionScope
	for _, scope := range rc.data.PermissionScopes {
		if permIDSet[scope.PermissionID] && scope.ScopeTableName == tableName {
			result = append(result, scope)
		}
	}
	return result
}

// GetAllPermissionScopes 获取所有权限数据范围
func (rc *RBACCache) GetAllPermissionScopes() []PermissionScope {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	return rc.data.PermissionScopes
}

// GetAllRoles 获取所有角色
func (rc *RBACCache) GetAllRoles() []Role {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	return rc.data.Roles
}

// GetAllPermissions 获取所有权限
func (rc *RBACCache) GetAllPermissions() []Permission {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	return rc.data.Permissions
}

// GetRolePermissions 获取角色权限映射
func (rc *RBACCache) GetRolePermissions() RolePermissionMap {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	return rc.data.RolePermissions
}

// IsReady 检查缓存是否已加载数据
func (rc *RBACCache) IsReady() bool {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	return len(rc.data.Roles) > 0 || len(rc.data.Permissions) > 0
}

// -------------------------------------------------------------------
// Cache Space
// -------------------------------------------------------------------

// CacheSpace 缓存空间 - 每个模块独立的缓存存储
type CacheSpace struct {
	module string
	data   map[string]string // key -> raw JSON string
	mu     sync.RWMutex
}

// NewCacheSpace 创建缓存空间
func NewCacheSpace(module string) *CacheSpace {
	return &CacheSpace{
		module: module,
		data:   make(map[string]string),
	}
}

// Set 设置缓存
func (cs *CacheSpace) Set(key string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal cache value: %w", err)
	}

	cs.mu.Lock()
	cs.data[key] = string(data)
	cs.mu.Unlock()

	return nil
}

// Get 获取缓存并反序列化
func (cs *CacheSpace) Get(key string, dest any) error {
	cs.mu.RLock()
	raw, ok := cs.data[key]
	cs.mu.RUnlock()

	if !ok {
		return fmt.Errorf("cache key not found: %s", key)
	}

	return json.Unmarshal([]byte(raw), dest)
}

// GetRaw 获取原始JSON字符串
func (cs *CacheSpace) GetRaw(key string) (string, bool) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	raw, ok := cs.data[key]
	return raw, ok
}

// Delete 删除缓存
func (cs *CacheSpace) Delete(key string) {
	cs.mu.Lock()
	delete(cs.data, key)
	cs.mu.Unlock()
}

// Clear 清空所有缓存
func (cs *CacheSpace) Clear() {
	cs.mu.Lock()
	cs.data = make(map[string]string)
	cs.mu.Unlock()
}

// Keys 获取所有缓存键
func (cs *CacheSpace) Keys() []string {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	keys := make([]string, 0, len(cs.data))
	for k := range cs.data {
		keys = append(keys, k)
	}
	return keys
}

// BaseAppConfig defines a BaseApp configuration option
type BaseAppConfig struct {
	// DataDir 数据目录
	DataDir string

	// EncryptionEnv 加密环境变量名
	EncryptionEnv string

	// IsDev 是否开发模式
	IsDev bool

	// ServiceName 服务名称
	ServiceName string

	// ServiceVersion 服务版本
	ServiceVersion string
}

// Broadcaster 广播接口
type Broadcaster interface {
	Start() error
	Stop() error
	Send(topic string, payload []byte, sender string) error
	Subscribe(topic string, handler func(payload []byte)) error
	HandleMessage(topic string, payload []byte)
}

// -------------------------------------------------------------------
// JWT Types
// -------------------------------------------------------------------

// JWTClaims JWT声明
type JWTClaims struct {
	UserID   int64  `json:"userId"`
	Username string `json:"username"`
	RoleID   int64  `json:"roleId"`
	RoleCode string `json:"roleCode"`
}

// JWTValidator JWT验证器接口
type JWTValidator interface {
	ParseToken(token string) (*JWTClaims, error)
}

// ServiceInfo 服务注册信息
type ServiceInfo struct {
	Name     string            `json:"name"`
	Version  string            `json:"version"`
	Address  string            `json:"address"`
	BasePath string            `json:"basePath"`
	Metadata map[string]string `json:"metadata"`
}

// ServiceInfoBuilder 服务信息构建器
type ServiceInfoBuilder struct {
	info *ServiceInfo
}

// NewServiceBuilder 创建服务构建器
func NewServiceBuilder(name, version string) *ServiceInfoBuilder {
	return &ServiceInfoBuilder{
		info: &ServiceInfo{
			Name:     name,
			Version:  version,
			Metadata: make(map[string]string),
		},
	}
}

// WithAddress 设置服务地址
func (b *ServiceInfoBuilder) WithAddress(addr string) *ServiceInfoBuilder {
	b.info.Address = addr
	return b
}

// WithBasePath 设置基础路径
func (b *ServiceInfoBuilder) WithBasePath(basePath string) *ServiceInfoBuilder {
	b.info.BasePath = basePath
	return b
}

// WithMetadata 设置元数据
func (b *ServiceInfoBuilder) WithMetadata(key, value string) *ServiceInfoBuilder {
	b.info.Metadata[key] = value
	return b
}

// Build 构建服务信息
func (b *ServiceInfoBuilder) Build() *ServiceInfo {
	return b.info
}

// ensures that the BaseApp implements the App interface.
var _ App = (*BaseApp)(nil)

// BaseApp implements core.App and defines the base app structure.
type BaseApp struct {
	config              *BaseAppConfig
	store               *store.Store[string, any]
	cronInstance        *cron.Cron
	subscriptionsBroker *subscriptions.Broker
	logger              *slog.Logger

	// 服务相关 - 直接使用 go-micro registry
	registry     registry.Registry
	regService   *registry.Service
	serviceInfo  *ServiceInfo // 简化的服务信息（用于内部）
	broadcaster  Broadcaster

	// 缓存相关
	cacheSpaces map[string]*CacheSpace // module -> CacheSpace
	rbacCache   *RBACCache
	cacheMu     sync.RWMutex

	// 运行时状态
	isBootstrapped bool
	mu             sync.RWMutex

	// app event hooks
	onBootstrap *hook.Hook[*BootstrapEvent]
	onServe     *hook.Hook[*ServeEvent]
	onTerminate *hook.Hook[*TerminateEvent]

	// Lifecycle event hooks (硬编码的生命周期钩子)
	onServiceStarted  *hook.Hook[*LifecycleEvent] // 其他服务启动
	onServiceReady    *hook.Hook[*LifecycleEvent] // 其他服务就绪
	onServiceStopping *hook.Hook[*LifecycleEvent] // 其他服务正在停止
	onServiceStopped  *hook.Hook[*LifecycleEvent] // 其他服务已停止

	// realtime api event hooks
	onRealtimeConnectRequest   *hook.Hook[*RealtimeConnectRequestEvent]
	onRealtimeMessageSend      *hook.Hook[*RealtimeMessageEvent]
	onRealtimeSubscribeRequest *hook.Hook[*RealtimeSubscribeRequestEvent]
}

// NewBaseApp creates and returns a new BaseApp instance
// configured with the provided arguments.
//
// To initialize the app, you need to call `app.Bootstrap()`.
func NewBaseApp(config BaseAppConfig) *BaseApp {
	app := &BaseApp{
		config:              &config,
		store:               store.New[string, any](nil),
		cronInstance:        cron.New(),
		subscriptionsBroker: subscriptions.NewBroker(),
		cacheSpaces:         make(map[string]*CacheSpace),
		rbacCache:           NewRBACCache(),
	}

	// apply config defaults
	if app.config.DataDir == "" {
		app.config.DataDir = "."
	}
	if app.config.ServiceName == "" {
		app.config.ServiceName = "app"
	}
	if app.config.ServiceVersion == "" {
		app.config.ServiceVersion = "1.0.0"
	}

	app.initHooks()

	return app
}

// initHooks initializes all app hook handlers.
func (app *BaseApp) initHooks() {
	// app event hooks
	app.onBootstrap = &hook.Hook[*BootstrapEvent]{}
	app.onServe = &hook.Hook[*ServeEvent]{}
	app.onTerminate = &hook.Hook[*TerminateEvent]{}

	// lifecycle event hooks (硬编码)
	app.onServiceStarted = &hook.Hook[*LifecycleEvent]{}
	app.onServiceReady = &hook.Hook[*LifecycleEvent]{}
	app.onServiceStopping = &hook.Hook[*LifecycleEvent]{}
	app.onServiceStopped = &hook.Hook[*LifecycleEvent]{}

	// realtime API event hooks
	app.onRealtimeConnectRequest = &hook.Hook[*RealtimeConnectRequestEvent]{}
	app.onRealtimeMessageSend = &hook.Hook[*RealtimeMessageEvent]{}
	app.onRealtimeSubscribeRequest = &hook.Hook[*RealtimeSubscribeRequestEvent]{}
}

// UnsafeWithoutHooks returns a shallow copy of the current app WITHOUT any registered hooks.
func (app *BaseApp) UnsafeWithoutHooks() App {
	clone := *app
	clone.initHooks()
	return &clone
}

// Logger returns the default app logger.
func (app *BaseApp) Logger() *slog.Logger {
	if app.logger == nil {
		return slog.Default()
	}
	return app.logger
}

// IsBootstrapped checks if the application was initialized.
func (app *BaseApp) IsBootstrapped() bool {
	app.mu.RLock()
	defer app.mu.RUnlock()
	return app.isBootstrapped
}

// IsTransactional checks if the current app instance is part of a transaction.
func (app *BaseApp) IsTransactional() bool {
	return false // 简化实现，不支持事务
}

// Bootstrap initializes the application.
func (app *BaseApp) Bootstrap() error {
	event := &BootstrapEvent{}
	event.App = app

	return app.OnBootstrap().Trigger(event, func(e *BootstrapEvent) error {
		// clear resources of previous core state (if any)
		if err := app.ResetBootstrapState(); err != nil {
			return err
		}

		// ensure that data dir exist
		if err := os.MkdirAll(app.DataDir(), os.ModePerm); err != nil {
			return err
		}

		// init logger
		app.logger = slog.Default()

		// try to cleanup the temp directory (if any)
		_ = os.RemoveAll(filepath.Join(app.DataDir(), LocalTempDirName))

		app.mu.Lock()
		app.isBootstrapped = true
		app.mu.Unlock()

		return nil
	})
}

// ResetBootstrapState releases the initialized core app resources.
func (app *BaseApp) ResetBootstrapState() error {
	app.Cron().Stop()

	app.mu.Lock()
	app.isBootstrapped = false
	app.mu.Unlock()

	return nil
}

// DataDir returns the app data directory path.
func (app *BaseApp) DataDir() string {
	return app.config.DataDir
}

// EncryptionEnv returns the name of the app secret env key.
func (app *BaseApp) EncryptionEnv() string {
	return app.config.EncryptionEnv
}

// IsDev returns whether the app is in dev mode.
func (app *BaseApp) IsDev() bool {
	return app.config.IsDev
}

// Store returns the app runtime store.
func (app *BaseApp) Store() *store.Store[string, any] {
	return app.store
}

// Cron returns the app cron instance.
func (app *BaseApp) Cron() *cron.Cron {
	return app.cronInstance
}

// SubscriptionsBroker returns the app realtime subscriptions broker instance.
func (app *BaseApp) SubscriptionsBroker() *subscriptions.Broker {
	return app.subscriptionsBroker
}

// NewFilesystem creates a new local filesystem instance.
func (app *BaseApp) NewFilesystem() (*filesystem.System, error) {
	return filesystem.NewLocal(filepath.Join(app.DataDir(), LocalStorageDirName))
}

// NewBackupsFilesystem creates a new local filesystem instance for backups.
func (app *BaseApp) NewBackupsFilesystem() (*filesystem.System, error) {
	return filesystem.NewLocal(filepath.Join(app.DataDir(), LocalBackupsDirName))
}

// ReloadSettings reinitializes and reloads the stored application settings.
func (app *BaseApp) ReloadSettings() error {
	// 简化实现
	return nil
}

// CreateBackup creates a new backup.
func (app *BaseApp) CreateBackup(ctx context.Context, name string) error {
	return fmt.Errorf("backup not implemented")
}

// RestoreBackup restores a backup.
func (app *BaseApp) RestoreBackup(ctx context.Context, name string) error {
	return fmt.Errorf("restore backup not implemented")
}

// Restart restarts the current running application process.
func (app *BaseApp) Restart() error {
	return fmt.Errorf("restart not supported")
}

// RunSystemMigrations applies all new system migrations.
func (app *BaseApp) RunSystemMigrations() error {
	return nil
}

// RunAppMigrations applies all new app migrations.
func (app *BaseApp) RunAppMigrations() error {
	return nil
}

// RunAllMigrations applies all system and app migrations.
func (app *BaseApp) RunAllMigrations() error {
	return nil
}

// -------------------------------------------------------------------
// Service Registry Methods
// -------------------------------------------------------------------

// SetRegistry sets the go-micro service registry.
func (app *BaseApp) SetRegistry(reg registry.Registry) *BaseApp {
	app.registry = reg
	return app
}

// Registry returns the go-micro service registry.
func (app *BaseApp) Registry() registry.Registry {
	return app.registry
}

// SetService sets the go-micro service for registration.
func (app *BaseApp) SetService(svc *registry.Service) *BaseApp {
	app.regService = svc
	// 同时更新简化的服务信息
	if svc != nil && len(svc.Nodes) > 0 {
		node := svc.Nodes[0]
		basePath := ""
		if node.Metadata != nil {
			basePath = node.Metadata["base_path"]
		}
		app.serviceInfo = &ServiceInfo{
			Name:     svc.Name,
			Version:  svc.Version,
			Address:  node.Address,
			BasePath: basePath,
			Metadata: node.Metadata,
		}
	}
	return app
}

// Service returns the go-micro service.
func (app *BaseApp) Service() *registry.Service {
	return app.regService
}

// -------------------------------------------------------------------
// Cache Methods
// -------------------------------------------------------------------

// GetCacheSpace 获取或创建缓存空间
func (app *BaseApp) GetCacheSpace(module string) *CacheSpace {
	app.cacheMu.RLock()
	cs, ok := app.cacheSpaces[module]
	app.cacheMu.RUnlock()

	if ok {
		return cs
	}

	// 创建新的缓存空间
	app.cacheMu.Lock()
	defer app.cacheMu.Unlock()

	// 双重检查
	if cs, ok = app.cacheSpaces[module]; ok {
		return cs
	}

	cs = NewCacheSpace(module)
	app.cacheSpaces[module] = cs
	return cs
}

// RBACCache 获取RBAC缓存
func (app *BaseApp) RBACCache() *RBACCache {
	return app.rbacCache
}

// UpdateRBACCache 更新RBAC缓存数据
func (app *BaseApp) UpdateRBACCache(data RBACData) {
	app.rbacCache.Update(data)
	app.Logger().Info("RBAC cache updated",
		"roles", len(data.Roles),
		"permissions", len(data.Permissions),
		"scopes", len(data.PermissionScopes),
	)
}

// ClearModuleCache 清空指定模块的缓存
func (app *BaseApp) ClearModuleCache(module string) {
	app.cacheMu.Lock()
	defer app.cacheMu.Unlock()

	if cs, ok := app.cacheSpaces[module]; ok {
		cs.Clear()
	}
}

// ClearAllCache 清空所有缓存
func (app *BaseApp) ClearAllCache() {
	app.cacheMu.Lock()
	defer app.cacheMu.Unlock()

	for _, cs := range app.cacheSpaces {
		cs.Clear()
	}
}

// -------------------------------------------------------------------
// Lifecycle Event Methods
// -------------------------------------------------------------------

// OnServiceStarted 返回服务启动事件钩子（收到其他服务启动通知时触发）
func (app *BaseApp) OnServiceStarted() *hook.Hook[*LifecycleEvent] {
	return app.onServiceStarted
}

// OnServiceReady 返回服务就绪事件钩子（收到其他服务就绪通知时触发）
func (app *BaseApp) OnServiceReady() *hook.Hook[*LifecycleEvent] {
	return app.onServiceReady
}

// OnServiceStopping 返回服务停止中事件钩子（收到其他服务停止通知时触发）
func (app *BaseApp) OnServiceStopping() *hook.Hook[*LifecycleEvent] {
	return app.onServiceStopping
}

// OnServiceStopped 返回服务已停止事件钩子（收到其他服务已停止通知时触发）
func (app *BaseApp) OnServiceStopped() *hook.Hook[*LifecycleEvent] {
	return app.onServiceStopped
}

// publishLifecycleEvent 发布生命周期事件（内部使用）
func (app *BaseApp) publishLifecycleEvent(event string, metadata any) error {
	if app.broadcaster == nil {
		return nil
	}

	msg := LifecycleMessage{
		Service:   app.ServiceName(),
		NodeID:    app.getNodeID(),
		Event:     event,
		Timestamp: time.Now(),
		Metadata:  metadata,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal lifecycle message: %w", err)
	}

	return app.broadcaster.Send(lifecycleTopic, data, app.ServiceName())
}

// getNodeID 获取节点ID
func (app *BaseApp) getNodeID() string {
	if app.regService != nil && len(app.regService.Nodes) > 0 {
		return app.regService.Nodes[0].Address
	}
	hostname, _ := os.Hostname()
	return hostname
}

// SubscribeLifecycleTopic 订阅生命周期主题
func (app *BaseApp) SubscribeLifecycleTopic() error {
	if app.broadcaster == nil {
		return nil
	}
	return app.broadcaster.Subscribe(lifecycleTopic, app.handleLifecycleBroadcast)
}

// SubscribeTopic 订阅自定义主题
func (app *BaseApp) SubscribeTopic(topic string, handler func(payload []byte)) error {
	if app.broadcaster == nil {
		return fmt.Errorf("broadcaster not configured")
	}
	return app.broadcaster.Subscribe(topic, handler)
}

// PublishTopic 发布消息到自定义主题
func (app *BaseApp) PublishTopic(topic string, payload []byte) error {
	if app.broadcaster == nil {
		return fmt.Errorf("broadcaster not configured")
	}
	return app.broadcaster.Send(topic, payload, app.ServiceName())
}

// SetServiceInfo sets the simplified service info (deprecated, use SetService).
func (app *BaseApp) SetServiceInfo(info *ServiceInfo) *BaseApp {
	app.serviceInfo = info
	return app
}

// ServiceInfo returns the simplified service info.
func (app *BaseApp) ServiceInfo() *ServiceInfo {
	return app.serviceInfo
}

// SetBroadcaster sets the broadcaster.
func (app *BaseApp) SetBroadcaster(b Broadcaster) *BaseApp {
	app.broadcaster = b
	return app
}

// Broadcaster returns the broadcaster.
func (app *BaseApp) GetBroadcaster() Broadcaster {
	return app.broadcaster
}

// ServiceName returns the service name.
func (app *BaseApp) ServiceName() string {
	return app.config.ServiceName
}

// ServiceVersion returns the service version.
func (app *BaseApp) ServiceVersion() string {
	return app.config.ServiceVersion
}

// -------------------------------------------------------------------
// App event hooks
// -------------------------------------------------------------------

func (app *BaseApp) OnBootstrap() *hook.Hook[*BootstrapEvent] {
	return app.onBootstrap
}

func (app *BaseApp) OnServe() *hook.Hook[*ServeEvent] {
	return app.onServe
}

func (app *BaseApp) OnTerminate() *hook.Hook[*TerminateEvent] {
	return app.onTerminate
}

func (app *BaseApp) OnRealtimeConnectRequest() *hook.Hook[*RealtimeConnectRequestEvent] {
	return app.onRealtimeConnectRequest
}

func (app *BaseApp) OnRealtimeMessageSend() *hook.Hook[*RealtimeMessageEvent] {
	return app.onRealtimeMessageSend
}

func (app *BaseApp) OnRealtimeSubscribeRequest() *hook.Hook[*RealtimeSubscribeRequestEvent] {
	return app.onRealtimeSubscribeRequest
}

// -------------------------------------------------------------------
// Serve - HTTP Server Management
// -------------------------------------------------------------------

// ServeConfig defines the HTTP server configuration.
type ServeConfig struct {
	// ShowStartBanner indicates whether to show the server start console message.
	ShowStartBanner bool

	// HttpAddr is the TCP address to listen for the HTTP server.
	HttpAddr string

	// AllowedOrigins is an optional list of CORS origins (default to "*").
	AllowedOrigins []string
}

// Serve starts the HTTP server with the given configuration.
func (app *BaseApp) Serve(config ServeConfig) error {
	if !app.IsBootstrapped() {
		if err := app.Bootstrap(); err != nil {
			return err
		}
	}

	if config.HttpAddr == "" {
		config.HttpAddr = ":8080"
	}
	if len(config.AllowedOrigins) == 0 {
		config.AllowedOrigins = []string{"*"}
	}

	// 创建路由
	pbRouter := router.NewRouter(func(w http.ResponseWriter, r *http.Request) (*RequestEvent, router.EventCleanupFunc) {
		event := new(RequestEvent)
		event.Response = w
		event.Request = r
		event.App = app
		return event, nil
	})

	// 基础请求上下文
	baseCtx, cancelBaseCtx := context.WithCancel(context.Background())
	defer cancelBaseCtx()

	server := &http.Server{
		WriteTimeout:      5 * time.Minute,
		ReadTimeout:       5 * time.Minute,
		ReadHeaderTimeout: 1 * time.Minute,
		Addr:              config.HttpAddr,
		BaseContext: func(l net.Listener) context.Context {
			return baseCtx
		},
	}

	var listener net.Listener
	var wg sync.WaitGroup

	// 注册优雅关闭处理
	app.OnTerminate().Bind(&hook.Handler[*TerminateEvent]{
		Id: "pbGracefulShutdown",
		Func: func(te *TerminateEvent) error {
			cancelBaseCtx()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			wg.Add(1)
			_ = server.Shutdown(ctx)
			wg.Done()

			return te.Next()
		},
		Priority: -9999,
	})

	defer func() {
		wg.Wait()
		if listener != nil {
			_ = listener.Close()
		}
	}()

	// 触发 OnServe 事件
	serveEvent := &ServeEvent{
		App:    app,
		Router: pbRouter,
		Server: server,
	}

	serveHookErr := app.OnServe().Trigger(serveEvent, func(e *ServeEvent) error {
		handler, err := e.Router.BuildMux()
		if err != nil {
			return err
		}
		e.Server.Handler = handler

		var lErr error
		listener, lErr = net.Listen("tcp", e.Server.Addr)
		if lErr != nil {
			return lErr
		}

		return nil
	})
	if serveHookErr != nil {
		return serveHookErr
	}

	if listener == nil {
		return fmt.Errorf("the OnServe listener was not initialized")
	}

	// 注册服务
	if app.registry != nil && app.regService != nil {
		if err := app.registry.Register(app.regService); err != nil {
			app.Logger().Error("register service failed", "error", err)
		}
	}

	// 启动广播器
	if app.broadcaster != nil {
		app.broadcaster.Subscribe(lifecycleTopic, app.handleLifecycleBroadcast)
		if err := app.broadcaster.Start(); err != nil {
			app.Logger().Error("start broadcaster failed", "error", err)
		}
		// 发布启动事件
		_ = app.publishLifecycleEvent("started", nil)
	}

	if config.ShowStartBanner {
		fmt.Printf("Server [%s] started at http://%s\n", app.ServiceName(), config.HttpAddr)
	}

	// 发布就绪事件
	_ = app.publishLifecycleEvent("ready", nil)

	// 处理退出信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	errCh := make(chan error, 1)
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case <-quit:
		app.Logger().Info("Received shutdown signal")
		// 发布停止事件
		_ = app.publishLifecycleEvent("stopping", nil)
	case err := <-errCh:
		return err
	}

	// 触发终止事件
	terminateEvent := &TerminateEvent{App: app}
	if err := app.OnTerminate().Trigger(terminateEvent, func(e *TerminateEvent) error {
		return e.Next()
	}); err != nil {
		app.Logger().Error("terminate hook failed", "error", err)
	}

	// 注销服务
	if app.registry != nil && app.regService != nil {
		if err := app.registry.Deregister(app.regService); err != nil {
			app.Logger().Error("deregister service failed", "error", err)
		}
	}

	// 停止广播器
	if app.broadcaster != nil {
		// 发布已停止事件
		_ = app.publishLifecycleEvent("stopped", nil)
		app.broadcaster.Stop()
	}

	return nil
}

// handleLifecycleBroadcast handles lifecycle broadcast messages.
func (app *BaseApp) handleLifecycleBroadcast(payload []byte) {
	var msg LifecycleMessage
	if err := json.Unmarshal(payload, &msg); err != nil {
		app.Logger().Error("unmarshal lifecycle message failed", "error", err)
		return
	}

	app.Logger().Debug("received lifecycle message",
		"service", msg.Service,
		"event", msg.Event,
	)

	// 构造钩子事件
	event := &LifecycleEvent{
		App:     app,
		Message: &msg,
	}

	// 根据事件类型触发对应的硬编码钩子
	var hook *hook.Hook[*LifecycleEvent]
	switch msg.Event {
	case "started":
		hook = app.onServiceStarted
	case "ready":
		hook = app.onServiceReady
	case "stopping":
		hook = app.onServiceStopping
	case "stopped":
		hook = app.onServiceStopped
	default:
		app.Logger().Warn("unknown lifecycle event", "event", msg.Event)
		return
	}

	if err := hook.Trigger(event, func(e *LifecycleEvent) error {
		return e.Next()
	}); err != nil {
		app.Logger().Error("lifecycle event hook failed", "error", err, "event", msg.Event)
	}
}
