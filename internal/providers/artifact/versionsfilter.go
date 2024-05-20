// Copyright 2024 Stacklok, Inc.
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

// Package artifact provides functions and utilities for artifact providers
package artifact

import (
	"fmt"
	"regexp"
	"time"

	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/stacklok/minder/internal/verifier"
	provifv1 "github.com/stacklok/minder/pkg/providers/v1"
)

// BuildFilter builds a container image filter based on the tags and tag regex as well as the creation time
func BuildFilter(tags []string, tagRegex string) (*filter, error) {
	if len(tags) > 0 && tagRegex != "" {
		return nil, fmt.Errorf("cannot specify both tags and tag_regex")
	}

	// tags specified, build a list matcher
	if len(tags) > 0 {
		stags := sets.New(tags...)
		if stags.HasAny("") {
			return nil, fmt.Errorf("cannot specify empty tag")
		}
		return &filter{
			tagMatcher:      &tagListMatcher{tags: tags},
			retentionPeriod: provifv1.ArtifactTypeContainerRetentionPeriod,
		}, nil
	}

	// no tags specified, but a regex was, compile it
	if tagRegex != "" {
		re, err := regexp.Compile(tagRegex)
		if err != nil {
			return nil, fmt.Errorf("error compiling tag regex: %w", err)
		}
		return &filter{
			tagMatcher:      &tagRegexMatcher{re: re},
			retentionPeriod: provifv1.ArtifactTypeContainerRetentionPeriod,
		}, nil
	}

	// no tags specified, match all
	return &filter{
		tagMatcher:      &tagAllMatcher{},
		retentionPeriod: provifv1.ArtifactTypeContainerRetentionPeriod,
	}, nil
}

type filter struct {
	tagMatcher
	retentionPeriod time.Time
}

// IsSkippable determines if an artifact should be skipped
func (f *filter) IsSkippable(createdAt time.Time, tags []string) error {
	// if the artifact is older than the retention period, skip it
	if createdAt.Before(f.retentionPeriod) {
		return fmt.Errorf("artifact is older than retention period - %s",
			f.retentionPeriod)
	}

	if len(tags) == 0 {
		// if the artifact has no tags, skip it
		return fmt.Errorf("artifact has no tags")
	}

	// Check if there is an empty tag using contains
	if sets.New(tags...).Has("") {
		return fmt.Errorf("artifact has empty tag")
	}

	// if the artifact has a .sig tag it's a signature, skip it
	if verifier.GetSignatureTag(tags) != "" {
		return fmt.Errorf("artifact is a signature")
	}
	// if the artifact tags don't match the tag matcher, skip it
	if !f.MatchTag(tags...) {
		return fmt.Errorf("artifact tags does not match")
	}
	return nil
}

// tagMatcher is an interface for matching tags
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
