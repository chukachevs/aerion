// Package email provides email content processing utilities
package email

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/quotedprintable"
	"path/filepath"
	"strings"

	gomessage "github.com/emersion/go-message"
	"github.com/google/uuid"
	"github.com/hkdb/aerion/internal/message"
	"github.com/teamwork/tnef"
)

// AttachmentExtractor extracts attachment metadata and content from emails
type AttachmentExtractor struct{}

// NewAttachmentExtractor creates a new attachment extractor
func NewAttachmentExtractor() *AttachmentExtractor {
	return &AttachmentExtractor{}
}

// AttachmentData holds both metadata and content for an attachment
type AttachmentData struct {
	Attachment *message.Attachment
	Content    []byte
}

// ExtractAttachments extracts all attachments from a raw email message
func (e *AttachmentExtractor) ExtractAttachments(messageID string, raw []byte) ([]*AttachmentData, error) {
	reader := bytes.NewReader(raw)

	entity, err := gomessage.Read(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to parse message: %w", err)
	}

	var attachments []*AttachmentData

	// Check if it's a multipart message
	if mr := entity.MultipartReader(); mr != nil {
		attachments = e.extractFromMultipart(messageID, mr)
	}

	return attachments, nil
}

// extractFromMultipart extracts attachments from a multipart message
func (e *AttachmentExtractor) extractFromMultipart(messageID string, mr gomessage.MultipartReader) []*AttachmentData {
	var attachments []*AttachmentData

	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}

		contentType, params, _ := mime.ParseMediaType(part.Header.Get("Content-Type"))
		disposition, dispParams, _ := mime.ParseMediaType(part.Header.Get("Content-Disposition"))
		contentID := strings.Trim(part.Header.Get("Content-ID"), "<>")

		// Handle nested multipart
		if strings.HasPrefix(contentType, "multipart/") {
			if nestedMr := part.MultipartReader(); nestedMr != nil {
				nested := e.extractFromMultipart(messageID, nestedMr)
				attachments = append(attachments, nested...)
			}
			continue
		}

		// Check for TNEF (winmail.dat)
		if contentType == "application/ms-tnef" ||
			(disposition == "attachment" && strings.EqualFold(dispParams["filename"], "winmail.dat")) {
			tnefAttachments := e.extractFromTNEF(messageID, part.Body)
			attachments = append(attachments, tnefAttachments...)
			continue
		}

		// Determine if this is an attachment
		isAttachment := disposition == "attachment"
		isInline := disposition == "inline" || contentID != ""

		// Skip text/plain and text/html unless they're explicit attachments
		if !isAttachment && (contentType == "text/plain" || contentType == "text/html") {
			continue
		}

		// If it's not text and has content, treat it as an attachment
		if isAttachment || isInline || (!strings.HasPrefix(contentType, "text/") && contentType != "") {
			// Get filename
			filename := dispParams["filename"]
			if filename == "" {
				filename = params["name"]
			}
			if filename == "" && contentID != "" {
				// Generate filename from content-id for inline images
				ext := getExtensionForMimeType(contentType)
				filename = contentID + ext
			}
			if filename == "" {
				// Generate a default filename
				ext := getExtensionForMimeType(contentType)
				filename = "attachment" + ext
			}

			// Decode filename if encoded
			decodedFilename, err := decodeRFC2047(filename)
			if err == nil {
				filename = decodedFilename
			}

			// Read content
			content, err := io.ReadAll(part.Body)
			if err != nil {
				continue
			}

			// Decode content if transfer-encoded
			transferEncoding := strings.ToLower(part.Header.Get("Content-Transfer-Encoding"))
			decodedContent := decodeContent(content, transferEncoding)

			att := &message.Attachment{
				ID:          uuid.New().String(),
				MessageID:   messageID,
				Filename:    filename,
				ContentType: contentType,
				Size:        len(decodedContent),
				ContentID:   contentID,
				IsInline:    isInline && contentID != "",
			}

			attachments = append(attachments, &AttachmentData{
				Attachment: att,
				Content:    decodedContent,
			})
		}
	}

	return attachments
}

