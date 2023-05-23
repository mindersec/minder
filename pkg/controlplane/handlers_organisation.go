package controlplane

import (
	"context"

	"github.com/stacklok/mediator/internal/organisation"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// CreateOrganisation is a service for creating an organisation
func (s *Server) CreateOrganisation(ctx context.Context, in *pb.CreateOrganisationRequest) (*pb.CreateOrganisationResponse, error) {
	organisation, err := organisation.CreateOrganisation(s.store, ctx, in.GetCompany(), in.GetName())
	if err != nil {
		return nil, err
	} else {
		return &pb.CreateOrganisationResponse{Id: organisation.ID, Name: organisation.Name,
			Company: organisation.Company, CreatedAt: timestamppb.New(organisation.CreatedAt),
			UpdatedAt: timestamppb.New(organisation.UpdatedAt)}, nil
	}
}
