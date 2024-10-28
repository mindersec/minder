// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package email

import "testing"

func TestIsValidField(t *testing.T) {
	t.Parallel() // make this sub-test run in parallel
	tests := []struct {
		input          string
		expectedErr    bool
		expectedErrMsg string
	}{
		// Test case 1: Empty string
		{"", true, "string is empty"},

		// Test case 2: Valid plain text
		{"Just plain text", false, ""},

		// Test case 3: String with HTML tags
		{"<b>Bold Text</b>", true, "string <b>Bold Text</b> contains HTML tags, entities, or comments"},

		// Test case 4: String with HTML entity
		{"This is a test &amp; example.", true, "string This is a test &amp; example. contains HTML tags, entities, or comments"},

		// Test case 5: String with multiple HTML entities
		{"This &amp; that &lt; should &gt; work.", true, "string This &amp; that &lt; should &gt; work. contains HTML tags, entities, or comments"},

		// Test case 6: String with special characters, but no HTML
		{"Special chars! #$%^&*", false, ""},

		// Test case 7: Numeric HTML entity
		{"This is a test &#1234;", true, "string This is a test &#1234; contains HTML tags, entities, or comments"},

		// Test case 8: Valid URL (no HTML tags or entities)
		{"https://example.com", false, ""},

		// Test case 9: Script tag injection
		{"<script>alert('test');</script>", true, "string <script>alert('test');</script> contains HTML tags, entities, or comments"},

		// Test case 10: Mixed content with HTML tag and entity
		{"Hello <b>World</b> &amp; Universe.", true, "string Hello <b>World</b> &amp; Universe. contains HTML tags, entities, or comments"},

		// Test case 11: Plain text with ampersand not forming an entity
		{"AT&T is a company.", false, ""},

		// Test case 12: Plain text with angle brackets but no tags
		{"Angle brackets < and > in text.", false, ""},

		// Test case 13: HTML-style comment
		{"<!-- This is a comment -->", true, "string <!-- This is a comment --> contains HTML tags, entities, or comments"},
	}

	for _, tt := range tests {
		tt := tt // capture range variable to avoid issues with parallel execution
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			err := isValidField(tt.input)
			if (err != nil) != tt.expectedErr {
				t.Errorf("isValidField(%q) got error: %v, expected error: %v", tt.input, err, tt.expectedErr)
			}
			if err != nil && err.Error() != tt.expectedErrMsg {
				t.Errorf("isValidField(%q) got error message: %v, expected message: %v", tt.input, err.Error(), tt.expectedErrMsg)
			}
		})
	}
}

func TestValidateDataSourceTemplate(t *testing.T) {
	t.Parallel() // make this sub-test run in parallel

	tests := []struct {
		input          bodyData
		expectedErr    bool
		expectedErrMsg string
	}{
		// Test case 1: All fields are valid plain text
		{
			bodyData{
				AdminName:        "John Doe",
				OrganizationName: "Acme Corp",
				InvitationURL:    "https://invitation.com",
				RecipientEmail:   "john.doe@example.com",
				MinderURL:        "https://minder.com",
				TermsURL:         "https://terms.com",
				PrivacyURL:       "https://privacy.com",
				SignInURL:        "https://signin.com",
				RoleName:         "Administrator",
				RoleVerb:         "manage",
			},
			false, "",
		},

		// Test case 2: One field contains HTML tag
		{
			bodyData{
				AdminName:        "John <b>Doe</b>",
				OrganizationName: "Acme Corp",
				InvitationURL:    "https://invitation.com",
				RecipientEmail:   "john.doe@example.com",
				MinderURL:        "https://minder.com",
				TermsURL:         "https://terms.com",
				PrivacyURL:       "https://privacy.com",
				SignInURL:        "https://signin.com",
				RoleName:         "Administrator",
				RoleVerb:         "manage",
			},
			true, "field AdminName is empty or contains HTML injection - John <b>Doe</b>",
		},

		// Test case 3: One field contains HTML entity
		{
			bodyData{
				AdminName:        "John Doe",
				OrganizationName: "Acme Corp",
				InvitationURL:    "https://invitation.com",
				RecipientEmail:   "john.doe@example.com",
				MinderURL:        "https://minder.com",
				TermsURL:         "https://terms.com",
				PrivacyURL:       "https://privacy.com",
				SignInURL:        "https://signin.com",
				RoleName:         "Administrator",
				RoleVerb:         "approve &amp; manage",
			},
			true, "field RoleVerb is empty or contains HTML injection - approve &amp; manage",
		},

		// Test case 4: Multiple fields contain HTML content
		{
			bodyData{
				AdminName:        "John Doe",
				OrganizationName: "<script>alert('Hack');</script>",
				InvitationURL:    "<a href='https://phishing.com'>Click here</a>",
				RecipientEmail:   "john.doe@example.com",
				MinderURL:        "https://minder.com",
				TermsURL:         "https://terms.com",
				PrivacyURL:       "https://privacy.com",
				SignInURL:        "https://signin.com",
				RoleName:         "Administrator",
				RoleVerb:         "manage",
			},
			true, "field OrganizationName is empty or contains HTML injection - <script>alert('Hack');</script>",
		},

		// Test case 5: All fields contain valid plain text with some URLs
		{
			bodyData{
				AdminName:        "Plain Text User",
				OrganizationName: "No HTML Corp",
				InvitationURL:    "https://example.com",
				RecipientEmail:   "user@example.com",
				MinderURL:        "https://example.com/minder",
				TermsURL:         "https://example.com/terms",
				PrivacyURL:       "https://example.com/privacy",
				SignInURL:        "https://example.com/signin",
				RoleName:         "User",
				RoleVerb:         "view",
			},
			false, "",
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable to avoid issues with parallel execution
		t.Run(tt.input.AdminName, func(t *testing.T) {
			t.Parallel()

			err := validateDataSourceTemplate(&tt.input)
			if (err != nil) != tt.expectedErr {
				t.Errorf("validateDataSourceTemplate(%+v) got error: %v, expected error: %v", tt.input, err, tt.expectedErr)
			}
			if err != nil && err.Error() != tt.expectedErrMsg {
				t.Errorf("validateDataSourceTemplate(%+v) got error message: %v, expected message: %v", tt.input, err.Error(), tt.expectedErrMsg)
			}
		})
	}
}
