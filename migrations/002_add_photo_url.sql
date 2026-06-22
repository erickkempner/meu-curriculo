-- 002_add_photo_url.sql
-- Add photo_url column to resumes table for profile photo support.

---- create
ALTER TABLE resumes ADD COLUMN photo_url VARCHAR(500) NOT NULL DEFAULT '';

---- drop
ALTER TABLE resumes DROP COLUMN IF EXISTS photo_url;
