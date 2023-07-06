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

// Package events provides implementations of all the event handlers
// the GitHub provider supports.
package events

import (
	"log"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/stacklok/mediator/internal/events"
)

var (
	reg *events.Registrar
)

func init() {
	reg = initRegistrar()
}

func initRegistrar() (r *events.Registrar) {
	r = events.NewRegistrar()

	r.RegisterHandler("security_and_analysis", handleSecurityAndAnalysisEvent)

	return r
}

// GetRegistrar returns the registrar for GitHub events
func GetRegistrar() *events.Registrar {
	return reg
}

func handleSecurityAndAnalysisEvent(msg *message.Message) error {
	log.Printf("Got a security_and_analysis event: %v", msg)
	return nil
}