// extractFromTNEF extracts attachments from a TNEF (winmail.dat) file
func (e *AttachmentExtractor) extractFromTNEF(messageID string, reader io.Reader) []*AttachmentData {
	var attachments []*AttachmentData

	data, err := io.ReadAll(reader)
	if err != nil {
		return attachments
	}

	// Parse TNEF
	tnefData, err := tnef.Decode(data)
	if err != nil {
		return attachments
	}

	// Extract attachments
	for _, att := range tnefData.Attachments {
		filename := att.Title
		if filename == "" {
			filename = "attachment"
		}

		// Try to guess content type from filename
		contentType := "application/octet-stream"
		if guessed := mime.TypeByExtension(filepath.Ext(filename)); guessed != "" {
			contentType = guessed
		}

		attachment := &message.Attachment{
			ID:          uuid.New().String(),
			MessageID:   messageID,
			Filename:    filename,
			ContentType: contentType,
			Size:        len(att.Data),
			IsInline:    false,
		}

		attachments = append(attachments, &AttachmentData{
			Attachment: attachment,
			Content:    att.Data,
		})
	}

	return attachments
}

// decodeContent decodes content based on transfer encoding
func decodeContent(content []byte, encoding string) []byte {
	switch encoding {
	case "base64":
		decoded := make([]byte, base64.StdEncoding.DecodedLen(len(content)))
		n, err := base64.StdEncoding.Decode(decoded, content)
		if err != nil {
			return content
		}
		return decoded[:n]
	case "quoted-printable":
		reader := quotedprintable.NewReader(bytes.NewReader(content))
		decoded, err := io.ReadAll(reader)
		if err != nil {
			return content
		}
		return decoded
	default:
		return content
	}
}

// decodeRFC2047 decodes RFC 2047 encoded strings (like filenames)
func decodeRFC2047(s string) (string, error) {
	dec := new(mime.WordDecoder)
	return dec.DecodeHeader(s)
}

// getExtensionForMimeType returns a file extension for a MIME type
func getExtensionForMimeType(mimeType string) string {
	extensions, err := mime.ExtensionsByType(mimeType)
	if err != nil || len(extensions) == 0 {
		switch mimeType {
		case "image/jpeg":
			return ".jpg"
		case "image/png":
			return ".png"
		case "image/gif":
			return ".gif"
		case "application/pdf":
			return ".pdf"
		case "text/plain":
			return ".txt"
		case "text/html":
			return ".html"
		case "application/zip":
			return ".zip"
		case "application/msword":
			return ".doc"
		case "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
			return ".docx"
		case "application/vnd.ms-excel":
			return ".xls"
		case "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":
			return ".xlsx"
		default:
			return ""
		}
	}
	return extensions[0]
}

// GetAttachmentIcon returns an icon name based on content type
func GetAttachmentIcon(contentType string) string {
	switch {
	case strings.HasPrefix(contentType, "image/"):
		return "mdi:file-image"
	case strings.HasPrefix(contentType, "video/"):
		return "mdi:file-video"
	case strings.HasPrefix(contentType, "audio/"):
		return "mdi:file-music"
	case contentType == "application/pdf":
		return "mdi:file-pdf-box"
	case strings.Contains(contentType, "word") || contentType == "application/msword":
		return "mdi:file-word"
	case strings.Contains(contentType, "excel") || contentType == "application/vnd.ms-excel":
		return "mdi:file-excel"
	case strings.Contains(contentType, "powerpoint") || contentType == "application/vnd.ms-powerpoint":
		return "mdi:file-powerpoint"
	case strings.Contains(contentType, "zip") || strings.Contains(contentType, "compressed"):
		return "mdi:folder-zip"
	case contentType == "text/plain":
		return "mdi:file-document-outline"
	case contentType == "text/html":
		return "mdi:language-html5"
	default:
		return "mdi:file"
	}
}
