// Package contact provides contact management for email autocomplete
package contact

import "time"

// Contact represents a contact for email autocomplete
type Contact struct {
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	Source      string    `json:"source"` // "aerion", "google", "vcard", "sent-history"
	AvatarURL   string    `json:"avatar_url,omitempty"`
	SendCount   int       `json:"send_count"` // Number of times user sent to this address
	LastUsed    time.Time `json:"last_used"`  // Last time this contact was used
	CreatedAt   time.Time `json:"created_at"`
}

// LocalContact represents a contact stored in Aerion's local database
// This is for contacts the user has emailed (sent mail recipients)
type LocalContact struct {
	Email       string    `json:"email"`        // Primary key
	DisplayName string    `json:"display_name"` // Last known name
	SendCount   int       `json:"send_count"`   // Times user sent to this address
	LastUsed    time.Time `json:"last_used"`    // For ranking
	CreatedAt   time.Time `json:"created_at"`
}

// ContactSource represents a source of contacts
type ContactSource struct {
	ID      string `json:"id"`
	Type    string `json:"type"` // "aerion", "google", "vcard"
	Name    string `json:"name"` // Display name
	Enabled bool   `json:"enabled"`
	Path    string `json:"path,omitempty"` // For vCard sources
}

// SearchResult represents a contact search result with ranking info
type SearchResult struct {
	Contact
	Score float64 `json:"score"` // Relevance score for ranking
}
