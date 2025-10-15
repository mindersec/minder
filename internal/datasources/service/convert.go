// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/mindersec/minder/internal/db"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	v1datasources "github.com/mindersec/minder/pkg/datasources/v1"
)

// DataSourceMetadata is used to serialize additional datasource-level fields
type DataSourceMetadata struct {
	Type         string `json:"type"`
	ProviderAuth bool   `json:"providerAuth"`
}

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

	var metadata DataSourceMetadata
	if ds.Metadata.Valid {
		if err := json.Unmarshal(ds.Metadata.RawMessage, &metadata); err != nil {
			return nil, fmt.Errorf("unable to unmarshal metadata: %w", err)
		}
	}

	// If we didn't record the type in metadata, use the first function to guess.
	dsfType := cmp.Or(metadata.Type, dsfuncs[0].Type)
	switch dsfType {
	case v1datasources.DataSourceDriverStruct:
		outds.Driver = &minderv1.DataSource_Structured{
			Structured: &minderv1.StructDataSource{},
		}
		return dataSourceStructDBToProtobuf(outds, dsfuncs)
	case v1datasources.DataSourceDriverRest:
		outds.Driver = &minderv1.DataSource_Rest{
			Rest: &minderv1.RestDataSource{},
		}
		outds.GetRest().ProviderAuth = metadata.ProviderAuth
		return dataSourceRestDBToProtobuf(outds, dsfuncs)
	default:
		return nil, fmt.Errorf("unknown data source type: %s", dsfType)
	}
}

func dataSourceRestDBToProtobuf(ds *minderv1.DataSource, dsfuncs []db.DataSourcesFunction) (*minderv1.DataSource, error) {
	// At this point we have already validated that we have at least one function.
	ds.GetRest().Def = make(map[string]*minderv1.RestDataSource_Def, len(dsfuncs))

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

func dataSourceStructDBToProtobuf(ds *minderv1.DataSource, dsfuncs []db.DataSourcesFunction) (*minderv1.DataSource, error) {
	ds.GetStructured().Def = make(map[string]*minderv1.StructDataSource_Def, len(dsfuncs))

	for _, dsf := range dsfuncs {
		key := dsf.Name
		dsfToParse := &minderv1.StructDataSource_Def{
			Path: &minderv1.StructDataSource_Def_Path{},
		}
		if err := protojson.Unmarshal(dsf.Definition, dsfToParse); err != nil {
			return nil, fmt.Errorf("failed to unmarshal data source definition for %s: %w", key, err)
		}

		ds.GetStructured().Def[key] = dsfToParse
	}

	return ds, nil
}

func metadataForDataSource(ds *minderv1.DataSource) (json.RawMessage, error) {
	metadata := DataSourceMetadata{
		Type: v1datasources.DataSourceDriverStruct,
	}
	switch ds.Driver.(type) {
	case *minderv1.DataSource_Rest:
		metadata.Type = v1datasources.DataSourceDriverRest
		metadata.ProviderAuth = ds.GetRest().GetProviderAuth()
	case *minderv1.DataSource_Structured:
		metadata.Type = v1datasources.DataSourceDriverStruct
	default:
		return nil, fmt.Errorf("unknown datasource driver %T", ds.Driver)
	}
	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}
	return metadataBytes, nil
}
