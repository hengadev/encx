package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDatabaseType_String(t *testing.T) {
	tests := []struct {
		name string
		dt   DatabaseType
		want string
	}{
		{
			name: "PostgreSQL",
			dt:   PostgreSQL,
			want: "postgresql",
		},
		{
			name: "SQLite",
			dt:   SQLite,
			want: "sqlite",
		},
		{
			name: "MySQL",
			dt:   MySQL,
			want: "mysql",
		},
		{
			name: "Empty",
			dt:   "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.dt.String())
		})
	}
}

func TestDatabaseType_IsValid(t *testing.T) {
	tests := []struct {
		name string
		dt   DatabaseType
		want bool
	}{
		{
			name: "PostgreSQL valid",
			dt:   PostgreSQL,
			want: true,
		},
		{
			name: "SQLite valid",
			dt:   SQLite,
			want: true,
		},
		{
			name: "MySQL valid",
			dt:   MySQL,
			want: true,
		},
		{
			name: "Empty invalid",
			dt:   "",
			want: false,
		},
		{
			name: "Unknown invalid",
			dt:   "unknown",
			want: false,
		},
		{
			name: "Invalid case",
			dt:   "POSTGRESQL",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.dt.IsValid())
		})
	}
}

func TestParseDatabaseType(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  DatabaseType
	}{
		{
			name:  "postgresql lowercase",
			input: "postgresql",
			want:  PostgreSQL,
		},
		{
			name:  "postgresql uppercase",
			input: "POSTGRESQL",
			want:  PostgreSQL,
		},
		{
			name:  "postgres alias",
			input: "postgres",
			want:  PostgreSQL,
		},
		{
			name:  "pgsql alias",
			input: "pgsql",
			want:  PostgreSQL,
		},
		{
			name:  "sqlite lowercase",
			input: "sqlite",
			want:  SQLite,
		},
		{
			name:  "sqlite3 alias",
			input: "sqlite3",
			want:  SQLite,
		},
		{
			name:  "mysql lowercase",
			input: "mysql",
			want:  MySQL,
		},
		{
			name:  "mariadb alias",
			input: "mariadb",
			want:  MySQL,
		},
		{
			name:  "unknown database",
			input: "oracle",
			want:  "",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ParseDatabaseType(tt.input))
		})
	}
}

func TestDatabaseType_GetJSONColumnType(t *testing.T) {
	tests := []struct {
		name string
		dt   DatabaseType
		want string
	}{
		{
			name: "PostgreSQL returns JSONB",
			dt:   PostgreSQL,
			want: "JSONB",
		},
		{
			name: "SQLite returns TEXT",
			dt:   SQLite,
			want: "TEXT",
		},
		{
			name: "MySQL returns JSON",
			dt:   MySQL,
			want: "JSON",
		},
		{
			name: "Unknown returns TEXT",
			dt:   "unknown",
			want: "TEXT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.dt.GetJSONColumnType())
		})
	}
}

func TestDatabaseType_GetBlobColumnType(t *testing.T) {
	tests := []struct {
		name string
		dt   DatabaseType
		want string
	}{
		{
			name: "PostgreSQL returns BYTEA",
			dt:   PostgreSQL,
			want: "BYTEA",
		},
		{
			name: "SQLite returns BLOB",
			dt:   SQLite,
			want: "BLOB",
		},
		{
			name: "MySQL returns BLOB",
			dt:   MySQL,
			want: "BLOB",
		},
		{
			name: "Unknown returns BLOB",
			dt:   "unknown",
			want: "BLOB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.dt.GetBlobColumnType())
		})
	}
}

func TestDatabaseType_GetIntegerColumnType(t *testing.T) {
	tests := []struct {
		name string
		dt   DatabaseType
		want string
	}{
		{
			name: "PostgreSQL returns INTEGER",
			dt:   PostgreSQL,
			want: "INTEGER",
		},
		{
			name: "SQLite returns INTEGER",
			dt:   SQLite,
			want: "INTEGER",
		},
		{
			name: "MySQL returns INT",
			dt:   MySQL,
			want: "INT",
		},
		{
			name: "Unknown returns INTEGER",
			dt:   "unknown",
			want: "INTEGER",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.dt.GetIntegerColumnType())
		})
	}
}

func TestDatabaseType_GetTimestampColumnType(t *testing.T) {
	tests := []struct {
		name string
		dt   DatabaseType
		want string
	}{
		{
			name: "PostgreSQL returns TIMESTAMP WITH TIME ZONE",
			dt:   PostgreSQL,
			want: "TIMESTAMP WITH TIME ZONE",
		},
		{
			name: "SQLite returns DATETIME",
			dt:   SQLite,
			want: "DATETIME",
		},
		{
			name: "MySQL returns TIMESTAMP",
			dt:   MySQL,
			want: "TIMESTAMP",
		},
		{
			name: "Unknown returns TIMESTAMP",
			dt:   "unknown",
			want: "TIMESTAMP",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.dt.GetTimestampColumnType())
		})
	}
}

