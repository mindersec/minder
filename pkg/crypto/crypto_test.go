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
	// "crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetCert(t *testing.T) {
	cert, err := GetCert([]byte(provenance))
	assert.Nil(t, err)
	assert.Contains(t, string(cert), "-----BEGIN CERTIFICATE-----")
}

func TestGetPubKeyFromCert(t *testing.T) {
	cert, err := GetCert([]byte(provenance))
	assert.Nil(t, err)
	pubKey, err := GetPubKeyFromCert(cert)
	assert.Nil(t, err)
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(pubKey)
	assert.Nil(t, err)

	pubKeyPem := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubKeyBytes,
	})
	assert.Nil(t, err)
	assert.Contains(t, string(pubKeyPem), "-----BEGIN PUBLIC KEY-----")
}

func TestCertificateChain(t *testing.T) {
	roots := x509.NewCertPool()
	cert, err := GetCert([]byte(provenance))
	assert.Nil(t, err)
	verified, err := VerifyCertChain(cert, roots)
	assert.Nil(t, err)
	assert.True(t, verified)
}

func TestEncryptDecryptBytes(t *testing.T) {
	encrypted, err := EncryptBytes("test", []byte("test"))
	assert.Nil(t, err)
	decrypted, err := DecryptBytes("test", encrypted)
	assert.Nil(t, err)
	assert.Equal(t, "test", string(decrypted))
}

func TestGenerateNonce(t *testing.T) {
	state, err := GenerateNonce()
	if err != nil {
		t.Errorf("Error in generateState: %v", err)
	}

	if len(state) != 54 {
		t.Errorf("Unexpected length of state: %v", len(state))
	}
}

func TestIsNonceValid(t *testing.T) {
	nonce, err := GenerateNonce()
	if err != nil {
		t.Errorf("Error in generateState: %v", err)
	}

	valid, err := IsNonceValid(nonce)
	if err != nil {
		t.Errorf("Error in isNonceValid: %v", err)
	}

	if !valid {
		t.Errorf("Expected nonce to be valid, got invalid")
	}

	invalid := "AAAAAGSDmJ_tKMkuUmeoOBdSQGWXq3BE_Zp7IrUFVUau5HcPa-yvzQ"

	valid, err = IsNonceValid(invalid)
	if err != nil {
		t.Errorf("Error in isNonceValid: %v", err)
	}

	if valid {
		t.Errorf("Expected nonce to be invalid, got valid")
	}
}

func TestGenerateKeyPair(t *testing.T) {
	passphrase := "passphrase"

	privKey, publicKey, err := GenerateKeyPair(passphrase)
	if err != nil {
		t.Errorf("Error in generateKeyPair: %v", err)
	}

	if privKey == nil {
		t.Errorf("Expected private key to be generated, got nil")
	}

	if publicKey == nil {
		t.Errorf("Expected public key to be generated, got nil")
	}

	block, _ := pem.Decode(publicKey)
	if block == nil || block.Type != "PUBLIC KEY" {
		t.Errorf("Failed to decode PEM block containing public key")
		return
	}

	ecdsaPubKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		t.Errorf("Failed to parse DER encoded public key: %v", err)
		return
	}

	ecdsaPubKeyBytes, err := x509.MarshalPKIXPublicKey(ecdsaPubKey)
	if err != nil {
		t.Errorf("Failed to marshal ECDSA public key: %v", err)
		return
	}

	ecdsaPubKeyPem := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: ecdsaPubKeyBytes,
	})

	if string(publicKey) != string(ecdsaPubKeyPem) {
		t.Errorf("Expected public key to match, got %v", string(publicKey))
	}
}
