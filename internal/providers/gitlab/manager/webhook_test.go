// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package manager

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gitlablib "gitlab.com/gitlab-org/api/client-go"
	"go.uber.org/mock/gomock"

	entmsg "github.com/mindersec/minder/internal/entities/handlers/message"
	"github.com/mindersec/minder/internal/providers/gitlab"
	"github.com/mindersec/minder/internal/providers/gitlab/webhooksecret"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/entities/properties"
	"github.com/mindersec/minder/pkg/eventer/constants"
	mockinterfaces "github.com/mindersec/minder/pkg/eventer/interfaces/mock"
)

func TestGetWebhookHandler_RoutingAndAuth(t *testing.T) {
	t.Parallel()

	const currentSecret = "current-gitlab-webhook-secret-key"
	const prevSecret = "prev-gitlab-webhook-secret-key"
	hookUUID := uuid.New().String()
	validToken, err := webhooksecret.New(currentSecret, hookUUID)
	require.NoError(t, err)

	validPrevToken, err := webhooksecret.New(prevSecret, hookUUID)
	require.NoError(t, err)

	tests := []struct {
		name           string
		path           string
		headers        map[string]string
		body           string
		setupMock      func(pub *mockinterfaces.MockPublisher)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "missing X-Gitlab-Token header",
			path: fmt.Sprintf("/api/v1/webhook/%s", hookUUID),
			headers: map[string]string{
				"X-Gitlab-Event": string(gitlablib.EventTypePush),
			},
			body:           `{"project_id": 1234}`,
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "invalid webhook request",
		},
		{
			name: "invalid unique UUID in URL path",
			path: "/api/v1/webhook/not-a-uuid",
			headers: map[string]string{
				"X-Gitlab-Token": "some-token",
				"X-Gitlab-Event": string(gitlablib.EventTypePush),
			},
			body:           `{"project_id": 1234}`,
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "invalid webhook request",
		},
		{
			name: "invalid token signature returns 401",
			path: fmt.Sprintf("/api/v1/webhook/%s", hookUUID),
			headers: map[string]string{
				"X-Gitlab-Token": "wrong-secret-token",
				"X-Gitlab-Event": string(gitlablib.EventTypePush),
			},
			body:           `{"project_id": 1234}`,
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "invalid webhook request",
		},
		{
			name: "missing X-Gitlab-Event header returns 400",
			path: fmt.Sprintf("/api/v1/webhook/%s", hookUUID),
			headers: map[string]string{
				"X-Gitlab-Token": validToken,
			},
			body:           `{"project_id": 1234}`,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "missing X-Gitlab-Event header",
		},
		{
			name: "valid request with previous rotated secret succeeds",
			path: fmt.Sprintf("/api/v1/webhook/%s", hookUUID),
			headers: map[string]string{
				"X-Gitlab-Token": validPrevToken,
				"X-Gitlab-Event": string(gitlablib.EventTypePush),
			},
			body: `{"project_id": 1234}`,
			setupMock: func(pub *mockinterfaces.MockPublisher) {
				pub.EXPECT().Publish(constants.TopicQueueRefreshEntityAndEvaluate, gomock.Any()).Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "unhandled webhook event routes to noop and returns 200",
			path: fmt.Sprintf("/api/v1/webhook/%s", hookUUID),
			headers: map[string]string{
				"X-Gitlab-Token": validToken,
				"X-Gitlab-Event": "Issue Hook",
			},
			body:           `{"object_kind": "issue"}`,
			expectedStatus: http.StatusOK,
		},
		{
			name: "handler error returns 500 internal server error",
			path: fmt.Sprintf("/api/v1/webhook/%s", hookUUID),
			headers: map[string]string{
				"X-Gitlab-Token": validToken,
				"X-Gitlab-Event": string(gitlablib.EventTypePush),
			},
			body:           `{"project_id": 0}`, // Missing project ID causes error
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "error handling webhook event",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			mockPub := mockinterfaces.NewMockPublisher(ctrl)
			if tt.setupMock != nil {
				tt.setupMock(mockPub)
			}

			mgr := helperNewTestManager(t, mockPub, currentSecret, []string{prevSecret})
			handler := mgr.GetWebhookHandler()

			req := httptest.NewRequest(http.MethodPost, tt.path, bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != "" {
				assert.Contains(t, w.Body.String(), tt.expectedBody)
			}
		})
	}
}

