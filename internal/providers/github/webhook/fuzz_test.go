// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/go-github/v63/github"

	"github.com/mindersec/minder/internal/controlplane/metrics"
	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/events"
	"github.com/mindersec/minder/pkg/config/server"
)

var eventTypes = [23]string{
	"branch_protection_configuration",
	"branch_protection_rule",
	"code_scanning_alert",
	"create",
	"member",
	"meta",
	"repository_vulnerability_alert",
	"org_block",
	"organization",
	"public",
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
	"package",
	"pull_request",
}

// FuzzGithubEventParsers tests Minder's GH event parsers:
//
//   - processPingEvent
//   - processRelevantRepositoryEvent
//   - processRepositoryEvent
//   - processPackageEvent
//   - processInstallationAppEvent
//
// It also tests validatePayloadSignature given it contains a fair
// amount of logic specific to Minder that depends on external input.
//
// The fuzzer does not validate return values of the parsers. It tests if any
// input can cause code-level issues.
func FuzzGitHubEventParsers(f *testing.F) {
	f.Fuzz(func(t *testing.T, rawWHPayload []byte, target, eventEnum uint) {
		mac := hmac.New(sha256.New, []byte("test"))
		mac.Write(rawWHPayload)
		expectedMAC := hex.EncodeToString(mac.Sum(nil))

		req, err := http.NewRequest("POST", "/", bytes.NewBuffer(rawWHPayload))
		if err != nil {
			t.Fatal(err)
		}

		eventType := eventTypes[eventEnum%uint(len(eventTypes))]

		req.Header.Add("X-GitHub-Event", eventType)
		req.Header.Add("X-GitHub-Delivery", "12345")
		req.Header.Add("Content-Type", "application/json")
		req.Header.Add("X-Hub-Signature-256", fmt.Sprintf("sha256=%s", expectedMAC))

		wes := &metrics.WebhookEventState{
			Typ:      "unknown",
			Accepted: false,
			Error:    true,
		}

		wes.Typ = github.WebHookType(req)

		m := message.NewMessage("", nil)
		m.Metadata.Set(events.ProviderDeliveryIdKey, github.DeliveryID(req))
		m.Metadata.Set(events.ProviderTypeKey, string(db.ProviderTypeGithub))
		m.Metadata.Set(events.ProviderSourceKey, "")
		m.Metadata.Set(events.GithubWebhookEventTypeKey, wes.Typ)

		// Create whConfig
		whSecretFile, err := os.CreateTemp("", "webhooksecret*")
		if err != nil {
			t.Fatal(err)
		}
		secret := "test"
		_, err = whSecretFile.WriteString(secret)
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(whSecretFile.Name())

		whConfig := &server.WebhookConfig{
			WebhookSecrets: server.WebhookSecrets{
				WebhookSecretFile: whSecretFile.Name(),
			},
		}

		ctx := context.Background()

		switch target % 6 {
		case 0:
			//nolint:gosec // The fuzzer does not validate the return values
			processInstallationAppEvent(ctx, rawWHPayload)
		case 1:
			//nolint:gosec // The fuzzer does not validate the return values
			processRelevantRepositoryEvent(ctx, rawWHPayload)
		case 2:
			//nolint:gosec // The fuzzer does not validate the return values
			processRepositoryEvent(ctx, rawWHPayload)
		case 3:
			//nolint:gosec // The fuzzer does not validate the return values
			processPackageEvent(ctx, rawWHPayload)
		case 4:
			//nolint:gosec // The fuzzer does not validate the return values
			processPingEvent(ctx, rawWHPayload)
		case 5:
			//nolint:gosec // The fuzzer does not validate the return values
			validatePayloadSignature(req, whConfig)
		}
	})
}