func TestDatabaseType_GetJSONExtractFunction(t *testing.T) {
	tests := []struct {
		name   string
		dt     DatabaseType
		column string
		path   string
		want   string
	}{
		{
			name:   "PostgreSQL JSON extract",
			dt:     PostgreSQL,
			column: "metadata",
			path:   "kek_alias",
			want:   "metadata->>'kek_alias'",
		},
		{
			name:   "SQLite JSON extract",
			dt:     SQLite,
			column: "metadata",
			path:   "serializer_type",
			want:   "json_extract(metadata, '$.serializer_type')",
		},
		{
			name:   "MySQL JSON extract",
			dt:     MySQL,
			column: "data",
			path:   "user_id",
			want:   "data->>'$.user_id'",
		},
		{
			name:   "Unknown database defaults to PostgreSQL syntax",
			dt:     "unknown",
			column: "info",
			path:   "version",
			want:   "info->>'version'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.dt.GetJSONExtractFunction(tt.column, tt.path)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestDatabaseType_GetMetadataIndexSQL(t *testing.T) {
	tests := []struct {
		name      string
		dt        DatabaseType
		tableName string
		indexName string
		path      string
		want      string
	}{
		{
			name:      "PostgreSQL metadata index",
			dt:        PostgreSQL,
			tableName: "users_encx",
			indexName: "idx_kek_alias",
			path:      "kek_alias",
			want:      "CREATE INDEX idx_kek_alias ON users_encx USING GIN ((metadata->>'kek_alias'))",
		},
		{
			name:      "SQLite metadata index",
			dt:        SQLite,
			tableName: "users_encx",
			indexName: "idx_serializer",
			path:      "serializer_type",
			want:      "CREATE INDEX idx_serializer ON users_encx (json_extract(metadata, '$.serializer_type'))",
		},
		{
			name:      "MySQL metadata index",
			dt:        MySQL,
			tableName: "data_table",
			indexName: "idx_version",
			path:      "version",
			want:      "CREATE INDEX idx_version ON data_table ((metadata->>'$.version'))",
		},
		{
			name:      "Unknown database returns unsupported message",
			dt:        "unknown",
			tableName: "test_table",
			indexName: "idx_test",
			path:      "test_path",
			want:      "-- Unsupported database type for metadata index",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.dt.GetMetadataIndexSQL(tt.tableName, tt.indexName, tt.path)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestDatabaseType_SupportsJSONIndexing(t *testing.T) {
	tests := []struct {
		name string
		dt   DatabaseType
		want bool
	}{
		{
			name: "PostgreSQL supports JSON indexing",
			dt:   PostgreSQL,
			want: true,
		},
		{
			name: "SQLite supports JSON indexing",
			dt:   SQLite,
			want: true,
		},
		{
			name: "MySQL supports JSON indexing",
			dt:   MySQL,
			want: true,
		},
		{
			name: "Unknown database doesn't support JSON indexing",
			dt:   "unknown",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.dt.SupportsJSONIndexing())
		})
	}
}

func TestDatabaseType_RequiresJSONValidation(t *testing.T) {
	tests := []struct {
		name string
		dt   DatabaseType
		want bool
	}{
		{
			name: "PostgreSQL doesn't require JSON validation",
			dt:   PostgreSQL,
			want: false,
		},
		{
			name: "SQLite requires JSON validation",
			dt:   SQLite,
			want: true,
		},
		{
			name: "MySQL doesn't require JSON validation",
			dt:   MySQL,
			want: false,
		},
		{
			name: "Unknown database requires JSON validation",
			dt:   "unknown",
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.dt.RequiresJSONValidation())
		})
	}
}

func TestDatabaseType_GetJSONValidationConstraint(t *testing.T) {
	tests := []struct {
		name       string
		dt         DatabaseType
		columnName string
		want       string
	}{
		{
			name:       "PostgreSQL returns empty constraint",
			dt:         PostgreSQL,
			columnName: "metadata",
			want:       "",
		},
		{
			name:       "SQLite returns JSON validation constraint",
			dt:         SQLite,
			columnName: "metadata",
			want:       "CHECK (json_valid(metadata))",
		},
		{
			name:       "MySQL returns empty constraint",
			dt:         MySQL,
			columnName: "data",
			want:       "",
		},
		{
			name:       "Unknown database returns empty constraint",
			dt:         "unknown",
			columnName: "info",
			want:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.dt.GetJSONValidationConstraint(tt.columnName)
			assert.Equal(t, tt.want, result)
		})
	}
}