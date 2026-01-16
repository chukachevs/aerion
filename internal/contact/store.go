// Package contact provides contact management for email autocomplete
package contact

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/hkdb/aerion/internal/logging"
	"github.com/rs/zerolog"
)

// CardDAVSearchFunc is a function type for searching CardDAV contacts
// This avoids circular imports with the carddav package
type CardDAVSearchFunc func(query string, limit int) ([]*Contact, error)

// Store handles contact storage and retrieval
type Store struct {
	db              *sql.DB
	vcardScanner    *VCardScanner
	carddavSearchFn CardDAVSearchFunc
	log             zerolog.Logger
}

// NewStore creates a new contact store
func NewStore(db *sql.DB) *Store {
	store := &Store{
		db:  db,
		log: logging.WithComponent("contact"),
	}

	// Ensure contacts table exists
	if err := store.ensureTable(); err != nil {
		store.log.Error().Err(err).Msg("Failed to create contacts table")
	}

	return store
}

// SetVCardScanner sets the vCard scanner for additional contact sources
func (s *Store) SetVCardScanner(scanner *VCardScanner) {
	s.vcardScanner = scanner
}

// SetCardDAVSearchFunc sets the CardDAV search function for synced contacts
func (s *Store) SetCardDAVSearchFunc(fn CardDAVSearchFunc) {
	s.carddavSearchFn = fn
}

// ensureTable creates the contacts table if it doesn't exist
func (s *Store) ensureTable() error {
	query := `
		CREATE TABLE IF NOT EXISTS contacts (
			email TEXT PRIMARY KEY,
			display_name TEXT,
			send_count INTEGER DEFAULT 0,
			last_used DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		
		CREATE INDEX IF NOT EXISTS idx_contacts_send_count ON contacts(send_count DESC);
		CREATE INDEX IF NOT EXISTS idx_contacts_last_used ON contacts(last_used DESC);
	`

	_, err := s.db.Exec(query)
	return err
}

// Search searches for contacts matching the query
// Results are merged from multiple sources:
// - Local SQLite (aerion - sent recipients)
// - vCard files (local .vcf files)
// - CardDAV (synced from servers)
// Ranked by: send count > recency > source priority (aerion > vcard > carddav > google)
func (s *Store) Search(query string, limit int) ([]*Contact, error) {
	if limit <= 0 {
		limit = 10
	}

	// 1. Query local SQLite (aerion contacts - sent recipients)
	aerionContacts, err := s.searchLocal(query, limit)
	if err != nil {
		s.log.Warn().Err(err).Msg("Failed to search local contacts")
		aerionContacts = []*Contact{}
	}

	// 2. Query vCard cache
	var vcardContacts []*Contact
	if s.vcardScanner != nil {
		vcardContacts, err = s.vcardScanner.Search(query, limit)
		if err != nil {
			s.log.Warn().Err(err).Msg("Failed to search vCard contacts")
			vcardContacts = []*Contact{}
		}

		// If few results and cache is expired, the scanner will have triggered a live scan
		// If we got very few results, try refreshing in background for next time
		if len(aerionContacts)+len(vcardContacts) < 3 {
			s.vcardScanner.RefreshIfNeeded()
		}
	}

	// 3. Query CardDAV synced contacts
	var carddavContacts []*Contact
	if s.carddavSearchFn != nil {
		carddavContacts, err = s.carddavSearchFn(query, limit)
		if err != nil {
			s.log.Warn().Err(err).Msg("Failed to search CardDAV contacts")
			carddavContacts = []*Contact{}
		}
	}

	// 4. Merge results (priority order: aerion > vcard > carddav)
	// MergeResults handles deduplication by email
	merged := MergeResults(aerionContacts, vcardContacts, carddavContacts)

	// 5. Apply limit
	if len(merged) > limit {
		merged = merged[:limit]
	}

	return merged, nil
}

// searchLocal searches the local SQLite database for contacts
func (s *Store) searchLocal(query string, limit int) ([]*Contact, error) {
	// Prepare search pattern
	pattern := "%" + strings.ToLower(query) + "%"

	// Search in email and display_name
	sqlQuery := `
		SELECT email, display_name, send_count, last_used, created_at
		FROM contacts
		WHERE LOWER(email) LIKE ? OR LOWER(display_name) LIKE ?
		ORDER BY send_count DESC, last_used DESC
		LIMIT ?
	`

	rows, err := s.db.Query(sqlQuery, pattern, pattern, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search contacts: %w", err)
	}
	defer rows.Close()

	var contacts []*Contact
	for rows.Next() {
		var c Contact
		var lastUsed, createdAt sql.NullTime

		if err := rows.Scan(&c.Email, &c.DisplayName, &c.SendCount, &lastUsed, &createdAt); err != nil {
			s.log.Warn().Err(err).Msg("Failed to scan contact row")
			continue
		}

		if lastUsed.Valid {
			c.LastUsed = lastUsed.Time
		}
		if createdAt.Valid {
			c.CreatedAt = createdAt.Time
		}
		c.Source = "aerion"

		contacts = append(contacts, &c)
	}

	return contacts, nil
}

