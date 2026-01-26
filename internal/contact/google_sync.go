// Package contact provides contact sync and autocomplete functionality
package contact

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/hkdb/aerion/internal/logging"
	"github.com/rs/zerolog"
)

// SyncedContact represents a contact fetched from a sync source (Google/Microsoft)
type SyncedContact struct {
	Email       string
	DisplayName string
	RemoteID    string // Provider-specific ID for change detection
}

// SyncResult represents the result of an incremental sync
type SyncResult struct {
	Contacts      []SyncedContact // New or updated contacts
	DeletedIDs    []string        // Remote IDs of deleted contacts
	NextSyncToken string          // Token for next incremental sync
	IsFullSync    bool            // True if this was a full sync (no valid token)
}

// GoogleContactsSyncer syncs contacts from Google People API.
// Uses the people.connections endpoint to fetch all user's saved contacts.
type GoogleContactsSyncer struct {
	httpClient *http.Client
	log        zerolog.Logger
}

// NewGoogleContactsSyncer creates a new Google contacts syncer.
func NewGoogleContactsSyncer() *GoogleContactsSyncer {
	return &GoogleContactsSyncer{
		httpClient: &http.Client{Timeout: 60 * time.Second}, // Longer timeout for sync
		log:        logging.WithComponent("google-contacts-sync"),
	}
}

// SyncContacts fetches all contacts from Google People API (full sync).
// Uses the connections endpoint with pagination to get all user's contacts.
// The accessToken should be a valid Google OAuth2 access token with contacts.readonly scope.
func (s *GoogleContactsSyncer) SyncContacts(accessToken string) ([]SyncedContact, error) {
	result, err := s.SyncContactsDelta(accessToken, "")
	if err != nil {
		return nil, err
	}
	return result.Contacts, nil
}

