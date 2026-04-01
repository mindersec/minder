// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package providers

import (
	"testing"

	ghclient "github.com/mindersec/minder/internal/providers/github/clients"
)

func TestGetProviderClassDefinition_Github(t *testing.T) {
	t.Parallel()
	def, err := GetProviderClassDefinition(ghclient.Github)
	if err != nil {
		t.Fatalf("GetProviderClassDefinition(%q) error = %v", ghclient.Github, err)
	}
	if len(def.Traits) == 0 {
		t.Error("Github definition has no traits")
	}
}

func TestGetProviderClassDefinition_GithubApp(t *testing.T) {
	t.Parallel()
	def, err := GetProviderClassDefinition(ghclient.GithubApp)
	if err != nil {
		t.Fatalf("GetProviderClassDefinition(%q) error = %v", ghclient.GithubApp, err)
	}
	if len(def.Traits) == 0 {
		t.Error("GithubApp definition has no traits")
	}
}

func TestGetProviderClassDefinition_Unknown_ReturnsError(t *testing.T) {
	t.Parallel()
	_, err := GetProviderClassDefinition("nonexistent-provider")
	if err == nil {
		t.Error("GetProviderClassDefinition(unknown) expected error")
	}
}

func TestGetProviderClassDefinition_EmptyString_ReturnsError(t *testing.T) {
	t.Parallel()
	_, err := GetProviderClassDefinition("")
	if err == nil {
		t.Error("GetProviderClassDefinition('') expected error")
	}
}

func TestErrProviderNotFoundBy_Name(t *testing.T) {
	t.Parallel()
	e := ErrProviderNotFoundBy{Name: "my-provider"}
	msg := e.Error()
	if msg == "" {
		t.Error("ErrProviderNotFoundBy.Error() returned empty string")
	}
}

func TestErrProviderNotFoundBy_Trait(t *testing.T) {
	t.Parallel()
	e := ErrProviderNotFoundBy{Trait: "git"}
	msg := e.Error()
	if msg == "" {
		t.Error("ErrProviderNotFoundBy.Error() returned empty string")
	}
}

func TestErrProviderNotFoundBy_Both(t *testing.T) {
	t.Parallel()
	e := ErrProviderNotFoundBy{Name: "my-provider", Trait: "git"}
	msg := e.Error()
	if msg == "" {
		t.Error("ErrProviderNotFoundBy.Error() returned empty string")
	}
}
