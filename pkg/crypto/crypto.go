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

package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"

	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"

	"io"
	"time"

	"golang.org/x/crypto/argon2"
)

func GetCert(envelope []byte) ([]byte, error) {
	env := &Envelope{}
	if err := json.Unmarshal(envelope, env); err != nil {
		return nil, err
	}
	return []byte(env.Signatures[0].Cert), nil
}

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

func VerifySignature(pubKey *ecdsa.PublicKey, payload []byte, sig []byte) (bool, error) {
	hash := sha256.Sum256(payload)
	verified := ecdsa.VerifyASN1(pubKey, hash[:], sig)
	return verified, nil
}

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
		return false, fmt.Errorf("failed to verify certificate: %w", err)
	}

	return true, nil
}

func EncryptRow(key, data string) ([]byte, error) {
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

// Function to decrypt data using AES
func DecryptRow(key string, ciphertext []byte) (string, error) {
	block, err := aes.NewCipher(deriveKey(key))
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// The IV needs to be extracted from the ciphertext.
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(ciphertext, ciphertext)

	return string(ciphertext), nil
}

// Function to derive a key from a passphrase using Argon2
func deriveKey(passphrase string) []byte {
	salt := []byte("somesalt") // In a real application, you should use a unique salt for each key and save it with the encrypted data.
	return argon2.IDKey([]byte(passphrase), salt, 1, 64*1024, 4, 32)
}
