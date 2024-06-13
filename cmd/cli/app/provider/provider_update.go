//
// Copyright 2024 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package provider

import (
	"context"
	// "encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/stacklok/minder/internal/util/cli"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

var (
	errMissingProviderName = errors.New("provider name flag is missing")
	errMissingProject      = errors.New("project flag is missing")
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Updates a provider's configuration",
	Long: `The minder provider update command allows a user to update a provider's
configuration after enrollement.`,
	RunE: cli.GRPCClientWrapRunE(UpdateProviderCommand),
}

type parser func(string) (*protoreflect.Value, error)

func parseBool(val string) (*protoreflect.Value, error) {
	v, err := strconv.ParseBool(val)
	if err != nil {
		return nil, fmt.Errorf("expected bool, got %s", val)
	}
	res := protoreflect.ValueOf(v)
	return &res, nil
}

func parseInt(size int) parser {
	return func(val string) (*protoreflect.Value, error) {
		v, err := strconv.ParseInt(val, 10, size)
		if err != nil {
			return nil, fmt.Errorf("expected integer, got %s", val)
		}
		res := protoreflect.ValueOf(v)
		return &res, nil
	}
}

func parseUint(size int) parser {
	return func(val string) (*protoreflect.Value, error) {
		v, err := strconv.ParseUint(val, 10, size)
		if err != nil {
			return nil, fmt.Errorf("expected integer, got %s", val)
		}
		res := protoreflect.ValueOf(v)
		return &res, nil
	}
}

func parseFloat(size int) parser {
	return func(val string) (*protoreflect.Value, error) {
		v, err := strconv.ParseFloat(val, size)
		if err != nil {
			return nil, fmt.Errorf("expected integer, got %s", val)
		}
		res := protoreflect.ValueOf(v)
		return &res, nil
	}
}

func identity(val string) (*protoreflect.Value, error) {
	res := protoreflect.ValueOf(val)
	return &res, nil
}

var (
	parserMap = map[protoreflect.Kind]parser{
		protoreflect.BoolKind:     parseBool,
		protoreflect.Int32Kind:    parseInt(32),
		protoreflect.Sint32Kind:   parseInt(32),
		protoreflect.Sfixed32Kind: parseInt(32),
		protoreflect.Int64Kind:    parseInt(64),
		protoreflect.Sint64Kind:   parseInt(64),
		protoreflect.Sfixed64Kind: parseInt(64),
		protoreflect.Uint32Kind:   parseUint(32),
		protoreflect.Fixed32Kind:  parseUint(32),
		protoreflect.Uint64Kind:   parseUint(64),
		protoreflect.Fixed64Kind:  parseUint(64),
		protoreflect.FloatKind:    parseFloat(32),
		protoreflect.DoubleKind:   parseFloat(64), // double
		protoreflect.StringKind:   identity,       // identity
		protoreflect.BytesKind:    identity,       // identity
	}
)

// UpdateProviderCommand is the command for enrolling a provider
func UpdateProviderCommand(
	ctx context.Context,
	cmd *cobra.Command,
	_ []string,
	conn *grpc.ClientConn,
) error {
	// TODO: get rid of provider flag, only use class
	providerName := viper.GetString("name")
	if providerName == "" {
		return cli.MessageAndError(
			"missing mandatory flag",
			errMissingProviderName,
		)
	}
	project := viper.GetString("project")
	if project == "" {
		return cli.MessageAndError(
			"missing mandatory flag",
			errMissingProject,
		)
	}

	fields := make(map[string]any)

	setAttrs := viper.GetStringSlice("set-attribute")
	unsetAttrs := viper.GetStringSlice("unset-attribute")

	config := &minderv1.ProviderConfig{}
	for _, attr := range setAttrs {
		// Parameters received from the command line must be
		// of the form <path>=<value>.
		attrName, attrValue, err := parseConfigAttribute(attr)
		if err != nil {
			return cli.MessageAndError(
				"invalid attribute",
				err,
			)
		}

		if err := configAttribute(config.ProtoReflect(), attrName, &attrValue); err != nil {
			return cli.MessageAndError(
				"invalid attribute",
				err,
			)
		}

		// At this point we've set the value via reflection
		// and just have to track the attribute name for the
		// FieldMask.
		fields[attrName] = attrValue
	}

	for _, attr := range unsetAttrs {
		if err := configAttribute(config.ProtoReflect(), attr, nil); err != nil {
			return cli.MessageAndError(
				"invalid attribute",
				err,
			)
		}

		// At this point we've ensured the value is a scalar
		// and is there and just have to track the attribute
		// name for the FieldMask.
		fields[attr] = nil
	}

	fieldMask, err := fieldmaskpb.New(config)
	if err != nil {
		return cli.MessageAndError("invalid configuration", err)
	}
	for attrName := range fields {
		if err := fieldMask.Append(config, attrName); err != nil {
			return cli.MessageAndError(
				"invalid configuration",
				fmt.Errorf("error adding attribute %s", attrName),
			)
		}
	}
	config.UpdateMask = fieldMask

	cfg, err := anypb.New(config)
	if err != nil {
		return cli.MessageAndError("invalid config patch", err)
	}

	req := &minderv1.PatchProviderRequest{
		Context: &minderv1.Context{
			Project:  &project,
			Provider: &providerName,
		},
		Patch: &minderv1.Provider{
			Config: cfg,
		},
	}

	client := minderv1.NewProvidersServiceClient(conn)
	_, err = client.PatchProvider(ctx, req)
	if err != nil {
		return cli.MessageAndError("failed calling minder", err)
	}

	cmd.Println("Provider updated successfully")

	return nil
}

func parseConfigAttribute(attr string) (string, string, error) {
	parts := strings.SplitN(attr, "=", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("can't set attribute %s with no value", attr)
	}
	return parts[0], parts[1], nil
}

func configAttribute(
	config protoreflect.Message,
	attrName string,
	attrValue *string,
) error {
	// Attribute name is meant to be a path traversing struct
	// fields of the form <root>.<field1>.<field2>...
	attrPath := strings.Split(attrName, ".")

	if err := recurConfigAttribute(config, attrPath, attrValue); err != nil {
		return fmt.Errorf("%s is not a valid attribute: %w", attrName, err)
	}

	return nil
}

func recurConfigAttribute(
	config protoreflect.Message,
	path []string,
	attrValue *string,
) error {
	if len(path) == 0 {
		return errors.New("too short")
	}

	fieldName := path[0]
	// We retrieve the field by name. ByTextName and the other
	// functions used to lookup fields return nil in case it does
	// not exist.
	fd := config.Descriptor().
		Fields().
		ByTextName(fieldName)
	// We treat non-existing field lookups as a user errors.
	if fd == nil {
		return fmt.Errorf("config does not have %s field", fieldName)
	}

	// Check here this link for the relation between Go types and
	// Protobuf types
	// https://pkg.go.dev/google.golang.org/protobuf/reflect/protoreflect#Value
	if fd.Kind() == protoreflect.MessageKind {
		return recurConfigAttribute(
			config.Mutable(fd).Message(),
			path[1:],
			attrValue,
		)
	}

	if attrValue != nil {
		parserFunc, found := parserMap[fd.Kind()]
		if !found {
			return fmt.Errorf("field has unexpected kind: %s", fd.Kind())
		}
		v, err := parserFunc(*attrValue)
		if err != nil {
			return fmt.Errorf("expected bool, got %s", *attrValue)
		}
		config.Set(fd, *v)
	}

	return nil
}

func init() {
	ProviderCmd.AddCommand(updateCmd)
	// Flags
	updateCmd.Flags().StringP("name", "n", "", "Name of the provider.")
	updateCmd.Flags().StringP("patch", "", "", "JSON config for the provider.")
	updateCmd.Flags().StringSliceP(
		"set-attribute", "s", []string{},
		"List of attributes to set in the config in <name>=<value> format",
	)
	updateCmd.Flags().StringSliceP(
		"unset-attribute", "u", []string{},
		"List of attributes to unset in the config in <name>=<value> format",
	)
	updateCmd.MarkFlagsMutuallyExclusive("patch", "set-attribute")
	updateCmd.MarkFlagsMutuallyExclusive("patch", "unset-attribute")
}
