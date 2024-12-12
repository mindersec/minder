package github

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/google/go-github/v63/github"
	"github.com/mindersec/minder/internal/db"
	provifv1 "github.com/mindersec/minder/pkg/providers/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"golang.org/x/oauth2"

	mock_github "github.com/mindersec/minder/internal/providers/github/mock"
	mock_ratecache "github.com/mindersec/minder/internal/providers/ratecache/mock"
)

type testGitHub struct {
	gh       *GitHub
	delegate *mock_github.MockDelegate
	cache    *mock_ratecache.MockRestClientCache
	client   *github.Client
}

func setupTest(t *testing.T) *testGitHub {
	ctrl := gomock.NewController(t)
	delegate := mock_github.NewMockDelegate(ctrl)
	cache := mock_ratecache.NewMockRestClientCache(ctrl)

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "dummy-token"})
	tc := oauth2.NewClient(context.Background(), ts)
	client := github.NewClient(tc)

	gh := &GitHub{
		client:   client,
		cache:    cache,
		delegate: delegate,
	}

	return &testGitHub{
		gh:       gh,
		delegate: delegate,
		cache:    cache,
		client:   client,
	}
}

func TestWaitForRateLimitReset(t *testing.T) {
	tests := []struct {
		name        string
		inputErr    error
		setupMocks  func(*testGitHub)
		wantErr     bool
		expectedErr error
	}{
		{
			name: "primary rate limit error with reset time under max wait",
			inputErr: &github.RateLimitError{
				Rate: github.Rate{
					Remaining: 0,
					Reset:     github.Timestamp{Time: time.Now().Add(100 * time.Millisecond)},
				},
				Response: &http.Response{
					Request: &http.Request{
						Method: "GET",
						URL:    &url.URL{Path: "/test"},
					},
				},
			},
			setupMocks: func(th *testGitHub) {
				th.delegate.EXPECT().GetCredential().Return(&mockCredential{}).AnyTimes()
				th.delegate.EXPECT().GetOwner().Return("test-owner").AnyTimes()
				th.cache.EXPECT().Set(
					"test-owner",
					gomock.Any(),
					db.ProviderTypeGithub,
					th.gh,
				).Return()
			},
			wantErr: false,
		},
		{
			name: "primary rate limit error with reset time over max wait",
			inputErr: &github.RateLimitError{
				Rate: github.Rate{
					Remaining: 0,
					Reset:     github.Timestamp{Time: time.Now().Add(10 * time.Minute)},
				},
				Response: &http.Response{
					Request: &http.Request{
						Method: "GET",
						URL:    &url.URL{Path: "/test"},
					},
				},
			},
			setupMocks: func(th *testGitHub) {
				th.delegate.EXPECT().GetCredential().Return(&mockCredential{}).AnyTimes()
				th.delegate.EXPECT().GetOwner().Return("test-owner").AnyTimes()
				th.cache.EXPECT().Set(
					"test-owner",
					gomock.Any(),
					db.ProviderTypeGithub,
					th.gh,
				).Return()
			},
			wantErr:     true,
			expectedErr: &github.RateLimitError{},
		},
		{
			name: "abuse rate limit error with retry after under max wait",
			inputErr: &github.AbuseRateLimitError{
				RetryAfter: func() *time.Duration {
					d := 100 * time.Millisecond
					return &d
				}(),
				Response: &http.Response{
					Request: &http.Request{
						Method: "GET",
						URL:    &url.URL{Path: "/test"},
					},
				},
			},
			setupMocks: func(th *testGitHub) {
				th.delegate.EXPECT().GetCredential().Return(&mockCredential{}).AnyTimes()
				th.delegate.EXPECT().GetOwner().Return("test-owner").AnyTimes()
				th.cache.EXPECT().Set(
					"test-owner",
					gomock.Any(),
					db.ProviderTypeGithub,
					th.gh,
				).Return()
			},
			wantErr: false,
		},
		{
			name:       "non-rate limit error",
			inputErr:   errors.New("random error"),
			setupMocks: func(th *testGitHub) {},
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			th := setupTest(t)
			tt.setupMocks(th)

			// Create context with timeout
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			err := th.gh.waitForRateLimitReset(ctx, tt.inputErr)

			if tt.wantErr {
				require.Error(t, err)
				if tt.expectedErr != nil {
					assert.IsType(t, tt.expectedErr, err)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPerformWithRetry(t *testing.T) {
	tests := []struct {
		name       string
		operation  func() (string, error)
		wantErr    bool
		wantResult string
	}{
		{
			name: "successful operation",
			operation: func() (string, error) {
				return "success", nil
			},
			wantErr:    false,
			wantResult: "success",
		},
		{
			name: "permanent error",
			operation: func() (string, error) {
				return "", errors.New("permanent error")
			},
			wantErr: true,
		},
		{
			name: "rate limit error then success",
			operation: func() (string, error) {
				return "", &github.RateLimitError{
					Rate: github.Rate{
						Remaining: 0,
						Reset:     github.Timestamp{Time: time.Now().Add(1 * time.Second)},
					},
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := performWithRetry(context.Background(), tt.operation)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantResult, result)
			}
		})
	}
}

type mockCredential struct {
	provifv1.GitHubCredential
}

func (m *mockCredential) GetCacheKey() string {
	return "mock-cache-key"
}
