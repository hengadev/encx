package crypto

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockKeyRotationService implements KeyRotationService for testing
type mockKeyRotationService struct {
	encryptDEKFunc func(ctx context.Context, keyID string, plaintext []byte) ([]byte, error)
	decryptDEKFunc func(ctx context.Context, keyID string, ciphertext []byte) ([]byte, error)
	createKeyFunc  func(ctx context.Context, alias string) (string, error)
	getKeyIDFunc   func(ctx context.Context, alias string) (string, error)
}

func (m *mockKeyRotationService) EncryptDEK(ctx context.Context, keyID string, plaintext []byte) ([]byte, error) {
	if m.encryptDEKFunc != nil {
		return m.encryptDEKFunc(ctx, keyID, plaintext)
	}
	return plaintext, nil
}

func (m *mockKeyRotationService) DecryptDEK(ctx context.Context, keyID string, ciphertext []byte) ([]byte, error) {
	if m.decryptDEKFunc != nil {
		return m.decryptDEKFunc(ctx, keyID, ciphertext)
	}
	return ciphertext, nil
}

func (m *mockKeyRotationService) CreateKey(ctx context.Context, alias string) (string, error) {
	if m.createKeyFunc != nil {
		return m.createKeyFunc(ctx, alias)
	}
	return "new-kms-key-id", nil
}

func (m *mockKeyRotationService) GetKeyID(ctx context.Context, alias string) (string, error) {
	if m.getKeyIDFunc != nil {
		return m.getKeyIDFunc(ctx, alias)
	}
	return "existing-kms-key-id", nil
}

// mockObservabilityHook implements ObservabilityHook for testing
type mockObservabilityHook struct {
	processStarts   []string
	processComplete []string
	errors          []string
	keyOperations   []string
}

func (m *mockObservabilityHook) OnProcessStart(ctx context.Context, operation string, metadata map[string]any) {
	m.processStarts = append(m.processStarts, operation)
}

func (m *mockObservabilityHook) OnProcessComplete(ctx context.Context, operation string, duration time.Duration, err error, metadata map[string]any) {
	m.processComplete = append(m.processComplete, operation)
}

func (m *mockObservabilityHook) OnError(ctx context.Context, operation string, err error, metadata map[string]any) {
	m.errors = append(m.errors, operation)
}

func (m *mockObservabilityHook) OnKeyOperation(ctx context.Context, operation string, alias string, version int, metadata map[string]any) {
	m.keyOperations = append(m.keyOperations, operation)
}

// mockVersionManager implements KMSVersionManager for testing
type mockVersionManager struct {
	currentVersion int
	getVersionErr  error
}

func (m *mockVersionManager) GetCurrentKEKVersion(ctx context.Context, alias string) (int, error) {
	if m.getVersionErr != nil {
		return 0, m.getVersionErr
	}
	return m.currentVersion, nil
}

func (m *mockVersionManager) GetKMSKeyIDForVersion(ctx context.Context, alias string, version int) (string, error) {
	return "kms-key-id", nil
}

// setupTestDB creates an in-memory SQLite database with the required schema
func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)

	// Create the kek_versions table
	_, err = db.Exec(`
		CREATE TABLE kek_versions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			alias TEXT NOT NULL,
			version INTEGER NOT NULL,
			kms_key_id TEXT NOT NULL,
			is_deprecated BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(alias, version)
		)
	`)
	require.NoError(t, err)

	return db
}

func TestNewKeyRotationOperations(t *testing.T) {
	kms := &mockKeyRotationService{}
	obs := &mockObservabilityHook{}
	db := setupTestDB(t)
	defer db.Close()

	kr := NewKeyRotationOperations(kms, "test-alias", db, obs)

	assert.NotNil(t, kr)
	assert.Equal(t, "test-alias", kr.kekAlias)
}

