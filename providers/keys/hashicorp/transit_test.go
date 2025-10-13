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

	vs, err := NewTransitService()
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

	vs, err := NewTransitService()
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

	vs, err := NewTransitService()
	require.NoError(t, err)
	assert.NotNil(t, vs)
	assert.Equal(t, "test-token-12345", vs.client.Token())
}

func TestGetKeyID(t *testing.T) {
	vs := &TransitService{client: &api.Client{}}
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

	vs := &TransitService{client: client}
	ctx := context.Background()

	keyID, err := vs.CreateKey(ctx, "new-test-key")
	assert.NoError(t, err)
	assert.Equal(t, "new-test-key", keyID)
}

func TestEncryptDEK(t *testing.T) {
	server := mockVaultServer(t)
	defer server.Close()

	config := api.DefaultConfig()
	config.Address = server.URL
	client, err := api.NewClient(config)
	require.NoError(t, err)

	vs := &TransitService{client: client}
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

	vs := &TransitService{client: client}
	ctx := context.Background()

	plaintext, err := vs.DecryptDEK(ctx, "test-key", []byte("vault:v1:mockencrypteddata"))
	assert.NoError(t, err)
	assert.Equal(t, []byte("decrypted-data"), plaintext)
}
