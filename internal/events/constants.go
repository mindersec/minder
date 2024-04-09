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

package events

// Metadata added to Messages
const (
	ProviderDeliveryIdKey     = "id"
	ProviderTypeKey           = "provider"
	ProviderSourceKey         = "source"
	GithubWebhookEventTypeKey = "type"

	GoChannelDriver = "go-channel"
	SQLDriver       = "sql"

	DeadLetterQueueTopic = "dead_letter_queue"
	PublishedKey         = "published_at"
)

const (
	metricsNamespace = "minder"
	metricsSubsystem = "eventer"
)

const (
	// TopicQueueEntityEvaluate is the topic for entity evaluation events from webhooks
	TopicQueueEntityEvaluate = "execute.entity.event"
	// TopicQueueEntityFlush is the topic for flushing internal webhook events
	TopicQueueEntityFlush = "flush.entity.event"
	// TopicQueueReconcileRepoInit is the topic for reconciling repository events, i.e. when a new repository is registered
	TopicQueueReconcileRepoInit = "internal.repo.reconciler.event"
	// TopicQueueReconcileProfileInit is the topic for reconciling when a profile is created or updated
	TopicQueueReconcileProfileInit = "internal.profile.init.event"
	// TopicQueueReconcileEntityDelete is the topic for reconciling when an entity is deleted
	TopicQueueReconcileEntityDelete = "internal.entity.delete.event"
)
