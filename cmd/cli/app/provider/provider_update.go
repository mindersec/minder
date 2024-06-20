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
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/stacklok/minder/internal/util/cli"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

var (
	errMissingProviderName = errors.New("provider name flag is missing")
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Updates a provider's configuration",
	Long: `The minder provider update command allows a user to update a provider's
configuration after enrollement.`,
	RunE: cli.GRPCClientWrapRunE(UpdateProviderCommand),
}

type parser func(string) (*reflect.Value, error)

func parseBool(val string) (*reflect.Value, error) {
	v, err := strconv.ParseBool(val)
	if err != nil {
		return nil, fmt.Errorf("expected bool, got %s", val)
	}
	res := reflect.ValueOf(v)
	return &res, nil
}

func parseInt(size int) parser {
	return func(val string) (*reflect.Value, error) {
		v, err := strconv.ParseInt(val, 10, size)
		if err != nil {
			return nil, fmt.Errorf("expected integer, got %s", val)
		}
		res := reflect.ValueOf(v)
		return &res, nil
	}
}

func parseUint(size int) parser {
	return func(val string) (*reflect.Value, error) {
		v, err := strconv.ParseUint(val, 10, size)
		if err != nil {
			return nil, fmt.Errorf("expected integer, got %s", val)
		}
		res := reflect.ValueOf(v)
		return &res, nil
	}
}

func parseFloat(size int) parser {
	return func(val string) (*reflect.Value, error) {
		v, err := strconv.ParseFloat(val, size)
		if err != nil {
			return nil, fmt.Errorf("expected integer, got %s", val)
		}
		res := reflect.ValueOf(v)
		return &res, nil
	}
}

func identity(val string) (*reflect.Value, error) {
	res := reflect.ValueOf(val)
	return &res, nil
}

var (
	parserMap = map[reflect.Kind]parser{
		reflect.Bool:    parseBool,
		reflect.Int32:   parseInt(32),
		reflect.Int64:   parseInt(64),
		reflect.Uint32:  parseUint(32),
		reflect.Uint64:  parseUint(64),
		reflect.Float32: parseFloat(32),
		reflect.Float64: parseFloat(64), // double
		reflect.String:  identity,       // identity
	}
)

type configStruct struct {
	*minderv1.ProviderConfig
	//nolint:lll
	GitHub *minderv1.GitHubProviderConfig `json:"github,omitempty" yaml:"github" mapstructure:"github" validate:"required"`
	//nolint:lll
	GitHubApp *minderv1.GitHubAppProviderConfig `json:"github_app,omitempty" yaml:"github_app" mapstructure:"github_app" validate:"required"`
}

