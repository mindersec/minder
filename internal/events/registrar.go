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

import "fmt"

// Registrar allows users to register handlers for specific topics
type Registrar struct {
	r map[string]MessageHandler
}

// NewRegistrar creates a new Registrar
func NewRegistrar() *Registrar {
	return &Registrar{
		r: make(map[string]MessageHandler),
	}
}

// RegisterHandler registers a handler for a specific topic
// If a handler for the topic already exists, it panics, given that
// a topic can only have one handler and this would be a programming
// error.
func (r *Registrar) RegisterHandler(topic string, handler MessageHandler) {
	if _, ok := r.r[topic]; ok {
		panic(fmt.Sprintf("handler for topic %s already registered", topic))
	}
	r.r[topic] = handler
}

// GetHandler returns the handler for a specific topic
func (r *Registrar) GetHandler(topic string) MessageHandler {
	return r.r[topic]
}

// Walk iterates over all registered handlers
// This is useful for subscribing all topics to the final subscriber
func (r *Registrar) Walk(f func(topic string, handler MessageHandler)) {
	for topic, handler := range r.r {
		f(topic, handler)
	}
}
