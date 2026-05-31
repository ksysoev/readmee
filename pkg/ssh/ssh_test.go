package ssh

import (
	context "context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	testPrivateKey, err := rsa.GenerateKey(nil, 2048)
	require.NoError(t, err, "Failed to generate test private key")

	encodedKey := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(testPrivateKey),
	})

	tests := []struct {
		svc     Service
		cfg     Config
		name    string
		wantErr bool
	}{
		{
			name: "Valid configuration",
			cfg: Config{
				Listen:     ":0",
				PrivateKey: string(encodedKey),
			},
			svc:     NewMockService(t),
			wantErr: false,
		},
		{
			name: "Missing private key",
			cfg: Config{
				Listen: ":0",
			},
			svc:     NewMockService(t),
			wantErr: true,
		},
		{
			name: "Invalid private key format",
			cfg: Config{
				Listen:     ":0",
				PrivateKey: "invalid-key",
			},
			svc:     NewMockService(t),
			wantErr: true,
		},
		{
			name: "Empty listen address",
			cfg: Config{
				PrivateKey: string(encodedKey),
			},
			svc:     NewMockService(t),
			wantErr: true,
		},
		{
			name: "Nil service",
			cfg: Config{
				Listen:     ":0",
				PrivateKey: string(encodedKey),
			},
			svc:     nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := New(tt.cfg, tt.svc)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("New() failed: %v", gotErr)
				}

				return
			}

			if tt.wantErr {
				t.Fatal("New() succeeded unexpectedly")
			}

			assert.NotNil(t, got, "New() should return a non-nil SSH instance")
		})
	}
}

func TestSSH_Run(t *testing.T) {
	testPrivateKey, err := rsa.GenerateKey(nil, 2048)
	require.NoError(t, err, "Failed to generate test private key")

	encodedKey := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(testPrivateKey),
	})

	tests := []struct {
		svc     Service
		cfg     Config
		name    string
		wantErr bool
	}{
		{
			name: "Valid configuration",
			cfg: Config{
				Listen:     ":0",
				PrivateKey: string(encodedKey),
			},
			svc:     NewMockService(t),
			wantErr: false,
		},
		{
			name: "Invalid listen address",
			cfg: Config{
				Listen:     "invalid-address",
				PrivateKey: string(encodedKey),
			},

			svc:     NewMockService(t),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := New(tt.cfg, tt.svc)
			require.NoError(t, err, "Failed to create SSH instance")

			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			gotErr := s.Run(ctx)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("Run() failed: %v", gotErr)
				}

				return
			}

			if tt.wantErr {
				t.Fatal("Run() succeeded unexpectedly")
			}
		})
	}
}
