package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/rs/zerolog"
	"github.com/stacklok/minder/internal/controlplane/metrics"
	"github.com/stacklok/minder/internal/util"
	"log"
	"strconv"
	"strings"
)

// errRepoNotFound is returned when a repository is not found
var errRepoNotFound = errors.New("repository not found")

// errArtifactNotFound is returned when an artifact is not found
var errArtifactNotFound = errors.New("artifact not found")

// errArtifactVersionSkipped is returned when an artifact is skipped because it has no tags
var errArtifactVersionSkipped = errors.New("artifact version skipped, has no tags")

// errRepoIsPrivate is returned when a repository is private
var errRepoIsPrivate = errors.New("repository is private")

// errNotHandled is returned when a webhook event is not handled
var errNotHandled = errors.New("webhook event not handled")

// newErrNotHandled returns a new errNotHandled error
func newErrNotHandled(smft string, args ...any) error {
	msg := fmt.Sprintf(smft, args...)
	return fmt.Errorf("%w: %s", errNotHandled, msg)
}

// https://docs.github.com/en/webhooks/webhook-events-and-payloads#about-webhook-events-and-payloads
var repoEvents = []string{
	"branch_protection_configuration",
	"branch_protection_rule",
	"code_scanning_alert",
	"create", // a tag or branch is created
	"member",
	"meta", // webhook itself
	"repository_vulnerability_alert",
	"org_block",
	"organization",
	"public",
	// listening to push makes sure we evaluate on pushes to branches we need to check, but might be too noisy
	// for topic branches
	"push",
	"repository",
	"repository_advisory",
	"repository_import",
	"repository_ruleset",
	"secret_scanning_alert",
	"secret_scanning_alert_location",
	"security_advisory",
	"security_and_analysis",
	"team",
	"team_add",
}

// WebhookActionEventDeleted is the action for a deleted event
const (
	WebhookActionEventDeleted     = "deleted"
	WebhookActionEventOpened      = "opened"
	WebhookActionEventClosed      = "closed"
	WebhookActionEventSynchronize = "synchronize"
	WebhookActionEventPublished   = "published"
)

// logPingReceivedEvent logs the type of token used to authenticate the webhook. The idea is to log a link between the
// repo and the token type. Since this is done only for the ping event, we can assume that the sender is the app that
// installed the webhook on the repository.
func logPingReceivedEvent(ctx context.Context, rawWHPayload []byte) {
	l := zerolog.Ctx(ctx).With().Logger()

	var payload map[string]any
	err := json.Unmarshal(rawWHPayload, &payload)
	if err == nil {
		repoInfo, ok := payload["repository"].(map[string]any)
		if ok {
			// Log the repository ID and URL if available
			repoID, err := parseRepoID(repoInfo["id"])
			if err == nil {
				l = l.With().Int64("github-repository-id", repoID).Logger()
			}
			repoUrl := repoInfo["html_url"].(string)
			l = l.With().Str("github-repository-url", repoUrl).Logger()
		}

		// During the ping event, the sender corresponds to the app that installed the webhook on the repository
		if payload["sender"] != nil {
			// Log the sender if available
			senderLogin, err := util.JQReadFrom[string](ctx, ".sender.login", payload)
			if err == nil {
				l = l.With().Str("sender-login", senderLogin).Logger()
			}
			senderHTMLUrl, err := util.JQReadFrom[string](ctx, ".sender.html_url", payload)
			if err == nil {
				if strings.Contains(senderHTMLUrl, "github.com/apps") {
					l = l.With().Str("sender-token-type", "github-app").Logger()
				} else {
					l = l.With().Str("sender-token-type", "oauth-app").Logger()
				}
			}
		}
	}
	l.Debug().Msg("ping received")
}

func handleParseError(typ string, parseErr error) *metrics.WebhookEventState {
	state := &metrics.WebhookEventState{Typ: typ, Accepted: false, Error: true}

	var logMsg string
	switch {
	case errors.Is(parseErr, errRepoNotFound):
		state.Error = false
		logMsg = "repository not found"
	case errors.Is(parseErr, errArtifactNotFound):
		state.Error = false
		logMsg = "artifact not found"
	case errors.Is(parseErr, errRepoIsPrivate):
		state.Error = false
		logMsg = "repository is private"
	case errors.Is(parseErr, errNotHandled):
		state.Error = false
		logMsg = fmt.Sprintf("webhook event not handled (%v)", parseErr)
	case errors.Is(parseErr, errArtifactVersionSkipped):
		state.Error = false
		logMsg = "artifact version skipped, has no tags"
	default:
		logMsg = fmt.Sprintf("Error parsing github webhook message: %v", parseErr)
	}
	log.Print(logMsg)
	return state
}

func parseRepoID(repoID any) (int64, error) {
	switch v := repoID.(type) {
	case int32:
		return int64(v), nil
	case int64:
		return v, nil
	case float64:
		return int64(v), nil
	case string:
		// convert string to int
		return strconv.ParseInt(v, 10, 64)
	default:
		return 0, fmt.Errorf("unknown type for repoID: %T", v)
	}
}
