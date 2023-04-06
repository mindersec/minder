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

package attestation

import (
	"time"

	cjson "github.com/docker/go/canonical/json"
)

// Provenance is a wrapper around dsse.Envelope
type Envelope struct {
	PayloadType string      `json:"payloadType"`
	Payload     string      `json:"payload"`
	Signatures  []Signature `json:"signatures"`
}

type Signature struct {
	KeyID string `json:"keyid"`
	Sig   string `json:"sig"`
	Cert  string `json:"cert"`
}

type PayLoadSLSA struct {
	Type          string `json:"_type"`
	PredicateType string `json:"predicateType"`
	Subject       []struct {
		Name   string `json:"name"`
		Digest struct {
			Sha256 string `json:"sha256"`
		} `json:"digest"`
	} `json:"subject"`
	Predicate struct {
		Builder struct {
			ID string `json:"id"`
		} `json:"builder"`
		BuildType  string `json:"buildType"`
		Invocation struct {
			ConfigSource struct {
				URI    string `json:"uri"`
				Digest struct {
					Sha1 string `json:"sha1"`
				} `json:"digest"`
				EntryPoint string `json:"entryPoint"`
			} `json:"configSource"`
			Parameters  struct{} `json:"parameters"`
			Environment struct {
				Arch               string `json:"arch"`
				GithubActor        string `json:"github_actor"`
				GithubActorID      string `json:"github_actor_id"`
				GithubBaseRef      string `json:"github_base_ref"`
				GithubEventName    string `json:"github_event_name"`
				GithubEventPayload struct {
					After      string        `json:"after"`
					BaseRef    string        `json:"base_ref"`
					Before     string        `json:"before"`
					Commits    []interface{} `json:"commits"`
					Compare    string        `json:"compare"`
					Created    bool          `json:"created"`
					Deleted    bool          `json:"deleted"`
					Forced     bool          `json:"forced"`
					HeadCommit struct {
						Author struct {
							Email    string `json:"email"`
							Name     string `json:"name"`
							Username string `json:"username"`
						} `json:"author"`
						Committer struct {
							Email    string `json:"email"`
							Name     string `json:"name"`
							Username string `json:"username"`
						} `json:"committer"`
						Distinct  bool   `json:"distinct"`
						ID        string `json:"id"`
						Message   string `json:"message"`
						Timestamp string `json:"timestamp"`
						TreeID    string `json:"tree_id"`
						URL       string `json:"url"`
					} `json:"head_commit"`
					Organization struct {
						AvatarURL        string `json:"avatar_url"`
						Description      string `json:"description"`
						EventsURL        string `json:"events_url"`
						HooksURL         string `json:"hooks_url"`
						ID               int    `json:"id"`
						IssuesURL        string `json:"issues_url"`
						Login            string `json:"login"`
						MembersURL       string `json:"members_url"`
						NodeID           string `json:"node_id"`
						PublicMembersURL string `json:"public_members_url"`
						ReposURL         string `json:"repos_url"`
						URL              string `json:"url"`
					} `json:"organization"`
					Pusher struct {
						Email string `json:"email"`
						Name  string `json:"name"`
					} `json:"pusher"`
					Ref        string `json:"ref"`
					Repository struct {
						AllowForking     bool        `json:"allow_forking"`
						ArchiveURL       string      `json:"archive_url"`
						Archived         bool        `json:"archived"`
						AssigneesURL     string      `json:"assignees_url"`
						BlobsURL         string      `json:"blobs_url"`
						BranchesURL      string      `json:"branches_url"`
						CloneURL         string      `json:"clone_url"`
						CollaboratorsURL string      `json:"collaborators_url"`
						CommentsURL      string      `json:"comments_url"`
						CommitsURL       string      `json:"commits_url"`
						CompareURL       string      `json:"compare_url"`
						ContentsURL      string      `json:"contents_url"`
						ContributorsURL  string      `json:"contributors_url"`
						CreatedAt        time.Time   `json:"created_at"`
						DefaultBranch    string      `json:"default_branch"`
						DeploymentsURL   string      `json:"deployments_url"`
						Description      interface{} `json:"description"`
						Disabled         bool        `json:"disabled"`
						DownloadsURL     string      `json:"downloads_url"`
						EventsURL        string      `json:"events_url"`
						Fork             bool        `json:"fork"`
						Forks            int         `json:"forks"`
						ForksCount       int         `json:"forks_count"`
						ForksURL         string      `json:"forks_url"`
						FullName         string      `json:"full_name"`
						GitCommitsURL    string      `json:"git_commits_url"`
						GitRefsURL       string      `json:"git_refs_url"`
						GitTagsURL       string      `json:"git_tags_url"`
						GitURL           string      `json:"git_url"`
						HasDownloads     bool        `json:"has_downloads"`
						HasIssues        bool        `json:"has_issues"`
						HasPages         bool        `json:"has_pages"`
						HasProjects      bool        `json:"has_projects"`
						HasWiki          bool        `json:"has_wiki"`
						Homepage         interface{} `json:"homepage"`
						HooksURL         string      `json:"hooks_url"`
						HTMLURL          string      `json:"html_url"`
						ID               int         `json:"id"`
						IsTemplate       bool        `json:"is_template"`
						IssueCommentURL  string      `json:"issue_comment_url"`
						IssueEventsURL   string      `json:"issue_events_url"`
						IssuesURL        string      `json:"issues_url"`
						KeysURL          string      `json:"keys_url"`
						LabelsURL        string      `json:"labels_url"`
						Language         string      `json:"language"`
						LanguagesURL     string      `json:"languages_url"`
						License          struct {
							Key    string `json:"key"`
							Name   string `json:"name"`
							NodeID string `json:"node_id"`
							SpdxID string `json:"spdx_id"`
							URL    string `json:"url"`
						} `json:"license"`
						MasterBranch     string      `json:"master_branch"`
						MergesURL        string      `json:"merges_url"`
						MilestonesURL    string      `json:"milestones_url"`
						MirrorURL        interface{} `json:"mirror_url"`
						Name             string      `json:"name"`
						NodeID           string      `json:"node_id"`
						NotificationsURL string      `json:"notifications_url"`
						OpenIssues       int         `json:"open_issues"`
						OpenIssuesCount  int         `json:"open_issues_count"`
						Organization     string      `json:"organization"`
						Owner            struct {
							AvatarURL         string      `json:"avatar_url"`
							Email             interface{} `json:"email"`
							EventsURL         string      `json:"events_url"`
							FollowersURL      string      `json:"followers_url"`
							FollowingURL      string      `json:"following_url"`
							GistsURL          string      `json:"gists_url"`
							GravatarID        string      `json:"gravatar_id"`
							HTMLURL           string      `json:"html_url"`
							ID                int         `json:"id"`
							Login             string      `json:"login"`
							Name              string      `json:"name"`
							NodeID            string      `json:"node_id"`
							OrganizationsURL  string      `json:"organizations_url"`
							ReceivedEventsURL string      `json:"received_events_url"`
							ReposURL          string      `json:"repos_url"`
							SiteAdmin         bool        `json:"site_admin"`
							StarredURL        string      `json:"starred_url"`
							SubscriptionsURL  string      `json:"subscriptions_url"`
							Type              string      `json:"type"`
							URL               string      `json:"url"`
						} `json:"owner"`
						Private                  bool          `json:"private"`
						PullsURL                 string        `json:"pulls_url"`
						PushedAt                 time.Time     `json:"pushed_at"`
						ReleasesURL              string        `json:"releases_url"`
						Size                     int           `json:"size"`
						SSHURL                   string        `json:"ssh_url"`
						Stargazers               int           `json:"stargazers"`
						StargazersCount          int           `json:"stargazers_count"`
						StargazersURL            string        `json:"stargazers_url"`
						StatusesURL              string        `json:"statuses_url"`
						SubscribersURL           string        `json:"subscribers_url"`
						SubscriptionURL          string        `json:"subscription_url"`
						SvnURL                   string        `json:"svn_url"`
						TagsURL                  string        `json:"tags_url"`
						TeamsURL                 string        `json:"teams_url"`
						Topics                   []interface{} `json:"topics"`
						TreesURL                 string        `json:"trees_url"`
						UpdatedAt                time.Time     `json:"updated_at"`
						URL                      string        `json:"url"`
						Visibility               string        `json:"visibility"`
						Watchers                 int           `json:"watchers"`
						WatchersCount            int           `json:"watchers_count"`
						WebCommitSignoffRequired bool          `json:"web_commit_signoff_required"`
					} `json:"repository"`
					Sender struct {
						AvatarURL         string `json:"avatar_url"`
						EventsURL         string `json:"events_url"`
						FollowersURL      string `json:"followers_url"`
						FollowingURL      string `json:"following_url"`
						GistsURL          string `json:"gists_url"`
						GravatarID        string `json:"gravatar_id"`
						HTMLURL           string `json:"html_url"`
						ID                int    `json:"id"`
						Login             string `json:"login"`
						NodeID            string `json:"node_id"`
						OrganizationsURL  string `json:"organizations_url"`
						ReceivedEventsURL string `json:"received_events_url"`
						ReposURL          string `json:"repos_url"`
						SiteAdmin         bool   `json:"site_admin"`
						StarredURL        string `json:"starred_url"`
						SubscriptionsURL  string `json:"subscriptions_url"`
						Type              string `json:"type"`
						URL               string `json:"url"`
					} `json:"sender"`
				} `json:"github_event_payload"`
				GithubHeadRef           string `json:"github_head_ref"`
				GithubRef               string `json:"github_ref"`
				GithubRefType           string `json:"github_ref_type"`
				GithubRepositoryID      string `json:"github_repository_id"`
				GithubRepositoryOwner   string `json:"github_repository_owner"`
				GithubRepositoryOwnerID string `json:"github_repository_owner_id"`
				GithubRunAttempt        string `json:"github_run_attempt"`
				GithubRunID             string `json:"github_run_id"`
				GithubRunNumber         string `json:"github_run_number"`
				GithubSha1              string `json:"github_sha1"`
				Os                      string `json:"os"`
			} `json:"environment"`
		} `json:"invocation"`
		BuildConfig struct {
			Version int `json:"version"`
			Steps   []struct {
				Command    []string    `json:"command"`
				Env        interface{} `json:"env"`
				WorkingDir string      `json:"workingDir"`
			} `json:"steps"`
		} `json:"buildConfig"`
		Metadata struct {
			BuildInvocationID string `json:"buildInvocationID"`
			Completeness      struct {
				Parameters  bool `json:"parameters"`
				Environment bool `json:"environment"`
				Materials   bool `json:"materials"`
			} `json:"completeness"`
			Reproducible bool `json:"reproducible"`
		} `json:"metadata"`
		Materials []struct {
			URI    string `json:"uri"`
			Digest struct {
				Sha1 string `json:"sha1"`
			} `json:"digest,omitempty"`
		} `json:"materials"`
	} `json:"predicate"`
}

// GetPayload returns the payload
func (p *Envelope) GetPayload() string {
	return p.Payload
}

// UnmarshalProvenance unmarshals a provenance
func UnmarshalProvenance(provenance []byte) (*Envelope, error) {
	prov := &Envelope{}
	err := cjson.Unmarshal(provenance, prov)
	if err != nil {
		return nil, err
	}
	return prov, nil
}
