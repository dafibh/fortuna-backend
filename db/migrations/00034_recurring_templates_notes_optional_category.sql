-- +goose Up
-- Add notes field to recurring templates and make category optional

-- Add notes column
ALTER TABLE recurring_templates ADD COLUMN notes TEXT NULL;

-- Make category_id nullable (drop NOT NULL constraint)
ALTER TABLE recurring_templates ALTER COLUMN category_id DROP NOT NULL;

-- +goose Down
-- Remove notes column and restore category_id NOT NULL constraint
ALTER TABLE recurring_templates DROP COLUMN IF EXISTS notes;

-- Note: This down migration will fail if there are rows with NULL category_id
-- You may need to set a default category first
ALTER TABLE recurring_templates ALTER COLUMN category_id SET NOT NULL;
