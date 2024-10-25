// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package ruletypes

import (
	"errors"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"

	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/db"
)

// RuleDefFromDB converts a rule type definition from the database to a protobuf
// rule type definition
func RuleDefFromDB(r *db.RuleType) (*pb.RuleType_Definition, error) {
	def := &pb.RuleType_Definition{}

	if err := protojson.Unmarshal(r.Definition, def); err != nil {
		return nil, fmt.Errorf("cannot unmarshal rule type definition: %w", err)
	}
	return def, nil
}

// RuleTypePBFromDB converts a rule type from the database to a protobuf
// rule type
func RuleTypePBFromDB(rt *db.RuleType) (*pb.RuleType, error) {
	def, err := RuleDefFromDB(rt)
	if err != nil {
		return nil, fmt.Errorf("cannot get rule type definition: %w", err)
	}

	id := rt.ID.String()
	project := rt.ProjectID.String()

	var seval pb.Severity_Value
	if err := seval.FromString(string(rt.SeverityValue)); err != nil {
		seval = pb.Severity_VALUE_UNKNOWN
	}

	displayName := rt.DisplayName
	if displayName == "" {
		displayName = rt.Name
	}

	var releasePhase pb.RuleTypeReleasePhase
	if err := releasePhase.FromString(string(rt.ReleasePhase)); err != nil {
		releasePhase = pb.RuleTypeReleasePhase_RULE_TYPE_RELEASE_PHASE_UNSPECIFIED
	}

	// TODO: (2024/03/28) this is for compatibility with old CLI versions that expect provider, remove this eventually
	noProvider := ""
	return &pb.RuleType{
		Id:                  &id,
		Name:                rt.Name,
		DisplayName:         displayName,
		ShortFailureMessage: rt.ShortFailureMessage,
		Context: &pb.Context{
			Provider: &noProvider,
			Project:  &project,
		},
		Description: rt.Description,
		Guidance:    rt.Guidance,
		Def:         def,
		Severity: &pb.Severity{
			Value: seval,
		},
		ReleasePhase: releasePhase,
	}, nil
}

// GetDBReleaseStatusFromPBReleasePhase converts a protobuf release phase to a database release status
func GetDBReleaseStatusFromPBReleasePhase(in pb.RuleTypeReleasePhase) (*db.ReleaseStatus, error) {
	sev, err := in.InitializedStringValue()
	if err != nil {
		return nil, errors.Join(ErrRuleTypeInvalid, err)
	}
	var rel db.ReleaseStatus

	if err := rel.Scan(sev); err != nil {
		// errors from the `Scan` method appear to be caused entirely by bad
		// input
		return nil, errors.Join(ErrRuleTypeInvalid, err)
	}

	return &rel, nil
}

// GetPBReleasePhaseFromDBReleaseStatus converts a database release status to a protobuf release phase
func GetPBReleasePhaseFromDBReleaseStatus(s *db.ReleaseStatus) (pb.RuleTypeReleasePhase, error) {
	if s == nil {
		return pb.RuleTypeReleasePhase_RULE_TYPE_RELEASE_PHASE_UNSPECIFIED, nil
	}

	var rel pb.RuleTypeReleasePhase
	if err := rel.FromString(string(*s)); err != nil {
		return pb.RuleTypeReleasePhase_RULE_TYPE_RELEASE_PHASE_UNSPECIFIED, err
	}

	return rel, nil
}
