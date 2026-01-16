package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/hkdb/aerion/internal/email"
	"github.com/hkdb/aerion/internal/logging"
	"github.com/hkdb/aerion/internal/message"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// ============================================================================
// Attachment API - Exposed to frontend via Wails bindings
// ============================================================================

// GetAttachments returns all attachments for a message
func (a *App) GetAttachments(messageID string) ([]*message.Attachment, error) {
	return a.attachmentStore.GetByMessage(messageID)
}

// GetAttachment returns a single attachment by ID
func (a *App) GetAttachment(attachmentID string) (*message.Attachment, error) {
	return a.attachmentStore.Get(attachmentID)
}

// GetInlineAttachments returns a map of content-id to data URL for all inline attachments
// This is used to resolve cid: references in HTML email bodies
// Content is read from the database (stored during sync) for fast offline access
func (a *App) GetInlineAttachments(messageID string) (map[string]string, error) {
	log := logging.WithComponent("app")

	log.Info().Str("messageID", messageID).Msg("GetInlineAttachments called")

	// Get inline attachments with content from database
	// This is fast and works offline since content is stored during sync
	result, err := a.attachmentStore.GetInlineByMessage(messageID)
	if err != nil {
		log.Error().Err(err).Str("messageID", messageID).Msg("Failed to get inline attachments from database")
		return nil, fmt.Errorf("failed to get inline attachments: %w", err)
	}

	// Log the content IDs we found
	contentIDs := make([]string, 0, len(result))
	for cid := range result {
		contentIDs = append(contentIDs, cid)
	}
	log.Info().Int("count", len(result)).Strs("contentIDs", contentIDs).Str("messageID", messageID).Msg("Returning inline attachments")

	return result, nil
}

// DownloadAttachment downloads an attachment and saves it to disk
// If savePath is empty, saves to the default attachments directory
// Returns the path where the file was saved
func (a *App) DownloadAttachment(attachmentID, savePath string) (string, error) {
	log := logging.WithComponent("app")

	log.Debug().Str("attachmentID", attachmentID).Str("savePath", savePath).Msg("DownloadAttachment called")

	// Get attachment metadata
	att, err := a.attachmentStore.Get(attachmentID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get attachment from store")
		return "", fmt.Errorf("failed to get attachment: %w", err)
	}
	if att == nil {
		log.Error().Str("attachmentID", attachmentID).Msg("Attachment not found")
		return "", fmt.Errorf("attachment not found: %s", attachmentID)
	}

	log.Debug().Str("filename", att.Filename).Int("size", att.Size).Msg("Got attachment metadata")

	// Check if already downloaded (only for default location, not custom paths)
	if savePath == "" && att.LocalPath != "" {
		if _, err := os.Stat(att.LocalPath); err == nil {
			log.Debug().Str("localPath", att.LocalPath).Msg("Attachment already downloaded")
			return att.LocalPath, nil
		}
	}

	// Get the message to find folder and UID
	msg, err := a.messageStore.Get(att.MessageID)
	if err != nil {
		log.Error().Err(err).Str("messageID", att.MessageID).Msg("Failed to get message")
		return "", fmt.Errorf("failed to get message: %w", err)
	}
	if msg == nil {
		log.Error().Str("messageID", att.MessageID).Msg("Message not found")
		return "", fmt.Errorf("message not found: %s", att.MessageID)
	}

	log.Debug().Uint32("uid", msg.UID).Str("folderID", msg.FolderID).Msg("Got message info")

	// Fetch raw message from IMAP
	raw, err := a.syncEngine.FetchRawMessage(a.ctx, msg.AccountID, msg.FolderID, msg.UID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch raw message from IMAP")
		return "", fmt.Errorf("failed to fetch message: %w", err)
	}

	log.Debug().Int("rawSize", len(raw)).Msg("Fetched raw message from IMAP")

	// Extract attachment content
	downloader := email.NewAttachmentDownloader(a.paths.AttachmentsPath())
	content, err := downloader.ExtractAttachmentContent(raw, att.Filename)
	if err != nil {
		log.Error().Err(err).Str("filename", att.Filename).Msg("Failed to extract attachment content")
		return "", fmt.Errorf("failed to extract attachment: %w", err)
	}

	log.Debug().Int("contentSize", len(content)).Msg("Extracted attachment content")

	// Save to disk
	localPath, err := downloader.SaveAttachment(att, content, savePath)
	if err != nil {
		log.Error().Err(err).Str("savePath", savePath).Msg("Failed to save attachment to disk")
		return "", fmt.Errorf("failed to save attachment: %w", err)
	}

	// Update attachment record with local path (only for default location)
	if savePath == "" {
		if err := a.attachmentStore.UpdateLocalPath(attachmentID, localPath); err != nil {
			log.Warn().Err(err).Msg("Failed to update attachment local path")
		}
	}

	log.Info().Str("attachment", att.Filename).Str("path", localPath).Int("size", len(content)).Msg("Attachment downloaded")
	return localPath, nil
}

// OpenAttachment downloads (if needed) and opens an attachment with the default application
func (a *App) OpenAttachment(attachmentID string) error {
	// Download if not already downloaded
	localPath, err := a.DownloadAttachment(attachmentID, "")
	if err != nil {
		return err
	}

	// Open with default application using runtime
	return a.openFile(localPath)
}

