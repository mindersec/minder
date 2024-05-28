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

package controlplane

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/go-github/v61/github"

	"github.com/stacklok/minder/internal/config/server"
	"github.com/stacklok/minder/internal/controlplane/metrics"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/events"
)

var eventTypes = map[int]string{
	0:  "branch_protection_configuration",
	1:  "branch_protection_rule",
	2:  "code_scanning_alert",
	3:  "create",
	4:  "member",
	5:  "meta",
	6:  "repository_vulnerability_alert",
	7:  "org_block",
	8:  "organization",
	9:  "public",
	10: "push",
	11: "repository",
	12: "repository_advisory",
	13: "repository_import",
	14: "repository_ruleset",
	15: "secret_scanning_alert",
	16: "secret_scanning_alert_location",
	17: "security_advisory",
	18: "security_and_analysis",
	19: "team",
	20: "team_add",
	21: "package",
	22: "pull_request",
}

func FuzzGithubEventParsers(f *testing.F) {
	f.Fuzz(func(t *testing.T, rawWHPayload []byte, target, eventEnum int) {
		mac := hmac.New(sha256.New, []byte("test"))
		mac.Write(rawWHPayload)
		expectedMAC := hex.EncodeToString(mac.Sum(nil))

		req, err := http.NewRequest("POST", "/", bytes.NewBuffer(rawWHPayload))
		if err != nil {
			t.Fatal(err)
		}

		eventType := eventTypes[eventEnum%len(eventTypes)]

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

		whConfig := &server.WebhookConfig{WebhookSecretFile: whSecretFile.Name()}

		s := &Server{}

		switch target % 2 {
		case 0:
			payload, err := validatePayloadSignature(req, whConfig)
			if err != nil {
				return
			}
			//nolint:gosec
			s.parseGithubEventForProcessing(payload, m)
		case 1:
			payload, err := github.ValidatePayload(req, []byte(secret))
			if err != nil {
				return
			}
			//nolint:gosec
			s.parseGithubAppEventForProcessing(payload, m)
		}
	})
}
