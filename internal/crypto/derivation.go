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
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"math/big"

	"golang.org/x/crypto/argon2"
)

const (
	// Argon2id parameters
	argon2Time    = 4
	argon2Memory  = 64 * 1024 // 64 MB
	argon2Threads = 1
	argon2KeyLen  = 32

	// Base62 alphabet for password generation
	base62Alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
)

// DeriveSecret derives a secret using Argon2id with the given master password and context.
// The context is used as the salt for the KDF.
// The derived secret is encoded as Base62 (A-Za-z0-9) and truncated/padded to the specified length.
func DeriveSecret(masterPassword, context string, length int) (string, error) {
	if length < 22 || length > 256 {
		return "", fmt.Errorf("length must be between 22 and 256, got %d", length)
	}

	// Use context as salt
	salt := []byte(context)

	// Derive key using Argon2id
	derivedKey := argon2.IDKey(
		[]byte(masterPassword),
		salt,
		argon2Time,
		argon2Memory,
		argon2Threads,
		argon2KeyLen,
	)

	// Convert to base62
	// Use the derived key as a seed to generate base62 characters deterministically
	result := make([]byte, length)
	for i := 0; i < length; i++ {
		// Use multiple bytes from the derived key for each character to increase entropy
		index := int(derivedKey[i%len(derivedKey)]) % len(base62Alphabet)

		// XOR with subsequent bytes to increase randomness
		if i+1 < len(derivedKey) {
			index = (index + int(derivedKey[(i+1)%len(derivedKey)])) % len(base62Alphabet)
		}

		result[i] = base62Alphabet[index]

		// Re-derive if we've used all bytes from the key
		if i > 0 && i%argon2KeyLen == 0 {
			// Use previous result as additional context for more bytes
			newSalt := append(salt, derivedKey...)
			derivedKey = argon2.IDKey(
				[]byte(masterPassword),
				newSalt,
				argon2Time,
				argon2Memory,
				argon2Threads,
				argon2KeyLen,
			)
		}
	}

	return string(result), nil
}

// GenerateRandomPassword generates a cryptographically secure random password
// of the specified length using Base62 alphabet.
func GenerateRandomPassword(length int) (string, error) {
	if length < 22 || length > 256 {
		return "", fmt.Errorf("length must be between 22 and 256, got %d", length)
	}

	result := make([]byte, length)
	alphabetLen := big.NewInt(int64(len(base62Alphabet)))

	for i := 0; i < length; i++ {
		num, err := rand.Int(rand.Reader, alphabetLen)
		if err != nil {
			return "", fmt.Errorf("failed to generate random number: %w", err)
		}
		result[i] = base62Alphabet[num.Int64()]
	}

	return string(result), nil
}

// GenerateRandomBytes generates cryptographically secure random bytes.
func GenerateRandomBytes(length int) ([]byte, error) {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return bytes, nil
}

// Base64Encode encodes bytes to base64.
func Base64Encode(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// GetSecretLength returns the length for a given secret type.
func GetSecretLength(secretType string, customLength int) int {
	switch secretType {
	case "password":
		return 26
	case "encryption-key":
		return 48
	case "custom":
		if customLength > 0 {
			return customLength
		}
		return 26 // default to password length if not specified
	default:
		return 26
	}
}

// BuildContext builds the context string for derivation from namespace, name, and key.
func BuildContext(namespace, name, key string) string {
	return fmt.Sprintf("%s/%s/%s", namespace, name, key)
}
