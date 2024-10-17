// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package crypto

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mindersec/minder/internal/config/server"
)

func FuzzEncryptDecrypt(f *testing.F) {
	rawKey := []byte("2hcGLimy2i7LAknby2AFqYx87CaaCAtjxDiorRxYq8Q=")

	file, err := os.CreateTemp("", "-fuzz-key-test")
	if err != nil {
		f.Fatal(err)
	}
	fileName := file.Name()

	defer os.Remove(fileName)
	err = os.WriteFile(fileName, rawKey, 0600)
	if err != nil {
		f.Fatal(err)
	}

	fuzzConfig := &server.Config{
		Crypto: server.CryptoConfig{
			KeyStore: server.KeyStoreConfig{
				Type: "local",
				Local: server.LocalKeyStoreConfig{
					KeyDir: os.TempDir(),
				},
			},
			Default: server.DefaultCrypto{
				KeyID: filepath.Base(fileName),
			},
		},
	}

	fuzzEngine, err := NewEngineFromConfig(fuzzConfig)
	if err != nil {
		panic(err)
	}

	f.Fuzz(func(_ *testing.T, data string) {
		encrypted, err := fuzzEngine.EncryptString(data)
		if err != nil {
			return
		}
		decrypted, err := fuzzEngine.DecryptString(encrypted)
		if err != nil {
			panic(err)
		}
		if !strings.EqualFold(data, decrypted) {
			panic(fmt.Sprintf("data '%s' and decrypted '%s' should be equal but are not", data, decrypted))
		}
	})
}
