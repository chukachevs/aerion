package app

import (
	"github.com/hkdb/aerion/internal/message"
)

// ============================================================================
// Search API - Exposed to frontend via Wails bindings
// ============================================================================

// SearchConversations searches for conversations in a folder using full-text search
// Returns matching conversations with highlighted text
func (a *App) SearchConversations(accountID, folderID, query string, offset, limit int) ([]*message.ConversationSearchResult, error) {
	results, _, err := a.messageStore.SearchConversations(folderID, query, offset, limit)
	return results, err
}

// GetSearchCount returns the total count of search results in a folder
func (a *App) GetSearchCount(accountID, folderID, query string) (int, error) {
	_, count, err := a.messageStore.SearchConversations(folderID, query, 0, 0)
	return count, err
}

// SearchUnifiedInbox searches across all inbox folders for all accounts
func (a *App) SearchUnifiedInbox(query string, offset, limit int) ([]*message.ConversationSearchResult, error) {
	results, _, err := a.messageStore.SearchConversationsUnifiedInbox(query, offset, limit)
	return results, err
}

// GetSearchCountUnifiedInbox returns the total count of search results across all inboxes
func (a *App) GetSearchCountUnifiedInbox(query string) (int, error) {
	_, count, err := a.messageStore.SearchConversationsUnifiedInbox(query, 0, 0)
	return count, err
}

// GetFTSIndexStatus returns the indexing status for a specific folder
func (a *App) GetFTSIndexStatus(folderID string) (*message.FTSIndexStatus, error) {
	return a.ftsIndexer.GetIndexStatus(folderID)
}

// GetFTSIndexStatusAll returns the indexing status for all folders
func (a *App) GetFTSIndexStatusAll() (map[string]*message.FTSIndexStatus, error) {
	return a.ftsIndexer.GetAllIndexStatuses()
}

// IsFTSIndexComplete checks if a folder is fully indexed
func (a *App) IsFTSIndexComplete(folderID string) bool {
	return a.ftsIndexer.IsIndexComplete(folderID)
}

// IsFTSIndexing returns true if any folder is currently being indexed
func (a *App) IsFTSIndexing() bool {
	return a.ftsIndexer.IsAnyIndexing()
}

// RebuildFTSIndex forces a rebuild of the FTS index for a folder
func (a *App) RebuildFTSIndex(folderID string) error {
	return a.ftsIndexer.RebuildIndex(a.ctx, folderID)
}
