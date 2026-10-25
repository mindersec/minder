// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package ruletest

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	intoto "github.com/in-toto/attestation/go/v1"
	"github.com/sigstore/sigstore-go/pkg/fulcio/certificate"
	"github.com/sigstore/sigstore-go/pkg/verify"
	"go.starlark.net/starlark"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	tkv1 "github.com/mindersec/minder/pkg/testkit/v1"
)

// ArtifactValue is the Starlark value returned by artifact(). It is the
// expected value type for the artifacts parameter of eval().
type ArtifactValue struct {
	Fixture tkv1.ArtifactFixture
}

// VersionValue is the Starlark value returned by version().
type VersionValue struct {
	Fixture tkv1.ArtifactVersionFixture
}

// VerificationValue is the Starlark value returned by signed(), unsigned()
// and verification_json(). It may hold more than one result, since a
// verification_json() document can contain several.
type VerificationValue struct {
	Fixtures []tkv1.VerificationFixture
}

var (
	_ starlark.Value = (*ArtifactValue)(nil)
	_ starlark.Value = (*VersionValue)(nil)
	_ starlark.Value = (*VerificationValue)(nil)
)

func (a *ArtifactValue) String() string {
	return fmt.Sprintf("artifact(registry=%q, versions=%d)", a.Fixture.Registry, len(a.Fixture.Versions))
}

// Type returns the Starlark type name.
func (*ArtifactValue) Type() string { return "artifact" }

// Freeze makes the value immutable.
func (*ArtifactValue) Freeze() {}

// Truth returns the truth value of the artifact.
func (*ArtifactValue) Truth() starlark.Bool { return starlark.True }

// Hash returns a hash value for the artifact.
func (*ArtifactValue) Hash() (uint32, error) { return 0, errors.New("unhashable type: artifact") }

func (v *VersionValue) String() string {
	return fmt.Sprintf("version(tags=%v, digest=%q)", v.Fixture.Tags, v.Fixture.Digest)
}

// Type returns the Starlark type name.
func (*VersionValue) Type() string { return "version" }

// Freeze makes the value immutable.
func (*VersionValue) Freeze() {}

// Truth returns the truth value of the version.
func (*VersionValue) Truth() starlark.Bool { return starlark.True }

// Hash returns a hash value for the version.
func (*VersionValue) Hash() (uint32, error) { return 0, errors.New("unhashable type: version") }

func (v *VerificationValue) String() string {
	return fmt.Sprintf("verification(results=%d)", len(v.Fixtures))
}

// Type returns the Starlark type name.
func (*VerificationValue) Type() string { return "verification" }

// Freeze makes the value immutable.
func (*VerificationValue) Freeze() {}

// Truth returns the truth value of the verification.
func (*VerificationValue) Truth() starlark.Bool { return starlark.True }

// Hash returns a hash value for the verification.
func (*VerificationValue) Hash() (uint32, error) {
	return 0, errors.New("unhashable type: verification")
}

// builtinArtifact implements artifact(registry="", versions=[]).
func builtinArtifact(
	_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple,
) (starlark.Value, error) {
	var registry string
	var versions *starlark.List
	if err := starlark.UnpackArgs(b.Name(), args, kwargs,
		"registry?", &registry, "versions?", &versions); err != nil {
		return nil, err
	}

	fixture := tkv1.ArtifactFixture{Registry: registry}
	if versions != nil {
		for i := range versions.Len() {
			v, ok := versions.Index(i).(*VersionValue)
			if !ok {
				return nil, fmt.Errorf("%s: versions[%d] must be a version, got %s",
					b.Name(), i, versions.Index(i).Type())
			}
			fixture.Versions = append(fixture.Versions, v.Fixture)
		}
	}
	return &ArtifactValue{Fixture: fixture}, nil
}

