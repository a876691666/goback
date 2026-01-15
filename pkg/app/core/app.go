// Package core is the backbone of PocketBase.
//
// It defines the main PocketBase App interface and its base implementation.
package core

import (
	"context"
	"log/slog"

	"go-micro.dev/v5/registry"

	"github.com/goback/pkg/app/tools/cron"
	"github.com/goback/pkg/app/tools/filesystem"
	"github.com/goback/pkg/app/tools/hook"
	"github.com/goback/pkg/app/tools/store"
	"github.com/goback/pkg/app/tools/subscriptions"
)

// App defines the main PocketBase app interface.
//
// Note that the interface is not intended to be implemented manually by users
// and instead they should use core.BaseApp (either directly or as embedded field in a custom struct).
//
// This interface exists to make testing easier and to allow users to
// create common and pluggable helpers and methods that doesn't rely
// on a specific wrapped app struct (hence the large interface size).
type App interface {
	// UnsafeWithoutHooks returns a shallow copy of the current app WITHOUT any registered hooks.
	//
	// NB! Note that using the returned app instance may cause data integrity errors
	// since the Record validations and data normalizations (including files uploads)
	// rely on the app hooks to work.
	UnsafeWithoutHooks() App

	// Logger returns the default app logger.
	//
	// If the application is not bootstrapped yet, fallbacks to slog.Default().
	Logger() *slog.Logger

	// IsBootstrapped checks if the application was initialized
	// (aka. whether Bootstrap() was called).
	IsBootstrapped() bool

	// IsTransactional checks if the current app instance is part of a transaction.
	IsTransactional() bool

	// Bootstrap initializes the application
	// (aka. create data dir, open db connections, load settings, etc.).
	//
	// It will call ResetBootstrapState() if the application was already bootstrapped.
	Bootstrap() error

	// ResetBootstrapState releases the initialized core app resources
	// (closing db connections, stopping cron ticker, etc.).
	ResetBootstrapState() error

	// DataDir returns the app data directory path.
	DataDir() string

	// EncryptionEnv returns the name of the app secret env key
	// (currently used primarily for optional settings encryption but this may change in the future).
	EncryptionEnv() string

	// IsDev returns whether the app is in dev mode.
	//
	// When enabled logs, executed sql statements, etc. are printed to the stderr.
	IsDev() bool

	// Store returns the app runtime store.
	Store() *store.Store[string, any]

	// Cron returns the app cron instance.
	Cron() *cron.Cron

	// SubscriptionsBroker returns the app realtime subscriptions broker instance.
	SubscriptionsBroker() *subscriptions.Broker

	// NewFilesystem creates a new local or S3 filesystem instance
	// for managing regular app files (ex. record uploads)
	// based on the current app settings.
	//
	// NB! Make sure to call Close() on the returned result
	// after you are done working with it.
	NewFilesystem() (*filesystem.System, error)

	// NewBackupsFilesystem creates a new local or S3 filesystem instance
	// for managing app backups based on the current app settings.
	//
	// NB! Make sure to call Close() on the returned result
	// after you are done working with it.
	NewBackupsFilesystem() (*filesystem.System, error)

	// ReloadSettings reinitializes and reloads the stored application settings.
	ReloadSettings() error

	// CreateBackup creates a new backup of the current app pb_data directory.
	//
	// Backups can be stored on S3 if it is configured in app.Settings().Backups.
	//
	// Please refer to the godoc of the specific core.App implementation
	// for details on the backup procedures.
	CreateBackup(ctx context.Context, name string) error

	// RestoreBackup restores the backup with the specified name and restarts
	// the current running application process.
	//
	// The safely perform the restore it is recommended to have free disk space
	// for at least 2x the size of the restored pb_data backup.
	//
	// Please refer to the godoc of the specific core.App implementation
	// for details on the restore procedures.
	//
	// NB! This feature is experimental and currently is expected to work only on UNIX based systems.
	RestoreBackup(ctx context.Context, name string) error

	// Restart restarts (aka. replaces) the current running application process.
	//
	// NB! It relies on execve which is supported only on UNIX based systems.
	Restart() error

	// RunSystemMigrations applies all new migrations registered in the [core.SystemMigrations] list.
	RunSystemMigrations() error

	// RunAppMigrations applies all new migrations registered in the [core.AppMigrations] list.
	RunAppMigrations() error

	// RunAllMigrations applies all system and app migrations
	// (aka. from both [core.SystemMigrations] and [core.AppMigrations]).
	RunAllMigrations() error

	// ---------------------------------------------------------------
	// App event hooks
	// ---------------------------------------------------------------

	// OnBootstrap hook is triggered when initializing the main application
	// resources (db, app settings, etc).
	OnBootstrap() *hook.Hook[*BootstrapEvent]

	// OnServe hook is triggered when the app web server is started
	// (after starting the TCP listener but before initializing the blocking serve task),
	// allowing you to adjust its options and attach new routes or middlewares.
	OnServe() *hook.Hook[*ServeEvent]

	// OnTerminate hook is triggered when the app is in the process
	// of being terminated (ex. on SIGTERM signal).
	//
	// Note that the app could be terminated abruptly without awaiting the hook completion.
	OnTerminate() *hook.Hook[*TerminateEvent]

	// ---------------------------------------------------------------
	// Realtime API event hooks
	// ---------------------------------------------------------------

	// OnRealtimeConnectRequest hook is triggered when establishing the SSE client connection.
	//
	// Any execution after e.Next() of a hook handler happens after the client disconnects.
	OnRealtimeConnectRequest() *hook.Hook[*RealtimeConnectRequestEvent]

	// OnRealtimeMessageSend hook is triggered when sending an SSE message to a client.
	OnRealtimeMessageSend() *hook.Hook[*RealtimeMessageEvent]

	// OnRealtimeSubscribeRequest hook is triggered when updating the
	// client subscriptions, allowing you to further validate and
	// modify the submitted change.
	OnRealtimeSubscribeRequest() *hook.Hook[*RealtimeSubscribeRequestEvent]

	// ---------------------------------------------------------------
	// Lifecycle event hooks (硬编码钩子，收到其他服务的生命周期广播时触发)
	// ---------------------------------------------------------------

	// OnServiceStarted hook is triggered when receiving a "started" lifecycle message from another service.
	OnServiceStarted() *hook.Hook[*LifecycleEvent]

	// OnServiceReady hook is triggered when receiving a "ready" lifecycle message from another service.
	OnServiceReady() *hook.Hook[*LifecycleEvent]

	// OnServiceStopping hook is triggered when receiving a "stopping" lifecycle message from another service.
	OnServiceStopping() *hook.Hook[*LifecycleEvent]

	// OnServiceStopped hook is triggered when receiving a "stopped" lifecycle message from another service.
	OnServiceStopped() *hook.Hook[*LifecycleEvent]

	// ---------------------------------------------------------------
	// Cache and RBAC Methods
	// ---------------------------------------------------------------

	// GetCacheSpace returns or creates a cache space for the given module.
	GetCacheSpace(module string) *CacheSpace

	// RBACCache returns the RBAC cache manager.
	RBACCache() *RBACCache

	// UpdateRBACCache updates the RBAC cache with the given data.
	UpdateRBACCache(data RBACData)

	// ClearModuleCache clears the cache for the given module.
	ClearModuleCache(module string)

	// ClearAllCache clears all caches.
	ClearAllCache()

	// ---------------------------------------------------------------
	// Service Registry Methods
	// ---------------------------------------------------------------

	// SetRegistry sets the go-micro service registry.
	SetRegistry(reg registry.Registry) *BaseApp

	// Registry returns the go-micro service registry.
	Registry() registry.Registry

	// SetService sets the go-micro service for registration.
	SetService(svc *registry.Service) *BaseApp

	// Service returns the go-micro service.
	Service() *registry.Service

	// SetServiceInfo sets the simplified service info (deprecated, use SetService).
	SetServiceInfo(info *ServiceInfo) *BaseApp

	// ServiceInfo returns the simplified service info.
	ServiceInfo() *ServiceInfo

	// SetBroadcaster sets the broadcaster.
	SetBroadcaster(b Broadcaster) *BaseApp

	// GetBroadcaster returns the broadcaster.
	GetBroadcaster() Broadcaster

	// ServiceName returns the service name.
	ServiceName() string

	// ServiceVersion returns the service version.
	ServiceVersion() string

	// ---------------------------------------------------------------
	// Broadcaster/PubSub Methods
	// ---------------------------------------------------------------

	// SubscribeLifecycleTopic subscribes to the lifecycle topic.
	SubscribeLifecycleTopic() error

	// SubscribeTopic subscribes to a custom topic.
	SubscribeTopic(topic string, handler func(payload []byte)) error

	// PublishTopic publishes a message to a custom topic.
	PublishTopic(topic string, payload []byte) error
}
