-- Remove indexes
DROP INDEX IF EXISTS idx_ideas_user_id;
DROP INDEX IF EXISTS idx_ideas_status;
DROP INDEX IF EXISTS idx_ideas_category;
DROP INDEX IF EXISTS idx_ideas_tags;

-- Remove columns
ALTER TABLE ideas 
    DROP COLUMN IF EXISTS learning_outcome,
    DROP COLUMN IF EXISTS recommended_level,
    DROP COLUMN IF EXISTS github_link,
    DROP COLUMN IF EXISTS website_link,
    DROP COLUMN IF EXISTS user_id;