//go:build windows

package notification

import (
	"context"

	"github.com/hkdb/aerion/internal/logging"
	"github.com/rs/zerolog"
)

// windowsNotifier is a stub for Windows notification support.
// TODO: Implement using Windows Toast notifications with activation handling.
type windowsNotifier struct {
	appName      string
	clickHandler ClickHandler
	log          zerolog.Logger
}

func newPlatformNotifier(appName string, useDirectDBus bool) Notifier {
	// useDirectDBus is Linux-only, ignored on Windows
	return &windowsNotifier{
		appName: appName,
		log:     logging.WithComponent("notification"),
	}
}

func (n *windowsNotifier) Start(ctx context.Context) error {
	n.log.Info().Msg("Windows notification support not yet implemented")
	return nil
}

func (n *windowsNotifier) Stop() {
	// Nothing to clean up
}

func (n *windowsNotifier) Show(notif Notification) (uint32, error) {
	// TODO: Implement Windows Toast notifications
	n.log.Debug().Str("title", notif.Title).Str("body", notif.Body).Msg("Windows notification (not implemented)")
	return 0, nil
}

func (n *windowsNotifier) SetClickHandler(handler ClickHandler) {
	n.clickHandler = handler
	// TODO: Implement click handling via Toast activation
}