// SyncContactsDelta performs an incremental sync using Google's syncToken mechanism.
// If syncToken is empty, performs a full sync and returns a token for future incremental syncs.
// If the syncToken is expired (410 Gone), automatically falls back to full sync.
func (s *GoogleContactsSyncer) SyncContactsDelta(accessToken, syncToken string) (*SyncResult, error) {
	var allContacts []SyncedContact
	var deletedIDs []string
	pageToken := ""
	isFullSync := syncToken == ""
	requestSync := true // Whether to request a sync token in response

	if isFullSync {
		s.log.Info().Msg("Starting Google contacts full sync")
	} else {
		s.log.Info().Msg("Starting Google contacts incremental sync")
	}

	for {
		// Build API URL with pagination and sync token
		apiURL := "https://people.googleapis.com/v1/people/me/connections?personFields=names,emailAddresses&pageSize=1000"
		if pageToken != "" {
			apiURL += "&pageToken=" + pageToken
		}
		if syncToken != "" && pageToken == "" {
			// Only include syncToken on first request (not pagination requests)
			apiURL += "&syncToken=" + syncToken
		}
		if requestSync {
			apiURL += "&requestSyncToken=true"
		}

		req, err := http.NewRequest("GET", apiURL, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+accessToken)

		resp, err := s.httpClient.Do(req)
		if err != nil {
			s.log.Error().Err(err).Msg("Google People API request failed")
			return nil, fmt.Errorf("Google People API request failed: %w", err)
		}

		// Handle 410 Gone - sync token expired, need full sync
		if resp.StatusCode == http.StatusGone {
			resp.Body.Close()
			if !isFullSync {
				s.log.Warn().Msg("Google sync token expired, falling back to full sync")
				return s.SyncContactsDelta(accessToken, "")
			}
			return nil, fmt.Errorf("Google API returned 410 Gone on full sync")
		}

		if resp.StatusCode != http.StatusOK {
			// Read the error response body for error handling
			bodyBytes, _ := io.ReadAll(resp.Body)
			resp.Body.Close()

			// Check for expired sync token in 400 Bad Request
			if resp.StatusCode == http.StatusBadRequest {
				var errorResp googleErrorResponse
				if err := json.Unmarshal(bodyBytes, &errorResp); err == nil {
					// Check if this is an EXPIRED_SYNC_TOKEN error
					for _, detail := range errorResp.Error.Details {
						if detail.Reason == "EXPIRED_SYNC_TOKEN" {
							if !isFullSync {
								s.log.Warn().Msg("Google sync token expired (400 EXPIRED_SYNC_TOKEN), falling back to full sync")
								return s.SyncContactsDelta(accessToken, "")
							}
							return nil, fmt.Errorf("Google API returned EXPIRED_SYNC_TOKEN on full sync")
						}
					}
				}
			}

			s.log.Error().
				Int("status", resp.StatusCode).
				Msg("Google People API error response")

			switch resp.StatusCode {
			case http.StatusUnauthorized:
				return nil, fmt.Errorf("Google API authentication failed (token may be expired): %s", string(bodyBytes))
			case http.StatusForbidden:
				return nil, fmt.Errorf("Google API access denied: %s", string(bodyBytes))
			case http.StatusTooManyRequests:
				return nil, fmt.Errorf("Google API rate limit exceeded")
			default:
				return nil, fmt.Errorf("Google People API error %d: %s", resp.StatusCode, string(bodyBytes))
			}
		}

		// Parse response
		var result googleConnectionsResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("failed to parse Google API response: %w", err)
		}
		resp.Body.Close()

		// Convert to SyncedContact structs
		for _, conn := range result.Connections {
			// Check if this is a deleted contact (incremental sync only)
			if conn.Metadata != nil && conn.Metadata.Deleted {
				deletedIDs = append(deletedIDs, conn.ResourceName)
				continue
			}

			if len(conn.EmailAddresses) == 0 {
				continue
			}

			name := ""
			if len(conn.Names) > 0 {
				name = conn.Names[0].DisplayName
			}

			// Create one contact entry per email address
			for _, email := range conn.EmailAddresses {
				if email.Value == "" {
					continue
				}
				allContacts = append(allContacts, SyncedContact{
					Email:       email.Value,
					DisplayName: name,
					RemoteID:    conn.ResourceName, // e.g., "people/c12345"
				})
			}
		}

		s.log.Debug().
			Int("page_count", len(result.Connections)).
			Int("contacts_so_far", len(allContacts)).
			Int("deleted_so_far", len(deletedIDs)).
			Msg("Fetched Google contacts page")

		// Check for more pages
		if result.NextPageToken == "" {
			// Store the next sync token for future incremental syncs
			syncResult := &SyncResult{
				Contacts:      allContacts,
				DeletedIDs:    deletedIDs,
				NextSyncToken: result.NextSyncToken,
				IsFullSync:    isFullSync,
			}

			if isFullSync {
				s.log.Info().
					Int("total_contacts", len(allContacts)).
					Bool("has_sync_token", result.NextSyncToken != "").
					Msg("Google contacts full sync completed")
			} else {
				s.log.Info().
					Int("updated_contacts", len(allContacts)).
					Int("deleted_contacts", len(deletedIDs)).
					Msg("Google contacts incremental sync completed")
			}

			return syncResult, nil
		}
		pageToken = result.NextPageToken
	}
}

// Google People API connections response structures

type googleConnectionsResponse struct {
	Connections   []googleConnection `json:"connections"`
	NextPageToken string             `json:"nextPageToken"`
	NextSyncToken string             `json:"nextSyncToken"` // For incremental sync
	TotalPeople   int                `json:"totalPeople"`
	TotalItems    int                `json:"totalItems"`
}

type googleConnection struct {
	ResourceName   string                    `json:"resourceName"` // e.g., "people/c12345"
	Names          []googleName              `json:"names"`
	EmailAddresses []googleEmail             `json:"emailAddresses"`
	Metadata       *googleConnectionMetadata `json:"metadata,omitempty"` // For detecting deleted contacts
}

type googleConnectionMetadata struct {
	Deleted bool `json:"deleted"` // True if contact was deleted (in incremental sync)
}

// Google API error response structures
type googleErrorResponse struct {
	Error googleErrorDetails `json:"error"`
}

type googleErrorDetails struct {
	Code    int                 `json:"code"`
	Message string              `json:"message"`
	Status  string              `json:"status"`
	Details []googleErrorDetail `json:"details"`
}

type googleErrorDetail struct {
	Type   string `json:"@type"`
	Reason string `json:"reason"` // e.g., "EXPIRED_SYNC_TOKEN"
	Domain string `json:"domain"`
}
