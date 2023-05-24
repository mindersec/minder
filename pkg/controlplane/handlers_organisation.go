package controlplane

import (
	"context"

	"github.com/stacklok/mediator/internal/organisation"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// CreateOrganisation is a service for creating an organisation
func (s *Server) CreateOrganisation(ctx context.Context,
	in *pb.CreateOrganisationRequest) (*pb.CreateOrganisationResponse, error) {
	org, err := organisation.CreateOrganisation(ctx, s.store, in.GetName(), in.GetCompany())
	if err != nil {
		return nil, err
	}
	return &pb.CreateOrganisationResponse{Id: org.ID, Name: org.Name,
		Company: org.Company, CreatedAt: timestamppb.New(org.CreatedAt),
		UpdatedAt: timestamppb.New(org.UpdatedAt)}, nil
}
