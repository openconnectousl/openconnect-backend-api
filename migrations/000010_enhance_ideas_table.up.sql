-- Add new columns to ideas table
ALTER TABLE ideas 
    ADD COLUMN learning_outcome TEXT,
    ADD COLUMN recommended_level VARCHAR(50),
    ADD COLUMN github_link TEXT,
    ADD COLUMN website_link TEXT,
    ADD COLUMN user_id UUID REFERENCES users(id) ON DELETE CASCADE;

-- Fix the existing table structure by removing trailing comma if it exists
-- You may need to recreate the table if this doesn't work
ALTER TABLE ideas DROP CONSTRAINT IF EXISTS ideas_pkey;
ALTER TABLE ideas ADD PRIMARY KEY (id);

-- Create indexes for faster searches
CREATE INDEX idx_ideas_user_id ON ideas(user_id);
CREATE INDEX idx_ideas_status ON ideas(status);
CREATE INDEX idx_ideas_category ON ideas(category);
CREATE INDEX idx_ideas_tags ON ideas USING gin(tags);