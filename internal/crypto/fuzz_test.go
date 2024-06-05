//
// Copyright 2024 Stacklok, Inc.
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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stacklok/minder/internal/config/server"
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
