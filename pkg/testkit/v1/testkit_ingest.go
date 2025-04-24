// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"context"
	"errors"

	"github.com/go-git/go-billy/v5/osfs"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/mindersec/minder/internal/engine/ingester/git"
	"github.com/mindersec/minder/pkg/engine/v1/interfaces"
)

var (
	// ErrIngestUnimplemented is returned when the ingester is not implemented
	ErrIngestUnimplemented = errors.New("ingester not implemented")
)

// Ensure that TestKit implements the Ingester interface
var _ interfaces.Ingester = &TestKit{}

// Ingest is a stub implementation of the ingester
func (tk *TestKit) Ingest(
	ctx context.Context, ent protoreflect.ProtoMessage, params map[string]any,
) (*interfaces.Ingested, error) {
	if tk.ingestType == git.GitRuleDataIngestType {
		return tk.fakeGit(ctx, ent, params)
	}
	return nil, ErrIngestUnimplemented
}

// ShouldOverrideIngest returns true if the ingester should override the ingest
func (tk *TestKit) ShouldOverrideIngest() bool {
	return tk.ingestType == git.GitRuleDataIngestType
}

// GetType returns the type of the ingester
func (*TestKit) GetType() string {
	return "testkit"
}

// GetConfig returns the config for the ingester
func (*TestKit) GetConfig() protoreflect.ProtoMessage {
	return nil
}

func (tk *TestKit) fakeGit(
	_ context.Context, _ protoreflect.ProtoMessage, _ map[string]any,
) (*interfaces.Ingested, error) {
	fs := osfs.New(tk.gitDir)
	return &interfaces.Ingested{
		Fs: fs,
	}, nil
}
