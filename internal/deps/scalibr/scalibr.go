// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package scalibr implements a dependency extractor using the osv-scalibr
// library.
package scalibr

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"reflect"
	"slices"
	"time"

	scalibr "github.com/google/osv-scalibr"
	scalibr_cpb "github.com/google/osv-scalibr/binary/proto/config_go_proto"
	"github.com/google/osv-scalibr/extractor"
	scalibr_fs "github.com/google/osv-scalibr/fs"
	scalibr_plugin "github.com/google/osv-scalibr/plugin"
	scalibr_config "github.com/google/osv-scalibr/plugin/config"
	"github.com/google/osv-scalibr/plugin/list"
	"github.com/google/osv-scalibr/stats"
	"github.com/google/uuid"
	"github.com/protobom/protobom/pkg/sbom"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
)

// Extractor is a dependency extractor based on osv-scalibr.
type Extractor struct {
}

// NewExtractor creates a new scalibr dependency extractor
func NewExtractor() *Extractor {
	return &Extractor{}
}

// ScanFilesystem takes
func (*Extractor) ScanFilesystem(ctx context.Context, iofs fs.FS) (*sbom.NodeList, error) {
	return scanFilesystem(ctx, iofs)
}

func scanFilesystem(ctx context.Context, iofs fs.FS) (*sbom.NodeList, error) {
	if iofs == nil {
		return nil, errors.New("unable to scan dependencies, no filesystem")
	}
	// have to down-cast here, because scalibr needs multiple io/fs types
	wrapped, ok := iofs.(scalibr_fs.FS)
	if !ok {
		return nil, errors.New("error converting filesystem to ReadDirFS")
	}

	desiredCaps := scalibr_plugin.Capabilities{
		OS:            scalibr_plugin.OSLinux,
		Network:       scalibr_plugin.NetworkOffline, // Don't fetch over the network, as we may be running in a trusted context.
		DirectFS:      false,
		RunningSystem: false,
	}

	// TODO: it's unfortunate that scalibr spills files to disk.  File an upstream bug?
	// NOTE: since we require NetworkOffline, we may not actually download anything...
	tmpDir, err := os.MkdirTemp("", "minder-scalibr-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary scalibr directory: %w", err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()
	cfg := scalibr_config.PluginConfig{
		ProtoConfig: &scalibr_cpb.PluginConfig{
			MaxFileSizeBytes:  1024 * 1024,
			LocalRegistry:     tmpDir,
			DisableGoogleAuth: true,
		},
		ClientFactories: &DisabledClientFactory{},
	}

	scalibrFs := scalibr_fs.ScanRoot{FS: wrapped}
	plugins, err := list.FromCapabilities(&desiredCaps, &cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to assemble plugins list: %w", err)
	}
	// unknownbinariesextr uses file extension to determine "binary-ness", and triggers on e.g. .py files
	skipPlugins := []string{"ffa/unknownbinariesextr"}
	plugins = slices.DeleteFunc(plugins, func(p scalibr_plugin.Plugin) bool {
		return slices.Contains(skipPlugins, p.Name())
	})
	// Ugly way to get statistics from each plugin, see https://github.com/google/osv-scalibr/issues/2316
	errStats := ErrorStats{}
	PatchExtractorStats(plugins, &errStats)
	scanConfig := scalibr.ScanConfig{
		ScanRoots:    []*scalibr_fs.ScanRoot{&scalibrFs},
		Plugins:      plugins,
		Capabilities: &desiredCaps,
	}

	scanner := scalibr.New()
	scanResults := scanner.Scan(ctx, &scanConfig)

	if scanResults == nil || scanResults.Status == nil {
		return nil, fmt.Errorf("error scanning files: no results")
	}
	switch scanResults.Status.Status {
	case scalibr_plugin.ScanStatusSucceeded:
		// success, continue
	case scalibr_plugin.ScanStatusPartiallySucceeded:
		// Scalibr runs a lot of plugins and aggregates the result.  Some of these are picky, and
		// fail for random reasons.  Accept partial success, but log the failing plugins.
		known_bad := []string{}
		for _, ps := range scanResults.PluginStatus {
			if ps.Status.Status != scalibr_plugin.ScanStatusSucceeded {
				if !slices.Contains(known_bad, ps.Name) {
					zerolog.Ctx(ctx).Warn().Str("plugin", ps.Name).Str("status", ps.Status.FailureReason).
						Msg("Scalibr plugin failed")
				}
			}
		}
	case scalibr_plugin.ScanStatusUnspecified, scalibr_plugin.ScanStatusFailed:
		fallthrough
	default:
		return nil, fmt.Errorf("error scanning files: %s", scanResults.Status)
	}

	// Log skipped files for server-side telemetry to adjust the 1MB limit.
	// We don't return anything to clients, sorry!
	for _, statErr := range errStats.Errs {
		zerolog.Ctx(ctx).Info().
			Str("plugin", statErr.Plugin).Str("path", statErr.Path).Str("res", string(statErr.Result)).
			Msg("Scalibr require warning on file")
	}

	res := sbom.NewNodeList()
	for _, inv := range scanResults.Inventory.Packages {
		res.AddNode(nodeFromPackage(inv))
	}

	return res, nil
}

func nodeFromPackage(inv *extractor.Package) *sbom.Node {
	// TODO: use repo and commit from inv.SourceCode
	node := &sbom.Node{
		Type:        sbom.Node_PACKAGE,
		Id:          uuid.New().String(),
		Name:        inv.Name,
		Version:     inv.Version,
		Identifiers: map[int32]string{
			// TODO: scalibr returns a _list_ of CPEs, but protobom will store one.
			// use the first?
			// int32(sbom.SoftwareIdentifierType_CPE23):  inv.Extractor.ToCPEs(inv),
		},
	}
	if inv.PURL() != nil {
		node.Identifiers[int32(sbom.SoftwareIdentifierType_PURL)] = inv.PURL().String()
	}
	if inv.Location.Descriptor.PathOrEmpty() != "" {
		node.Properties = append(node.Properties, &sbom.Property{
			Name: "sourceFile",
			Data: inv.Location.Descriptor.PathOrEmpty(),
			// TODO: add Descriptor.File.LineNumber if available
		})
	}
	for _, l := range inv.Location.Related {
		node.Properties = append(node.Properties, &sbom.Property{
			Name: "sourceFile",
			Data: l.PathOrEmpty(),
		})
	}
	return node
}

// PatchExtractorStats monkey-patches each plugin with stats.Collector,
// as Scalibr does not provide a nice interface for setting the collector
// which almost every plugin exposes. See https://github.com/google/osv-scalibr/issues/2316
func PatchExtractorStats(plugins []scalibr_plugin.Plugin, collector stats.Collector) {
	for _, p := range plugins {
		v := reflect.ValueOf(p)
		if v.Kind() != reflect.Ptr || v.IsNil() {
			continue
		}
		elem := v.Elem()
		if elem.Kind() != reflect.Struct {
			continue
		}
		statsField := elem.FieldByName("Stats")
		if !statsField.IsValid() || !statsField.CanSet() {
			continue
		}
		collectorVal := reflect.ValueOf(collector)
		if collectorVal.Type().AssignableTo(statsField.Type()) {
			statsField.Set(collectorVal)
			continue
		}
		if collectorVal.Type().Implements(statsField.Type()) {
			statsField.Set(collectorVal)
		}
	}
}

var _ stats.Collector = (*ErrorStats)(nil)

// StatErr contains plugin error reports (via the stats module) for specific paths
// which were unable to be processed.
type StatErr struct {
	Plugin string
	Path   string
	Result string
}

// ErrorStats is a scalibr stat collector for monitoring cases where
// (for example) files are too large to be scanned.
type ErrorStats struct {
	Errs   []StatErr
	maxRSS int64
}

// AfterDetectorRun implements [stats.Collector].
func (*ErrorStats) AfterDetectorRun(string, time.Duration, error) {}

// AfterExtractorRun implements [stats.Collector].
func (*ErrorStats) AfterExtractorRun(string, *stats.AfterExtractorStats) {
}

// AfterFileExtracted implements [stats.Collector].
func (e *ErrorStats) AfterFileExtracted(pluginName string, filestats *stats.FileExtractedStats) {
	if filestats.Result != stats.FileExtractedResultSuccess {
		e.Errs = append(e.Errs, StatErr{pluginName, filestats.Path, string(filestats.Result)})
	}
}

// AfterFileRequired implements [stats.Collector].
func (e *ErrorStats) AfterFileRequired(pluginName string, filestats *stats.FileRequiredStats) {
	if filestats.Result != stats.FileRequiredResultOK {
		e.Errs = append(e.Errs, StatErr{pluginName, filestats.Path, string(filestats.Result)})
	}
}

// AfterInodeVisited implements [stats.Collector].
func (*ErrorStats) AfterInodeVisited(string) {}

// AfterResultsExported implements [stats.Collector].
func (*ErrorStats) AfterResultsExported(string, int, error) {}

// AfterScan implements [stats.Collector].
func (*ErrorStats) AfterScan(time.Duration, *scalibr_plugin.ScanStatus) {}

// MaxRSS implements [stats.Collector].
func (e *ErrorStats) MaxRSS(maxRSS int64) {
	e.maxRSS = maxRSS
}

// DisabledClientFactory is a scalibr ClientFactory that produces network clients
// which exist but always throw errors when used.  (Non-existent clients cause
// plugin initialization to fail: https://github.com/google/osv-scalibr/issues/2377)
type DisabledClientFactory struct{}

var _ scalibr_config.ClientFactories = (*DisabledClientFactory)(nil)
var _ http.RoundTripper = (*DisabledClientFactory)(nil)
var _ grpc.ClientConnInterface = (*DisabledClientFactory)(nil)

// GRPCClientConn implements [config.ClientFactories].
func (d *DisabledClientFactory) GRPCClientConn(string, ...grpc.DialOption) (grpc.ClientConnInterface, error) {
	return d, nil
}

// GoogleHTTPClient implements [config.ClientFactories].
func (*DisabledClientFactory) GoogleHTTPClient(context.Context, ...string) (*http.Client, error) {
	return nil, errors.New("network access is prohibited")
}

// HTTPClient implements [config.ClientFactories].
// This needs to return non-nil for
func (d *DisabledClientFactory) HTTPClient() *http.Client {
	return &http.Client{Transport: d}
}

// RoundTrip implements [http.RoundTripper].
func (*DisabledClientFactory) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("network access is prohibited")
}

// Invoke implements [grpc.ClientConnInterface].
func (*DisabledClientFactory) Invoke(context.Context, string, any, any, ...grpc.CallOption) error {
	return errors.New("network access is prohibited")
}

// NewStream implements [grpc.ClientConnInterface].
func (*DisabledClientFactory) NewStream(
	context.Context, *grpc.StreamDesc, string, ...grpc.CallOption) (grpc.ClientStream, error) {
	return nil, errors.New("network access is prohibited")
}