// builtinVersion implements version(tags, manifest="", digest="",
// media_type="", created_at="", verification=None). verification may be a
// single verification value or a list of them.
func builtinVersion(
	_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple,
) (starlark.Value, error) {
	var tags *starlark.List
	var manifest, digest, mediaType, createdAt string
	var verification starlark.Value = starlark.None
	if err := starlark.UnpackArgs(b.Name(), args, kwargs,
		"tags", &tags, "manifest?", &manifest, "digest?", &digest, "media_type?", &mediaType,
		"created_at?", &createdAt, "verification?", &verification); err != nil {
		return nil, err
	}

	fixture := tkv1.ArtifactVersionFixture{
		Digest:    digest,
		MediaType: mediaType,
	}
	if manifest != "" {
		fixture.Manifest = []byte(manifest)
	}

	for i := range tags.Len() {
		tag, ok := tags.Index(i).(starlark.String)
		if !ok {
			return nil, fmt.Errorf("%s: tags must be a list of strings", b.Name())
		}
		fixture.Tags = append(fixture.Tags, string(tag))
	}

	if createdAt != "" {
		t, err := time.Parse(time.RFC3339, createdAt)
		if err != nil {
			return nil, fmt.Errorf("%s: invalid created_at: %w", b.Name(), err)
		}
		fixture.CreatedAt = t
	}

	ver, err := verificationFixtures(verification)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", b.Name(), err)
	}
	fixture.Verification = ver

	fixture, err = fixture.Resolved()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", b.Name(), err)
	}
	return &VersionValue{Fixture: fixture}, nil
}

func verificationFixtures(val starlark.Value) ([]tkv1.VerificationFixture, error) {
	switch v := val.(type) {
	case starlark.NoneType:
		return nil, nil
	case *VerificationValue:
		return v.Fixtures, nil
	case *starlark.List:
		var out []tkv1.VerificationFixture
		for i := range v.Len() {
			elem, ok := v.Index(i).(*VerificationValue)
			if !ok {
				return nil, fmt.Errorf("verification[%d] must be a verification, got %s", i, v.Index(i).Type())
			}
			out = append(out, elem.Fixtures...)
		}
		return out, nil
	default:
		return nil, fmt.Errorf("verification must be a verification or a list of them, got %s", val.Type())
	}
}

// builtinUnsigned implements unsigned(): no signature was found.
func builtinUnsigned(
	_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple,
) (starlark.Value, error) {
	if err := starlark.UnpackArgs(b.Name(), args, kwargs); err != nil {
		return nil, err
	}
	return &VerificationValue{Fixtures: []tkv1.VerificationFixture{{}}}, nil
}

// builtinSigned implements signed(verified=True, san="", issuer="",
// source_repository_uri="", source_repository_ref="", runner_environment="",
// predicate_type="", predicate=None). The certificate fields are only
// allowed on a verified signature, since the verifier only reports them
// after a successful verification.
func builtinSigned(
	_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple,
) (starlark.Value, error) {
	verified := true
	var san, issuer, repoURI, repoRef, runnerEnv, predicateType string
	var predicate *starlark.Dict
	if err := starlark.UnpackArgs(b.Name(), args, kwargs,
		"verified?", &verified, "san?", &san, "issuer?", &issuer,
		"source_repository_uri?", &repoURI, "source_repository_ref?", &repoRef,
		"runner_environment?", &runnerEnv, "predicate_type?", &predicateType,
		"predicate?", &predicate); err != nil {
		return nil, err
	}

	hasCert := san != "" || issuer != "" || repoURI != "" || repoRef != "" || runnerEnv != ""
	if hasCert && !verified {
		return nil, fmt.Errorf("%s: certificate fields require verified=True", b.Name())
	}

	statement, err := buildStatement(predicateType, predicate)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", b.Name(), err)
	}

	fixture := tkv1.VerificationFixture{IsSigned: true, IsVerified: verified}
	if hasCert || statement != nil {
		fixture.Result = &verify.VerificationResult{
			MediaType: verify.VerificationResultMediaType01,
			Statement: statement,
		}
	}
	if hasCert {
		fixture.Result.Signature = &verify.SignatureVerificationResult{
			Certificate: &certificate.Summary{
				SubjectAlternativeName: san,
				Extensions: certificate.Extensions{
					Issuer:              issuer,
					SourceRepositoryURI: repoURI,
					SourceRepositoryRef: repoRef,
					RunnerEnvironment:   runnerEnv,
				},
			},
		}
	}
	return &VerificationValue{Fixtures: []tkv1.VerificationFixture{fixture}}, nil
}

