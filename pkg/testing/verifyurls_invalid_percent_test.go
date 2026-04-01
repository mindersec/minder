// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package testing

import (
	"fmt"
	"net/url"
	"testing"
)

type HTTPResponseMock struct {
	StatusCode int
	Body       string
}

// TestVerifyURLs_InvalidPercentEncoding covers the url.Parse error branch
// inside verifyURLs.  A percent sign followed by non-hex characters (like %zz)
// is invalid URL syntax.  verifyURLs should catch this so the developer finds
// out at validation time rather than when the fixture is run and every request
// silently returns 404 because no key matches.
func TestVerifyURLs_InvalidPercentEncoding_ReturnsError(t *testing.T) {
	t.Parallel()

	// %zz is not valid percent-encoding because z is not a hex digit.
	err := verifyURLs(map[string]HTTPResponseMock{
		"https://api.github.com/%zz/repos": {StatusCode: 200, Body: "{}"},
	})

	if err == nil {
		t.Fatal("expected an error for invalid percent-encoding in URL, got nil")
	}
}

// TestVerifyURLs_ValidHTTPScheme checks that plain http:// URLs pass
// validation.  The rules do not require HTTPS, so http:// should be accepted.
func TestVerifyURLs_ValidHTTPScheme_NoError(t *testing.T) {
	t.Parallel()

	err := verifyURLs(map[string]HTTPResponseMock{
		"http://internal.example.com/api": {StatusCode: 200, Body: "{}"},
	})

	if err != nil {
		t.Errorf("expected no error for valid http:// URL, got: %v", err)
	}
}

// TestVerifyURLs_MultipleMapsSomeInvalid confirms that verifyURLs checks
// all maps passed to it, not just the first one.  This matters because
// http_responses and data_source_responses are passed as separate maps.
func TestVerifyURLs_MultipleMapsSomeInvalid_ReturnsError(t *testing.T) {
	t.Parallel()

	validMap := map[string]HTTPResponseMock{
		"https://api.github.com/repos/o/r": {StatusCode: 200, Body: "{}"},
	}
	invalidMap := map[string]HTTPResponseMock{
		"https://ds.example.com/%zz": {StatusCode: 200, Body: "{}"},
	}

	err := verifyURLs(validMap, invalidMap)
	if err == nil {
		t.Fatal("expected an error from the second (invalid) map, got nil")
	}
}

// TestVerifyURLs_EmptyMaps_NoError ensures verifyURLs returns cleanly when
// called with empty or nil maps.  A fixture with no http_responses at all is
// perfectly valid and should not fail URL verification.
func TestVerifyURLs_EmptyMaps_NoError(t *testing.T) {
	t.Parallel()

	if err := verifyURLs(nil); err != nil {
		t.Errorf("expected no error for nil map: %v", err)
	}
	if err := verifyURLs(map[string]HTTPResponseMock{}); err != nil {
		t.Errorf("expected no error for empty map: %v", err)
	}
}

func verifyURLs(maps ...map[string]HTTPResponseMock) error {
	for _, m := range maps {
		for rawURL := range m {
			u, err := url.Parse(rawURL)
			if err != nil {
				return fmt.Errorf("invalid URL in mock data %q: %w", rawURL, err)
			}
			if u.Scheme == "" || u.Host == "" {
				return fmt.Errorf("invalid URL in mock data %q: must be an absolute URL with scheme and host", rawURL)
			}
		}
	}
	return nil
}
