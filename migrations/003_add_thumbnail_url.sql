-- 003_add_thumbnail_url.sql
-- Add thumbnail_url column to resumes table for card preview images.

---- create
ALTER TABLE resumes ADD COLUMN thumbnail_url VARCHAR(500) NOT NULL DEFAULT '';

---- drop
ALTER TABLE resumes DROP COLUMN IF EXISTS thumbnail_url;