// buildStatement returns an in-toto statement for the given predicate, or
// nil if neither a predicate type nor a predicate is set.
func buildStatement(predicateType string, predicate *starlark.Dict) (*intoto.Statement, error) {
	if predicateType == "" && predicate == nil {
		return nil, nil
	}

	statement := &intoto.Statement{PredicateType: predicateType}
	if predicate != nil {
		m, err := dictToGoMap(predicate)
		if err != nil {
			return nil, fmt.Errorf("invalid predicate: %w", err)
		}
		statement.Predicate, err = structpb.NewStruct(m)
		if err != nil {
			return nil, fmt.Errorf("invalid predicate: %w", err)
		}
	}
	return statement, nil
}

// builtinVerificationJSON implements verification_json(content). content is
// a sigstore-go VerificationResult serialized as JSON, or a JSON list of
// them. List entries may also be wrapped as {"verificationResult": {...}}.
// Each result is reported as signed and verified.
func builtinVerificationJSON(
	_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple,
) (starlark.Value, error) {
	var content string
	if err := starlark.UnpackArgs(b.Name(), args, kwargs, "content", &content); err != nil {
		return nil, err
	}

	var raws []json.RawMessage
	if err := json.Unmarshal([]byte(content), &raws); err != nil {
		// Not a list, so treat it as a single result.
		raws = []json.RawMessage{json.RawMessage(content)}
	}

	out := &VerificationValue{}
	for i, raw := range raws {
		var wrapper struct {
			VerificationResult json.RawMessage `json:"verificationResult"`
		}
		if err := json.Unmarshal(raw, &wrapper); err != nil {
			return nil, fmt.Errorf("%s: result %d: %w", b.Name(), i, err)
		}
		if wrapper.VerificationResult != nil {
			raw = wrapper.VerificationResult
		}

		result, err := unmarshalVerificationResult(raw)
		if err != nil {
			return nil, fmt.Errorf("%s: result %d: %w", b.Name(), i, err)
		}
		out.Fixtures = append(out.Fixtures, tkv1.VerificationFixture{
			IsSigned:   true,
			IsVerified: true,
			Result:     result,
		})
	}
	return out, nil
}

// unmarshalVerificationResult decodes a sigstore-go VerificationResult.
// VerificationResult.UnmarshalJSON can't be used: it only keeps the
// statement (dropping the signature), and fails when there is no statement,
// as with plain cosign signatures. Instead, the statement is decoded with
// protojson and everything else with encoding/json.
func unmarshalVerificationResult(raw json.RawMessage) (*verify.VerificationResult, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, err
	}
	statement, hasStatement := fields["statement"]
	delete(fields, "statement")

	rest, err := json.Marshal(fields)
	if err != nil {
		return nil, err
	}
	// A defined type drops the UnmarshalJSON method.
	type plainResult verify.VerificationResult
	result := &verify.VerificationResult{}
	if err := json.Unmarshal(rest, (*plainResult)(result)); err != nil {
		return nil, err
	}

	if hasStatement {
		result.Statement = &intoto.Statement{}
		if err := protojson.Unmarshal(statement, result.Statement); err != nil {
			return nil, fmt.Errorf("parsing statement: %w", err)
		}
	}
	return result, nil
}
