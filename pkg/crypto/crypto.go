//
// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package crypto provides cryptographic functions
package crypto

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"encoding/pem"
	"errors"
	"strings"
	"time"

	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/subtle"
	"fmt"
	"io"

	"github.com/spf13/viper"
	"golang.org/x/crypto/argon2"
)

type params struct {
	memory      uint32
	iterations  uint32
	parallelism uint
	saltLength  uint32
	keyLength   uint32
}

func getParams() *params {
	// set default values when not set
	viper.SetDefault("salt.memory", 64*1024)
	viper.SetDefault("salt.iterations", 3)
	viper.SetDefault("salt.parallelism", 2)
	viper.SetDefault("salt.salt_length", 16)
	viper.SetDefault("salt.key_length", 32)

	return &params{
		memory:      viper.GetUint32("salt.memory"),
		iterations:  viper.GetUint32("salt.iterations"),
		parallelism: viper.GetUint("salt.parallelism"),
		saltLength:  viper.GetUint32("salt.salt_length"),
		keyLength:   viper.GetUint32("salt.key_length"),
	}
}

var (
	// ErrInvalidHash is returned when the encoded hash is not in the correct format
	ErrInvalidHash = errors.New("the encoded hash is not in the correct format")
	// ErrIncompatibleVersion is returned when the encoded hash was created with a different version of argon2
	ErrIncompatibleVersion = errors.New("incompatible version of argon2")
)

// GetCert gets a certificate from an envelope
func GetCert(envelope []byte) ([]byte, error) {
	env := &Envelope{}
	if err := json.Unmarshal(envelope, env); err != nil {
		return nil, err
	}
	return []byte(env.Signatures[0].Cert), nil
}

// GetPubKeyFromCert gets a public key from a certificate
func GetPubKeyFromCert(certIn []byte) (*ecdsa.PublicKey, error) {
	block, _ := pem.Decode(certIn)
	if block == nil {
		return nil, errors.New("failed to parse certificate PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, errors.New("failed to parse certificate: " + err.Error())
	}

	pubKey := cert.PublicKey.(*ecdsa.PublicKey)
	return pubKey, nil
}

// VerifySignature verifies a signature
func VerifySignature(pubKey *ecdsa.PublicKey, payload []byte, sig []byte) (bool, error) {
	hash := sha256.Sum256(payload)
	verified := ecdsa.VerifyASN1(pubKey, hash[:], sig)
	return verified, nil
}

// VerifyCertChain verifies a certificate chain
func VerifyCertChain(certIn []byte, roots *x509.CertPool) (bool, error) {
	block, _ := pem.Decode(certIn)
	if block == nil {
		return false, errors.New("failed to parse certificate PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return false, errors.New("failed to parse certificate: " + err.Error())
	}

	// combine the roots with the intermediates to get a full chain
	roots.AppendCertsFromPEM([]byte(certIn))

	opts := x509.VerifyOptions{
		Roots: roots,
		// skip expiry check
		CurrentTime: cert.NotBefore.Add(1 * time.Minute),
		KeyUsages: []x509.ExtKeyUsage{
			x509.ExtKeyUsageCodeSigning,
		},
	}

	if _, err := cert.Verify(opts); err != nil {
		return false, err
	}

	return true, nil
}

// EncryptBytes encrypts a row of data using AES-CFB.
func EncryptBytes(key string, data []byte) ([]byte, error) {
	block, err := aes.NewCipher(deriveKey(key))
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// The IV needs to be unique, but not secure. Therefore it's common to include it at the beginning of the ciphertext.
	ciphertext := make([]byte, aes.BlockSize+len(data))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, fmt.Errorf("failed to read random bytes: %w", err)
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], []byte(data))

	return ciphertext, nil
}

// DecryptBytes decrypts a row of data
func DecryptBytes(key string, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(deriveKey(key))
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// The IV needs to be extracted from the ciphertext.
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(ciphertext, ciphertext)

	return ciphertext, nil
}

