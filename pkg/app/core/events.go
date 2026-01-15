package core

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/goback/pkg/app/tools/hook"
	"github.com/goback/pkg/app/tools/router"
	"github.com/goback/pkg/app/tools/subscriptions"
	"golang.org/x/crypto/acme/autocert"
)

// -------------------------------------------------------------------
// App events data
// -------------------------------------------------------------------

type BootstrapEvent struct {
	hook.Event
	App App
}

type TerminateEvent struct {
	hook.Event
	App       App
	IsRestart bool
}

type BackupEvent struct {
	hook.Event
	App     App
	Context context.Context
	Name    string   // the name of the backup to create/restore.
	Exclude []string // list of dir entries to exclude from the backup create/restore.
}

type ServeEvent struct {
	hook.Event
	App         App
	Router      *router.Router[*RequestEvent]
	Server      *http.Server
	CertManager *autocert.Manager

	// Listener allow specifying a custom network listener.
	//
	// Leave it nil to use the default net.Listen("tcp", e.Server.Addr).
	Listener net.Listener
}

// -------------------------------------------------------------------
// Realtime API events data
// -------------------------------------------------------------------

type RealtimeConnectRequestEvent struct {
	hook.Event
	*RequestEvent

	Client subscriptions.Client

	// note: modifying it after the connect has no effect
	IdleTimeout time.Duration
}

type RealtimeMessageEvent struct {
	hook.Event
	*RequestEvent

	Client  subscriptions.Client
	Message *subscriptions.Message
}

type RealtimeSubscribeRequestEvent struct {
	hook.Event
	*RequestEvent

	Client        subscriptions.Client
	Subscriptions []string
}
