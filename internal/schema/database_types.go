package schema

import "strings"

// DatabaseType represents supported database types
type DatabaseType string

const (
	PostgreSQL DatabaseType = "postgresql"
	SQLite     DatabaseType = "sqlite"
	MySQL      DatabaseType = "mysql"
)

// String returns the string representation of the database type
func (dt DatabaseType) String() string {
	return string(dt)
}

// IsValid checks if the database type is supported
func (dt DatabaseType) IsValid() bool {
	switch dt {
	case PostgreSQL, SQLite, MySQL:
		return true
	default:
		return false
	}
}

// ParseDatabaseType parses a string into a DatabaseType
func ParseDatabaseType(s string) DatabaseType {
	switch strings.ToLower(s) {
	case "postgresql", "postgres", "pgsql":
		return PostgreSQL
	case "sqlite", "sqlite3":
		return SQLite
	case "mysql", "mariadb":
		return MySQL
	default:
		return ""
	}
}

// GetJSONColumnType returns the appropriate JSON column type for each database
func (dt DatabaseType) GetJSONColumnType() string {
	switch dt {
	case PostgreSQL:
		return "JSONB"
	case SQLite:
		return "TEXT" // SQLite stores JSON as TEXT with JSON functions
	case MySQL:
		return "JSON"
	default:
		return "TEXT"
	}
}

// GetBlobColumnType returns the appropriate BLOB column type for each database
func (dt DatabaseType) GetBlobColumnType() string {
	switch dt {
	case PostgreSQL:
		return "BYTEA"
	case SQLite:
		return "BLOB"
	case MySQL:
		return "BLOB"
	default:
		return "BLOB"
	}
}

// GetIntegerColumnType returns the appropriate integer column type for each database
func (dt DatabaseType) GetIntegerColumnType() string {
	switch dt {
	case PostgreSQL:
		return "INTEGER"
	case SQLite:
		return "INTEGER"
	case MySQL:
		return "INT"
	default:
		return "INTEGER"
	}
}

// GetTimestampColumnType returns the appropriate timestamp column type for each database
func (dt DatabaseType) GetTimestampColumnType() string {
	switch dt {
	case PostgreSQL:
		return "TIMESTAMP WITH TIME ZONE"
	case SQLite:
		return "DATETIME"
	case MySQL:
		return "TIMESTAMP"
	default:
		return "TIMESTAMP"
	}
}

// GetJSONExtractFunction returns the JSON extraction function for each database
func (dt DatabaseType) GetJSONExtractFunction(column, path string) string {
	switch dt {
	case PostgreSQL:
		return column + "->>" + "'" + path + "'"
	case SQLite:
		return "json_extract(" + column + ", '$." + path + "')"
	case MySQL:
		return column + "->>'" + "$." + path + "'"
	default:
		return column + "->>" + "'" + path + "'"
	}
}

// GetMetadataIndexSQL returns the SQL for creating metadata indexes
func (dt DatabaseType) GetMetadataIndexSQL(tableName, indexName, path string) string {
	switch dt {
	case PostgreSQL:
		return "CREATE INDEX " + indexName + " ON " + tableName + " USING GIN ((metadata->>" + "'" + path + "'))"
	case SQLite:
		return "CREATE INDEX " + indexName + " ON " + tableName + " (json_extract(metadata, '$." + path + "'))"
	case MySQL:
		return "CREATE INDEX " + indexName + " ON " + tableName + " ((metadata->>'" + "$." + path + "'))"
	default:
		return "-- Unsupported database type for metadata index"
	}
}

// SupportsJSONIndexing returns true if the database supports efficient JSON indexing
func (dt DatabaseType) SupportsJSONIndexing() bool {
	switch dt {
	case PostgreSQL, MySQL:
		return true
	case SQLite:
		return true // SQLite 3.38+ supports JSON functions with indexes
	default:
		return false
	}
}

// RequiresJSONValidation returns true if the database requires JSON validation constraints
func (dt DatabaseType) RequiresJSONValidation() bool {
	switch dt {
	case SQLite:
		return true // SQLite requires CHECK(json_valid(column))
	case PostgreSQL, MySQL:
		return false // These enforce JSON validity at the column type level
	default:
		return true
	}
}

// GetJSONValidationConstraint returns the JSON validation constraint for databases that need it
func (dt DatabaseType) GetJSONValidationConstraint(columnName string) string {
	switch dt {
	case SQLite:
		return "CHECK (json_valid(" + columnName + "))"
	default:
		return ""
	}
}