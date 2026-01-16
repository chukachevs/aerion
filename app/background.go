package app

import (
	"context"
	"fmt"
	"time"

	"github.com/hkdb/aerion/internal/folder"
	"github.com/hkdb/aerion/internal/imap"
	"github.com/hkdb/aerion/internal/logging"
	"github.com/hkdb/aerion/internal/notification"
	"github.com/hkdb/aerion/internal/platform"
	"github.com/hkdb/aerion/internal/sync"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// ============================================================================
// Background Email Sync (Polling + IDLE)
// ============================================================================

// initBackgroundSync initializes and starts the background sync scheduler
// and IMAP IDLE manager for real-time email notifications
func (a *App) initBackgroundSync(ctx context.Context) {
	log := logging.WithComponent("app")

	// Initialize the sync scheduler for periodic polling
	a.syncScheduler = sync.NewScheduler(a.syncEngine, a.accountStore, a.folderStore)

	// Set callback for new mail notifications
	a.syncScheduler.SetNewMailCallback(func(info sync.NewMailInfo) {
		a.handleNewMailNotification(info)
	})

	// Set callback for sync completion (so frontend clears progress)
	a.syncScheduler.SetSyncCompletedCallback(func(accountID, folderID string, err error) {
		if err != nil {
			wailsRuntime.EventsEmit(a.ctx, "folder:syncError", map[string]interface{}{
				"accountId": accountID,
				"folderId":  folderID,
				"error":     err.Error(),
			})
		} else {
			wailsRuntime.EventsEmit(a.ctx, "folder:synced", map[string]interface{}{
				"accountId": accountID,
				"folderId":  folderID,
			})
		}
	})

	// Start the polling scheduler
	a.syncScheduler.Start(ctx)
	log.Info().Msg("Email sync scheduler started")

	// Initialize the IDLE manager for real-time push notifications
	idleConfig := imap.DefaultIdleConfig()
	a.idleManager = imap.NewIdleManager(idleConfig, a.getIMAPCredentials)
	a.idleManager.Start(ctx)

	// Start IDLE for all enabled accounts
	accounts, err := a.accountStore.List()
	if err != nil {
		log.Error().Err(err).Msg("Failed to list accounts for IDLE")
	} else {
		for _, acc := range accounts {
			if acc.Enabled {
				a.idleManager.StartAccount(acc.ID, acc.Name)
			}
		}
	}

	// Start goroutine to process IDLE events
	go a.processIdleEvents(ctx)

	log.Info().Msg("IDLE manager started")
}

// processIdleEvents processes mail events from IDLE connections
func (a *App) processIdleEvents(ctx context.Context) {
	log := logging.WithComponent("app.idle")

	for {
		select {
		case <-ctx.Done():
			return
		case event := <-a.idleManager.Events():
			log.Debug().
				Str("type", event.Type.String()).
				Str("accountID", event.AccountID).
				Str("folder", event.Folder).
				Uint32("count", event.Count).
				Msg("Received IDLE event")

			switch event.Type {
			case imap.EventNewMail:
				// New mail arrived - trigger sync for this account's INBOX
				go a.handleIdleNewMail(event)

			case imap.EventExpunge:
				// Message deleted - could refresh the folder view
				// For now, just emit an event to the frontend
				wailsRuntime.EventsEmit(a.ctx, "mail:expunge", map[string]interface{}{
					"accountId": event.AccountID,
					"folder":    event.Folder,
					"seqNum":    event.SeqNum,
				})

			case imap.EventFlagsChanged:
				// Flags changed - could refresh the message
				wailsRuntime.EventsEmit(a.ctx, "mail:flagsChanged", map[string]interface{}{
					"accountId": event.AccountID,
					"folder":    event.Folder,
					"seqNum":    event.SeqNum,
				})
			}
		}
	}
}

// handleIdleNewMail handles a new mail event from IDLE
func (a *App) handleIdleNewMail(event imap.MailEvent) {
	log := logging.WithComponent("app.idle")

	log.Info().
		Str("accountID", event.AccountID).
		Uint32("count", event.Count).
		Msg("New mail detected via IDLE, triggering sync")

	// Get the INBOX folder ID for events
	inbox, _ := a.folderStore.GetByType(event.AccountID, folder.TypeInbox)
	var folderID string
	if inbox != nil {
		folderID = inbox.ID
	}

	// Use composite key for sync tracking
	syncKey := event.AccountID + ":" + folderID

	// Check if a sync is already running for this folder - skip IDLE sync if so
	a.syncMu.Lock()
	if _, exists := a.syncContexts[syncKey]; exists {
		a.syncMu.Unlock()
		log.Debug().Str("syncKey", syncKey).Msg("Skipping IDLE sync - sync already in progress")
		return
	}
	a.syncMu.Unlock()

	// Use the scheduler's blocking sync to get new mail info
	newMailInfo, err := a.syncScheduler.SyncAccountInboxBlocking(event.AccountID)

	if err != nil {
		log.Error().Err(err).Str("accountID", event.AccountID).Msg("Failed to sync after IDLE notification")
		// Emit folder:synced to clear syncing state even on error
		if folderID != "" {
			wailsRuntime.EventsEmit(a.ctx, "folder:synced", map[string]interface{}{
				"accountId": event.AccountID,
				"folderId":  folderID,
			})
		}
		return
	}

	// Fetch bodies in background (same as SyncFolder does)
	if folderID != "" {
		// Get account's sync period
		syncPeriodDays := 30 // default
		if acc, accErr := a.accountStore.Get(event.AccountID); accErr == nil && acc != nil {
			syncPeriodDays = acc.SyncPeriodDays
		}

		// Register IDLE sync context so manual sync can cancel it
		a.syncMu.Lock()
		// Double-check no sync started while we were processing
		if _, exists := a.syncContexts[syncKey]; exists {
			a.syncMu.Unlock()
			log.Debug().Str("syncKey", syncKey).Msg("Skipping IDLE body fetch - sync started during processing")
			return
		}
		ctx, cancel := context.WithCancel(a.ctx)
		a.syncContexts[syncKey] = cancel
		a.syncMu.Unlock()

		go func(syncCtx context.Context, syncDays int, fID string, key string) {
			// Cleanup context on completion
			defer func() {
				a.syncMu.Lock()
				delete(a.syncContexts, key)
				a.syncMu.Unlock()

				// Also emit messages:updated so the message list refreshes
				wailsRuntime.EventsEmit(a.ctx, "messages:updated", map[string]interface{}{
					"accountId": event.AccountID,
					"folderId":  fID,
				})
				// Emit folder counts changed so sidebar unread badge updates
				if updatedFolder, err := a.folderStore.Get(fID); err == nil && updatedFolder != nil {
					wailsRuntime.EventsEmit(a.ctx, "folders:countsChanged", map[string]int{
						fID: updatedFolder.UnreadCount,
					})
				}
			}()

			// Panic recovery - ensure we always emit an event so UI doesn't get stuck
			defer func() {
				if r := recover(); r != nil {
					log.Error().Interface("panic", r).Str("folder", fID).Msg("IDLE body fetch goroutine panicked")
					wailsRuntime.EventsEmit(a.ctx, "folder:syncError", map[string]interface{}{
						"accountId": event.AccountID,
						"folderId":  fID,
						"error":     fmt.Sprintf("body fetch panic: %v", r),
					})
				}
			}()

			bodyErr := a.syncEngine.FetchBodiesInBackground(syncCtx, event.AccountID, fID, syncDays)
			if bodyErr != nil {
				if syncCtx.Err() != nil {
					// Cancelled - not an error, emit synced
					log.Debug().Str("folder", fID).Msg("IDLE body fetch cancelled")
					wailsRuntime.EventsEmit(a.ctx, "folder:synced", map[string]interface{}{
						"accountId": event.AccountID,
						"folderId":  fID,
					})
				} else {
					// Actual error - emit error event
					log.Error().Err(bodyErr).Str("folder", fID).Msg("Background body fetch failed after IDLE sync")
					wailsRuntime.EventsEmit(a.ctx, "folder:syncError", map[string]interface{}{
						"accountId": event.AccountID,
						"folderId":  fID,
						"error":     bodyErr.Error(),
					})
				}
			} else {
				// Success
				wailsRuntime.EventsEmit(a.ctx, "folder:synced", map[string]interface{}{
					"accountId": event.AccountID,
					"folderId":  fID,
				})
			}
		}(ctx, syncPeriodDays, folderID, syncKey)
	}

	// Notify about new mail if any
	if newMailInfo != nil && newMailInfo.Count > 0 {
		a.handleNewMailNotification(*newMailInfo)
	}
}

// handleNewMailNotification handles notifications for new mail
func (a *App) handleNewMailNotification(info sync.NewMailInfo) {
	log := logging.WithComponent("app.notify")

	log.Info().
		Str("account", info.AccountName).
		Int("count", info.Count).
		Msg("New mail notification")

	// Get the most recent conversation for the notification
	var subject, fromName, fromEmail, threadID string

	inbox, err := a.folderStore.GetByType(info.AccountID, folder.TypeInbox)
	if err == nil && inbox != nil {
		// Get the most recent conversation (sorted by newest first)
		conversations, err := a.messageStore.ListConversationsByFolder(info.FolderID, 0, 1, "newest")
		if err == nil && len(conversations) > 0 {
			conv := conversations[0]
			subject = conv.Subject
			threadID = conv.ThreadID
			// Get sender info from participants
			if len(conv.Participants) > 0 {
				fromName = conv.Participants[0].Name
				fromEmail = conv.Participants[0].Email
			}
		}
	}

	// Emit event to frontend for UI updates
	wailsRuntime.EventsEmit(a.ctx, "mail:newMail", map[string]interface{}{
		"accountId":   info.AccountID,
		"accountName": info.AccountName,
		"folderId":    info.FolderID,
		"count":       info.Count,
		"subject":     subject,
		"fromName":    fromName,
		"fromEmail":   fromEmail,
	})

	// Send system notification
	a.sendSystemNotification(info, subject, fromName, fromEmail, threadID)
}

// sendSystemNotification sends a desktop notification for new mail
func (a *App) sendSystemNotification(info sync.NewMailInfo, subject, fromName, fromEmail, threadID string) {
	log := logging.WithComponent("app.notify")

	// Build notification title and body
	var title, body string

	if info.Count == 1 && subject != "" {
		// Single message notification
		sender := fromName
		if sender == "" {
			sender = fromEmail
		}
		title = "New email from " + sender
		body = subject
	} else {
		// Multiple messages notification
		title = "New emails"
		body = info.AccountName
	}

	// Use the notifier if available
	if a.notifier != nil {
		_, err := a.notifier.Show(notification.Notification{
			Title: title,
			Body:  body,
			Icon:  "mail-unread",
			Data: notification.NotificationData{
				AccountID: info.AccountID,
				FolderID:  info.FolderID,
				ThreadID:  threadID,
			},
		})
		if err != nil {
			log.Debug().Err(err).Msg("Failed to send notification")
		}
	}
}

// ============================================================================
// Desktop Notifications with Click Handling
// ============================================================================

// initNotifications initializes the desktop notification system with click handling
func (a *App) initNotifications(ctx context.Context) {
	log := logging.WithComponent("app.notify")

	a.notifier = notification.New("Aerion")

	// Set click handler to navigate to the message
	a.notifier.SetClickHandler(func(data notification.NotificationData) {
		log.Info().
			Str("accountId", data.AccountID).
			Str("folderId", data.FolderID).
			Str("threadId", data.ThreadID).
			Msg("Notification clicked, navigating to message")

		// Bring window to foreground
		wailsRuntime.WindowShow(a.ctx)

		// Emit event to frontend to navigate to the message
		wailsRuntime.EventsEmit(a.ctx, "notification:clicked", map[string]interface{}{
			"accountId": data.AccountID,
			"folderId":  data.FolderID,
			"threadId":  data.ThreadID,
		})
	})

	// Start the notification listener
	if err := a.notifier.Start(ctx); err != nil {
		log.Warn().Err(err).Msg("Failed to start notification listener (click handling may not work)")
	}
}

// ============================================================================
// Sleep/Wake Detection for Auto-Sync
// ============================================================================

// initSleepWakeMonitor initializes the sleep/wake monitor for auto-sync on wake
func (a *App) initSleepWakeMonitor(ctx context.Context) {
	log := logging.WithComponent("app.sleep-wake")

	// Create the platform-specific monitor
	a.sleepWakeMonitor = platform.NewSleepWakeMonitor()

	// Start the monitor
	if err := a.sleepWakeMonitor.Start(ctx); err != nil {
		log.Warn().Err(err).Msg("Failed to start sleep/wake monitor - auto-sync on wake disabled")
		return
	}

	// Process events in background
	go a.processSleepWakeEvents(ctx)

	log.Info().Msg("Sleep/wake monitor initialized")
}

// processSleepWakeEvents handles sleep/wake events from the monitor
func (a *App) processSleepWakeEvents(ctx context.Context) {
	if a.sleepWakeMonitor == nil {
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-a.sleepWakeMonitor.Events():
			if !ok {
				return
			}

			if event.IsSleeping {
				a.handleSystemSleep()
			} else {
				a.handleSystemWake()
			}
		}
	}
}

