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

package cli

import (
	"github.com/charmbracelet/glamour"
)

// RenderMarkdown renders the given string as markdown.
func RenderMarkdown(payload string) (string, error) {
	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
	)
	// We can't really fail here, but in case we just return the
	// payload as-is.
	if err != nil {
		return "", err
	}
	rendered, err := r.Render(payload)
	// We don't want to fail rendering when input is not valid
	// markdown, and we just output it as-is instead.
	if err != nil {
		return "", err
	}
	return rendered, nil
}

// MaybeRenderMarkdown tries to render the given string as
// markdown. In case of error it silently ignores the error and
// returns the string as-is.
func MaybeRenderMarkdown(payload string) string {
	rendered, err := RenderMarkdown(payload)
	if err != nil {
		return payload
	}
	return rendered
}