func TestRotateKEK_Success(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	defer db.Close()

	// Insert initial KEK version
	_, err := db.Exec(`
		INSERT INTO kek_versions (alias, version, kms_key_id)
		VALUES ('test-alias', 1, 'old-kms-key-id')
	`)
	require.NoError(t, err)

	kms := &mockKeyRotationService{
		createKeyFunc: func(ctx context.Context, alias string) (string, error) {
			return "new-kms-key-id", nil
		},
	}
	obs := &mockObservabilityHook{}
	versionMgr := &mockVersionManager{currentVersion: 1}

	kr := NewKeyRotationOperations(kms, "test-alias", db, obs)
	err = kr.RotateKEK(ctx, versionMgr)

	assert.NoError(t, err)
	assert.Contains(t, obs.processStarts, "RotateKEK")
	assert.Contains(t, obs.processComplete, "RotateKEK")
	assert.Contains(t, obs.keyOperations, "rotate")

	// Verify old version is deprecated
	var isDeprecated bool
	err = db.QueryRow(`
		SELECT is_deprecated FROM kek_versions
		WHERE alias = 'test-alias' AND version = 1
	`).Scan(&isDeprecated)
	require.NoError(t, err)
	assert.True(t, isDeprecated)

	// Verify new version exists
	var kmsKeyID string
	err = db.QueryRow(`
		SELECT kms_key_id FROM kek_versions
		WHERE alias = 'test-alias' AND version = 2
	`).Scan(&kmsKeyID)
	require.NoError(t, err)
	assert.Equal(t, "new-kms-key-id", kmsKeyID)
}

func TestRotateKEK_GetVersionError(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	defer db.Close()

	kms := &mockKeyRotationService{}
	obs := &mockObservabilityHook{}
	versionMgr := &mockVersionManager{
		getVersionErr: errors.New("version fetch failed"),
	}

	kr := NewKeyRotationOperations(kms, "test-alias", db, obs)
	err := kr.RotateKEK(ctx, versionMgr)

	assert.Error(t, err)
	assert.Contains(t, obs.errors, "RotateKEK")
	assert.Contains(t, obs.processComplete, "RotateKEK")
}

func TestRotateKEK_CreateKeyError(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	defer db.Close()

	kms := &mockKeyRotationService{
		createKeyFunc: func(ctx context.Context, alias string) (string, error) {
			return "", errors.New("KMS create key failed")
		},
	}
	obs := &mockObservabilityHook{}
	versionMgr := &mockVersionManager{currentVersion: 1}

	kr := NewKeyRotationOperations(kms, "test-alias", db, obs)
	err := kr.RotateKEK(ctx, versionMgr)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create new KEK version in KMS")
	assert.Contains(t, obs.errors, "RotateKEK")
}

func TestRotateKEK_DeprecateOldVersionError(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	defer db.Close()

	// Don't insert initial version - this will cause UPDATE to fail silently or we close DB
	db.Close() // Force DB error

	kms := &mockKeyRotationService{
		createKeyFunc: func(ctx context.Context, alias string) (string, error) {
			return "new-kms-key-id", nil
		},
	}
	obs := &mockObservabilityHook{}
	versionMgr := &mockVersionManager{currentVersion: 1}

	kr := NewKeyRotationOperations(kms, "test-alias", db, obs)
	err := kr.RotateKEK(ctx, versionMgr)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to deprecate old KEK version")
}

func TestRotateKEK_RecordNewVersionError(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	defer db.Close()

	// Insert initial version
	_, err := db.Exec(`
		INSERT INTO kek_versions (alias, version, kms_key_id)
		VALUES ('test-alias', 1, 'old-kms-key-id')
	`)
	require.NoError(t, err)

	// Insert version 2 to create a conflict
	_, err = db.Exec(`
		INSERT INTO kek_versions (alias, version, kms_key_id)
		VALUES ('test-alias', 2, 'conflict-key-id')
	`)
	require.NoError(t, err)

	kms := &mockKeyRotationService{
		createKeyFunc: func(ctx context.Context, alias string) (string, error) {
			return "new-kms-key-id", nil
		},
	}
	obs := &mockObservabilityHook{}
	versionMgr := &mockVersionManager{currentVersion: 1}

	kr := NewKeyRotationOperations(kms, "test-alias", db, obs)
	err = kr.RotateKEK(ctx, versionMgr)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to record new KEK version in metadata DB")
}

func TestEnsureInitialKEK_CreateNew(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	defer db.Close()

	kms := &mockKeyRotationService{
		getKeyIDFunc: func(ctx context.Context, alias string) (string, error) {
			return "", errors.New("key not found")
		},
		createKeyFunc: func(ctx context.Context, alias string) (string, error) {
			return "initial-kms-key-id", nil
		},
	}
	obs := &mockObservabilityHook{}
	versionMgr := &mockVersionManager{currentVersion: 0}

	kr := NewKeyRotationOperations(kms, "test-alias", db, obs)
	err := kr.EnsureInitialKEK(ctx, versionMgr)

	assert.NoError(t, err)

	// Verify initial version exists
	var kmsKeyID string
	err = db.QueryRow(`
		SELECT kms_key_id FROM kek_versions
		WHERE alias = 'test-alias' AND version = 1
	`).Scan(&kmsKeyID)
	require.NoError(t, err)
	assert.Equal(t, "initial-kms-key-id", kmsKeyID)
}

