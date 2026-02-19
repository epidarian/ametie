-- Migration: Add encrypted_key column to api_keys table
-- Run this if you have an existing database

ALTER TABLE api_keys ADD COLUMN encrypted_key TEXT NOT NULL AFTER key_hash;

-- Note: Existing keys will need to be re-encrypted
-- You'll need to update them with encrypted versions