// AddOrUpdate adds a new contact or updates an existing one
// This is called when a message is sent to update send statistics
func (s *Store) AddOrUpdate(email, displayName string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	displayName = strings.TrimSpace(displayName)

	if email == "" {
		return fmt.Errorf("email cannot be empty")
	}

	now := time.Now()

	// Use upsert pattern
	query := `
		INSERT INTO contacts (email, display_name, send_count, last_used, created_at)
		VALUES (?, ?, 1, ?, ?)
		ON CONFLICT(email) DO UPDATE SET
			display_name = CASE 
				WHEN ? != '' THEN ? 
				ELSE contacts.display_name 
			END,
			send_count = contacts.send_count + 1,
			last_used = ?
	`

	_, err := s.db.Exec(query,
		email, displayName, now, now,
		displayName, displayName, now)
	if err != nil {
		return fmt.Errorf("failed to add/update contact: %w", err)
	}

	s.log.Debug().
		Str("email", email).
		Str("name", displayName).
		Msg("Contact added/updated")

	return nil
}

// AddFromSentMail adds contacts from a sent email's recipients
func (s *Store) AddFromSentMail(recipients []struct{ Email, Name string }) error {
	for _, r := range recipients {
		if err := s.AddOrUpdate(r.Email, r.Name); err != nil {
			s.log.Warn().Err(err).
				Str("email", r.Email).
				Msg("Failed to add contact from sent mail")
		}
	}
	return nil
}

// Get returns a contact by email
func (s *Store) Get(email string) (*Contact, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	query := `
		SELECT email, display_name, send_count, last_used, created_at
		FROM contacts
		WHERE email = ?
	`

	var c Contact
	var lastUsed, createdAt sql.NullTime

	err := s.db.QueryRow(query, email).Scan(
		&c.Email, &c.DisplayName, &c.SendCount, &lastUsed, &createdAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get contact: %w", err)
	}

	if lastUsed.Valid {
		c.LastUsed = lastUsed.Time
	}
	if createdAt.Valid {
		c.CreatedAt = createdAt.Time
	}
	c.Source = "aerion"

	return &c, nil
}

// Delete removes a contact by email
func (s *Store) Delete(email string) error {
	email = strings.ToLower(strings.TrimSpace(email))

	_, err := s.db.Exec("DELETE FROM contacts WHERE email = ?", email)
	if err != nil {
		return fmt.Errorf("failed to delete contact: %w", err)
	}

	return nil
}

// List returns all contacts, optionally limited
func (s *Store) List(limit int) ([]*Contact, error) {
	query := `
		SELECT email, display_name, send_count, last_used, created_at
		FROM contacts
		ORDER BY send_count DESC, last_used DESC
	`
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list contacts: %w", err)
	}
	defer rows.Close()

	var contacts []*Contact
	for rows.Next() {
		var c Contact
		var lastUsed, createdAt sql.NullTime

		if err := rows.Scan(&c.Email, &c.DisplayName, &c.SendCount, &lastUsed, &createdAt); err != nil {
			continue
		}

		if lastUsed.Valid {
			c.LastUsed = lastUsed.Time
		}
		if createdAt.Valid {
			c.CreatedAt = createdAt.Time
		}
		c.Source = "aerion"

		contacts = append(contacts, &c)
	}

	return contacts, nil
}

// Count returns the total number of contacts
func (s *Store) Count() (int, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM contacts").Scan(&count)
	return count, err
}

// MergeResults merges contacts from multiple sources and deduplicates by email
// Results are ranked by: send count > recency > source priority
func MergeResults(sources ...[]*Contact) []*Contact {
	// Use map to dedupe by email
	byEmail := make(map[string]*Contact)

	for _, contacts := range sources {
		for _, c := range contacts {
			email := strings.ToLower(c.Email)
			existing, exists := byEmail[email]

			if !exists {
				byEmail[email] = c
			} else {
				// Keep the one with higher send count, or more recent, or better source
				if c.SendCount > existing.SendCount {
					byEmail[email] = c
				} else if c.SendCount == existing.SendCount && c.LastUsed.After(existing.LastUsed) {
					byEmail[email] = c
				} else if c.SendCount == existing.SendCount && sourcePriority(c.Source) > sourcePriority(existing.Source) {
					byEmail[email] = c
				}
			}
		}
	}

	// Convert to slice
	result := make([]*Contact, 0, len(byEmail))
	for _, c := range byEmail {
		result = append(result, c)
	}

	// Sort by send count (desc), then last used (desc), then alphabetically
	sort.Slice(result, func(i, j int) bool {
		if result[i].SendCount != result[j].SendCount {
			return result[i].SendCount > result[j].SendCount
		}
		if !result[i].LastUsed.Equal(result[j].LastUsed) {
			return result[i].LastUsed.After(result[j].LastUsed)
		}
		return result[i].Email < result[j].Email
	})

	return result
}

// sourcePriority returns the priority of a contact source
// Higher is better
// Priority order: aerion > vcard > carddav > google
func sourcePriority(source string) int {
	switch source {
	case "aerion":
		return 4
	case "vcard":
		return 3
	case "carddav":
		return 2
	case "google":
		return 1
	default:
		return 0
	}
}
