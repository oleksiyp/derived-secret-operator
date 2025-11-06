/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package crypto

import (
	"strings"
	"testing"
)

func TestDeriveSecret(t *testing.T) {
	tests := []struct {
		name            string
		masterPassword  string
		context         string
		length          int
		wantErr         bool
		checkDeterminism bool
	}{
		{
			name:            "derive password length",
			masterPassword:  "test-master-password",
			context:         "namespace/name/key1",
			length:          26,
			wantErr:         false,
			checkDeterminism: true,
		},
		{
			name:            "derive encryption key length",
			masterPassword:  "test-master-password",
			context:         "namespace/name/key2",
			length:          48,
			wantErr:         false,
			checkDeterminism: true,
		},
		{
			name:           "length too short",
			masterPassword: "test-master-password",
			context:        "namespace/name/key3",
			length:         21,
			wantErr:        true,
		},
		{
			name:           "length too long",
			masterPassword: "test-master-password",
			context:        "namespace/name/key4",
			length:         257,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DeriveSecret(tt.masterPassword, tt.context, tt.length)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeriveSecret() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Check length
				if len(got) != tt.length {
					t.Errorf("DeriveSecret() returned length = %d, want %d", len(got), tt.length)
				}

				// Check that all characters are from base62 alphabet
				for _, c := range got {
					if !strings.ContainsRune(base62Alphabet, c) {
						t.Errorf("DeriveSecret() returned character not in base62 alphabet: %c", c)
					}
				}

				// Check determinism - same input should produce same output
				if tt.checkDeterminism {
					got2, err2 := DeriveSecret(tt.masterPassword, tt.context, tt.length)
					if err2 != nil {
						t.Errorf("DeriveSecret() second call error = %v", err2)
					}
					if got != got2 {
						t.Errorf("DeriveSecret() is not deterministic: first=%s, second=%s", got, got2)
					}
				}

				// Check that different contexts produce different secrets
				got3, err3 := DeriveSecret(tt.masterPassword, tt.context+"different", tt.length)
				if err3 != nil {
					t.Errorf("DeriveSecret() third call error = %v", err3)
				}
				if got == got3 {
					t.Errorf("DeriveSecret() produced same secret for different contexts")
				}
			}
		})
	}
}

func TestGenerateRandomPassword(t *testing.T) {
	tests := []struct {
		name    string
		length  int
		wantErr bool
	}{
		{
			name:    "generate password",
			length:  26,
			wantErr: false,
		},
		{
			name:    "generate encryption key",
			length:  48,
			wantErr: false,
		},
		{
			name:    "length too short",
			length:  21,
			wantErr: true,
		},
		{
			name:    "length too long",
			length:  257,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GenerateRandomPassword(tt.length)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateRandomPassword() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Check length
				if len(got) != tt.length {
					t.Errorf("GenerateRandomPassword() returned length = %d, want %d", len(got), tt.length)
				}

				// Check that all characters are from base62 alphabet
				for _, c := range got {
					if !strings.ContainsRune(base62Alphabet, c) {
						t.Errorf("GenerateRandomPassword() returned character not in base62 alphabet: %c", c)
					}
				}

				// Check randomness - two calls should produce different results
				got2, err2 := GenerateRandomPassword(tt.length)
				if err2 != nil {
					t.Errorf("GenerateRandomPassword() second call error = %v", err2)
				}
				if got == got2 {
					t.Errorf("GenerateRandomPassword() produced same password twice (very unlikely if random)")
				}
			}
		})
	}
}

func TestGetSecretLength(t *testing.T) {
	tests := []struct {
		name         string
		secretType   string
		customLength int
		want         int
	}{
		{
			name:       "password type",
			secretType: "password",
			want:       26,
		},
		{
			name:       "encryption-key type",
			secretType: "encryption-key",
			want:       48,
		},
		{
			name:         "custom type with length",
			secretType:   "custom",
			customLength: 64,
			want:         64,
		},
		{
			name:       "custom type without length",
			secretType: "custom",
			want:       26,
		},
		{
			name:       "unknown type",
			secretType: "unknown",
			want:       26,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetSecretLength(tt.secretType, tt.customLength); got != tt.want {
				t.Errorf("GetSecretLength() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildContext(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		objName   string
		key       string
		want      string
	}{
		{
			name:      "build context",
			namespace: "test-namespace",
			objName:   "test-secret",
			key:       "test-key",
			want:      "test-namespace/test-secret/test-key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BuildContext(tt.namespace, tt.objName, tt.key); got != tt.want {
				t.Errorf("BuildContext() = %v, want %v", got, tt.want)
			}
		})
	}
}
