package handlers

import (
	"context"
	"fmt"
	"github.com/stacklok/minder/internal/engine/entities"
	"github.com/stacklok/minder/internal/entities/models"
	mock_service "github.com/stacklok/minder/internal/entities/properties/service/mock"
	ghprops "github.com/stacklok/minder/internal/providers/github/properties"
	mock_manager "github.com/stacklok/minder/internal/providers/manager/mock"
	"google.golang.org/protobuf/reflect/protoreflect"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/stacklok/minder/internal/entities/properties"
	"github.com/stacklok/minder/internal/entities/properties/service"
	stubeventer "github.com/stacklok/minder/internal/events/stubs"
	"github.com/stacklok/minder/internal/providers/manager"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

type stubConverterProvider struct{}

func (scp *stubConverterProvider) CanImplement(_ minderv1.ProviderType) bool {
	panic("implement me")
}

func (scp *stubConverterProvider) FetchAllProperties(_ context.Context, _ *properties.Properties, entType minderv1.Entity, cachedProps *properties.Properties) (*properties.Properties, error) {
	panic("implement me")
}

func (scp *stubConverterProvider) FetchProperty(_ context.Context, _ *properties.Properties, entType minderv1.Entity, key string) (*properties.Property, error) {
	panic("implement me")
}

func (scp *stubConverterProvider) GetEntityName(_ minderv1.Entity, _ *properties.Properties) (string, error) {
	panic("implement me")
}

func (scp *stubConverterProvider) SupportsEntity(_ minderv1.Entity) bool {
	panic("implement me")
}

func (scp *stubConverterProvider) RegisterEntity(_ context.Context, _ minderv1.Entity, props *properties.Properties) (*properties.Properties, error) {
	panic("implement me")
}

func (scp *stubConverterProvider) DeregisterEntity(_ context.Context, _ minderv1.Entity, props *properties.Properties) error {
	panic("implement me")
}

func (scp *stubConverterProvider) PropertiesToProtoMessage(entType minderv1.Entity, props *properties.Properties) (protoreflect.ProtoMessage, error) {
	if entType == minderv1.Entity_ENTITY_REPOSITORIES {
		return ghprops.RepoV1FromProperties(props)
	}

	return nil, fmt.Errorf("unexpected entity type %s", entType)
}

func TestRefreshEntityAndDoHandler_HandleRefreshEntityAndDo(t *testing.T) {
	tests := []struct {
		name            string
		lookupPropMap   map[string]any
		entPropMap      map[string]any
		nextHandler     string
		providerHint    string
		ewp             *models.EntityWithProperties
		setupMocks      func(*gomock.Controller, *models.EntityWithProperties) (service.PropertiesService, manager.ProviderManager)
		expectedError   string
		expectedPublish bool
	}{
		{
			name: "successful refresh and publish of a repo",
			lookupPropMap: map[string]any{
				properties.PropertyUpstreamID: "123",
			},
			ewp: &models.EntityWithProperties{
				Entity: models.EntityInstance{
					ID:         uuid.New(),
					Type:       minderv1.Entity_ENTITY_REPOSITORIES,
					Name:       "testorg/testrepo",
					ProviderID: uuid.New(),
					ProjectID:  uuid.New(),
				},
			},
			nextHandler:  "call.me.next",
			providerHint: "github",
			entPropMap: map[string]any{
				properties.PropertyName:          "testorg/testrepo",
				ghprops.RepoPropertyName:         "testrepo",
				ghprops.RepoPropertyOwner:        "testorg",
				ghprops.RepoPropertyId:           int64(123),
				properties.RepoPropertyIsPrivate: false,
				properties.RepoPropertyIsFork:    false,
			},
			setupMocks: func(ctrl *gomock.Controller, ewp *models.EntityWithProperties) (service.PropertiesService, manager.ProviderManager) {
				mockPropSvc := mock_service.NewMockPropertiesService(ctrl)
				mockProvMgr := mock_manager.NewMockProviderManager(ctrl)

				mockPropSvc.EXPECT().
					EntityWithPropertiesByUpstreamID(gomock.Any(), ewp.Entity.Type, gomock.Any(), gomock.Any(), gomock.Any()).
					Return(ewp, nil)
				mockPropSvc.EXPECT().
					RetrieveAllPropertiesForEntity(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil)
				mockProvMgr.EXPECT().
					InstantiateFromID(gomock.Any(), ewp.Entity.ProviderID).
					Return(&stubConverterProvider{}, nil)

				return mockPropSvc, mockProvMgr
			},
			expectedPublish: true,
		},
		{
			name: "error unpacking message",
			setupMocks: func(ctrl *gomock.Controller, _ *models.EntityWithProperties) (service.PropertiesService, manager.ProviderManager) {
				return mock_service.NewMockPropertiesService(ctrl), mock_manager.NewMockProviderManager(ctrl)

			},
			expectedError: "error unpacking message",
		},
		// Add more test cases here
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			getByProps, err := properties.NewProperties(tt.lookupPropMap)
			require.NoError(t, err)

			entityMsg := NewEntityRefreshAndDoMessage(tt.ewp.Entity.Type, getByProps, tt.nextHandler, tt.providerHint)
			msg, err := entityMsg.ToMessage()
			require.NoError(t, err)

			entProps, err := properties.NewProperties(tt.entPropMap)
			require.NoError(t, err)
			tt.ewp.Properties = entProps

			mockPropSvc, mockProvMgr := tt.setupMocks(ctrl, tt.ewp)

			stubEventer := &stubeventer.StubEventer{}
			handler := NewRefreshEntityAndEvaluateHandler(stubEventer, mockPropSvc, mockProvMgr)

			refreshHandlerStruct, ok := handler.(*handleEntityAndDoBase)
			require.True(t, ok)
			err = refreshHandlerStruct.handleRefreshEntityAndDo(msg)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}

			if !tt.expectedPublish {
				assert.Equal(t, 0, len(stubEventer.Sent), "Expected no publish calls")
				return
			}

			assert.Equal(t, 1, len(stubEventer.Sent), "Expected one publish call")
			sentMsg := stubEventer.Sent[0]
			eiw, err := entities.ParseEntityEvent(sentMsg)
			require.NoError(t, err)
			require.NotNil(t, eiw)

			assert.Equal(t, tt.ewp.Entity.Type, eiw.Type)
			assert.Equal(t, tt.ewp.Entity.ProjectID, eiw.ProjectID)
			assert.Equal(t, tt.ewp.Entity.ProviderID, eiw.ProviderID)

			pbrepo, ok := eiw.Entity.(*minderv1.Repository)
			require.True(t, ok)
			assert.Equal(t, tt.entPropMap[ghprops.RepoPropertyName].(string), pbrepo.Name)
		})
	}
}
