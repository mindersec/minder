package controlplane

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-github/v53/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"golang.org/x/oauth2"
)

// MockClient is a mock implementation of the GitHub client.
type MockClient struct {
	mock.Mock
}

// RunUnitTestSuite runs the unit test suite.
func RunUnitTestSuite(t *testing.T) {
	suite.Run(t, new(UnitTestSuite))
}

// Repositories is a mock implementation of the GitHub client's Repositories service.
func (m *MockClient) Repositories() *github.RepositoriesService {
	args := m.Called()
	return args.Get(0).(*github.RepositoriesService)
}

// CreateHook is a mock implementation of the GitHub client's CreateHook method.
func (m *MockClient) CreateHook(ctx context.Context, owner, repo string, hook *github.Hook) (*github.Hook, *github.Response, error) {
	args := m.Called(ctx, owner, repo, hook)
	return args.Get(0).(*github.Hook), args.Get(1).(*github.Response), args.Error(2)
}

// UnitTestSuite is the test suite for the unit tests.
type UnitTestSuite struct {
	suite.Suite
	mockClient *MockClient
}

// SetupTest is called before each test function is executed.
func (s *UnitTestSuite) SetupTest() {
	s.mockClient = new(MockClient)
}

// TestRegisterWebHook_Success tests the RegisterWebHook function when the webhook registration is successful.
func (s *UnitTestSuite) TestRegisterWebHook_Success() {
	// Set up the test data
	ctx := context.Background()
	token := oauth2.Token{AccessToken: "your-access-token"}
	repositories := []Repository{
		{Owner: "owner1", Repo: "repo1"},
	}
	events := []string{"push", "pull_request"}

	// Set up the expectations for the mock client
	s.mockClient.On("Repositories").Return(&github.RepositoriesService{})
	s.mockClient.On("CreateHook", ctx, "owner1", "repo1", mock.AnythingOfType("*github.Hook")).
		Return(&github.Hook{ID: github.Int64(0), CreatedAt: &github.Timestamp{Time: time.Now()}}, &github.Response{}, nil)

	// Inject the mock client into the RegisterWebHook function
	registerData, err := RegisterWebHook(ctx, token, repositories, events)
	require.NoError(s.T(), err)
	require.Len(s.T(), registerData, 1)

	// Assertions for the first result
	assert.Equal(s.T(), "repo1", registerData[0].Repository)
	assert.Equal(s.T(), "owner1", registerData[0].Owner)
	assert.Equal(s.T(), int64(0), registerData[0].HookID)
}

func (s *UnitTestSuite) TestHandleGitHubWebHook() {
	ctx := context.Background()
	token := oauth2.Token{AccessToken: "your-access-token"}
	repositories := []Repository{
		{Owner: "owner1", Repo: "repo1"},
		{Owner: "owner2", Repo: "repo2"},
	}

	events := []string{"push", "pull_request"}

	// Set up the expectations for the mock client
	s.mockClient.On("Repositories").Return(&github.RepositoriesService{})
	s.mockClient.On("CreateHook", ctx, "owner1", "repo1", mock.AnythingOfType("*github.Hook")).
		Return(&github.Hook{ID: github.Int64(0), CreatedAt: &github.Timestamp{Time: time.Now()}}, &github.Response{}, nil)
	// Call the function under test
	results, err := RegisterWebHook(ctx, token, repositories, events)

	// Assertions
	require.NoError(s.T(), err)
	require.Len(s.T(), results, 2)
	assert.Equal(s.T(), "repo1", results[0].Repository)
	assert.Equal(s.T(), "owner1", results[0].Owner)
	assert.Equal(s.T(), int64(0), results[0].HookID)
	assert.NotEmpty(s.T(), results[0].DeployURL)
}

func TestAll(t *testing.T) {
	RunUnitTestSuite(t)
	// Call other test runner functions for additional test suites
}