func TestHandleRepoPushAndTagPush(t *testing.T) {
	t.Parallel()

	const currentSecret = "gitlab-repo-events-secret"
	hookUUID := uuid.New().String()
	validToken, err := webhooksecret.New(currentSecret, hookUUID)
	require.NoError(t, err)

	tests := []struct {
		name           string
		eventType      gitlablib.EventType
		body           string
		setupMock      func(pub *mockinterfaces.MockPublisher)
		expectedStatus int
		expectedErr    string
	}{
		{
			name:      "push event with valid project ID publishes to refresh and eval",
			eventType: gitlablib.EventTypePush,
			body:      `{"project_id": 42001}`,
			setupMock: func(pub *mockinterfaces.MockPublisher) {
				pub.EXPECT().Publish(constants.TopicQueueRefreshEntityAndEvaluate, gomock.Any()).
					DoAndReturn(func(topic string, msg *message.Message) error {
						assert.Equal(t, constants.TopicQueueRefreshEntityAndEvaluate, topic)
						require.NotNil(t, msg)
						entMsg, err := entmsg.ToEntityRefreshAndDo(msg)
						require.NoError(t, err)
						assert.Equal(t, string(gitlab.Class), entMsg.Hint.ProviderClassHint)
						assert.Equal(t, minderv1.Entity_ENTITY_REPOSITORIES, entMsg.Entity.Type)
						assert.Equal(t, gitlab.FormatRepositoryUpstreamID(42001), entMsg.Entity.GetByProps[properties.PropertyUpstreamID])
						return nil
					})
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "push event missing project ID returns 500",
			eventType:      gitlablib.EventTypePush,
			body:           `{"project_id": 0}`,
			expectedStatus: http.StatusInternalServerError,
			expectedErr:    "error handling webhook event",
		},
		{
			name:           "push event with malformed JSON body returns 500",
			eventType:      gitlablib.EventTypePush,
			body:           `{not-valid-json`,
			expectedStatus: http.StatusInternalServerError,
			expectedErr:    "error handling webhook event",
		},
		{
			name:      "tag push event with valid project ID publishes to refresh and eval",
			eventType: gitlablib.EventTypeTagPush,
			body:      `{"project_id": 9988}`,
			setupMock: func(pub *mockinterfaces.MockPublisher) {
				pub.EXPECT().Publish(constants.TopicQueueRefreshEntityAndEvaluate, gomock.Any()).
					DoAndReturn(func(_ string, msg *message.Message) error {
						entMsg, err := entmsg.ToEntityRefreshAndDo(msg)
						require.NoError(t, err)
						assert.Equal(t, string(gitlab.Class), entMsg.Hint.ProviderClassHint)
						assert.Equal(t, minderv1.Entity_ENTITY_REPOSITORIES, entMsg.Entity.Type)
						assert.Equal(t, gitlab.FormatRepositoryUpstreamID(9988), entMsg.Entity.GetByProps[properties.PropertyUpstreamID])
						return nil
					})
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "tag push event missing project ID returns 500",
			eventType:      gitlablib.EventTypeTagPush,
			body:           `{"project_id": 0}`,
			expectedStatus: http.StatusInternalServerError,
			expectedErr:    "error handling webhook event",
		},
		{
			name:           "tag push event with malformed JSON returns 500",
			eventType:      gitlablib.EventTypeTagPush,
			body:           `{"project_id": `,
			expectedStatus: http.StatusInternalServerError,
			expectedErr:    "error handling webhook event",
		},
		{
			name:      "publisher error on push event returns 500",
			eventType: gitlablib.EventTypePush,
			body:      `{"project_id": 123}`,
			setupMock: func(pub *mockinterfaces.MockPublisher) {
				pub.EXPECT().Publish(constants.TopicQueueRefreshEntityAndEvaluate, gomock.Any()).
					Return(errors.New("watermill publish network error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedErr:    "error handling webhook event",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			mockPub := mockinterfaces.NewMockPublisher(ctrl)
			if tt.setupMock != nil {
				tt.setupMock(mockPub)
			}

			mgr := helperNewTestManager(t, mockPub, currentSecret, nil)
			handler := mgr.GetWebhookHandler()

			path := fmt.Sprintf("/api/v1/webhook/%s", hookUUID)
			req := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(tt.body))
			req.Header.Set("X-Gitlab-Token", validToken)
			req.Header.Set("X-Gitlab-Event", string(tt.eventType))

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedErr != "" {
				assert.Contains(t, w.Body.String(), tt.expectedErr)
			}
		})
	}
}

func TestHandleMergeRequest(t *testing.T) {
	t.Parallel()

	const currentSecret = "gitlab-mr-events-secret"
	hookUUID := uuid.New().String()
	validToken, err := webhooksecret.New(currentSecret, hookUUID)
	require.NoError(t, err)

	tests := []struct {
		name          string
		body          string
		setupMock     func(pub *mockinterfaces.MockPublisher)
		expectedCode  int
		expectedTopic string
	}{
		{
			name: "merge request action open publishes EntityAdd message",
			body: `{
				"project": {"id": 101},
				"object_attributes": {
					"id": 5001,
					"iid": 42,
					"action": "open"
				}
			}`,
			expectedTopic: constants.TopicQueueOriginatingEntityAdd,
			setupMock: func(pub *mockinterfaces.MockPublisher) {
				pub.EXPECT().Publish(constants.TopicQueueOriginatingEntityAdd, gomock.Any()).
					DoAndReturn(func(topic string, msg *message.Message) error {
						assert.Equal(t, constants.TopicQueueOriginatingEntityAdd, topic)
						entMsg, err := entmsg.ToEntityRefreshAndDo(msg)
						require.NoError(t, err)

						assert.Equal(t, string(gitlab.Class), entMsg.Hint.ProviderClassHint)

						// Check Entity (pull_requests)
						assert.Equal(t, minderv1.Entity_ENTITY_PULL_REQUESTS, entMsg.Entity.Type)
						assert.Equal(t, gitlab.FormatPullRequestUpstreamID(5001), entMsg.Entity.GetByProps[properties.PropertyUpstreamID])
						assert.Equal(t, gitlab.FormatPullRequestUpstreamID(42), entMsg.Entity.GetByProps[gitlab.PullRequestNumber])
						assert.Equal(t, gitlab.FormatRepositoryUpstreamID(101), entMsg.Entity.GetByProps[gitlab.PullRequestProjectID])

						// Check Originator (repositories)
						assert.Equal(t, minderv1.Entity_ENTITY_REPOSITORIES, entMsg.Originator.Type)
						assert.Equal(t, gitlab.FormatRepositoryUpstreamID(101), entMsg.Originator.GetByProps[properties.PropertyUpstreamID])
						return nil
					})
			},
			expectedCode: http.StatusOK,
		},
		{
			name: "merge request action reopen publishes EntityAdd message",
			body: `{
				"project": {"id": 101},
				"object_attributes": {
					"id": 5002,
					"iid": 43,
					"action": "reopen"
				}
			}`,
			expectedTopic: constants.TopicQueueOriginatingEntityAdd,
			setupMock: func(pub *mockinterfaces.MockPublisher) {
				pub.EXPECT().Publish(constants.TopicQueueOriginatingEntityAdd, gomock.Any()).Return(nil)
			},
			expectedCode: http.StatusOK,
		},
		{
			name: "merge request action close publishes EntityDelete message",
			body: `{
				"project": {"id": 101},
				"object_attributes": {
					"id": 5003,
					"iid": 44,
					"action": "close"
				}
			}`,
			expectedTopic: constants.TopicQueueOriginatingEntityDelete,
			setupMock: func(pub *mockinterfaces.MockPublisher) {
				pub.EXPECT().Publish(constants.TopicQueueOriginatingEntityDelete, gomock.Any()).Return(nil)
			},
			expectedCode: http.StatusOK,
		},
		{
			name: "merge request action update publishes RefreshEntityAndEvaluate message",
			body: `{
				"project": {"id": 101},
				"object_attributes": {
					"id": 5004,
					"iid": 45,
					"action": "update"
				}
			}`,
			expectedTopic: constants.TopicQueueRefreshEntityAndEvaluate,
			setupMock: func(pub *mockinterfaces.MockPublisher) {
				pub.EXPECT().Publish(constants.TopicQueueRefreshEntityAndEvaluate, gomock.Any()).Return(nil)
			},
			expectedCode: http.StatusOK,
		},
		{
			name: "unhandled merge request action (e.g. approved) does not publish and returns 200",
			body: `{
				"project": {"id": 101},
				"object_attributes": {
					"id": 5005,
					"iid": 46,
					"action": "approved"
				}
			}`,
			expectedCode: http.StatusOK,
		},
		{
			name: "merge request missing object_attributes id returns 500",
			body: `{
				"project": {"id": 101},
				"object_attributes": {
					"iid": 42,
					"action": "open"
				}
			}`,
			expectedCode: http.StatusInternalServerError,
		},
		{
			name: "merge request missing object_attributes iid returns 500",
			body: `{
				"project": {"id": 101},
				"object_attributes": {
					"id": 5001,
					"action": "open"
				}
			}`,
			expectedCode: http.StatusInternalServerError,
		},
		{
			name: "merge request missing project id returns 500",
			body: `{
				"object_attributes": {
					"id": 5001,
					"iid": 42,
					"action": "open"
				}
			}`,
			expectedCode: http.StatusInternalServerError,
		},
		{
			name:         "merge request invalid JSON payload returns 500",
			body:         `{invalid json`,
			expectedCode: http.StatusInternalServerError,
		},
		{
			name: "publisher error on merge request event returns 500",
			body: `{
				"project": {"id": 101},
				"object_attributes": {
					"id": 5001,
					"iid": 42,
					"action": "open"
				}
			}`,
			setupMock: func(pub *mockinterfaces.MockPublisher) {
				pub.EXPECT().Publish(constants.TopicQueueOriginatingEntityAdd, gomock.Any()).
					Return(errors.New("publisher broker error"))
			},
			expectedCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			mockPub := mockinterfaces.NewMockPublisher(ctrl)
			if tt.setupMock != nil {
				tt.setupMock(mockPub)
			}

			mgr := helperNewTestManager(t, mockPub, currentSecret, nil)
			handler := mgr.GetWebhookHandler()

			path := fmt.Sprintf("/api/v1/webhook/%s", hookUUID)
			req := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(tt.body))
			req.Header.Set("X-Gitlab-Token", validToken)
			req.Header.Set("X-Gitlab-Event", string(gitlablib.EventTypeMergeRequest))

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedCode, w.Code)
		})
	}
}

