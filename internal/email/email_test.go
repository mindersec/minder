// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package email

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestIsValidField(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input          string
		expectedErrMsg string
	}{
		// Test case 1: Valid plain text
		{"Just plain text", ""},

		// Test case 2: String with HTML tags
		{"<b>Bold Text</b>", "string <b>Bold Text</b> contains HTML injection"},

		// Test case 3: String with HTML entity
		{"This is a test &amp; example.", "string This is a test &amp; example. contains HTML injection"},

		// Test case 4: String with multiple HTML entities
		{"This &amp; that &lt; should &gt; work.", "string This &amp; that &lt; should &gt; work. contains HTML injection"},

		// Test case 5: Valid URL (no HTML or JavaScript injection)
		{"https://example.com", ""},

		// Test case 6: Mixed content with HTML and JS
		{"Hello <b>World</b> onload=alert('test');", "string Hello <b>World</b> onload=alert('test'); contains HTML injection"},

		// Test case 7: HTML-style comment
		{"<!-- This is a comment -->", "string <!-- This is a comment --> contains HTML injection"},

		// Test case 8: ensure allowed length is less than 200 characters
		{strings.Repeat("a", MaxFieldLength+1), fmt.Sprintf("field value %s is more than %d characters", strings.Repeat("a", MaxFieldLength+1), MaxFieldLength)},
	}

	for _, tt := range tests {
		tt := tt // capture range variable for parallel execution
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			err := isValidField(tt.input)
			if err != nil && err.Error() != tt.expectedErrMsg {
				t.Errorf("isValidField(%q) got error message: %v, expected message: %v", tt.input, err.Error(), tt.expectedErrMsg)
			}
		})
	}
}

func TestValidateDataSourceTemplate(t *testing.T) {
	t.Parallel()

	projectId := uuid.New()

	tests := []struct {
		input          bodyData
		expectedErrMsg string
	}{
		// Test case 1: All fields are valid plain text
		{
			bodyData{
				AdminName:        "John Doe",
				OrganizationId:   projectId,
				OrganizationName: "Acme Corp",
				InvitationCode:   "ABC123",
				InvitationURL:    "https://invitation.com",
				RecipientEmail:   "john.doe@example.com",
				MinderURL:        "https://minder.com",
				TermsURL:         "https://terms.com",
				PrivacyURL:       "https://privacy.com",
				SignInURL:        "https://signin.com",
				RoleName:         "Administrator",
				RoleVerb:         "manage",
			},
			"",
		},

		// Test case 2: AdminName contains HTML tags
		{
			bodyData{
				AdminName:        "John <b>Doe</b>",
				OrganizationId:   projectId,
				OrganizationName: "Acme Corp",
				InvitationCode:   "ABC123",
				InvitationURL:    "https://invitation.com",
				RecipientEmail:   "john.doe@example.com",
				MinderURL:        "https://minder.com",
				TermsURL:         "https://terms.com",
				PrivacyURL:       "https://privacy.com",
				SignInURL:        "https://signin.com",
				RoleName:         "Administrator",
				RoleVerb:         "manage",
			},
			"validation failed: field AdminName failed validation John <b>Doe</b>",
		},

		// Test case 3: OrganizationName contains HTML content
		{
			bodyData{
				AdminName:        "John Doe",
				OrganizationId:   projectId,
				OrganizationName: "<script>alert('Hack');</script>",
				InvitationCode:   "ABC123",
				InvitationURL:    "https://invitation.com",
				RecipientEmail:   "john.doe@example.com",
				MinderURL:        "https://minder.com",
				TermsURL:         "https://terms.com",
				PrivacyURL:       "https://privacy.com",
				SignInURL:        "https://signin.com",
				RoleName:         "Administrator",
				RoleVerb:         "manage",
			},
			"validation failed: field OrganizationName failed validation <script>alert('Hack');</script>",
		},

		// Test case 4: AdminName contains JavaScript code
		{
			bodyData{
				AdminName:        "onload=alert('test')",
				OrganizationId:   projectId,
				OrganizationName: "Acme Corp",
				InvitationCode:   "ABC123",
				InvitationURL:    "https://invitation.com",
				RecipientEmail:   "john.doe@example.com",
				MinderURL:        "https://minder.com",
				TermsURL:         "https://terms.com",
				PrivacyURL:       "https://privacy.com",
				SignInURL:        "https://signin.com",
				RoleName:         "Administrator",
				RoleVerb:         "manage",
			},
			"validation failed: field AdminName failed validation onload=alert('test')",
		},

		// Test case 5: All fields contain valid plain text with some URLs
		{
			bodyData{
				AdminName:        "Plain Text User",
				OrganizationId:   projectId,
				OrganizationName: "No HTML Corp",
				InvitationCode:   "ABC123",
				InvitationURL:    "https://example.com",
				RecipientEmail:   "user@example.com",
				MinderURL:        "https://example.com/minder",
				TermsURL:         "https://example.com/terms",
				PrivacyURL:       "https://example.com/privacy",
				SignInURL:        "https://example.com/signin",
				RoleName:         "User",
				RoleVerb:         "view",
			},
			"",
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable for parallel execution
		t.Run(tt.input.AdminName, func(t *testing.T) {
			t.Parallel()
			err := tt.input.Validate()
			if err != nil && !strings.Contains(err.Error(), tt.expectedErrMsg) {
				t.Errorf("validateDataSourceTemplate(%+v) got error message: %v, expected message: %v", tt.input, err.Error(), tt.expectedErrMsg)
			}
		})
	}
}
