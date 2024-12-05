// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package jwt

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"go.uber.org/mock/gomock"

	mockjwt "github.com/mindersec/minder/internal/auth/jwt/mock"
)

var (
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
)

func init() {
	var err error
	privateKey, err = rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return
	}
	publicKey = &privateKey.PublicKey
}

func FuzzParseAndValidate(f *testing.F) {
	f.Fuzz(func(t *testing.T, pubKeyId, privKeyId, token string) {

		privateJwk, _ := jwk.FromRaw(privateKey)
		err := privateJwk.Set(jwk.KeyIDKey, privKeyId)
		if err != nil {
			return
		}

		publicJwk, _ := jwk.FromRaw(publicKey)
		err = publicJwk.Set(jwk.KeyIDKey, pubKeyId)
		if err != nil {
			return
		}
		err = publicJwk.Set(jwk.AlgorithmKey, jwa.RS256)
		if err != nil {
			return
		}

		jwks := jwk.NewSet()
		err = jwks.AddKey(publicJwk)
		if err != nil {
			return
		}

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockKeyFetcher := mockjwt.NewMockKeySetFetcher(ctrl)
		mockKeyFetcher.EXPECT().GetKeySet().Return(jwks, nil)

		jwtValidator := JwkSetJwtValidator{jwksFetcher: mockKeyFetcher}
		//nolint:gosec // Ignore return values
		jwtValidator.ParseAndValidate(token)
	})
}