func TestEnsureInitialKEK_KeyExistsInKMS(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	defer db.Close()

	// Insert existing version
	_, err := db.Exec(`
		INSERT INTO kek_versions (alias, version, kms_key_id)
		VALUES ('test-alias', 1, 'existing-kms-key-id')
	`)
	require.NoError(t, err)

	kms := &mockKeyRotationService{
		getKeyIDFunc: func(ctx context.Context, alias string) (string, error) {
			return "existing-kms-key-id", nil
		},
	}
	obs := &mockObservabilityHook{}
	versionMgr := &mockVersionManager{currentVersion: 1}

	kr := NewKeyRotationOperations(kms, "test-alias", db, obs)
	err = kr.EnsureInitialKEK(ctx, versionMgr)

	assert.NoError(t, err)

	// Verify no duplicate version was created
	var count int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM kek_versions
		WHERE alias = 'test-alias'
	`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestEnsureInitialKEK_KeyExistsInKMSButNotDB(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	defer db.Close()

	kms := &mockKeyRotationService{
		getKeyIDFunc: func(ctx context.Context, alias string) (string, error) {
			return "existing-kms-key-id", nil
		},
	}
	obs := &mockObservabilityHook{}
	versionMgr := &mockVersionManager{currentVersion: 0}

	kr := NewKeyRotationOperations(kms, "test-alias", db, obs)
	err := kr.EnsureInitialKEK(ctx, versionMgr)

	assert.NoError(t, err)

	// Verify version was recorded
	var kmsKeyID string
	err = db.QueryRow(`
		SELECT kms_key_id FROM kek_versions
		WHERE alias = 'test-alias' AND version = 1
	`).Scan(&kmsKeyID)
	require.NoError(t, err)
	assert.Equal(t, "existing-kms-key-id", kmsKeyID)
}

func TestEnsureInitialKEK_CreateKeyError(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	defer db.Close()

	kms := &mockKeyRotationService{
		getKeyIDFunc: func(ctx context.Context, alias string) (string, error) {
			return "", errors.New("key not found")
		},
		createKeyFunc: func(ctx context.Context, alias string) (string, error) {
			return "", errors.New("KMS create failed")
		},
	}
	obs := &mockObservabilityHook{}
	versionMgr := &mockVersionManager{currentVersion: 0}

	kr := NewKeyRotationOperations(kms, "test-alias", db, obs)
	err := kr.EnsureInitialKEK(ctx, versionMgr)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create initial KEK in KMS")
}

func TestEnsureInitialKEK_RecordInitialError(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	defer db.Close()

	// Close DB to force error
	db.Close()

	kms := &mockKeyRotationService{
		getKeyIDFunc: func(ctx context.Context, alias string) (string, error) {
			return "", errors.New("key not found")
		},
		createKeyFunc: func(ctx context.Context, alias string) (string, error) {
			return "initial-kms-key-id", nil
		},
	}
	obs := &mockObservabilityHook{}
	versionMgr := &mockVersionManager{currentVersion: 0}

	kr := NewKeyRotationOperations(kms, "test-alias", db, obs)
	err := kr.EnsureInitialKEK(ctx, versionMgr)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to record initial KEK in metadata DB")
}

func TestEnsureInitialKEK_GetVersionError(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	defer db.Close()

	kms := &mockKeyRotationService{
		getKeyIDFunc: func(ctx context.Context, alias string) (string, error) {
			return "existing-kms-key-id", nil
		},
	}
	obs := &mockObservabilityHook{}
	versionMgr := &mockVersionManager{
		currentVersion: 0,
		getVersionErr:  errors.New("version fetch failed"),
	}

	kr := NewKeyRotationOperations(kms, "test-alias", db, obs)
	err := kr.EnsureInitialKEK(ctx, versionMgr)

	assert.Error(t, err)
	assert.Equal(t, "version fetch failed", err.Error())
}
