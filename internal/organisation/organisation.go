package organisation

import (
	"context"

	"github.com/stacklok/mediator/pkg/db"
)

func CreateOrganisation(store db.Store, ctx context.Context, company string, name string) (*db.Organisation, error) {
	org, err := store.CreateOrganisation(ctx, db.CreateOrganisationParams{Company: company, Name: name})
	if err != nil {
		return nil, err
	} else {
		return &org, nil
	}
}
