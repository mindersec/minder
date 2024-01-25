//
// Copyright 2023 Stacklok, Inc.
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

// Package util provides helper functions for the minder CLI.
package util

import (
	"context"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	_ "github.com/signalfx/splunk-otel-go/instrumentation/github.com/lib/pq/splunkpq" // nolint
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/util/jsonyaml"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

var (
	// PyRequestsVersionRegexp is a regexp to match a line in a requirements.txt file, including the package version
	// and the comparison operators
	PyRequestsVersionRegexp = regexp.MustCompile(`\s*(>=|<=|==|>|<|!=)\s*(\d+(\.\d+)*(\*)?)`)
	// PyRequestsNameRegexp is a regexp to match a line in a requirements.txt file, parsing out the package name
	PyRequestsNameRegexp = regexp.MustCompile(`\s*(>=|<=|==|>|<|!=)`)
)

// OpenIdCredentials is a struct to hold the access and refresh tokens
type OpenIdCredentials struct {
	AccessToken          string    `json:"access_token"`
	RefreshToken         string    `json:"refresh_token"`
	AccessTokenExpiresAt time.Time `json:"expiry"`
}

func getCredentialsPath() (string, error) {
	// Get the XDG_CONFIG_HOME environment variable
	xdgConfigHome := os.Getenv("XDG_CONFIG_HOME")

	// If XDG_CONFIG_HOME is not set or empty, use $HOME/.config as the base directory
	if xdgConfigHome == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("error getting home directory: %v", err)
		}
		xdgConfigHome = filepath.Join(homeDir, ".config")
	}

	filePath := filepath.Join(xdgConfigHome, "minder", "credentials.json")
	return filePath, nil
}

// JWTTokenCredentials is a helper struct for grpc
type JWTTokenCredentials struct {
	accessToken string
}

// GetRequestMetadata implements the PerRPCCredentials interface.
func (jwt JWTTokenCredentials) GetRequestMetadata(_ context.Context, _ ...string) (map[string]string, error) {
	return map[string]string{
		"authorization": "Bearer " + string(jwt.accessToken),
	}, nil
}

// RequireTransportSecurity implements the PerRPCCredentials interface.
func (JWTTokenCredentials) RequireTransportSecurity() bool {
	return false
}

// GetGrpcConnection is a helper for getting a testing connection for grpc
func GetGrpcConnection(
	grpc_host string, grpc_port int,
	allowInsecure bool,
	issuerUrl string, clientId string,
	opts ...grpc.DialOption) (
	*grpc.ClientConn, error) {
	address := fmt.Sprintf("%s:%d", grpc_host, grpc_port)

	// read credentials
	token := ""
	t, err := GetToken(issuerUrl, clientId)
	if err == nil {
		token = t
	}

	credentialOpts := credentials.NewTLS(&tls.Config{MinVersion: tls.VersionTLS13})
	if allowInsecure {
		credentialOpts = insecure.NewCredentials()
	}

	dialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(credentialOpts),
		grpc.WithPerRPCCredentials(JWTTokenCredentials{accessToken: token}),
	}
	dialOpts = append(dialOpts, opts...)

	// generate credentials
	conn, err := grpc.Dial(address, dialOpts...)
	if err != nil {
		return nil, fmt.Errorf("error connecting to gRPC server: %v", err)
	}

	return conn, nil
}

// TestWriter is a helper struct for testing
type TestWriter struct {
	Output string
}

func (tw *TestWriter) Write(p []byte) (n int, err error) {
	tw.Output += string(p)
	return len(p), nil
}

// SaveCredentials saves the credentials to a file
func SaveCredentials(tokens OpenIdCredentials) (string, error) {
	// marshal the credentials to json
	credsJSON, err := json.Marshal(tokens)
	if err != nil {
		return "", fmt.Errorf("error marshaling credentials: %v", err)
	}

	filePath, err := getCredentialsPath()
	if err != nil {
		return "", fmt.Errorf("error getting credentials path: %v", err)
	}

	err = os.MkdirAll(filepath.Dir(filePath), 0750)
	if err != nil {
		return "", fmt.Errorf("error creating directory: %v", err)
	}

	// Write the JSON data to the file
	err = os.WriteFile(filePath, credsJSON, 0600)
	if err != nil {
		return "", fmt.Errorf("error writing credentials to file: %v", err)
	}
	return filePath, nil
}

