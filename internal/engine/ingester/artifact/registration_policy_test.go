// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package artifact

import (
	"context"
	"io"
	"log"
	"net/http/httptest"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/random"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/proto"

	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/db/embedded"
	"github.com/mindersec/minder/internal/engine/eval/rego"
	propsvc "github.com/mindersec/minder/internal/entities/properties/service"
	entsvc "github.com/mindersec/minder/internal/entities/service"
	"github.com/mindersec/minder/internal/entities/service/validators"
	"github.com/mindersec/minder/internal/providers/credentials"
	"github.com/mindersec/minder/internal/providers/dockerhub"
	mockmanager "github.com/mindersec/minder/internal/providers/manager/mock"
	"github.com/mindersec/minder/internal/providers/oci"
	"github.com/mindersec/minder/internal/verifier/verifyif"
	mockverify "github.com/mindersec/minder/internal/verifier/verifyif/mock"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/engine/v1/interfaces"
	"github.com/mindersec/minder/pkg/entities/properties"
)

// The eval definition of rule-types/github/artifact_signature.yaml in
// mindersec/minder-rules-and-profiles.
const artifactSignatureRego = `package minder

import rego.v1

violations contains {"msg": msg} if {
  artifactVersion := input.ingested[_]

  keys := object.keys(input.profile)
  key := keys[_]
  value := input.profile[key]

  artifactVersion.Verification[key] != value

  msg := sprintf("Verification failed for on key '%s': expected '%v', got '%v'", [key, value, artifactVersion.Verification[key]])
}
`

func TestRegisteredDockerHubArtifactEvaluation(t *testing.T) {
	t.Parallel()

	const namespace = "testns"
	ctx := t.Context()

	srv := httptest.NewServer(registry.New(registry.Logger(log.New(io.Discard, "", 0))))
	t.Cleanup(srv.Close)
	host := strings.TrimPrefix(srv.URL, "http://")

	signedDigest := pushRecentImage(t, host, namespace+"/myimage", "signed")
	unsignedDigest := pushRecentImage(t, host, namespace+"/myimage", "unsigned")
	require.NotEqual(t, signedDigest, unsignedDigest)

	dh, err := dockerhub.New(credentials.NewOAuth2TokenCredential("token"),
		&pb.DockerHubProviderConfig{Namespace: proto.String(namespace)})
	require.NoError(t, err)
	dh.OCI = oci.New(nil, host, path.Join(host, namespace))

	store, cancel, err := embedded.GetFakeStore()
	if cancel != nil {
		t.Cleanup(cancel)
	}
	require.NoError(t, err)

	project, err := store.CreateProject(ctx, db.CreateProjectParams{Name: "test", Metadata: []byte(`{}`)})
	require.NoError(t, err)
	dbProv, err := store.CreateProvider(ctx, db.CreateProviderParams{
		Name:       "dockerhub",
		ProjectID:  project.ID,
		Class:      db.ProviderClassDockerhub,
		Implements: []db.ProviderType{db.ProviderTypeImageLister, db.ProviderTypeOci},
		Definition: []byte(`{"dockerhub": {"namespace": "` + namespace + `"}}`),
		AuthFlows:  []db.AuthorizationFlow{db.AuthorizationFlowUserInput},
	})
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	provMgr := mockmanager.NewMockProviderManager(ctrl)
	provMgr.EXPECT().InstantiateFromID(gomock.Any(), dbProv.ID).Return(dh, nil).AnyTimes()

	propSvc := propsvc.NewPropertiesService(store)
	validatorRegistry := validators.NewValidatorRegistry()
	validatorRegistry.AddValidator(pb.Entity_ENTITY_ARTIFACTS, validators.NewArtifactValidator())
	creator := entsvc.NewEntityCreator(store, propSvc, provMgr, nil, validatorRegistry)

	verifier := mockverify.NewMockArtifactVerifier(ctrl)
	verifier.EXPECT().
		Verify(gomock.Any(), verifyif.ArtifactTypeContainer, namespace, "myimage", gomock.Any()).
		DoAndReturn(func(_ context.Context, _ verifyif.ArtifactType, _, _, digest string) ([]verifyif.Result, error) {
			return []verifyif.Result{{IsSigned: digest == signedDigest}}, nil
		}).
		AnyTimes()

	evaluator, err := rego.NewRegoEvaluator(&pb.RuleType_Definition_Eval_Rego{
		Type: "constraints",
		Def:  artifactSignatureRego,
	})
	require.NoError(t, err)

	tests := []struct {
		tag        string
		wantDigest string
		wantPass   bool
	}{
		{tag: "signed", wantDigest: signedDigest, wantPass: true},
		{tag: "unsigned", wantDigest: unsignedDigest, wantPass: false},
	}

	for _, tt := range tests {
		t.Run(tt.tag, func(t *testing.T) {
			t.Parallel()

			created, err := creator.CreateEntity(ctx, &dbProv, project.ID, pb.Entity_ENTITY_ARTIFACTS,
				properties.NewProperties(map[string]any{properties.PropertyName: "myimage:" + tt.tag}), nil)
			require.NoError(t, err)

			stored, err := propSvc.EntityWithPropertiesByID(ctx, created.Entity.ID, nil)
			require.NoError(t, err)
			require.Equal(t, "myimage:"+tt.tag, stored.Properties.GetProperty(properties.PropertyUpstreamID).GetString())
			require.Equal(t, tt.wantDigest, stored.Properties.GetProperty(oci.ArtifactPropertyDigest).GetString())

			ent, err := propSvc.EntityWithPropertiesAsProto(ctx, stored, provMgr)
			require.NoError(t, err)

			ing, err := NewArtifactDataIngest(dh)
			require.NoError(t, err)
			ing.artifactVerifier = verifier

			// TODO(#7089): drop tags once ingest uses the entity's own tag.
			ingested, err := ing.Ingest(ctx, ent, map[string]any{"name": "myimage", "tags": []any{tt.tag}})
			require.NoError(t, err)
			versions, ok := ingested.Object.([]imageInfo)
			require.True(t, ok)
			require.Len(t, versions, 1)
			require.Equal(t, tt.wantDigest, versions[0].Identity.Digest)
			require.Equal(t, []string{tt.tag}, versions[0].Identity.Tags)

			_, err = evaluator.Eval(ctx, map[string]any{"is_signed": true}, ent, ingested)
			if tt.wantPass {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, interfaces.ErrEvaluationFailed)
				require.ErrorContains(t, err, "Verification failed for on key 'is_signed': expected 'true', got 'false'")
			}
		})
	}
}

func pushRecentImage(t *testing.T, host, repo, tag string) string {
	t.Helper()

	img, err := random.Image(256, 1)
	require.NoError(t, err)
	img, err = mutate.CreatedAt(img, v1.Time{Time: time.Now()})
	require.NoError(t, err)

	ref, err := name.NewTag(host + "/" + repo + ":" + tag)
	require.NoError(t, err)
	require.NoError(t, remote.Write(ref, img))

	digest, err := img.Digest()
	require.NoError(t, err)
	return digest.String()
}
