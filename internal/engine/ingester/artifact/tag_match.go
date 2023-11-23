// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// Package rule provides the CLI subcommand for managing rules

package artifact

import (
	"fmt"
	"regexp"

	"k8s.io/apimachinery/pkg/util/sets"
)

func buildTagMatcher(tags []string, tagRegex string) (tagMatcher, error) {
	if len(tags) > 0 && tagRegex != "" {
		return nil, fmt.Errorf("cannot specify both tags and tag_regex")
	}

	// tags specified, build a list matcher
	if len(tags) > 0 {
		stags := sets.New(tags...)
		if stags.HasAny("") {
			return nil, fmt.Errorf("cannot specify empty tag")
		}
		return &tagListMatcher{tags: tags}, nil
	}

	// no tags specified, but a regex was, compile it
	if tagRegex != "" {
		re, err := regexp.Compile(tagRegex)
		if err != nil {
			return nil, fmt.Errorf("error compiling tag regex: %w", err)
		}
		return &tagRegexMatcher{re: re}, nil
	}

	// no tags specified, match all
	return &tagAllMatcher{}, nil
}

type tagMatcher interface {
	MatchTag(tags ...string) bool
}

type tagRegexMatcher struct {
	re *regexp.Regexp
}

func (m *tagRegexMatcher) MatchTag(tags ...string) bool {
	for _, tag := range tags {
		if m.re.MatchString(tag) {
			return true
		}
	}

	return false
}

type tagListMatcher struct {
	tags []string
}

func (m *tagListMatcher) MatchTag(tags ...string) bool {
	haveTags := sets.New(tags...)
	return haveTags.HasAll(m.tags...)
}

type tagAllMatcher struct{}

func (*tagAllMatcher) MatchTag(_ ...string) bool {
	return true
}
