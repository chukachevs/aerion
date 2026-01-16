package app

import (
	"fmt"

	"github.com/hkdb/aerion/internal/folder"
	"github.com/hkdb/aerion/internal/logging"
)

// ============================================================================
// Folder API - Exposed to frontend via Wails bindings
// ============================================================================

// GetFolders returns all folders for an account
func (a *App) GetFolders(accountID string) ([]*folder.Folder, error) {
	folders, err := a.folderStore.List(accountID)
	if err != nil {
		return nil, err
	}

	// Sort folders (special folders first)
	folder.SortFolders(folders)
	return folders, nil
}

// GetFolderTree returns folders as a tree structure
func (a *App) GetFolderTree(accountID string) ([]*folder.FolderTree, error) {
	folders, err := a.folderStore.List(accountID)
	if err != nil {
		return nil, err
	}

	// Sort folders (INBOX first, then special folders, then alphabetically)
	folder.SortFolders(folders)
	return folder.BuildTree(folders), nil
}

// SyncFolders synchronizes the folder list with the IMAP server
func (a *App) SyncFolders(accountID string) error {
	log := logging.WithComponent("app")
	err := a.syncEngine.SyncFolders(a.ctx, accountID)
	if err == nil {
		// Checkpoint WAL after heavy sync operation
		if checkpointErr := a.db.Checkpoint(); checkpointErr != nil {
			log.Warn().Err(checkpointErr).Msg("WAL checkpoint after SyncFolders failed")
		}
	}
	return err
}

// GetAccountFoldersForMapping returns all folders for an account (for folder mapping UI).
// Triggers a folder sync if no folders exist yet.
func (a *App) GetAccountFoldersForMapping(accountID string) ([]*folder.Folder, error) {
	log := logging.WithComponent("app")

	folders, err := a.folderStore.List(accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to list folders: %w", err)
	}

	// If no folders, trigger sync first
	if len(folders) == 0 {
		log.Info().Str("accountID", accountID).Msg("No folders found, triggering sync")
		if err := a.syncEngine.SyncFolders(a.ctx, accountID); err != nil {
			return nil, fmt.Errorf("failed to sync folders: %w", err)
		}
		folders, err = a.folderStore.List(accountID)
		if err != nil {
			return nil, fmt.Errorf("failed to list folders after sync: %w", err)
		}
	}

	// Sort using existing logic
	folder.SortFolders(folders)
	return folders, nil
}

// GetAutoDetectedFolders returns the auto-detected special folders for an account.
// Returns map of folder type -> folder path (e.g., {"sent": "Sent Mail", "trash": "Deleted Items"}).
func (a *App) GetAutoDetectedFolders(accountID string) (map[string]string, error) {
	folders, err := a.folderStore.List(accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to list folders: %w", err)
	}

	result := make(map[string]string)
	for _, f := range folders {
		if f.Type != folder.TypeFolder && f.Type != folder.TypeInbox {
			result[string(f.Type)] = f.Path
		}
	}
	return result, nil
}

// GetSpecialFolder returns the folder for a special type, checking account mappings first.
// If no mapping is set, falls back to auto-detected folder type.
func (a *App) GetSpecialFolder(accountID string, folderType folder.Type) (*folder.Folder, error) {
	// Get account to check mappings
	acc, err := a.accountStore.Get(accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}
	if acc == nil {
		return nil, fmt.Errorf("account not found: %s", accountID)
	}

	// Check if account has a manual mapping
	mappedPath := acc.GetFolderMapping(string(folderType))
	if mappedPath != "" {
		f, err := a.folderStore.GetByPath(accountID, mappedPath)
		if err != nil {
			return nil, err
		}
		if f != nil {
			return f, nil
		}
		// Mapped folder not found, fall through to auto-detect
	}

	// Fall back to auto-detected type
	return a.folderStore.GetByType(accountID, folderType)
}