// Function to derive a key from a passphrase using Argon2
func deriveKey(passphrase string) []byte {
	// In a real application, you should use a unique salt for
	// each key and save it with the encrypted data.
	salt := []byte("somesalt")
	return argon2.IDKey([]byte(passphrase), salt, 1, 64*1024, 4, 32)
}

// GeneratePasswordHash generates a hash of a password using Argon2id.
func GeneratePasswordHash(password string) (encodedHash string, err error) {

	p := getParams()

	salt, err := generateRandomBytes(p.saltLength)
	if err != nil {
		return "", err
	}

	hash := argon2.IDKey([]byte(password), salt, p.iterations, p.memory, uint8(p.parallelism), p.keyLength)

	// Base64 encode the salt and hashed password.
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	// Return a string using the standard encoded hash representation.
	encodedHash = fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s", argon2.Version,
		p.memory, p.iterations, p.parallelism, b64Salt, b64Hash)

	return encodedHash, nil
}

func generateRandomBytes(n uint32) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}

	return b, nil
}

// VerifyPasswordHash compares a password with a hash and returns true if
func VerifyPasswordHash(password, encodedHash string) (match bool, err error) {
	// Extract the parameters, salt and derived key from the encoded password
	// hash.
	p, salt, hash, err := decodeHash(encodedHash)
	if err != nil {
		return false, err
	}

	// Derive the key from the other password using the same parameters.
	otherHash := argon2.IDKey([]byte(password), salt, p.iterations, p.memory, uint8(p.parallelism), p.keyLength)

	// Check that the contents of the hashed passwords are identical. Note
	// that we are using the subtle.ConstantTimeCompare() function for this
	// to help prevent timing attacks.
	if subtle.ConstantTimeCompare(hash, otherHash) == 1 {
		return true, nil
	}
	return false, nil
}

func decodeHash(encodedHash string) (p *params, salt, hash []byte, err error) {
	vals := strings.Split(encodedHash, "$")
	if len(vals) != 6 {
		return nil, nil, nil, ErrInvalidHash
	}

	var version int
	_, err = fmt.Sscanf(vals[2], "v=%d", &version)
	if err != nil {
		return nil, nil, nil, err
	}
	if version != argon2.Version {
		return nil, nil, nil, ErrIncompatibleVersion
	}

	p = &params{}
	_, err = fmt.Sscanf(vals[3], "m=%d,t=%d,p=%d", &p.memory, &p.iterations, &p.parallelism)
	if err != nil {
		return nil, nil, nil, err
	}

	salt, err = base64.RawStdEncoding.Strict().DecodeString(vals[4])
	if err != nil {
		return nil, nil, nil, err
	}
	p.saltLength = uint32(len(salt))

	hash, err = base64.RawStdEncoding.Strict().DecodeString(vals[5])
	if err != nil {
		return nil, nil, nil, err
	}
	p.keyLength = uint32(len(hash))

	return p, salt, hash, nil
}

// GenerateNonce generates a nonce for the OAuth2 flow. The nonce is a base64 encoded
func GenerateNonce() (string, error) {
	randomBytes := make([]byte, 32)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", err
	}

	nonceBytes := make([]byte, 8)
	timestamp := time.Now().Unix()
	binary.BigEndian.PutUint64(nonceBytes, uint64(timestamp))

	nonceBytes = append(nonceBytes, randomBytes...)
	nonce := base64.RawURLEncoding.EncodeToString(nonceBytes)
	return nonce, nil
}

// IsNonceValid checks if a nonce is valid. A nonce is valid if it is a base64 encoded string
func IsNonceValid(nonce string) (bool, error) {
	nonceBytes, err := base64.RawURLEncoding.DecodeString(nonce)
	if err != nil {
		return false, err
	}

	if len(nonceBytes) < 8 {
		return false, nil
	}

	storedTimestamp := int64(binary.BigEndian.Uint64(nonceBytes[:8]))
	currentTimestamp := time.Now().Unix()
	timeDiff := currentTimestamp - storedTimestamp

	if timeDiff > viper.GetInt64("auth.nonce_period") { // 5 minutes = 300 seconds
		return false, nil
	}

	return true, nil
}
