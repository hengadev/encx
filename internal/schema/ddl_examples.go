package schema

// DDL examples for different database systems

// PostgreSQL DDL example with JSONB support
const PostgreSQLExample = `
-- Example PostgreSQL table with hybrid schema approach
CREATE TABLE users_encx (
    id SERIAL PRIMARY KEY,

    -- Individual encrypted/hashed fields
    email_encrypted BYTEA,
    email_hash VARCHAR(64),
    phone_encrypted BYTEA,
    phone_hash VARCHAR(64),
    ssn_encrypted BYTEA,
    ssn_hash_secure TEXT,

    -- Essential encryption fields
    dek_encrypted BYTEA NOT NULL,
    key_version INTEGER NOT NULL,

    -- Flexible metadata column (JSONB for PostgreSQL)
    metadata JSONB NOT NULL DEFAULT '{}',

    -- Standard fields
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Index on metadata fields for efficient queries
CREATE INDEX idx_users_encx_metadata_serializer ON users_encx USING GIN ((metadata->>'serializer_type'));
CREATE INDEX idx_users_encx_metadata_kek_alias ON users_encx USING GIN ((metadata->>'kek_alias'));

-- Index on hash fields for lookups
CREATE INDEX idx_users_encx_email_hash ON users_encx (email_hash);
CREATE INDEX idx_users_encx_phone_hash ON users_encx (phone_hash);
`

// SQLite DDL example with JSON support
const SQLiteExample = `
-- Example SQLite table with hybrid schema approach
CREATE TABLE users_encx (
    id INTEGER PRIMARY KEY AUTOINCREMENT,

    -- Individual encrypted/hashed fields
    email_encrypted BLOB,
    email_hash TEXT,
    phone_encrypted BLOB,
    phone_hash TEXT,
    ssn_encrypted BLOB,
    ssn_hash_secure TEXT,

    -- Essential encryption fields
    dek_encrypted BLOB NOT NULL,
    key_version INTEGER NOT NULL,

    -- Flexible metadata column (JSON for SQLite)
    metadata TEXT NOT NULL DEFAULT '{}',

    -- Standard fields
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- SQLite indexes using JSON functions
CREATE INDEX idx_users_encx_metadata_serializer ON users_encx (json_extract(metadata, '$.serializer_type'));
CREATE INDEX idx_users_encx_metadata_kek_alias ON users_encx (json_extract(metadata, '$.kek_alias'));

-- Index on hash fields for lookups
CREATE INDEX idx_users_encx_email_hash ON users_encx (email_hash);
CREATE INDEX idx_users_encx_phone_hash ON users_encx (phone_hash);
`

// Example migration SQL for adding metadata column to existing tables
const MigrationExample = `
-- Migration to add metadata column to existing table
-- PostgreSQL version
ALTER TABLE users_encx ADD COLUMN metadata JSONB NOT NULL DEFAULT '{}';

-- SQLite version
ALTER TABLE users_encx ADD COLUMN metadata TEXT NOT NULL DEFAULT '{}';

-- Populate metadata for existing rows (example)
UPDATE users_encx SET metadata = json_object(
    'serializer_type', 'json',
    'generator_version', '1.0.0',
    'kek_alias', 'default',
    'encryption_time', strftime('%s', 'now'),
    'pepper_version', 1
) WHERE metadata = '{}';
`