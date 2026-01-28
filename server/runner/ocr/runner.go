// Package ocr provides a background runner for processing attachments with OCR and text extraction.
package ocr

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/hrygo/divinesense/internal/profile"
	"github.com/hrygo/divinesense/plugin/ocr"
	"github.com/hrygo/divinesense/plugin/textextract"
	"github.com/hrygo/divinesense/store"
)

// Runner processes attachments for OCR and text extraction.
type Runner struct {
	store               *store.Store
	ocrClient           *ocr.Client
	textExtractClient   *textextract.Client
	interval            time.Duration
	batchSize           int
	ocrEnabled          bool
	textExtractEnabled   bool
	semaphore           chan struct{} // Limits concurrent async processing
}

// NewRunner creates a new OCR runner.
func NewRunner(store *store.Store, profile *profile.Profile) *Runner {
	if profile == nil {
		return &Runner{
			store:     store,
			interval:  5 * time.Minute,
			batchSize: 5,
			semaphore: make(chan struct{}, 10), // Max 10 concurrent async processing
		}
	}

	// Only create OCR clients if enabled
	var ocrClient *ocr.Client
	var textExtractClient *textextract.Client

	if profile.OCREnabled {
		ocrConfig := &ocr.Config{
			TesseractPath: profile.TesseractPath,
			DataPath:      profile.TessdataPath,
			Languages:     profile.OCRLanguages,
		}
		ocrClient = ocr.NewClient(ocrConfig)
	}

	if profile.TextExtractEnabled {
		textConfig := &textextract.Config{
			TikaServerURL: profile.TikaServerURL,
		}
		textExtractClient = textextract.NewClient(textConfig)
	}

	return &Runner{
		store:               store,
		ocrClient:           ocrClient,
		textExtractClient:   textExtractClient,
		interval:            5 * time.Minute,
		batchSize:           5,
		ocrEnabled:          profile.OCREnabled,
		textExtractEnabled:  profile.TextExtractEnabled,
		semaphore:           make(chan struct{}, 10), // Max 10 concurrent async processing
	}
}

// Run starts the background task.
func (r *Runner) Run(ctx context.Context) {
	// Skip if both features are disabled
	if !r.ocrEnabled && !r.textExtractEnabled {
		slog.Info("OCR runner disabled (both OCR and text extraction are disabled)")
		return
	}

	slog.Info("OCR runner started",
		"ocr_enabled", r.ocrEnabled,
		"text_extract_enabled", r.textExtractEnabled,
	)

	// Process once on startup
	r.processPendingAttachments(ctx)

	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			r.processPendingAttachments(ctx)
		case <-ctx.Done():
			slog.Info("OCR runner stopped")
			return
		}
	}
}

// RunOnce processes attachments once (for manual trigger).
func (r *Runner) RunOnce(ctx context.Context) {
	if !r.ocrEnabled && !r.textExtractEnabled {
		slog.Info("OCR runner is disabled, skipping")
		return
	}
	r.processPendingAttachments(ctx)
}

// processPendingAttachments processes attachments that need OCR or text extraction.
func (r *Runner) processPendingAttachments(ctx context.Context) {
	// Find attachments needing processing
	attachments, err := r.findPendingAttachments(ctx)
	if err != nil {
		slog.Error("failed to find pending attachments", "error", err)
		return
	}

	if len(attachments) == 0 {
		return
	}

	slog.Info("processing attachments for OCR/text extraction", "count", len(attachments))

	// Process in batches
	for i := 0; i < len(attachments); i += r.batchSize {
		select {
		case <-ctx.Done():
			slog.Info("OCR processing cancelled", "processed", i, "total", len(attachments))
			return
		default:
		}

		end := i + r.batchSize
		if end > len(attachments) {
			end = len(attachments)
		}
		batch := attachments[i:end]

		for _, attachment := range batch {
			if err := r.processAttachment(ctx, attachment); err != nil {
				slog.Warn("failed to process attachment", "id", attachment.ID, "filename", attachment.Filename, "error", err)
			}
		}
		slog.Info("batch processed", "count", len(batch), "progress", fmt.Sprintf("%d/%d", end, len(attachments)))
	}
}

