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

package util

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	htmltemplate "html/template"
	"io"
	"reflect"
	"text/template"

	"github.com/rs/zerolog"
)

var (
	// ErrExceededSizeLimit is returned when the size limit is exceeded
	ErrExceededSizeLimit = errors.New("exceeded size limit")
)

var (
	// TemplateFuncs is a map of functions that can be used in templates
	// It introduces two custom functions:
	// - asMap: converts a structpb (or anything that implements the AsMap function call) to a map
	// - mapGet: returns the value of a key in a map
	TemplateFuncs = template.FuncMap{
		"asMap":  asMap,
		"mapGet": mapGet,
	}
)

// SafeTemplate is a `template` wrapper that ensures that the template is
// rendered in a safe and secure manner. That is, with memory limits
// and timeouts.
type SafeTemplate struct {
	t templater
}

type templater interface {
	Execute(io.Writer, interface{}) error
	Name() string
}

// This is a utility interface that allows us to accept any type
type asMapper interface {
	AsMap() map[string]interface{}
}

// asMap converts a structpb to a map
func asMap(s any) (reflect.Value, error) {
	if s == nil {
		return reflect.Value{}, fmt.Errorf("asMap called with nil")
	}

	inspb, ok := s.(asMapper)
	if !ok {
		return reflect.Value{}, fmt.Errorf("invalid type: %T", s)
	}

	return reflect.ValueOf(inspb.AsMap()), nil
}

// mapGet returns the value of a key in a map
// The map could be a map[string]interface{} or a asMapper
// So we need to handle both cases
func mapGet(m any, key string) (reflect.Value, error) {
	if m == nil {
		return reflect.Value{}, fmt.Errorf("map is nil")
	}

	// Check if the map is a map[string]interface{}
	if mm, ok := m.(map[string]interface{}); ok {
		return valueOfKey(mm, key)
	}

	if mm, ok := m.(asMapper); ok {
		mm := mm.AsMap()
		return valueOfKey(mm, key)
	}

	return reflect.Value{}, fmt.Errorf("invalid type: %T", m)
}

func valueOfKey(m map[string]interface{}, key string) (reflect.Value, error) {
	if v, ok := m[key]; ok {
		return reflect.ValueOf(v), nil
	}
	return reflect.Value{}, fmt.Errorf("key not found: %s", key)
}

// NewSafeTextTemplate creates a new SafeTemplate for text templates
func NewSafeTextTemplate(tmpl *string, name string) (*SafeTemplate, error) {
	t, err := parseNewTextTemplate(tmpl, name, TemplateFuncs)
	if err != nil {
		return nil, err
	}

	return &SafeTemplate{
		t: t,
	}, nil
}

// NewSafeHTMLTemplate creates a new SafeTemplate for HTML templates
func NewSafeHTMLTemplate(tmpl *string, name string) (*SafeTemplate, error) {
	t, err := parseNewHtmlTemplate(tmpl, name, TemplateFuncs)
	if err != nil {
		return nil, err
	}

	return &SafeTemplate{
		t: t,
	}, nil
}

// Render renders the template with the given data
func (t *SafeTemplate) Render(ctx context.Context, data any, limit int) (string, error) {
	buf := new(bytes.Buffer)
	if err := t.Execute(ctx, buf, data, limit); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// Execute executes the template with the given data
func (t *SafeTemplate) Execute(ctx context.Context, w io.Writer, data any, limit int) error {
	if limit <= 0 {
		return errors.New("limit must be greater than 0")
	}

	lw := NewLimitedWriter(w, limit)
	if err := t.t.Execute(lw, data); err != nil {
		if errors.Is(err, ErrExceededSizeLimit) {
			zerolog.Ctx(ctx).Error().Err(err).Str("template", t.t.Name()).Msg("expanding template exceeded size limit")
		}
		return err
	}

	return nil
}

// parseNewTextTemplate parses a named template from a string, ensuring it is not empty
func parseNewTextTemplate(tmpl *string, name string, fnmap template.FuncMap) (*template.Template, error) {
	if tmpl == nil || len(*tmpl) == 0 {
		return nil, fmt.Errorf("missing template")
	}

	t := template.New(name).Option("missingkey=error")
	if fnmap != nil {
		t = t.Funcs(fnmap)
	}
	t, err := t.Parse(*tmpl)
	if err != nil {
		return nil, fmt.Errorf("cannot parse template: %w", err)
	}

	return t, nil
}

// parseNewHtmlTemplate parses a named template from a string, ensuring it is not empty
func parseNewHtmlTemplate(tmpl *string, name string, fnmap template.FuncMap) (*htmltemplate.Template, error) {
	if tmpl == nil || len(*tmpl) == 0 {
		return nil, fmt.Errorf("missing template")
	}

	t := htmltemplate.New(name).Option("missingkey=error")
	if fnmap != nil {
		t = t.Funcs(fnmap)
	}
	t, err := t.Parse(*tmpl)
	if err != nil {
		return nil, fmt.Errorf("cannot parse template: %w", err)
	}

	return t, nil
}

// LimitedWriter is an io.Writer that limits the number of bytes written
type LimitedWriter struct {
	w     io.Writer
	n     int
	limit int
}

// NewLimitedWriter creates a new LimitedWriter
func NewLimitedWriter(w io.Writer, limit int) *LimitedWriter {
	return &LimitedWriter{
		w:     w,
		limit: limit,
	}
}

// Write implements the io.Writer interface
func (lw *LimitedWriter) Write(p []byte) (int, error) {
	if lw.n+len(p) > lw.limit {
		return 0, ErrExceededSizeLimit
	}
	n, err := lw.w.Write(p)
	lw.n += n
	return n, err
}