// handleSystemSleep handles system going to sleep
// Gracefully disconnects IMAP connections to avoid stale connection errors on wake
func (a *App) handleSystemSleep() {
	log := logging.WithComponent("app.sleep-wake")
	log.Info().Msg("System going to sleep - disconnecting IMAP connections")

	// Stop all IDLE connections gracefully
	if a.idleManager != nil {
		a.idleManager.Stop()
	}

	// Close all IMAP pool connections to avoid stale connections on wake
	if a.imapPool != nil {
		a.imapPool.CloseAll()
	}

	log.Info().Msg("IMAP connections closed for sleep")
}

// handleSystemWake handles system waking from sleep
// Waits for network, restarts IDLE connections, and syncs all accounts
func (a *App) handleSystemWake() {
	log := logging.WithComponent("app.sleep-wake")
	log.Info().Msg("System woke from sleep - waiting for network...")

	// Wait for network to be available (3 seconds)
	time.Sleep(sleepWakeSyncDelay)

	log.Info().Msg("Restarting IDLE connections and syncing all accounts")

	// Update LastSync on all inbox folders BEFORE starting sync
	// This prevents the scheduler from thinking sync is overdue and interfering
	now := time.Now()
	accounts, err := a.accountStore.List()
	if err == nil {
		for _, acc := range accounts {
			if !acc.Enabled {
				continue
			}
			inbox, err := a.folderStore.GetByType(acc.ID, folder.TypeInbox)
			if err == nil && inbox != nil {
				inbox.LastSync = &now
				if err := a.folderStore.Update(inbox); err != nil {
					log.Warn().Err(err).Str("account", acc.Name).Msg("Failed to update LastSync before wake sync")
				}
			}
		}
	}

	// Restart IDLE manager
	if a.idleManager != nil {
		a.idleManager.Start(a.ctx)

		// Restart IDLE for all enabled accounts
		accounts, err := a.accountStore.List()
		if err == nil {
			for _, acc := range accounts {
				if acc.Enabled {
					a.idleManager.StartAccount(acc.ID, acc.Name)
				}
			}
			log.Info().Int("accounts", len(accounts)).Msg("IDLE restarted for accounts")
		}
	}

	// Trigger master sync for all accounts (same as manual sync button)
	// Run in goroutine since SyncAllComplete is blocking
	go func() {
		if err := a.SyncAllComplete(); err != nil {
			log.Warn().Err(err).Msg("Post-wake sync encountered errors")
		} else {
			log.Info().Msg("Post-wake sync completed successfully")
		}
	}()

	log.Info().Msg("Post-wake sync triggered for all accounts")
}