// findPendingAttachments finds attachments that need OCR or text extraction.
func (r *Runner) findPendingAttachments(ctx context.Context) ([]*store.Attachment, error) {
	// Find attachments where extracted_text and ocr_text are both empty
	// and the MIME type is supported
	attachments, err := r.store.ListAttachments(ctx, &store.FindAttachment{
		Limit: intPtr(r.batchSize * 10),
	})
	if err != nil {
		return nil, err
	}

	// Filter for attachments that need processing
	var pending []*store.Attachment
	for _, att := range attachments {
		if att.RowStatus != "NORMAL" {
			continue
		}
		if att.ExtractedText != "" || att.OCRText != "" {
			// Already processed
			continue
		}
		if !r.needsProcessing(att.Type) {
			continue
		}
		pending = append(pending, att)
	}

	return pending, nil
}

// needsProcessing checks if an attachment type needs OCR or text extraction.
func (r *Runner) needsProcessing(mimeType string) bool {
	// Check if it's an image (OCR)
	if r.ocrEnabled && r.ocrClient != nil {
		for _, supported := range ocr.SupportedMimeTypes {
			if strings.EqualFold(mimeType, supported) {
				return true
			}
		}
	}

	// Check if it's a document (text extraction)
	if r.textExtractEnabled && r.textExtractClient != nil {
		for _, supported := range textextract.SupportedMimeTypes {
			if strings.EqualFold(mimeType, supported) {
				return true
			}
		}
	}

	return false
}

// processAttachment processes a single attachment.
func (r *Runner) processAttachment(ctx context.Context, attachment *store.Attachment) error {
	// Get the blob data
	attachmentWithBlob, err := r.store.GetAttachment(ctx, &store.FindAttachment{
		ID:      &attachment.ID,
		GetBlob: true,
	})
	if err != nil {
		return err
	}
	if attachmentWithBlob == nil {
		return fmt.Errorf("attachment not found")
	}

	update := &store.UpdateAttachment{
		ID:        attachment.ID,
		UpdatedTs: int64Ptr(time.Now().Unix()),
	}

	// Process based on MIME type
	if r.ocrEnabled && r.ocrClient != nil && r.ocrClient.IsSupported(attachmentWithBlob.Type) {
		// OCR for images
		text, err := r.ocrClient.ExtractText(ctx, attachmentWithBlob.Blob, attachmentWithBlob.Type)
		if err != nil {
			slog.Warn("OCR failed", "id", attachment.ID, "error", err)
			// Don't return error, continue with empty text
		} else {
			update.OCRText = &text
			slog.Info("OCR completed", "id", attachment.ID, "text_length", len(text))
		}
	}

	if r.textExtractEnabled && r.textExtractClient != nil && r.textExtractClient.IsSupported(attachmentWithBlob.Type) {
		// Text extraction for documents
		result, err := r.textExtractClient.ExtractText(ctx, attachmentWithBlob.Blob, attachmentWithBlob.Type)
		if err != nil {
			slog.Warn("text extraction failed", "id", attachment.ID, "error", err)
		} else {
			update.ExtractedText = &result.Text
			slog.Info("text extraction completed", "id", attachment.ID, "text_length", len(result.Text))
		}
	}

	// Update attachment if we have new data
	if update.OCRText != nil || update.ExtractedText != nil {
		if err := r.store.UpdateAttachment(ctx, update); err != nil {
			return fmt.Errorf("failed to update attachment: %w", err)
		}
	}

	return nil
}

// ProcessAttachmentAsync processes a single attachment asynchronously.
// This can be called when a new attachment is uploaded.
func (r *Runner) ProcessAttachmentAsync(ctx context.Context, attachmentID int32) {
	if !r.ocrEnabled && !r.textExtractEnabled {
		slog.Debug("OCR runner is disabled, skipping async processing", "attachment_id", attachmentID)
		return
	}

	// Acquire semaphore (non-blocking if full, skip processing)
	select {
	case r.semaphore <- struct{}{}:
		// Got semaphore slot, proceed
	default:
		slog.Warn("async attachment processing skipped (concurrency limit reached)", "attachment_id", attachmentID)
		return
	}

	go func() {
		defer func() { <-r.semaphore }() // Release semaphore

		// Use a timeout context
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		attachment, err := r.store.GetAttachment(ctx, &store.FindAttachment{
			ID: &attachmentID,
		})
		if err != nil {
			slog.Error("failed to get attachment for async processing", "id", attachmentID, "error", err)
			return
		}

		if err := r.processAttachment(ctx, attachment); err != nil {
			slog.Error("async attachment processing failed", "id", attachmentID, "error", err)
		}
	}()
}

// intPtr returns a pointer to an int.
func intPtr(i int) *int {
	return &i
}

// int64Ptr returns a pointer to an int64.
func int64Ptr(i int64) *int64 {
	return &i
}
