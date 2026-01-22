-- Rollback attachment OCR and full-text extraction enhancements

-- Drop indexes
DROP INDEX IF EXISTS idx_attachment_text_gin;
DROP INDEX IF EXISTS idx_attachment_type;
DROP INDEX IF EXISTS idx_attachment_memo;
DROP INDEX IF EXISTS idx_attachment_creator_status;

-- Drop constraint
ALTER TABLE attachment DROP CONSTRAINT IF EXISTS chk_attachment_row_status;

-- Drop new columns
ALTER TABLE attachment DROP COLUMN IF EXISTS row_status;
ALTER TABLE attachment DROP COLUMN IF EXISTS thumbnail_path;
ALTER TABLE attachment DROP COLUMN IF EXISTS ocr_text;
ALTER TABLE attachment DROP COLUMN IF EXISTS extracted_text;
ALTER TABLE attachment DROP COLUMN IF EXISTS file_path;

-- Revert payload column back to TEXT
ALTER TABLE attachment ADD COLUMN IF NOT EXISTS payload TEXT DEFAULT '{}';

-- Note: Data in JSONB payload will be lost in rollback
-- In production, consider a more careful migration strategy
ALTER TABLE attachment DROP COLUMN IF EXISTS payload;
ALTER TABLE attachment RENAME COLUMN payload_old TO payload;