// UpdateProviderCommand is the command for enrolling a provider
//
//nolint:gocyclo
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
			"invalid option",
			errMissingProviderName,
		)
	}
	project := viper.GetString("project")

	fields := make(map[string]any)

	setAttrs := viper.GetStringSlice("set-attribute")
	unsetAttrs := viper.GetStringSlice("unset-attribute")

	client := minderv1.NewProvidersServiceClient(conn)
	resp, err := client.GetProvider(ctx, &minderv1.GetProviderRequest{
		Context: &minderv1.Context{
			Project: &project,
		},
		Name: providerName,
	})
	if err != nil {
		return cli.MessageAndError("Failed to get provider", err)
	}
	if resp.GetProvider() == nil {
		return cli.MessageAndError(
			"could not retrieve provider",
			errors.New("provider was empty"),
		)
	}

	provider := resp.GetProvider()
	bytes, err := provider.GetConfig().MarshalJSON()
	if err != nil {
		// TODO this is likely to be an internal error and
		// should be mapped to a more suitable user-facing
		// error.
		return cli.MessageAndError(
			"invalid config",
			fmt.Errorf("error marshalling provider config: %w", err),
		)
	}
	serde := &configStruct{}
	if err := json.Unmarshal(bytes, &serde); err != nil {
		// TODO this is likely to be an internal error and
		// should be mapped to a more suitable user-facing
		// error.
		return cli.MessageAndError(
			"invalid config",
			fmt.Errorf("error unmarshalling provider config: %w", err),
		)
	}

	config := serde.ProviderConfig
	if config == nil {
		config = &minderv1.ProviderConfig{}
	}
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

		if err := configAttribute(
			reflect.ValueOf(config),
			attrName,
			&attrValue,
		); err != nil {
			return cli.MessageAndError(
				"invalid attribute",
				err,
			)
		}

		// At this point we've set the value via reflection
		// and just have to track the attribute name for the
		// field mask.
		fields[attrName] = attrValue
	}

	for _, attr := range unsetAttrs {
		if err := configAttribute(
			reflect.ValueOf(config),
			attr,
			nil,
		); err != nil {
			return cli.MessageAndError(
				"invalid attribute",
				err,
			)
		}

		// At this point we've ensured the value is a scalar
		// and is there and just have to track the attribute
		// name for the field mask.
		fields[attr] = nil
	}

	serde.ProviderConfig = config
	var structConfig map[string]any
	bytes, err = json.Marshal(serde)
	if err != nil {
		// TODO this is likely to be an internal error and
		// should be mapped to a more suitable user-facing
		// error.
		return cli.MessageAndError(
			"invalid config",
			err,
		)
	}
	if err := json.Unmarshal(bytes, &structConfig); err != nil {
		// TODO this is likely to be an internal error and
		// should be mapped to a more suitable user-facing
		// error.
		return cli.MessageAndError(
			"invalid configuration",
			err,
		)
	}

	cfg, err := structpb.NewStruct(structConfig)
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
		return "", "", fmt.Errorf("invalid attribute format: %s", attr)
	}
	if parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid attribute format: %s", attr)
	}
	return parts[0], parts[1], nil
}

func configAttribute(
	config reflect.Value,
	attrName string,
	attrValue *string,
) error {
	if config.Kind() != reflect.Pointer {
		return errors.New("config must be passed by reference")
	}

	// Attribute name is meant to be a path traversing struct
	// fields of the form <root>.<field1>.<field2>...
	attrPath := strings.Split(attrName, ".")

	if err := recurConfigAttribute(config, attrPath, attrValue); err != nil {
		return fmt.Errorf("%s is not a valid attribute: %w", attrName, err)
	}

	return nil
}

func recurConfigAttribute(
	config reflect.Value,
	path []string,
	attrValue *string,
) error {
	//nolint:exhaustive
	switch config.Kind() {
	case reflect.Pointer:
		return recurConfigAttribute(
			reflect.Indirect(config),
			path, // we just dereference the pointer
			attrValue,
		)
	case reflect.Struct,
		reflect.Map:
		if len(path) == 0 {
			return errors.New("too short")
		}

		fd, err := next(config, path[0])
		if err != nil {
			return err
		}
		return recurConfigAttribute(
			fd,
			path[1:],
			attrValue,
		)
	case reflect.Bool,
		reflect.Int32,
		reflect.Int64,
		reflect.Uint32,
		reflect.Uint64,
		reflect.Float32,
		reflect.Float64,
		reflect.String:
		if len(path) != 0 {
			return errors.New("too long")
		}
		if attrValue != nil {
			parserFunc, found := parserMap[config.Kind()]
			if !found {
				return fmt.Errorf("field has unexpected kind: %s", config.Kind())
			}
			v, err := parserFunc(*attrValue)
			if err != nil {
				return fmt.Errorf("expected %s, got %s", config.Kind(), *attrValue)
			}
			config.Set(*v)
		} else {
			config.SetZero()
		}
	default:
		return errors.New("invalid config")
	}

	return nil
}

func next(config reflect.Value, fieldName string) (reflect.Value, error) {
	// Here we get the field from the current value.
	fd, err := getField(config, fieldName)
	if err != nil {
		return reflect.ValueOf(nil), err
	}

	// This ensures that the field is correctly initialized with
	// its zero value.
	if !fd.IsValid() {
		return reflect.ValueOf(nil), fmt.Errorf("got invalid field for %s", fieldName)
	}
	if isNilAndSettable(fd) {
		if err := initField(fd); err != nil {
			return reflect.ValueOf(nil), err
		}
	}

	return fd, nil
}

