package hashicorp

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/hashicorp/vault/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockVaultServer creates a mock Vault server for testing
func mockVaultServer(t *testing.T) *httptest.Server {
	mux := http.NewServeMux()

	// Mock AppRole login
	mux.HandleFunc("/v1/auth/approle/login", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"auth": {
				"client_token": "test-token-12345"
			}
		}`))
	})

	// Mock transit key creation
	mux.HandleFunc("/v1/transit/keys/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			w.WriteHeader(http.StatusNoContent)
		}
	})

	// Mock transit key rotation
	mux.HandleFunc("/v1/transit/keys/test-key/rotate", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	// Mock transit encrypt
	mux.HandleFunc("/v1/transit/encrypt/test-key", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"data": {
				"ciphertext": "vault:v1:mockencrypteddata"
			}
		}`))
	})

	// Mock transit decrypt
	mux.HandleFunc("/v1/transit/decrypt/test-key", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		plaintext := base64.StdEncoding.EncodeToString([]byte("decrypted-data"))
		w.Write([]byte(`{
			"data": {
				"plaintext": "` + plaintext + `"
			}
		}`))
	})

	// Mock secret read (KV v2)
	mux.HandleFunc("/v1/secret/data/pepper", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"data": {
				"data": {
					"value": "test-pepper-value"
				}
			}
		}`))
	})

	// Mock secret write (KV v2)
	mux.HandleFunc("/v1/secret/data/test-secret", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	return httptest.NewServer(mux)
}

func TestNew_Success(t *testing.T) {
	server := mockVaultServer(t)
	defer server.Close()

	os.Setenv("VAULT_ADDR", server.URL)
	defer os.Unsetenv("VAULT_ADDR")

	vs, err := New()
	require.NoError(t, err)
	assert.NotNil(t, vs)
	assert.NotNil(t, vs.client)
}

func TestNew_WithNamespace(t *testing.T) {
	server := mockVaultServer(t)
	defer server.Close()

	os.Setenv("VAULT_ADDR", server.URL)
	os.Setenv("VAULT_NAMESPACE", "admin/test")
	defer os.Unsetenv("VAULT_ADDR")
	defer os.Unsetenv("VAULT_NAMESPACE")

	vs, err := New()
	require.NoError(t, err)
	assert.NotNil(t, vs)
}

func TestNew_WithAppRole(t *testing.T) {
	server := mockVaultServer(t)
	defer server.Close()

	os.Setenv("VAULT_ADDR", server.URL)
	os.Setenv("VAULT_ROLE_ID", "test-role-id")
	os.Setenv("VAULT_SECRET_ID", "test-secret-id")
	defer os.Unsetenv("VAULT_ADDR")
	defer os.Unsetenv("VAULT_ROLE_ID")
	defer os.Unsetenv("VAULT_SECRET_ID")

	vs, err := New()
	require.NoError(t, err)
	assert.NotNil(t, vs)
	assert.Equal(t, "test-token-12345", vs.client.Token())
}

func TestGetKey(t *testing.T) {
	vs := &VaultService{client: &api.Client{}}
	ctx := context.Background()

	key, err := vs.GetKey(ctx, "test-key")
	assert.Error(t, err)
	assert.Nil(t, key)
	assert.Contains(t, err.Error(), "not supported")
}

func TestGetKeyID(t *testing.T) {
	vs := &VaultService{client: &api.Client{}}
	ctx := context.Background()

	keyID, err := vs.GetKeyID(ctx, "test-alias")
	assert.NoError(t, err)
	assert.Equal(t, "test-alias", keyID)
}

func TestCreateKey(t *testing.T) {
	server := mockVaultServer(t)
	defer server.Close()

	config := api.DefaultConfig()
	config.Address = server.URL
	client, err := api.NewClient(config)
	require.NoError(t, err)

	vs := &VaultService{client: client}
	ctx := context.Background()

	keyID, err := vs.CreateKey(ctx, "new-test-key")
	assert.NoError(t, err)
	assert.Equal(t, "new-test-key", keyID)
}

func TestRotateKey(t *testing.T) {
	server := mockVaultServer(t)
	defer server.Close()

	config := api.DefaultConfig()
	config.Address = server.URL
	client, err := api.NewClient(config)
	require.NoError(t, err)

	vs := &VaultService{client: client}
	ctx := context.Background()

	err = vs.RotateKey(ctx, "test-key")
	assert.NoError(t, err)
}

func TestGetSecret(t *testing.T) {
	server := mockVaultServer(t)
	defer server.Close()

	config := api.DefaultConfig()
	config.Address = server.URL
	client, err := api.NewClient(config)
	require.NoError(t, err)

	vs := &VaultService{client: client}
	ctx := context.Background()

	secret, err := vs.GetSecret(ctx, "secret/data/pepper")
	assert.NoError(t, err)
	assert.Equal(t, []byte("test-pepper-value"), secret)
}

func TestSetSecret(t *testing.T) {
	server := mockVaultServer(t)
	defer server.Close()

	config := api.DefaultConfig()
	config.Address = server.URL
	client, err := api.NewClient(config)
	require.NoError(t, err)

	vs := &VaultService{client: client}
	ctx := context.Background()

	err = vs.SetSecret(ctx, "secret/data/test-secret", []byte("test-value"))
	assert.NoError(t, err)
}

func TestEncryptDEK(t *testing.T) {
	server := mockVaultServer(t)
	defer server.Close()

	config := api.DefaultConfig()
	config.Address = server.URL
	client, err := api.NewClient(config)
	require.NoError(t, err)

	vs := &VaultService{client: client}
	ctx := context.Background()

	ciphertext, err := vs.EncryptDEK(ctx, "test-key", []byte("plaintext-dek"))
	assert.NoError(t, err)
	assert.Equal(t, []byte("vault:v1:mockencrypteddata"), ciphertext)
}

func TestDecryptDEK(t *testing.T) {
	server := mockVaultServer(t)
	defer server.Close()

	config := api.DefaultConfig()
	config.Address = server.URL
	client, err := api.NewClient(config)
	require.NoError(t, err)

	vs := &VaultService{client: client}
	ctx := context.Background()

	plaintext, err := vs.DecryptDEK(ctx, "test-key", []byte("vault:v1:mockencrypteddata"))
	assert.NoError(t, err)
	assert.Equal(t, []byte("decrypted-data"), plaintext)
}

func TestEncryptDEKWithVersion(t *testing.T) {
	server := mockVaultServer(t)
	defer server.Close()

	config := api.DefaultConfig()
	config.Address = server.URL
	client, err := api.NewClient(config)
	require.NoError(t, err)

	vs := &VaultService{client: client}
	ctx := context.Background()

	// Note: EncryptDEKWithVersion calls EncryptDEK with empty keyID
	// This is a limitation in the current implementation
	_, err = vs.EncryptDEKWithVersion(ctx, []byte("plaintext-dek"), 1)
	assert.Error(t, err) // Expecting error due to empty keyID
}

func TestDecryptDEKWithVersion(t *testing.T) {
	server := mockVaultServer(t)
	defer server.Close()

	config := api.DefaultConfig()
	config.Address = server.URL
	client, err := api.NewClient(config)
	require.NoError(t, err)

	vs := &VaultService{client: client}
	ctx := context.Background()

	// Note: DecryptDEKWithVersion calls DecryptDEK with empty keyID
	// This is a limitation in the current implementation
	_, err = vs.DecryptDEKWithVersion(ctx, []byte("vault:v1:mockencrypteddata"), 1)
	assert.Error(t, err) // Expecting error due to empty keyID
}