func TestHandleRelease(t *testing.T) {
	t.Parallel()

	const currentSecret = "gitlab-release-events-secret"
	hookUUID := uuid.New().String()
	validToken, err := webhooksecret.New(currentSecret, hookUUID)
	require.NoError(t, err)

	tests := []struct {
		name         string
		body         string
		setupMock    func(pub *mockinterfaces.MockPublisher)
		expectedCode int
	}{
		{
			name: "release action create publishes EntityAdd message",
			body: `{
				"id": 8801,
				"tag": "v1.0.0",
				"action": "create",
				"project": {"id": 202}
			}`,
			setupMock: func(pub *mockinterfaces.MockPublisher) {
				pub.EXPECT().Publish(constants.TopicQueueOriginatingEntityAdd, gomock.Any()).
					DoAndReturn(func(topic string, msg *message.Message) error {
						assert.Equal(t, constants.TopicQueueOriginatingEntityAdd, topic)

						entMsg, err := entmsg.ToEntityRefreshAndDo(msg)
						require.NoError(t, err)

						assert.Equal(t, string(gitlab.Class), entMsg.Hint.ProviderClassHint)

						// Check Entity (release)
						assert.Equal(t, minderv1.Entity_ENTITY_RELEASE, entMsg.Entity.Type)
						assert.Equal(t, gitlab.FormatPullRequestUpstreamID(8801), entMsg.Entity.GetByProps[properties.PropertyUpstreamID])
						assert.Equal(t, "v1.0.0", entMsg.Entity.GetByProps[gitlab.ReleasePropertyTag])
						assert.Equal(t, gitlab.FormatRepositoryUpstreamID(202), entMsg.Entity.GetByProps[gitlab.PullRequestProjectID])

						// Check Originator (repositories)
						assert.Equal(t, minderv1.Entity_ENTITY_REPOSITORIES, entMsg.Originator.Type)
						assert.Equal(t, gitlab.FormatRepositoryUpstreamID(202), entMsg.Originator.GetByProps[properties.PropertyUpstreamID])
						return nil
					})
			},
			expectedCode: http.StatusOK,
		},
		{
			name: "release action update publishes RefreshEntityAndEvaluate message",
			body: `{
				"id": 8802,
				"tag": "v1.0.1",
				"action": "update",
				"project": {"id": 202}
			}`,
			setupMock: func(pub *mockinterfaces.MockPublisher) {
				pub.EXPECT().Publish(constants.TopicQueueRefreshEntityAndEvaluate, gomock.Any()).Return(nil)
			},
			expectedCode: http.StatusOK,
		},
		{
			name: "release action delete publishes EntityDelete message",
			body: `{
				"id": 8803,
				"tag": "v1.0.2",
				"action": "delete",
				"project": {"id": 202}
			}`,
			setupMock: func(pub *mockinterfaces.MockPublisher) {
				pub.EXPECT().Publish(constants.TopicQueueOriginatingEntityDelete, gomock.Any()).Return(nil)
			},
			expectedCode: http.StatusOK,
		},
		{
			name: "unhandled release action returns 200 without publishing",
			body: `{
				"id": 8804,
				"tag": "v1.0.3",
				"action": "unknown_action",
				"project": {"id": 202}
			}`,
			expectedCode: http.StatusOK,
		},
		{
			name: "release missing id returns 500",
			body: `{
				"tag": "v1.0.0",
				"action": "create",
				"project": {"id": 202}
			}`,
			expectedCode: http.StatusInternalServerError,
		},
		{
			name: "release missing tag returns 500",
			body: `{
				"id": 8801,
				"action": "create",
				"project": {"id": 202}
			}`,
			expectedCode: http.StatusInternalServerError,
		},
		{
			name: "release missing project id returns 500",
			body: `{
				"id": 8801,
				"tag": "v1.0.0",
				"action": "create"
			}`,
			expectedCode: http.StatusInternalServerError,
		},
		{
			name:         "release invalid JSON returns 500",
			body:         `{not-json`,
			expectedCode: http.StatusInternalServerError,
		},
		{
			name: "publisher error on release event returns 500",
			body: `{
				"id": 8801,
				"tag": "v1.0.0",
				"action": "create",
				"project": {"id": 202}
			}`,
			setupMock: func(pub *mockinterfaces.MockPublisher) {
				pub.EXPECT().Publish(constants.TopicQueueOriginatingEntityAdd, gomock.Any()).
					Return(errors.New("watermill release publish failed"))
			},
			expectedCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			mockPub := mockinterfaces.NewMockPublisher(ctrl)
			if tt.setupMock != nil {
				tt.setupMock(mockPub)
			}

			mgr := helperNewTestManager(t, mockPub, currentSecret, nil)
			handler := mgr.GetWebhookHandler()

			path := fmt.Sprintf("/api/v1/webhook/%s", hookUUID)
			req := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(tt.body))
			req.Header.Set("X-Gitlab-Token", validToken)
			req.Header.Set("X-Gitlab-Event", string(gitlablib.EventTypeRelease))

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedCode, w.Code)
		})
	}
}

