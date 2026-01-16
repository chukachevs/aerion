//go:build darwin

package notification

import (
	"context"
	"os/exec"

	"github.com/hkdb/aerion/internal/logging"
	"github.com/rs/zerolog"
)

// darwinNotifier uses osascript for notifications on macOS.
// TODO: Implement using UNUserNotificationCenter for click handling.
type darwinNotifier struct {
	appName      string
	clickHandler ClickHandler
	log          zerolog.Logger
}

func newPlatformNotifier(appName string) Notifier {
	return &darwinNotifier{
		appName: appName,
		log:     logging.WithComponent("notification"),
	}
}

func (n *darwinNotifier) Start(ctx context.Context) error {
	n.log.Info().Msg("macOS notification support started (click handling not yet implemented)")
	return nil
}

func (n *darwinNotifier) Stop() {
	// Nothing to clean up for osascript
}

func (n *darwinNotifier) Show(notif Notification) (uint32, error) {
	// Use osascript for basic notifications (no click handling)
	script := `display notification "` + notif.Body + `" with title "` + notif.Title + `"`
	cmd := exec.Command("osascript", "-e", script)
	if err := cmd.Run(); err != nil {
		n.log.Debug().Err(err).Msg("Failed to send notification via osascript")
		return 0, err
	}
	return 0, nil
}

func (n *darwinNotifier) SetClickHandler(handler ClickHandler) {
	n.clickHandler = handler
	// TODO: Implement click handling via UNUserNotificationCenter
}