// RemoveCredentials removes the local credentials file
func RemoveCredentials() error {
	// remove credentials file
	xdgConfigHome := os.Getenv("XDG_CONFIG_HOME")

	// just delete token from credentials file
	if xdgConfigHome == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("error getting home directory: %v", err)
		}
		xdgConfigHome = filepath.Join(homeDir, ".config")
	}

	filePath := filepath.Join(xdgConfigHome, "minder", "credentials.json")
	err := os.Remove(filePath)
	if err != nil {
		return fmt.Errorf("error removing credentials file: %v", err)
	}
	return nil
}

// GetToken retrieves the access token from the credentials file and refreshes it if necessary
func GetToken(issuerUrl string, clientId string) (string, error) {
	refreshLimit := 10
	creds, err := LoadCredentials()
	if err != nil {
		return "", fmt.Errorf("error loading credentials: %v", err)
	}
	needsRefresh := time.Now().Add(time.Duration(refreshLimit) * time.Second).After(creds.AccessTokenExpiresAt)

	if needsRefresh {
		updatedCreds, err := RefreshCredentials(creds.RefreshToken, issuerUrl, clientId)
		if err != nil {
			return "", fmt.Errorf("error refreshing credentials: %v", err)
		}
		return updatedCreds.AccessToken, nil
	}

	return creds.AccessToken, nil
}

type refreshTokenResponse struct {
	AccessToken          string `json:"access_token"`
	RefreshToken         string `json:"refresh_token"`
	AccessTokenExpiresIn int    `json:"expires_in"`
}

// RefreshCredentials uses a refresh token to get and save a new set of credentials
func RefreshCredentials(refreshToken string, issuerUrl string, clientId string) (OpenIdCredentials, error) {

	parsedURL, err := url.Parse(issuerUrl)
	if err != nil {
		return OpenIdCredentials{}, fmt.Errorf("error parsing issuer URL: %v", err)
	}
	logoutUrl := parsedURL.JoinPath("realms/stacklok/protocol/openid-connect/token")

	data := url.Values{}
	data.Set("client_id", clientId)
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)

	req, err := http.NewRequest("POST", logoutUrl.String(), strings.NewReader(data.Encode()))
	if err != nil {
		return OpenIdCredentials{}, fmt.Errorf("error creating: %v", err)
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return OpenIdCredentials{}, fmt.Errorf("error fetching new credentials: %v", err)
	}
	defer resp.Body.Close()

	tokens := refreshTokenResponse{}
	err = json.NewDecoder(resp.Body).Decode(&tokens)
	if err != nil {
		return OpenIdCredentials{}, fmt.Errorf("error unmarshaling credentials: %v", err)
	}

	updatedCredentials := OpenIdCredentials{
		AccessToken:          tokens.AccessToken,
		RefreshToken:         tokens.RefreshToken,
		AccessTokenExpiresAt: time.Now().Add(time.Duration(tokens.AccessTokenExpiresIn) * time.Second),
	}
	_, err = SaveCredentials(updatedCredentials)
	if err != nil {
		return OpenIdCredentials{}, fmt.Errorf("error saving credentials: %v", err)
	}

	return updatedCredentials, nil
}

// LoadCredentials loads the credentials from a file
func LoadCredentials() (OpenIdCredentials, error) {
	filePath, err := getCredentialsPath()
	if err != nil {
		return OpenIdCredentials{}, fmt.Errorf("error getting credentials path: %v", err)
	}

	// Read the file
	credsJSON, err := os.ReadFile(filepath.Clean(filePath))
	if err != nil {
		return OpenIdCredentials{}, fmt.Errorf("error reading credentials file: %v", err)
	}

	var creds OpenIdCredentials
	err = json.Unmarshal(credsJSON, &creds)
	if err != nil {
		return OpenIdCredentials{}, fmt.Errorf("error unmarshaling credentials: %v", err)
	}
	return creds, nil
}

func getProtoMarshalOptions() protojson.MarshalOptions {
	return protojson.MarshalOptions{
		Multiline: true,
		Indent:    "  ",
	}

}