func TestWrapSafe(t *testing.T) {
	t.Parallel()

	reader := bytes.NewReader([]byte("test safe payload"))
	limited := wrapSafe(reader)
	require.NotNil(t, limited)

	buf := new(bytes.Buffer)
	n, err := buf.ReadFrom(limited)
	require.NoError(t, err)
	assert.Equal(t, int64(17), n)
	assert.Equal(t, "test safe payload", buf.String())

	// Test boundary with payload larger than MaxBytesLimit (1MB)
	oversized := bytes.Repeat([]byte("a"), int(MaxBytesLimit)+1024)
	limitedOversized := wrapSafe(bytes.NewReader(oversized))
	bufOversized := new(bytes.Buffer)
	nOversized, err := bufOversized.ReadFrom(limitedOversized)
	require.NoError(t, err)
	assert.Equal(t, MaxBytesLimit, nOversized)
	assert.Equal(t, int(MaxBytesLimit), bufOversized.Len())
}

type testSamplePayload struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

func TestDecodeJSONSafe(t *testing.T) {
	t.Parallel()

	t.Run("successfully decodes valid JSON", func(t *testing.T) {
		t.Parallel()

		rawJSON := `{"name": "minder", "count": 42}`
		rc := io.NopCloser(strings.NewReader(rawJSON))

		var out testSamplePayload
		err := decodeJSONSafe(rc, &out)
		require.NoError(t, err)
		assert.Equal(t, "minder", out.Name)
		assert.Equal(t, 42, out.Count)
	})

	t.Run("returns error on malformed JSON", func(t *testing.T) {
		t.Parallel()

		rawJSON := `{"name": "minder", broken`
		rc := io.NopCloser(strings.NewReader(rawJSON))

		var out testSamplePayload
		err := decodeJSONSafe(rc, &out)
		require.Error(t, err)
	})

	t.Run("returns error when payload exceeds limit or truncated JSON", func(t *testing.T) {
		t.Parallel()

		// Construct valid JSON structure that exceeds 1MB limit so the closing bracket is cut off
		oversized := fmt.Sprintf(`{"name": "%s", "count": 1}`, strings.Repeat("x", int(MaxBytesLimit)+10))
		rc := io.NopCloser(strings.NewReader(oversized))

		var out testSamplePayload
		err := decodeJSONSafe(rc, &out)
		require.Error(t, err)
	})
}
