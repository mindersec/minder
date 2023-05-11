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

func TestDecryptRow(t *testing.T) {
	encrypted, err := EncryptRow("w00t", "topsecret")
	assert.Nil(t, err)
	decrypted, err := DecryptRow("w00t", encrypted)
	assert.Nil(t, err)
	assert.Equal(t, "topsecret", decrypted)
}
