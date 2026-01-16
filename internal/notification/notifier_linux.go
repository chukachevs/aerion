//go:build linux

package notification

import (
	"context"
	"sync"

	"github.com/godbus/dbus/v5"
	"github.com/hkdb/aerion/internal/logging"
	"github.com/rs/zerolog"
)

const (
	dbusNotifyDest      = "org.freedesktop.Notifications"
	dbusNotifyPath      = "/org/freedesktop/Notifications"
	dbusNotifyInterface = "org.freedesktop.Notifications"
)

type linuxNotifier struct {
	appName       string
	conn          *dbus.Conn
	clickHandler  ClickHandler
	notifications map[uint32]NotificationData
	mu            sync.RWMutex
	log           zerolog.Logger
	cancel        context.CancelFunc
}

func newPlatformNotifier(appName string) Notifier {
	return &linuxNotifier{
		appName:       appName,
		notifications: make(map[uint32]NotificationData),
		log:           logging.WithComponent("notification"),
	}
}

func (n *linuxNotifier) Start(ctx context.Context) error {
	var err error
	n.conn, err = dbus.ConnectSessionBus()
	if err != nil {
		return err
	}

	// Subscribe to notification signals
	if err := n.conn.AddMatchSignal(
		dbus.WithMatchObjectPath(dbusNotifyPath),
		dbus.WithMatchInterface(dbusNotifyInterface),
	); err != nil {
		n.conn.Close()
		return err
	}

	// Create cancellable context for signal handling
	signalCtx, cancel := context.WithCancel(ctx)
	n.cancel = cancel

	// Start listening for signals
	signals := make(chan *dbus.Signal, 10)
	n.conn.Signal(signals)

	go n.handleSignals(signalCtx, signals)

	n.log.Info().Msg("Linux notification listener started")
	return nil
}

func (n *linuxNotifier) Stop() {
	if n.cancel != nil {
		n.cancel()
	}
	if n.conn != nil {
		n.conn.Close()
	}
	n.log.Info().Msg("Linux notification listener stopped")
}

func (n *linuxNotifier) Show(notif Notification) (uint32, error) {
	if n.conn == nil {
		// Fallback: not started, try to connect
		var err error
		n.conn, err = dbus.ConnectSessionBus()
		if err != nil {
			return 0, err
		}
	}

	obj := n.conn.Object(dbusNotifyDest, dbusNotifyPath)

	// Actions: "default" is the click action
	actions := []string{"default", "Open"}

	// Hints for better notification behavior
	hints := map[string]dbus.Variant{
		"urgency":  dbus.MakeVariant(byte(1)), // Normal urgency
		"category": dbus.MakeVariant("email.arrived"),
	}

	// Icon - use mail icon
	icon := notif.Icon
	if icon == "" {
		icon = "mail-unread"
	}

	// Call Notify method
	// Notify(app_name, replaces_id, icon, summary, body, actions, hints, timeout)
	call := obj.Call(
		dbusNotifyInterface+".Notify",
		0,
		n.appName,      // app_name
		uint32(0),      // replaces_id (0 = new notification)
		icon,           // icon
		notif.Title,    // summary
		notif.Body,     // body
		actions,        // actions
		hints,          // hints
		int32(-1),      // timeout (-1 = default)
	)

	if call.Err != nil {
		return 0, call.Err
	}

	var id uint32
	if err := call.Store(&id); err != nil {
		return 0, err
	}

	// Store notification data for click handling
	n.mu.Lock()
	n.notifications[id] = notif.Data
	n.mu.Unlock()

	n.log.Debug().Uint32("id", id).Str("title", notif.Title).Msg("Notification shown")
	return id, nil
}

func (n *linuxNotifier) SetClickHandler(handler ClickHandler) {
	n.mu.Lock()
	n.clickHandler = handler
	n.mu.Unlock()
}

func (n *linuxNotifier) handleSignals(ctx context.Context, signals chan *dbus.Signal) {
	for {
		select {
		case <-ctx.Done():
			return
		case signal := <-signals:
			if signal == nil {
				continue
			}
			n.handleSignal(signal)
		}
	}
}

func (n *linuxNotifier) handleSignal(signal *dbus.Signal) {
	switch signal.Name {
	case dbusNotifyInterface + ".ActionInvoked":
		// ActionInvoked(id uint32, action_key string)
		if len(signal.Body) >= 2 {
			id, ok1 := signal.Body[0].(uint32)
			action, ok2 := signal.Body[1].(string)
			if ok1 && ok2 {
				n.handleAction(id, action)
			}
		}

	case dbusNotifyInterface + ".NotificationClosed":
		// NotificationClosed(id uint32, reason uint32)
		if len(signal.Body) >= 1 {
			if id, ok := signal.Body[0].(uint32); ok {
				n.mu.Lock()
				delete(n.notifications, id)
				n.mu.Unlock()
			}
		}
	}
}

func (n *linuxNotifier) handleAction(id uint32, action string) {
	n.log.Debug().Uint32("id", id).Str("action", action).Msg("Notification action invoked")

	// Get notification data
	n.mu.RLock()
	data, exists := n.notifications[id]
	handler := n.clickHandler
	n.mu.RUnlock()

	if !exists {
		n.log.Debug().Uint32("id", id).Msg("Notification data not found")
		return
	}

	// Handle "default" action (click on notification body)
	if action == "default" && handler != nil {
		n.log.Info().
			Str("accountId", data.AccountID).
			Str("folderId", data.FolderID).
			Str("threadId", data.ThreadID).
			Msg("Notification clicked, invoking handler")
		handler(data)
	}

	// Clean up
	n.mu.Lock()
	delete(n.notifications, id)
	n.mu.Unlock()
}
