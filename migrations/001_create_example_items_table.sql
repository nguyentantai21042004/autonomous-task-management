-- Migration: 001_create_example_items_table
-- Description: Creates the example_items table for the example domain.
-- Replace this migration with your actual domain tables.

CREATE TABLE IF NOT EXISTS example_items (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(255) NOT NULL,
    description TEXT         NOT NULL DEFAULT '',
    status      VARCHAR(50)  NOT NULL DEFAULT 'active',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Index on status for fast filtering
CREATE INDEX IF NOT EXISTS idx_example_items_status ON example_items (status);

-- Index on name for uniqueness check (GetOne by name)
CREATE UNIQUE INDEX IF NOT EXISTS idx_example_items_name ON example_items (name);

COMMENT ON TABLE example_items IS 'Example domain items table. Replace with your actual business entity.';
