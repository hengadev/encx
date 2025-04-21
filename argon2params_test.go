package encx

import (
	// "fmt"
	"testing"

	"github.com/hengadev/errsx"
	"github.com/stretchr/testify/assert"
)

func TestArgon2Params_Validate(t *testing.T) {
	tests := []struct {
		name     string
		params   Argon2Params
		wantErr  bool
		errCount int
		errKeys  []string // expected error fields
	}{
		{
			name: "valid parameters",
			params: Argon2Params{
				Memory:      19456,
				Iterations:  2,
				Parallelism: 1,
				SaltLength:  16,
				KeyLength:   32,
			},
			wantErr:  false,
			errCount: 0,
		},
		{
			name: "all parameters too low",
			params: Argon2Params{
				Memory:      1000,
				Iterations:  1,
				Parallelism: 0,
				SaltLength:  8,
				KeyLength:   16,
			},
			wantErr:  true,
			errCount: 5,
			errKeys:  []string{"memory", "iterations", "parallelism", "saltLength", "keyLength"},
		},
		{
			name: "memory too low",
			params: Argon2Params{
				Memory:      1000,
				Iterations:  2,
				Parallelism: 1,
				SaltLength:  16,
				KeyLength:   32,
			},
			wantErr:  true,
			errCount: 1,
			errKeys:  []string{"memory"},
		},
		{
			name: "iterations too low",
			params: Argon2Params{
				Memory:      19456,
				Iterations:  1,
				Parallelism: 1,
				SaltLength:  16,
				KeyLength:   32,
			},
			wantErr:  true,
			errCount: 1,
			errKeys:  []string{"iterations"},
		},
		{
			name: "parallelism too low",
			params: Argon2Params{
				Memory:      19456,
				Iterations:  2,
				Parallelism: 0,
				SaltLength:  16,
				KeyLength:   32,
			},
			wantErr:  true,
			errCount: 1,
			errKeys:  []string{"parallelism"},
		},
		{
			name: "salt length too low",
			params: Argon2Params{
				Memory:      19456,
				Iterations:  2,
				Parallelism: 1,
				SaltLength:  8,
				KeyLength:   32,
			},
			wantErr:  true,
			errCount: 1,
			errKeys:  []string{"saltLength"},
		},
		{
			name: "key length too low",
			params: Argon2Params{
				Memory:      19456,
				Iterations:  2,
				Parallelism: 1,
				SaltLength:  16,
				KeyLength:   16,
			},
			wantErr:  true,
			errCount: 1,
			errKeys:  []string{"keyLength"},
		},
		{
			name: "multiple errors",
			params: Argon2Params{
				Memory:      1000,
				Iterations:  1,
				Parallelism: 1,
				SaltLength:  8,
				KeyLength:   32,
			},
			wantErr:  true,
			errCount: 3,
			errKeys:  []string{"memory", "iterations", "saltLength"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.params.Validate()

			if tt.wantErr {
				assert.Error(t, err)

				var errs errsx.Map
				errs, ok := err.(errsx.Map)
				if !ok {
					t.Fatal("expected error to be of type errsx.Map")
				}
				assert.Equal(t, tt.errCount, len(errs))

				for _, key := range tt.errKeys {
					if _, ok := errs[key]; !ok {
						t.Errorf("expected key '%s' in errsx.Map", key)
					}
				}
			} else {
				assert.NoError(t, err, nil)
			}
		})
	}
}
