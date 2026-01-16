package undo

import (
	"context"
	"fmt"

	"github.com/emersion/go-imap/v2"
	imapPkg "github.com/hkdb/aerion/internal/imap"
)

// UndoContext provides dependencies for undo operations
type UndoContext interface {
	// GetIMAPConnectionForUndo returns an IMAP client for the account
	GetIMAPConnectionForUndo(ctx context.Context, accountID string) (*imapPkg.Client, func(), error)
	// UpdateLocalFlags updates flags in local database
	UpdateLocalFlags(messageIDs []string, isRead, isStarred *bool) error
	// MoveLocalMessages moves messages in local database
	MoveLocalMessages(messageIDs []string, folderID string) error
	// DeleteLocalMessages deletes messages from local database
	DeleteLocalMessages(messageIDs []string) error
}

// FlagChangeCommand handles read/star flag changes
type FlagChangeCommand struct {
	BaseCommand
	ctx           context.Context
	undoCtx       UndoContext
	accountID     string
	folderPath    string
	messageIDs    []string
	uids          []uint32
	flagType      string // "read" or "starred"
	previousState bool   // What was the state before
}

// NewFlagChangeCommand creates a new FlagChangeCommand
func NewFlagChangeCommand(
	ctx context.Context,
	undoCtx UndoContext,
	accountID, folderPath string,
	messageIDs []string,
	uids []uint32,
	flagType string,
	previousState bool,
	description string,
) *FlagChangeCommand {
	return &FlagChangeCommand{
		BaseCommand:   NewBaseCommand(description),
		ctx:           ctx,
		undoCtx:       undoCtx,
		accountID:     accountID,
		folderPath:    folderPath,
		messageIDs:    messageIDs,
		uids:          uids,
		flagType:      flagType,
		previousState: previousState,
	}
}

// Execute performs the action (already done at creation time)
func (c *FlagChangeCommand) Execute() error { return nil }

// Undo reverses the flag change
func (c *FlagChangeCommand) Undo() error {
	// Get IMAP connection
	client, release, err := c.undoCtx.GetIMAPConnectionForUndo(c.ctx, c.accountID)
	if err != nil {
		return fmt.Errorf("failed to get IMAP connection: %w", err)
	}
	defer release()

	// Select mailbox
	if _, err := client.SelectMailbox(c.ctx, c.folderPath); err != nil {
		return fmt.Errorf("failed to select mailbox: %w", err)
	}

	// Convert UIDs
	imapUIDs := make([]imap.UID, len(c.uids))
	for i, uid := range c.uids {
		imapUIDs[i] = imap.UID(uid)
	}

	// Determine flag
	var flag imap.Flag
	switch c.flagType {
	case "read":
		flag = imap.FlagSeen
	case "starred":
		flag = imap.FlagFlagged
	default:
		return fmt.Errorf("unknown flag type: %s", c.flagType)
	}

	// Restore previous state on IMAP
	if c.previousState {
		if err := client.AddMessageFlags(imapUIDs, []imap.Flag{flag}); err != nil {
			return fmt.Errorf("failed to add flags: %w", err)
		}
	} else {
		if err := client.RemoveMessageFlags(imapUIDs, []imap.Flag{flag}); err != nil {
			return fmt.Errorf("failed to remove flags: %w", err)
		}
	}

	// Update local database
	var isRead, isStarred *bool
	switch c.flagType {
	case "read":
		isRead = &c.previousState
	case "starred":
		isStarred = &c.previousState
	}
	if err := c.undoCtx.UpdateLocalFlags(c.messageIDs, isRead, isStarred); err != nil {
		return fmt.Errorf("failed to update local flags: %w", err)
	}

	return nil
}

// MoveCommand handles moving messages between folders
type MoveCommand struct {
	BaseCommand
	ctx              context.Context
	undoCtx          UndoContext
	accountID        string
	messageIDs       []string
	uids             []uint32
	sourceFolderID   string
	sourceFolderPath string
	destFolderID     string
	destFolderPath   string
	newUIDs          []uint32 // UIDs in destination folder after move
}

// NewMoveCommand creates a new MoveCommand
func NewMoveCommand(
	ctx context.Context,
	undoCtx UndoContext,
	accountID string,
	messageIDs []string,
	uids []uint32,
	sourceFolderID, sourceFolderPath string,
	destFolderID, destFolderPath string,
	description string,
) *MoveCommand {
	return &MoveCommand{
		BaseCommand:      NewBaseCommand(description),
		ctx:              ctx,
		undoCtx:          undoCtx,
		accountID:        accountID,
		messageIDs:       messageIDs,
		uids:             uids,
		sourceFolderID:   sourceFolderID,
		sourceFolderPath: sourceFolderPath,
		destFolderID:     destFolderID,
		destFolderPath:   destFolderPath,
	}
}

// SetNewUIDs sets the UIDs of messages in the destination folder
// This should be called after the move is complete if UIDPLUS info is available
func (c *MoveCommand) SetNewUIDs(uids []uint32) {
	c.newUIDs = uids
}

// Execute performs the action (already done at creation time)
func (c *MoveCommand) Execute() error { return nil }

// Undo reverses the move operation
func (c *MoveCommand) Undo() error {
	// Get IMAP connection
	client, release, err := c.undoCtx.GetIMAPConnectionForUndo(c.ctx, c.accountID)
	if err != nil {
		return fmt.Errorf("failed to get IMAP connection: %w", err)
	}
	defer release()

	// Select destination mailbox (where messages currently are)
	if _, err := client.SelectMailbox(c.ctx, c.destFolderPath); err != nil {
		return fmt.Errorf("failed to select mailbox: %w", err)
	}

	// Use newUIDs if available, otherwise use original UIDs
	// Note: Original UIDs may not work after move - this is a limitation
	// For proper undo, we'd need the new UIDs from UIDPLUS or a resync
	uidsToUse := c.newUIDs
	if len(uidsToUse) == 0 {
		// Fallback: try original UIDs (may not work reliably)
		uidsToUse = c.uids
	}

	imapUIDs := make([]imap.UID, len(uidsToUse))
	for i, uid := range uidsToUse {
		imapUIDs[i] = imap.UID(uid)
	}

	// Copy back to source folder
	if _, err := client.CopyMessages(imapUIDs, c.sourceFolderPath); err != nil {
		return fmt.Errorf("failed to copy messages: %w", err)
	}

	// Delete from destination
	if err := client.DeleteMessagesByUID(imapUIDs); err != nil {
		return fmt.Errorf("failed to delete messages from destination: %w", err)
	}

	// Update local database
	if err := c.undoCtx.MoveLocalMessages(c.messageIDs, c.sourceFolderID); err != nil {
		return fmt.Errorf("failed to update local database: %w", err)
	}

	return nil
}
