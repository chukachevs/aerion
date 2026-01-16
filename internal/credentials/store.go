// Package credentials provides secure credential storage with fallback support
package credentials

import (
	"database/sql"
	"fmt"

	"github.com/hkdb/aerion/internal/crypto"
	"github.com/hkdb/aerion/internal/logging"
	"github.com/rs/zerolog"
	gokeyring "github.com/zalando/go-keyring"
)

const serviceName = "aerion"

// Store provides credential storage with OS keyring and encrypted DB fallback
type Store struct {
	db             *sql.DB
	encryptor      *crypto.Encryptor
	keyringEnabled bool
	log            zerolog.Logger
}

// NewStore creates a new credential store
// It tries to use the OS keyring, falling back to encrypted database storage
func NewStore(db *sql.DB, dataDir string) (*Store, error) {
	log := logging.WithComponent("credentials")

	// Create encryptor for fallback storage
	encryptor, err := crypto.NewEncryptor(dataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create encryptor: %w", err)
	}

	// Test if keyring is available
	keyringEnabled := testKeyring()
	if keyringEnabled {
		log.Info().Msg("OS keyring available, using as primary credential storage")
	} else {
		log.Warn().Msg("OS keyring not available, using encrypted database storage")
	}

	return &Store{
		db:             db,
		encryptor:      encryptor,
		keyringEnabled: keyringEnabled,
		log:            log,
	}, nil
}

// testKeyring checks if the OS keyring is available and functional
func testKeyring() bool {
	testKey := "aerion-test-keyring-check"
	testValue := "test"

	// Try to set a test value
	err := gokeyring.Set(serviceName, testKey, testValue)
	if err != nil {
		return false
	}

	// Clean up test value
	gokeyring.Delete(serviceName, testKey)

	return true
}

// SetPassword stores a password for an account
func (s *Store) SetPassword(accountID, password string) error {
	if password == "" {
		return nil
	}

	// Try OS keyring first if available
	if s.keyringEnabled {
		err := gokeyring.Set(serviceName, accountID, password)
		if err == nil {
			s.log.Debug().Str("account_id", accountID).Msg("Password stored in OS keyring")
			// Clear any fallback storage
			s.clearDBPassword(accountID)
			return nil
		}
		s.log.Warn().Err(err).Msg("Failed to store in OS keyring, using fallback")
	}

	// Fallback to encrypted database storage
	encrypted, err := s.encryptor.Encrypt(password)
	if err != nil {
		return fmt.Errorf("failed to encrypt password: %w", err)
	}

	_, err = s.db.Exec(
		"UPDATE accounts SET encrypted_password = ? WHERE id = ?",
		encrypted, accountID,
	)
	if err != nil {
		return fmt.Errorf("failed to store encrypted password: %w", err)
	}

	s.log.Debug().Str("account_id", accountID).Msg("Password stored in encrypted database")
	return nil
}

// GetPassword retrieves a password for an account
func (s *Store) GetPassword(accountID string) (string, error) {
	// Try OS keyring first if available
	if s.keyringEnabled {
		password, err := gokeyring.Get(serviceName, accountID)
		if err == nil {
			return password, nil
		}
		if err != gokeyring.ErrNotFound {
			s.log.Warn().Err(err).Msg("Error reading from OS keyring, trying fallback")
		}
	}

	// Try fallback encrypted database storage
	var encrypted sql.NullString
	err := s.db.QueryRow(
		"SELECT encrypted_password FROM accounts WHERE id = ?",
		accountID,
	).Scan(&encrypted)

	if err == sql.ErrNoRows {
		return "", ErrCredentialNotFound
	}
	if err != nil {
		return "", fmt.Errorf("failed to query password: %w", err)
	}

	if !encrypted.Valid || encrypted.String == "" {
		return "", ErrCredentialNotFound
	}

	// Decrypt
	password, err := s.encryptor.Decrypt(encrypted.String)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt password: %w", err)
	}

	return password, nil
}