// GetJsonFromProto given a proto message, formats into json
func GetJsonFromProto(msg protoreflect.ProtoMessage) (string, error) {
	m := getProtoMarshalOptions()
	out, err := m.Marshal(msg)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// GetYamlFromProto given a proto message, formats into yaml
func GetYamlFromProto(msg protoreflect.ProtoMessage) (string, error) {
	// first converts into json using the marshal options
	m := getProtoMarshalOptions()
	out, err := m.Marshal(msg)
	if err != nil {
		return "", err
	}

	// from byte, we get the raw message so we can convert into yaml
	var rawMsg json.RawMessage
	err = json.Unmarshal(out, &rawMsg)
	if err != nil {
		return "", err
	}
	yamlResult, err := jsonyaml.ConvertJsonToYaml(rawMsg)
	if err != nil {
		return "", err
	}
	return yamlResult, nil
}

// GetBytesFromProto given a proto message, formats into bytes
func GetBytesFromProto(message protoreflect.ProtoMessage) ([]byte, error) {
	m := getProtoMarshalOptions()
	return m.Marshal(message)
}

// OpenFileArg opens a file argument and returns a descriptor, closer, and error
// If the file is "-", it will return whatever is passed in as dashOpen and a no-op closer
func OpenFileArg(f string, dashOpen io.Reader) (desc io.Reader, closer func(), err error) {
	if f == "-" {
		desc = dashOpen
		closer = func() {}
		return desc, closer, nil
	}

	f = filepath.Clean(f)
	ftemp, err := os.Open(f)
	if err != nil {
		return nil, nil, fmt.Errorf("error opening file: %w", err)
	}

	closer = func() {
		err := ftemp.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error closing file: %v\n", err)
		}
	}

	desc = ftemp
	return desc, closer, nil
}

// ExpandFileArgs expands a list of file arguments into a list of files.
// If the file list contains "-" or regular files, it will leave them as-is.
// If the file list contains directories, it will expand them into a list of files.
func ExpandFileArgs(files []string) ([]string, error) {
	var expandedFiles []string
	for _, f := range files {
		if f == "-" {
			expandedFiles = append(expandedFiles, f)
			continue
		}
		f = filepath.Clean(f)
		fi, err := os.Stat(f)
		if err != nil {
			return nil, fmt.Errorf("error getting file info: %w", err)
		}

		if fi.IsDir() {
			// expand directory
			err := filepath.Walk(f, func(path string, info fs.FileInfo, err error) error {
				if err != nil {
					return fmt.Errorf("error walking directory: %w", err)
				}

				if !info.IsDir() {
					expandedFiles = append(expandedFiles, path)
				}

				return nil
			})
			if err != nil {
				return nil, fmt.Errorf("error walking directory: %w", err)
			}
		} else {
			// add file
			expandedFiles = append(expandedFiles, f)
		}
	}

	return expandedFiles, nil
}

// Int32FromString converts a string to an int32
func Int32FromString(v string) (int32, error) {
	if v == "" {
		return 0, fmt.Errorf("cannot convert empty string to int")
	}

	// convert string to int
	asInt32, err := strconv.ParseInt(v, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("error converting string to int: %w", err)
	}
	return int32(asInt32), nil
}

// PBRepositoryFromDB converts a database repository to a protobuf
// Note this doesn't set the context as that's assumed to be set
// on the caller's side.
func PBRepositoryFromDB(dbrepo db.Repository) *minderv1.Repository {
	strRepoID := dbrepo.ID.String()
	return &minderv1.Repository{
		Id:            &strRepoID,
		Owner:         dbrepo.RepoOwner,
		Name:          dbrepo.RepoName,
		RepoId:        dbrepo.RepoID,
		IsPrivate:     dbrepo.IsPrivate,
		IsFork:        dbrepo.IsFork,
		HookUrl:       dbrepo.WebhookUrl,
		DeployUrl:     dbrepo.DeployUrl,
		CloneUrl:      dbrepo.CloneUrl,
		DefaultBranch: dbrepo.DefaultBranch.String,
		CreatedAt:     timestamppb.New(dbrepo.CreatedAt),
		UpdatedAt:     timestamppb.New(dbrepo.UpdatedAt),
	}
}

