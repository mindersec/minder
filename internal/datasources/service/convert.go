// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"errors"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/mindersec/minder/internal/datasources"
	"github.com/mindersec/minder/internal/db"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

func dataSourceDBToProtobuf(ds db.DataSource, dsfuncs []db.DataSourcesFunction) (*minderv1.DataSource, error) {
	outds := &minderv1.DataSource{
		Version: minderv1.VersionV1,
		Type:    string(minderv1.DataSourceResource),
		Id:      ds.ID.String(),
		Name:    ds.Name,
		Context: &minderv1.ContextV2{
			ProjectId: ds.ProjectID.String(),
		},
	}

	if len(dsfuncs) == 0 {
		return nil, errors.New("data source is invalid and has no defintions")
	}

	// All data source types should be equal... so we'll just take the first one.
	dsfType := dsfuncs[0].Type

	switch dsfType {
	case datasources.DataSourceDriverRest:
		return dataSourceRestDBToProtobuf(outds, dsfuncs)
	default:
		return nil, fmt.Errorf("unknown data source type: %s", dsfType)
	}
}

func dataSourceRestDBToProtobuf(ds *minderv1.DataSource, dsfuncs []db.DataSourcesFunction) (*minderv1.DataSource, error) {
	// At this point we have already validated that we have at least one function.
	ds.Driver = &minderv1.DataSource_Rest{
		Rest: &minderv1.RestDataSource{
			Def: make(map[string]*minderv1.RestDataSource_Def, len(dsfuncs)),
		},
	}

	for _, dsf := range dsfuncs {
		key := dsf.Name
		dsfToParse := &minderv1.RestDataSource_Def{}
		if err := protojson.Unmarshal(dsf.Definition, dsfToParse); err != nil {
			return nil, fmt.Errorf("failed to unmarshal data source definition for %s: %w", key, err)
		}

		ds.GetRest().Def[key] = dsfToParse
	}

	return ds, nil
}