func isNilAndSettable(fd reflect.Value) bool {
	return (fd.Kind() == reflect.Pointer || fd.Kind() == reflect.Map) &&
		fd.IsNil() &&
		fd.CanSet()
}

// getField retrieves the field from the current value managing
// differences between container/indirect types like pointers, structs
// or maps.
//
// Caveat: arrays and slices are not currently managed.
func getField(
	val reflect.Value,
	fieldName string,
) (reflect.Value, error) {
	//nolint:exhaustive
	switch val.Kind() {
	case reflect.Struct:
		return byJSONName(val, fieldName)
	// When val is a Map we don't look up `json` tags and just
	// lookup the field.
	case reflect.Map:
		res := val.MapIndex(reflect.ValueOf(fieldName))
		if !res.IsValid() {
			// Watch out for parentheses, these two if
			// branches produce very different structs.
			//
			// The main branch produces a *something while
			// the second produces just e something.
			if val.Type().Elem().Kind() == reflect.Pointer {
				res = reflect.New(val.Type().Elem().Elem())
			} else {
				// Reflection package has only a few
				// generic creation routines, the main
				// one being `reflect.New` that
				// returns a pointer to an object of
				// the received type, thus the need
				// for a follow-up call to `Elem`.
				res = reflect.New(val.Type().Elem()).Elem()
			}
			val.SetMapIndex(
				reflect.ValueOf(fieldName),
				res,
			)
		}
		return res, nil
	default:
		return reflect.ValueOf(nil), fmt.Errorf("field name %s cannot be configured", fieldName)
	}
}

// byJSONName looks for `fieldName` inside the given `reflect.Value`
// by looking at existing JSON tags.
//
// This is supposed to be called only on `reflect.Value`s of type
// `reflect.Struct`, returns an error otherwise. Additionally, it
// returns an error if the looked up field was not found.
func byJSONName(
	val reflect.Value,
	fieldName string,
) (reflect.Value, error) {
	if val.Type().Kind() != reflect.Struct {
		return reflect.ValueOf(nil), fmt.Errorf(
			"expected struct, got %s",
			val.Type().Kind(),
		)
	}

	for i := 0; i < val.Type().NumField(); i++ {
		t := val.Type().Field(i)

		tag, ok := t.Tag.Lookup("json")
		if ok && tag != "" && tag != "-" {
			parts := strings.Split(tag, ",")
			n := parts[0]
			if n == fieldName {
				return val.FieldByName(t.Name), nil
			}
		}
	}

	return reflect.ValueOf(nil), fmt.Errorf(
		"no such field: %s",
		fieldName,
	)
}

// initField initializes its first argument to its correct zero value
// by means of side effect. It is supposed to be called only on maps,
// structs, and pointers.
func initField(
	fd reflect.Value,
) error {
	//nolint:exhaustive
	switch fd.Kind() {
	case reflect.Pointer:
		fd.Set(reflect.New(fd.Type().Elem()))
	case reflect.Map:
		// Initialize map to non-nil value
		fd.Set(reflect.MakeMap(fd.Type()))
	default:
		return fmt.Errorf("invalid type %s", fd.Kind())
	}

	return nil
}

func init() {
	ProviderCmd.AddCommand(updateCmd)
	// Flags
	updateCmd.Flags().StringP("name", "n", "", "Name of the provider.")
	updateCmd.Flags().StringSliceP(
		"set-attribute", "s", []string{},
		"List of attributes to set in the config in <name>=<value> format",
	)
	updateCmd.Flags().StringSliceP(
		"unset-attribute", "u", []string{},
		"List of attributes to unset in the config in <name> format",
	)
	if err := updateCmd.MarkFlagRequired("name"); err != nil {
		updateCmd.Printf("Error marking flag required: %s", err)
		os.Exit(1)
	}
}
