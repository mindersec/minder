// SPDX-FileCopyrightText: Copyright 2025 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package datasource

import (
	"fmt"
	"net/http"
	"os"
	"regexp"
	"slices"
	"strings"

	"buf.build/go/protoyaml"
	"github.com/go-openapi/loads"
	"github.com/go-openapi/spec"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/types/known/structpb"

	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

// CmdGenerate returns a cobra command for the 'datasource generate' subcommand.
func CmdGenerate() *cobra.Command {
	var generateCmd = &cobra.Command{
		Use:     "generate",
		Aliases: []string{"gen"},
		Short:   "generate datasource code from an OpenAPI specification",
		Long: `The 'datasource generate' subcommand allows you to generate datasource code from an OpenAPI
specification`,
		RunE:         generateCmdRun,
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
	}

	return generateCmd
}

// parseOpenAPI parses an OpenAPI specification from a byte slice.
func parseOpenAPI(filepath string) (*spec.Swagger, error) {
	doc, err := loads.Spec(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse OpenAPI spec: %w", err)
	}

	return doc.Spec(), nil
}

func initDataSourceStruct(name string) *minderv1.DataSource {
	return &minderv1.DataSource{
		Version: minderv1.VersionV1,
		Type:    "data-source",
		Name:    name,
		Context: &minderv1.ContextV2{},
	}
}

func initDriverStruct() *minderv1.RestDataSource {
	return &minderv1.RestDataSource{
		Def: make(map[string]*minderv1.RestDataSource_Def),
	}
}

// conver the title to a valid datasource name. It should only contain alphanumeric characters and dashes.
func swaggerTitleToDataSourceName(title string) string {
	re := regexp.MustCompile(`[^a-z0-9_-]+`)
	sanitized := strings.ToLower(strings.TrimSpace(title))
	sanitized = re.ReplaceAllString(sanitized, "-")
	sanitized = strings.Trim(sanitized, "-")
	if sanitized == "" {
		return "datasource"
	}
	if sanitized[0] < 'a' || sanitized[0] > 'z' {
		sanitized = "ds-" + sanitized
	}

	return sanitized
}

// swaggerToDataSource generates datasource code from an OpenAPI specification.
func swaggerToDataSource(cmd *cobra.Command, swagger *spec.Swagger) error {
	// Ensure the generator respects Cobra's configured output writer.
	if out := cmd.OutOrStdout(); out != os.Stdout {
		if f, ok := out.(*os.File); ok {
			prev := os.Stdout
			os.Stdout = f
			defer func() {
				os.Stdout = prev
			}()
		}
	}

	if swagger.Info == nil {
		return fmt.Errorf("info section is required in OpenAPI spec")
	}

	ds := initDataSourceStruct(swaggerTitleToDataSourceName(swagger.Info.Title))
	drv := initDriverStruct()
	ds.Driver = &minderv1.DataSource_Rest{Rest: drv}

	// Add the OpenAPI specification to the DataSource
	basepath := swagger.BasePath
	if basepath == "" {
		return fmt.Errorf("base path is required in OpenAPI spec")
	}

	for path, pathItem := range swagger.Paths.Paths {
		p := joinPaths(basepath, path)

		for method, op := range operations(pathItem) {
			opName := generateOpName(method, path)
			if _, ok := drv.Def[opName]; ok {
				return fmt.Errorf("duplicate generated operation name %q for %s %s", opName, method, path)
			}

			if err := validateParameters(op.Parameters); err != nil {
				return fmt.Errorf("%s %s: %w", method, path, err)
			}

			// Create a new REST DataSource definition
			def := &minderv1.RestDataSource_Def{
				Method:   method,
				Endpoint: p,
				// TODO: Make this configurable
				Parse: "json",
			}

			is := paramsToInputSchema(op.Parameters)

			if requiresMsgBody(method) {
				def.Body = &minderv1.RestDataSource_Def_BodyFromField{
					BodyFromField: "body",
				}

				// Add the `body` field to the input schema
				is = inputSchemaForBody(is)
			}

			pbs, err := structpb.NewStruct(is)
			if err != nil {
				return fmt.Errorf("error creating input schema: %w", err)
			}

			def.InputSchema = pbs

			// Add the operation to the DataSource
			drv.Def[opName] = def
		}
	}

	return writeDataSourceToFile(ds)
}

func joinPaths(basepath, path string) string {
	basepath = strings.TrimSuffix(basepath, "/")
	path = strings.TrimPrefix(path, "/")
	if path == "" {
		return basepath
	}

	return basepath + "/" + path
}

// Generates an operation name for a data source. Note that these names
// must be unique within a data source. They also should be only alphanumeric
// characters and underscores
func generateOpName(method, path string) string {
	// Replace all non-alphanumeric characters with underscores
	re := regexp.MustCompile("[^a-zA-Z0-9]+")
	return re.ReplaceAllString(fmt.Sprintf("%s_%s", strings.ToLower(method), strings.ToLower(path)), "_")
}

func operations(p spec.PathItem) map[string]*spec.Operation {
	out := make(map[string]*spec.Operation)
	for mstr, op := range map[string]*spec.Operation{
		http.MethodGet:     p.Get,
		http.MethodPut:     p.Put,
		http.MethodPost:    p.Post,
		http.MethodDelete:  p.Delete,
		http.MethodOptions: p.Options,
		http.MethodHead:    p.Head,
		http.MethodPatch:   p.Patch,
	} {
		if op != nil {
			out[mstr] = op
		}
	}

	return out
}

func requiresMsgBody(method string) bool {
	return method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch
}

func validateParameters(params []spec.Parameter) error {
	for _, p := range params {
		if !slices.Contains([]string{"path", "query"}, p.In) {
			return fmt.Errorf("unsupported parameter %q in %q", p.Name, p.In)
		}
	}

	return nil
}

func paramsToInputSchema(params []spec.Parameter) map[string]any {
	if len(params) == 0 {
		return nil
	}

	is := map[string]any{
		"type":       "object",
		"properties": make(map[string]any),
	}

	for _, p := range params {
		is["properties"].(map[string]any)[p.Name] = map[string]any{
			// TODO: Add support for more types
			"type": "string",
		}

		if p.Required {
			if _, ok := is["required"]; !ok {
				is["required"] = make([]any, 0)
			}

			is["required"] = append(is["required"].([]any), p.Name)
		}
	}

	return is
}

func inputSchemaForBody(is map[string]any) map[string]any {
	if is == nil {
		is = map[string]any{
			"type":       "object",
			"properties": make(map[string]any),
		}
	}

	is["properties"].(map[string]any)["body"] = map[string]any{
		"type": "object",
	}

	return is
}

func writeDataSourceToFile(ds *minderv1.DataSource) error {
	// Convert the DataSource to YAML
	dsYAML, err := protoyaml.MarshalOptions{
		Indent: 2,
	}.Marshal(ds)
	if err != nil {
		return fmt.Errorf("error marshalling DataSource to YAML: %w", err)
	}

	// Write the YAML to a file
	if _, err := os.Stdout.Write(dsYAML); err != nil {
		return fmt.Errorf("error writing DataSource to file: %w", err)
	}

	return nil
}

// generateCmdRun is the entry point for the 'datasource generate' command.
func generateCmdRun(cmd *cobra.Command, args []string) error {
	// We've already validated that there is exactly one argument via the cobra.ExactArgs(1) call
	filePath := args[0]

	// Parse the OpenAPI specification
	swagger, err := parseOpenAPI(filePath)
	if err != nil {
		return fmt.Errorf("error parsing OpenAPI spec: %w", err)
	}

	return swaggerToDataSource(cmd, swagger)
}
