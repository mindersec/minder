// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package pull_request_comment

import "fmt"

func formatTitle(displayName string) string {
	return fmt.Sprintf("Rule '%s' Alert", displayName)
}

// Formats the comment for a single alert as markdown
func alert(title, body string) string {
	return fmt.Sprintf("%s\n\n%s", title2(title), body)
}

func paragraph(text string) string {
	return fmt.Sprintf("%s\n\n", text)
}

func title1(title string) string {
	return fmt.Sprintf("# %s", title)
}

func title2(title string) string {
	return fmt.Sprintf("## %s", title)
}

func separator() string {
	return "---"
}
