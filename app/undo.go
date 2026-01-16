package app

import (
	"context"
	"fmt"

	"github.com/hkdb/aerion/internal/imap"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// ============================================================================
// Undo API - Exposed to frontend via Wails bindings
// ============================================================================

// Undo reverses the most recent undoable action
// Returns the description of what was undone, or error if nothing to undo
func (a *App) Undo() (string, error) {
	cmd := a.undoStack.Pop()
	if cmd == nil {
		return "", fmt.Errorf("nothing to undo")
	}

	if err := cmd.Undo(); err != nil {
		return "", fmt.Errorf("undo failed: %w", err)
	}

	// Emit event to refresh UI
	wailsRuntime.EventsEmit(a.ctx, "undo:completed", cmd.Description())

	return cmd.Description(), nil
}

// CanUndo returns true if there's an action that can be undone
func (a *App) CanUndo() bool {
	return a.undoStack.CanUndo()
}

// GetUndoDescription returns the description of what would be undone
func (a *App) GetUndoDescription() string {
	cmd := a.undoStack.Peek()
	if cmd == nil {
		return ""
	}
	return cmd.Description()
}

// ============================================================================
// UndoContext Implementation - Required for undo.Command operations
// ============================================================================

// GetIMAPConnectionForUndo implements undo.UndoContext
func (a *App) GetIMAPConnectionForUndo(ctx context.Context, accountID string) (*imap.Client, func(), error) {
	poolConn, err := a.imapPool.GetConnection(ctx, accountID)
	if err != nil {
		return nil, nil, err
	}
	return poolConn.Client(), func() { a.imapPool.Release(poolConn) }, nil
}

// UpdateLocalFlags implements undo.UndoContext
func (a *App) UpdateLocalFlags(messageIDs []string, isRead, isStarred *bool) error {
	err := a.messageStore.UpdateFlagsBatch(messageIDs, isRead, isStarred)
	if err == nil {
		wailsRuntime.EventsEmit(a.ctx, "messages:flagsChanged", messageIDs)
	}
	return err
}

// MoveLocalMessages implements undo.UndoContext
func (a *App) MoveLocalMessages(messageIDs []string, folderID string) error {
	err := a.messageStore.MoveMessages(messageIDs, folderID)
	if err == nil {
		wailsRuntime.EventsEmit(a.ctx, "messages:moved", map[string]interface{}{
			"messageIds":   messageIDs,
			"destFolderId": folderID,
		})
	}
	return err
}

// DeleteLocalMessages implements undo.UndoContext
func (a *App) DeleteLocalMessages(messageIDs []string) error {
	err := a.messageStore.DeleteBatch(messageIDs)
	if err == nil {
		wailsRuntime.EventsEmit(a.ctx, "messages:deleted", messageIDs)
	}
	return err
}
