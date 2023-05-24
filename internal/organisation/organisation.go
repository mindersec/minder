package organisation

import (
	"context"

	"github.com/go-playground/validator/v10"
	"github.com/stacklok/mediator/pkg/db"
)

type CreateOrganisationValidation struct {
	Name    string `db:"name" validate:"required"`
	Company string `db:"company" validate:"required"`
}

func CreateOrganisation(ctx context.Context, store db.Store, name string, company string) (*db.Organisation, error) {
	// validate that the company and name are not empty
	validator := validator.New()
	err := validator.Struct(CreateOrganisationValidation{Name: name, Company: company})
	if err != nil {
		return nil, err
	}
	org, err := store.CreateOrganisation(ctx, db.CreateOrganisationParams{Name: name, Company: company})
	if err != nil {
		return nil, err
	}
	return &org, nil
}