// DeletePassword removes a password for an account
func (s *Store) DeletePassword(accountID string) error {
	// Delete from OS keyring
	if s.keyringEnabled {
		gokeyring.Delete(serviceName, accountID)
	}

	// Delete from database
	s.clearDBPassword(accountID)

	return nil
}

// clearDBPassword clears the encrypted password from the database
func (s *Store) clearDBPassword(accountID string) {
	s.db.Exec("UPDATE accounts SET encrypted_password = NULL WHERE id = ?", accountID)
}

// DeleteAllCredentials removes all credentials for an account
func (s *Store) DeleteAllCredentials(accountID string) error {
	s.DeletePassword(accountID)
	s.DeleteOAuthTokens(accountID)
	return nil
}

// IsKeyringEnabled returns whether the OS keyring is being used
func (s *Store) IsKeyringEnabled() bool {
	return s.keyringEnabled
}

// SetCardDAVPassword stores a password for a CardDAV contact source
func (s *Store) SetCardDAVPassword(sourceID, password string) error {
	if password == "" {
		return nil
	}

	// Try OS keyring first if available
	if s.keyringEnabled {
		err := gokeyring.Set(serviceName, "carddav:"+sourceID, password)
		if err == nil {
			s.log.Debug().Str("source_id", sourceID).Msg("CardDAV password stored in OS keyring")
			// Clear any fallback storage
			s.clearCardDAVDBPassword(sourceID)
			return nil
		}
		s.log.Warn().Err(err).Msg("Failed to store CardDAV password in OS keyring, using fallback")
	}

	// Fallback to encrypted database storage
	encrypted, err := s.encryptor.Encrypt(password)
	if err != nil {
		return fmt.Errorf("failed to encrypt password: %w", err)
	}

	_, err = s.db.Exec(
		"UPDATE contact_sources SET encrypted_password = ? WHERE id = ?",
		encrypted, sourceID,
	)
	if err != nil {
		return fmt.Errorf("failed to store encrypted password: %w", err)
	}

	s.log.Debug().Str("source_id", sourceID).Msg("CardDAV password stored in encrypted database")
	return nil
}

// GetCardDAVPassword retrieves a password for a CardDAV contact source
func (s *Store) GetCardDAVPassword(sourceID string) (string, error) {
	// Try OS keyring first if available
	if s.keyringEnabled {
		password, err := gokeyring.Get(serviceName, "carddav:"+sourceID)
		if err == nil {
			return password, nil
		}
		if err != gokeyring.ErrNotFound {
			s.log.Warn().Err(err).Msg("Error reading CardDAV password from OS keyring, trying fallback")
		}
	}

	// Try fallback encrypted database storage
	var encrypted sql.NullString
	err := s.db.QueryRow(
		"SELECT encrypted_password FROM contact_sources WHERE id = ?",
		sourceID,
	).Scan(&encrypted)

	if err == sql.ErrNoRows {
		return "", ErrCredentialNotFound
	}
	if err != nil {
		return "", fmt.Errorf("failed to query password: %w", err)
	}

	if !encrypted.Valid || encrypted.String == "" {
		return "", ErrCredentialNotFound
	}

	// Decrypt
	password, err := s.encryptor.Decrypt(encrypted.String)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt password: %w", err)
	}

	return password, nil
}

// DeleteCardDAVPassword removes a password for a CardDAV contact source
func (s *Store) DeleteCardDAVPassword(sourceID string) error {
	// Delete from OS keyring
	if s.keyringEnabled {
		gokeyring.Delete(serviceName, "carddav:"+sourceID)
	}

	// Delete from database
	s.clearCardDAVDBPassword(sourceID)

	return nil
}

// clearCardDAVDBPassword clears the encrypted password from the contact_sources table
func (s *Store) clearCardDAVDBPassword(sourceID string) {
	s.db.Exec("UPDATE contact_sources SET encrypted_password = NULL WHERE id = ?", sourceID)
}
