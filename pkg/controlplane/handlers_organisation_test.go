package controlplane

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"

	"github.com/stacklok/mediator/pkg/db"
	"github.com/stretchr/testify/assert"

	mockdb "github.com/stacklok/mediator/database/mock"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
)

func TestCreateOrganisation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create a mock store
	mockStore := mockdb.NewMockStore(ctrl)

	// Create test data
	request := &pb.CreateOrganisationRequest{
		Name:    "TestOrg",
		Company: "TestCompany",
	}

	expectedOrg := db.Organisation{
		ID:        1,
		Name:      "TestOrg",
		Company:   "TestCompany",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Set expectations on the mock store
	mockStore.EXPECT().
		CreateOrganisation(gomock.Any(), gomock.Any()).
		Return(expectedOrg, nil)

	// Create an instance of the server with the mock store
	server := &Server{
		store: mockStore,
	}

	// Call the CreateOrganisation function
	response, err := server.CreateOrganisation(context.Background(), request)

	// Assert the expected behavior and outcomes
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, expectedOrg.ID, response.Id)
	assert.Equal(t, expectedOrg.Name, response.Name)
	assert.Equal(t, expectedOrg.Company, response.Company)
	expectedCreatedAt := expectedOrg.CreatedAt.In(time.UTC)
	assert.Equal(t, expectedCreatedAt, response.CreatedAt.AsTime().In(time.UTC))
	expectedUpdatedAt := expectedOrg.UpdatedAt.In(time.UTC)
	assert.Equal(t, expectedUpdatedAt, response.UpdatedAt.AsTime().In(time.UTC))
}