// SaveAttachmentAs shows a Save As dialog and saves the attachment to the user-selected location
// Returns the path where the file was saved, or empty string if cancelled
func (a *App) SaveAttachmentAs(attachmentID string) (string, error) {
	log := logging.WithComponent("app")

	log.Debug().Str("attachmentID", attachmentID).Msg("SaveAttachmentAs called")

	// Get attachment metadata for the filename
	att, err := a.attachmentStore.Get(attachmentID)
	if err != nil {
		log.Error().Err(err).Str("attachmentID", attachmentID).Msg("Failed to get attachment metadata")
		return "", fmt.Errorf("failed to get attachment: %w", err)
	}
	if att == nil {
		log.Error().Str("attachmentID", attachmentID).Msg("Attachment not found in database")
		return "", fmt.Errorf("attachment not found: %s", attachmentID)
	}

	log.Debug().Str("filename", att.Filename).Str("messageID", att.MessageID).Msg("Found attachment metadata")

	// Get user's home directory for default save location
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = ""
	}
	defaultDir := filepath.Join(homeDir, "Downloads")

	// Show Save As dialog
	savePath, err := wailsRuntime.SaveFileDialog(a.ctx, wailsRuntime.SaveDialogOptions{
		DefaultDirectory: defaultDir,
		DefaultFilename:  att.Filename,
		Title:            "Save Attachment",
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to show save dialog")
		return "", fmt.Errorf("failed to show save dialog: %w", err)
	}

	log.Debug().Str("savePath", savePath).Msg("User selected save path")

	// User cancelled the dialog
	if savePath == "" {
		log.Debug().Msg("User cancelled save dialog")
		return "", nil
	}

	// Download and save to the selected path
	resultPath, err := a.DownloadAttachment(attachmentID, savePath)
	if err != nil {
		log.Error().Err(err).Str("savePath", savePath).Msg("Failed to download attachment")
		return "", err
	}

	log.Info().Str("attachment", att.Filename).Str("path", resultPath).Msg("Attachment saved")
	return resultPath, nil
}

// openFile opens a file with the system default application
func (a *App) openFile(path string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", path)
	case "darwin":
		cmd = exec.Command("open", path)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", path)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	return cmd.Start()
}

// OpenFile opens a file with the system default application (exposed to frontend)
func (a *App) OpenFile(path string) error {
	return a.openFile(path)
}

// OpenFolder opens the folder containing a file in the system file manager
func (a *App) OpenFolder(path string) error {
	dir := filepath.Dir(path)
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "linux":
		// Try to select the file in the file manager if possible
		cmd = exec.Command("xdg-open", dir)
	case "darwin":
		// -R reveals the file in Finder
		cmd = exec.Command("open", "-R", path)
	case "windows":
		// /select highlights the file in Explorer
		cmd = exec.Command("explorer", "/select,", path)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	return cmd.Start()
}

// SaveAllAttachments shows a folder picker and saves all attachments from a message to that folder
// Returns the folder path where files were saved, or empty string if cancelled
func (a *App) SaveAllAttachments(messageID string) (string, error) {
	log := logging.WithComponent("app")

	// Get all attachments for the message
	attachments, err := a.attachmentStore.GetByMessage(messageID)
	if err != nil {
		return "", fmt.Errorf("failed to get attachments: %w", err)
	}
	if len(attachments) == 0 {
		return "", fmt.Errorf("no attachments found for message")
	}

	// Get user's home directory for default save location
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = ""
	}
	defaultDir := filepath.Join(homeDir, "Downloads")

	// Show folder picker dialog
	saveDir, err := wailsRuntime.OpenDirectoryDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		DefaultDirectory: defaultDir,
		Title:            "Save All Attachments",
	})
	if err != nil {
		return "", fmt.Errorf("failed to show folder dialog: %w", err)
	}

	// User cancelled the dialog
	if saveDir == "" {
		return "", nil
	}

	// Get the message to find folder and UID
	msg, err := a.messageStore.Get(messageID)
	if err != nil {
		return "", fmt.Errorf("failed to get message: %w", err)
	}
	if msg == nil {
		return "", fmt.Errorf("message not found: %s", messageID)
	}

	// Fetch raw message from IMAP
	raw, err := a.syncEngine.FetchRawMessage(a.ctx, msg.AccountID, msg.FolderID, msg.UID)
	if err != nil {
		return "", fmt.Errorf("failed to fetch message: %w", err)
	}

	// Save each attachment
	downloader := email.NewAttachmentDownloader(a.paths.AttachmentsPath())
	savedCount := 0

	for _, att := range attachments {
		content, err := downloader.ExtractAttachmentContent(raw, att.Filename)
		if err != nil {
			log.Warn().Err(err).Str("filename", att.Filename).Msg("Failed to extract attachment")
			continue
		}

		savePath := filepath.Join(saveDir, att.Filename)
		_, err = downloader.SaveAttachment(att, content, savePath)
		if err != nil {
			log.Warn().Err(err).Str("filename", att.Filename).Msg("Failed to save attachment")
			continue
		}
		savedCount++
	}

	log.Info().Int("count", savedCount).Str("folder", saveDir).Msg("Saved all attachments")
	return saveDir, nil
}
