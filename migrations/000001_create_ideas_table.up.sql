-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "citext";
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS ideas (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    created_at TIMESTAMP(0) with TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP(0) with TIME ZONE NOT NULL DEFAULT NOW(),
    idea_source_id UUID,
    category VARCHAR(100) NOT NULL,
    tags TEXT[] NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'Pending',
    version INT NOT NULL DEFAULT 1
);
