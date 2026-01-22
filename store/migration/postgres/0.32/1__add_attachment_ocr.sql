-- Add OCR and full-text extraction fields to attachment table
-- Phase 1: Attachment Management Enhancement

-- Add new columns to attachment table
ALTER TABLE attachment
ADD COLUMN IF NOT EXISTS extracted_text TEXT,
ADD COLUMN IF NOT EXISTS ocr_text TEXT,
ADD COLUMN IF NOT EXISTS thumbnail_path TEXT,
ADD COLUMN IF NOT EXISTS row_status TEXT NOT NULL DEFAULT 'NORMAL',
ADD COLUMN IF NOT EXISTS file_path TEXT;

-- Convert payload column from TEXT to JSONB for better structure
-- First create a new JSONB column
ALTER TABLE attachment
ADD COLUMN IF NOT EXISTS payload_jsonb JSONB NOT NULL DEFAULT '{}';

-- Migrate existing TEXT payload to JSONB (try to parse)
UPDATE attachment
SET payload_jsonb = CASE
    WHEN payload = '' THEN '{}'::jsonb
    WHEN payload::text = '{}' THEN '{}'::jsonb
    WHEN jsonb_typeof(payload::text::jsonb) IS NOT NULL THEN payload::text::jsonb
    ELSE '{"legacy": "' || replace(replace(payload, '\', '\\'), '"', '\"') || '"}'::jsonb
END;

-- Drop old payload column and rename new one
ALTER TABLE attachment DROP COLUMN IF EXISTS payload;
ALTER TABLE attachment RENAME COLUMN payload_jsonb TO payload;

-- Add indexes for OCR performance
CREATE INDEX IF NOT EXISTS idx_attachment_creator_status
ON attachment(creator_id, row_status);

CREATE INDEX IF NOT EXISTS idx_attachment_type
ON attachment(type);

CREATE INDEX IF NOT EXISTS idx_attachment_memo
ON attachment(memo_id) WHERE memo_id IS NOT NULL;

-- Full-text search index for extracted and OCR text
CREATE INDEX IF NOT EXISTS idx_attachment_text_gin
ON attachment USING gin(to_tsvector('simple', COALESCE(extracted_text, '') || ' ' || COALESCE(ocr_text, '')))
WHERE extracted_text IS NOT NULL OR ocr_text IS NOT NULL;

-- Add constraint for row_status
ALTER TABLE attachment
ADD CONSTRAINT IF NOT EXISTS chk_attachment_row_status
CHECK (row_status IN ('NORMAL', 'ARCHIVED', 'DELETED'));

-- Comments
COMMENT ON COLUMN attachment.extracted_text IS 'Text extracted from PDF/Office documents via Apache Tika';
COMMENT ON COLUMN attachment.ocr_text IS 'Text extracted from images via Tesseract OCR';
COMMENT ON COLUMN attachment.thumbnail_path IS 'Path to thumbnail image (for images/PDFs)';
COMMENT ON COLUMN attachment.row_status IS 'Record status: NORMAL, ARCHIVED, DELETED';
COMMENT ON COLUMN attachment.file_path IS 'Internal file path for LOCAL storage type';