// GetRepository retrieves a repository from the database
// and converts it to a protobuf
func GetRepository(ctx context.Context, store db.ExtendQuerier, repoID uuid.UUID) (*minderv1.Repository, error) {
	dbrepo, err := store.GetRepositoryByID(ctx, repoID)
	if err != nil {
		return nil, fmt.Errorf("error getting repository: %w", err)
	}

	return PBRepositoryFromDB(dbrepo), nil
}

// GetArtifactWithVersions retrieves an artifact and its versions from the database
func GetArtifactWithVersions(
	ctx context.Context, store db.ExtendQuerier, repoID, artifactID uuid.UUID) (*minderv1.Artifact, error) {
	// Get repository data - we need the owner and name
	dbrepo, err := store.GetRepositoryByID(ctx, repoID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("repository not found")
	} else if err != nil {
		return nil, fmt.Errorf("cannot read repository: %v", err)
	}

	// Retrieve artifact details
	artifact, err := store.GetArtifactByID(ctx, artifactID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("artifact not found")
	} else if err != nil {
		return nil, fmt.Errorf("failed to get artifact: %v", err)
	}

	// Get its versions
	dbArtifactVersions, err := store.ListArtifactVersionsByArtifactID(ctx, db.ListArtifactVersionsByArtifactIDParams{
		ArtifactID: artifact.ID,
		Limit:      sql.NullInt32{Valid: false},
	})
	if err != nil {
		log.Printf("error getting artifact versions for artifact %s: %v", artifact.ID, err)
	}

	// Translate each to protobuf so we can publish the event
	var listArtifactVersions []*minderv1.ArtifactVersion
	for _, dbVersion := range dbArtifactVersions {
		var tags []string
		if dbVersion.Tags.Valid {
			tags = strings.Split(dbVersion.Tags.String, ",")
		}
		sigVer := &minderv1.SignatureVerification{}
		if dbVersion.SignatureVerification.Valid {
			if err := protojson.Unmarshal(dbVersion.SignatureVerification.RawMessage, sigVer); err != nil {
				log.Printf("error unmarshalling signature verification: %v", err)
				continue
			}
		}
		ghWorkflow := &minderv1.GithubWorkflow{}
		if dbVersion.GithubWorkflow.Valid {
			if err := protojson.Unmarshal(dbVersion.GithubWorkflow.RawMessage, ghWorkflow); err != nil {
				log.Printf("error unmarshalling gh workflow: %v", err)
				continue
			}
		}
		listArtifactVersions = append(listArtifactVersions, &minderv1.ArtifactVersion{
			VersionId: dbVersion.Version,
			Tags:      tags,
			Sha:       dbVersion.Sha,
			CreatedAt: timestamppb.New(dbVersion.CreatedAt),
		})
	}

	// Build the artifact protobuf
	return &minderv1.Artifact{
		ArtifactPk: artifact.ID.String(),
		Owner:      dbrepo.RepoOwner,
		Name:       artifact.ArtifactName,
		Type:       artifact.ArtifactType,
		Visibility: artifact.ArtifactVisibility,
		Repository: dbrepo.RepoName,
		Versions:   listArtifactVersions,
		CreatedAt:  timestamppb.New(artifact.CreatedAt),
	}, nil
}

// GetPullRequest retrieves a pull request from the database
// and converts it to a protobuf
func GetPullRequest(
	ctx context.Context,
	store db.ExtendQuerier,
	repoID, pullRequestID uuid.UUID,
) (*minderv1.PullRequest, error) {
	// Get repository data - we need the owner and name
	dbrepo, err := store.GetRepositoryByID(ctx, repoID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("repository not found")
	} else if err != nil {
		return nil, fmt.Errorf("cannot read repository: %v", err)
	}

	dbpr, err := store.GetPullRequestByID(ctx, pullRequestID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("pull request not found")
	} else if err != nil {
		return nil, fmt.Errorf("cannot read pull request: %v", err)
	}

	// TODO: Do we need extra columns in the pull request table?
	return &minderv1.PullRequest{
		Number:    int32(dbpr.PrNumber), // TODO: this should be int64
		RepoOwner: dbrepo.RepoOwner,
		RepoName:  dbrepo.RepoName,
	}, nil
}
