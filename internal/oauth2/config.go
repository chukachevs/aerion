// Package oauth2 provides OAuth2 authentication for email providers
package oauth2

// Build-time variables injected via ldflags
// These are set during compilation using:
//
//	go build -ldflags "-X 'github.com/hkdb/aerion/internal/oauth2.GoogleClientID=xxx'"
//
// See Makefile for the complete build command.
var (
	// GoogleClientID is the OAuth2 client ID for Google/Gmail
	// Obtain from Google Cloud Console: https://console.cloud.google.com
	GoogleClientID string

	// GoogleClientSecret is the OAuth2 client secret for Google/Gmail
	// For desktop apps, this can be empty (public client)
	GoogleClientSecret string

	// MicrosoftClientID is the OAuth2 client ID for Microsoft/Outlook
	// Obtain from Azure Portal: https://portal.azure.com
	MicrosoftClientID string
)

// IsGoogleConfigured returns true if Google OAuth credentials are available
func IsGoogleConfigured() bool {
	return GoogleClientID != ""
}

// IsMicrosoftConfigured returns true if Microsoft OAuth credentials are available
func IsMicrosoftConfigured() bool {
	return MicrosoftClientID != ""
}

// IsProviderConfigured returns true if the specified provider has OAuth credentials
func IsProviderConfigured(provider string) bool {
	switch provider {
	case "google":
		return IsGoogleConfigured()
	case "microsoft":
		return IsMicrosoftConfigured()
	default:
		return false
	}
}
