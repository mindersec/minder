// SPDX-FileCopyrightText: Copyright 2025 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package merged

import (
	"errors"
	"testing"

	"github.com/lestrrat-go/jwx/v2/jwt/openid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	minder_jwt "github.com/mindersec/minder/internal/auth/jwt"
	mockjwt "github.com/mindersec/minder/internal/auth/jwt/mock"
)

func TestValidator_ParseAndValidate(t *testing.T) {
	t.Parallel()

	testToken := "some-token"
	mockToken, _ := openid.NewBuilder().Subject("subject1").Build()
	errTest := errors.New("test error")

	testCases := []struct {
		name          string
		setupMock     func(v1, v2 *mockjwt.MockValidator)
		expectedToken openid.Token
		expectedError string
	}{
		{
			name: "First validator succeeds",
			setupMock: func(v1, _ *mockjwt.MockValidator) {
				v1.EXPECT().ParseAndValidate(testToken).Return(mockToken, nil)
			},
			expectedToken: mockToken,
		},
		{
			name: "First validator fails, second succeeds",
			setupMock: func(v1, v2 *mockjwt.MockValidator) {
				v1.EXPECT().ParseAndValidate(testToken).Return(nil, errTest)
				v2.EXPECT().ParseAndValidate(testToken).Return(mockToken, nil)
			},
			expectedToken: mockToken,
		},
		{
			name: "Both validators fail",
			setupMock: func(v1, v2 *mockjwt.MockValidator) {
				v1.EXPECT().ParseAndValidate(testToken).Return(nil, errTest)
				v2.EXPECT().ParseAndValidate(testToken).Return(nil, errTest)
			},
			expectedError: "no validator could parse and validate the token",
		},
		{
			name: "No validators",
			setupMock: func(_, _ *mockjwt.MockValidator) {
			},
			expectedError: "no validator could parse and validate the token",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			v1 := mockjwt.NewMockValidator(ctrl)
			v2 := mockjwt.NewMockValidator(ctrl)
			tc.setupMock(v1, v2)

			var validators []minder_jwt.Validator
			if tc.name != "No validators" {
				validators = []minder_jwt.Validator{v1, v2}
			}

			mergedValidator := Validator{
				Validators: validators,
			}

			token, err := mergedValidator.ParseAndValidate(testToken)

			if tc.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
				assert.Nil(t, token)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedToken, token)
			}
		})
	}
}
