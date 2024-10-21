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

package events

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"github.com/open-feature/go-sdk/openfeature"
	"go.opentelemetry.io/otel"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	serverconfig "github.com/mindersec/minder/internal/config/server"
	"github.com/mindersec/minder/internal/engine/engcontext"
	"github.com/mindersec/minder/internal/flags"
)

func Test_flaggedDriver_Publish(t *testing.T) {
	// t.Parallel()

	experimentProject := uuid.New()

	flagFile := filepath.Clean(filepath.Join(t.TempDir(), "testflags.yaml"))
	tempFile, err := os.Create(flagFile)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	t.Cleanup(func() { _ = tempFile.Close() })
	configFile := fmt.Sprintf(`
alternate_message_driver:
  variations:
    Base: false
    NATS: true
  targeting:
  - query: project eq "%s"
    percentage:
      NATS: 100
      Base: 0
  defaultRule:
    variation: Base
`, experimentProject)
	if _, err := io.WriteString(tempFile, configFile); err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}

	existingProvider := otel.GetMeterProvider()
	reader := sdkmetric.NewManualReader()
	meterProvider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	t.Cleanup(func() {
		otel.SetMeterProvider(existingProvider)
		if err := reader.Shutdown(context.Background()); err != nil {
			t.Fatalf("failed to shutdown reader: %v", err)
		}
	})
	otel.SetMeterProvider(meterProvider)
	// Tests want t.Parallel(), but we don't want them to increment counters at
	// the same time, so we need to actually mutex their execution.
	metricMutex := &sync.Mutex{}

	tests := []struct {
		name           string
		sendExperiment bool
		messageContext func() context.Context
	}{{
		name: "No flags",
		messageContext: func() context.Context {
			return context.Background()
		},
	}, {
		name: "With context",
		messageContext: func() context.Context {
			return engcontext.WithEntityContext(
				context.Background(),
				&engcontext.EntityContext{
					Project: engcontext.Project{ID: uuid.New()},
				},
			)
		},
	}, {
		name:           "With experiment flag",
		sendExperiment: true,
		messageContext: func() context.Context {
			return engcontext.WithEntityContext(
				context.Background(),
				&engcontext.EntityContext{
					Project: engcontext.Project{ID: experimentProject},
				},
			)
		},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()

			config := serverconfig.Config{
				Events: serverconfig.EventConfig{
					Driver: FlaggedDriver,
					Flags: serverconfig.FlagDriverConfig{
						MainDriver:      GoChannelDriver,
						AlternateDriver: GoChannelDriver,
					},
					GoChannel: serverconfig.GoChannelEventConfig{
						BlockPublishUntilSubscriberAck: false,
					},
				},
				Flags: serverconfig.FlagsConfig{
					GoFeature: serverconfig.GoFeatureConfig{
						FilePath: flagFile,
					},
				},
			}
			flags.OpenFeatureProviderFromFlags(ctx, config.Flags)
			flagClient := openfeature.NewClient("test")

			sendMsg := message.NewMessage("test-id", []byte(t.Name()))
			sendMsg.SetContext(tt.messageContext())

			eventer, err := Setup(ctx, flagClient, &config.Events)
			if err != nil {
				t.Fatalf("failed to setup eventer: %v", err)
			}

			var recvMsg *message.Message
			done := make(chan struct{})
			eventer.Register("test-topic", func(msg *message.Message) error {
				recvMsg = msg
				done <- struct{}{}
				return nil
			})

			go eventer.Run(ctx)
			t.Cleanup(func() {
				if err := eventer.Close(); err != nil {
					t.Fatalf("failed to close eventer: %v", err)
				}
			})
			<-eventer.Running()

			metricMutex.Lock()
			defer metricMutex.Unlock()

			sendBefore := getMetric(t, reader, "events_published", tt.sendExperiment)
			readBefore := getMetric(t, reader, "events_read", tt.sendExperiment)

			if err := eventer.Publish("test-topic", sendMsg); err != nil {
				t.Fatalf("failed to publish message: %v", err)
			}

			select {
			case <-time.After(5 * time.Second):
				t.Fatal("timed out waiting for message")
			case <-done:
			}

			if !reflect.DeepEqual(recvMsg.Payload, sendMsg.Payload) {
				t.Errorf("received message %v does not match published message %v", recvMsg.Payload, sendMsg.Payload)
			}

			sendAfter := getMetric(t, reader, "events_published", tt.sendExperiment)
			readAfter := getMetric(t, reader, "events_read", tt.sendExperiment)

			if sendBefore+1 != sendAfter {
				t.Errorf("send metric not as expected: started at %d, ended at %d", sendAfter, sendBefore)
			}
			if readBefore+1 != readAfter {
				t.Errorf("read metric not as expected: started at %d, ended at %d", readAfter, readBefore)
			}
		})
	}
}

func getMetric(t *testing.T, r sdkmetric.Reader, name string, experiment bool) int64 {
	t.Helper()
	rm := metricdata.ResourceMetrics{}
	// read ResourceMetrics from r, and extract the metric with the given name, or return an empty metric
	if err := r.Collect(context.Background(), &rm); err != nil {
		t.Fatalf("failed to collect metrics: %v", err)
	}
	for _, metric := range rm.ScopeMetrics {
		for _, metric := range metric.Metrics {
			if metric.Name == name {
				sums, ok := metric.Data.(metricdata.Sum[int64])
				if !ok {
					t.Errorf("metric %q is not a Sum metric, it is a %T", name, metric.Data)
					break
				}
				for _, series := range sums.DataPoints {
					v, ok := series.Attributes.Value("experiment")
					if !ok {
						t.Errorf("metric %q does not have an experiment attribute", name)
						continue
					}
					if v.AsBool() == experiment {
						return series.Value
					}
				}
			}
		}
	}
	// It's okay if we don't have a data point with the correct attribute yet.
	// Attributes are created when the first value is recorded.
	return 0
}
