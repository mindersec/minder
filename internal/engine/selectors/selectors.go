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

//go:generate go run go.uber.org/mock/mockgen -package mock_$GOPACKAGE -destination=./mock/$GOFILE -source=./$GOFILE

// Package selectors provides utilities for selecting entities based on profiles using CEL
package selectors

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/interpreter"

	"github.com/stacklok/minder/internal/profiles/models"
	internalpb "github.com/stacklok/minder/internal/proto"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

var (
	// ErrResultUnknown is returned when the result of a selector expression is unknown
	// this tells the caller to try again with more information
	ErrResultUnknown = errors.New("result is unknown")
	// ErrUnsupported is returned when the entity type is not supported
	ErrUnsupported = errors.New("unsupported entity type")
	// ErrSelectorCheck is returned if the selector fails to be checked for syntax errors
	ErrSelectorCheck = errors.New("failed to check selector")
)

// ErrKind is a string for the kind of error that occurred
type ErrKind string

const (
	// ErrKindParse is an error kind for parsing errors, e.g. syntax errors
	ErrKindParse ErrKind = "parse"
	// ErrKindCheck is an error kind for checking errors, e.g. mismatched types
	ErrKindCheck ErrKind = "check"
)

// ErrInstance is one occurrence of an error in a CEL expression
type ErrInstance struct {
	Line int    `json:"line,omitempty"`
	Col  int    `json:"col,omitempty"`
	Msg  string `json:"msg,omitempty"`
}

// ErrDetails is a struct that holds the details of an error in a CEL expression
type ErrDetails struct {
	Errors []ErrInstance `json:"errors,omitempty"`
	Source string        `json:"source,omitempty"`
}

// AsJSON returns the ErrDetails as a JSON string
func (ed *ErrDetails) AsJSON() string {
	edBytes, err := json.Marshal(ed)
	if err != nil {
		return fmt.Sprintf(`{"error": "failed to marshal JSON: %s"}`, err)
	}
	return string(edBytes)
}

func errDetailsFromCelIssues(source string, issues *cel.Issues) ErrDetails {
	var ed ErrDetails

	ed.Source = source
	ed.Errors = make([]ErrInstance, 0, len(issues.Errors()))
	for _, err := range issues.Errors() {
		ed.Errors = append(ed.Errors, ErrInstance{
			Line: err.Location.Line(),
			Col:  err.Location.Column(),
			Msg:  err.Message,
		})
	}

	return ed
}

// ErrStructure is a struct that callers can use to deserialize the JSON error
type ErrStructure struct {
	Err     ErrKind    `json:"err"`
	Details ErrDetails `json:"details"`
}

// ParseError is an error type for syntax errors in CEL expressions
type ParseError struct {
	ErrDetails
	original error
}

// Error implements the error interface for ParseError
func (pe *ParseError) Error() string {
	return fmt.Sprintf(`{"err": "%s", "details": %s}`, ErrKindParse, pe.AsJSON())
}

// Is checks if the target error is a ParseError
func (_ *ParseError) Is(target error) bool {
	var t *ParseError
	return errors.As(target, &t)
}

func (pe *ParseError) Unwrap() error {
	return pe.original
}

// CheckError is an error type for type checking errors in CEL expressions, e.g.
// mismatched types
type CheckError struct {
	ErrDetails
	original error
}

// Error implements the error interface for CheckError
func (ce *CheckError) Error() string {
	return fmt.Sprintf(`{"err": "%s", "details": %s}`, ErrKindCheck, ce.AsJSON())
}

// Is checks if the target error is a CheckError
func (_ *CheckError) Is(target error) bool {
	var t *CheckError
	return errors.As(target, &t)
}

func (ce *CheckError) Unwrap() error {
	return ce.original
}

func newParseError(source string, issues *cel.Issues) error {
	return &ParseError{
		ErrDetails: errDetailsFromCelIssues(source, issues),
		original:   ErrSelectorCheck,
	}
}

func newCheckError(source string, issues *cel.Issues) error {
	return &CheckError{
		ErrDetails: errDetailsFromCelIssues(source, issues),
		original:   ErrSelectorCheck,
	}
}

// celEnvFactory is an interface for creating CEL environments
// for an entity. Each entity must implement this interface to be
// usable in selectors
type celEnvFactory func() (*cel.Env, error)

// genericEnvFactory is a factory for creating a CEL environment
// for the generic SelectorEntity type
func genericEnvFactory() (*cel.Env, error) {
	return newEnvForEntity(
		"entity",
		&internalpb.SelectorEntity{},
		"internal.SelectorEntity")
}

// repoEnvFactory is a factory for creating a CEL environment
// for the SelectorRepository type representing a repository
func repoEnvFactory() (*cel.Env, error) {
	return newEnvForEntity(
		"repository",
		&internalpb.SelectorRepository{},
		"internal.SelectorRepository")
}

// artifactEnvFactory is a factory for creating a CEL environment
// for the SelectorArtifact type representing an artifact
func artifactEnvFactory() (*cel.Env, error) {
	return newEnvForEntity(
		"artifact",
		&internalpb.SelectorArtifact{},
		"internal.SelectorArtifact")
}

// pullRequestEnvFactory is a factory for creating a CEL environment
// for the SelectorPullRequest type representing a pull request
func pullRequestEnvFactory() (*cel.Env, error) {
	return newEnvForEntity(
		"pull_request",
		&internalpb.SelectorArtifact{},
		"internal.SelectorPullRequest")
}

// newEnvForEntity creates a new CEL environment for an entity. All environments are allowed to
// use the generic "entity" variable plus the specific entity type is also declared as variable
// with the appropriate type.
func newEnvForEntity(varName string, typ any, typName string) (*cel.Env, error) {
	entityPtr := &internalpb.SelectorEntity{}

	env, err := cel.NewEnv(
		cel.Types(typ), cel.Types(&internalpb.SelectorEntity{}),
		cel.Declarations(
			decls.NewVar("entity",
				decls.NewObjectType(string(entityPtr.ProtoReflect().Descriptor().FullName())),
			),
			decls.NewVar(varName,
				decls.NewObjectType(typName),
			),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL environment for %s: %v", varName, err)
	}

	return env, nil
}

type compiledSelector struct {
	orig    string
	ast     *cel.Ast
	program cel.Program
}

// compileSelectorForEntity compiles a selector expression for a given entity type into a CEL program
func compileSelectorForEntity(env *cel.Env, selector string) (*compiledSelector, error) {
	checked, err := checkSelectorForEntity(env, selector)
	if err != nil {
		return nil, fmt.Errorf("failed to check expression %q: %w", selector, err)
	}

	program, err := env.Program(checked,
		// OptPartialEval is needed to enable partial evaluation of the expression
		// OptTrackState is needed to get the details about partial evaluation (aka what is missing)
		cel.EvalOptions(cel.OptTrackState, cel.OptPartialEval))
	if err != nil {
		return nil, fmt.Errorf("failed to create program for expression %q: %w", selector, err)
	}

	return &compiledSelector{
		ast:     checked,
		orig:    selector,
		program: program,
	}, nil
}

func checkSelectorForEntity(env *cel.Env, selector string) (*cel.Ast, error) {
	parsedAst, issues := env.Parse(selector)
	if issues.Err() != nil {
		return nil, newParseError(selector, issues)
	}

	checkedAst, issues := env.Check(parsedAst)
	if issues.Err() != nil {
		return nil, newCheckError(selector, issues)
	}

	return checkedAst, nil
}

// SelectionBuilder is an interface for creating Selections (a collection of compiled CEL expressions)
// for an entity type. This is what the user of this module uses. The interface makes it easier to pass
// mocks by the user of this module.
type SelectionBuilder interface {
	NewSelectionFromProfile(minderv1.Entity, []models.ProfileSelector) (Selection, error)
}

// SelectionChecker is an interface for checking if a selector expression is valid for a given entity type
type SelectionChecker interface {
	CheckSelector(*minderv1.Profile_Selector) error
}

// Env is a struct that holds the CEL environments for each entity type and the factories for creating
type Env struct {
	// entityEnvs is a map of entity types to their respective CEL environments. We keep them cached
	// and lazy-initialize on first use
	entityEnvs map[minderv1.Entity]*entityEnvCache
	// factories is a map of entity types to their respective factories for creating CEL environments
	factories map[minderv1.Entity]celEnvFactory
}

// entityEnvCache is a struct that holds a CEL environment for lazy-initialization. Since the initialization
// is done only once, we also keep track of the error
type entityEnvCache struct {
	once sync.Once
	env  *cel.Env
	err  error
}

// NewEnv creates a new Env struct with the default factories for each entity type. The factories
// are used on first access to create the CEL environments for each entity type.
func NewEnv() *Env {
	factoryMap := map[minderv1.Entity]celEnvFactory{
		minderv1.Entity_ENTITY_UNSPECIFIED:   genericEnvFactory,
		minderv1.Entity_ENTITY_REPOSITORIES:  repoEnvFactory,
		minderv1.Entity_ENTITY_ARTIFACTS:     artifactEnvFactory,
		minderv1.Entity_ENTITY_PULL_REQUESTS: pullRequestEnvFactory,
	}

	entityEnvs := make(map[minderv1.Entity]*entityEnvCache, len(factoryMap))
	for entity := range factoryMap {
		entityEnvs[entity] = &entityEnvCache{}
	}

	return &Env{
		entityEnvs: entityEnvs,
		factories:  factoryMap,
	}
}

// NewSelectionFromProfile creates a new Selection (compiled CEL programs for that entity type)
// from a profile
func (e *Env) NewSelectionFromProfile(
	entityType minderv1.Entity,
	profileSelection []models.ProfileSelector,
) (Selection, error) {
	selector := make([]*compiledSelector, 0, len(profileSelection))

	env, err := e.envForEntity(entityType)
	if err != nil {
		return nil, fmt.Errorf("failed to get environment for entity %v: %w", entityType, err)
	}

	for _, sel := range profileSelection {
		if sel.Entity != entityType && sel.Entity != minderv1.Entity_ENTITY_UNSPECIFIED {
			continue
		}

		compSel, err := compileSelectorForEntity(env, sel.Selector)
		if err != nil {
			return nil, fmt.Errorf("failed to compile selector %q: %w", sel.Selector, err)
		}

		selector = append(selector, compSel)
	}

	return &EntitySelection{
		env:      env,
		selector: selector,
		entity:   entityType,
	}, nil
}

// CheckSelector checks if a selector expression compiles and is valid for a given entity
func (e *Env) CheckSelector(sel *minderv1.Profile_Selector) error {
	ent := minderv1.EntityFromString(sel.GetEntity())
	env, err := e.envForEntity(ent)
	if err != nil {
		return fmt.Errorf("failed to get environment for entity %v: %w", ent, ErrUnsupported)
	}

	_, err = checkSelectorForEntity(env, sel.Selector)
	return err
}

// envForEntity gets the CEL environment for a given entity type. If the environment is not cached,
// it creates it using the factory for that entity type.
func (e *Env) envForEntity(entity minderv1.Entity) (*cel.Env, error) {
	cache, ok := e.entityEnvs[entity]
	if !ok {
		return nil, fmt.Errorf("no cache found for entity %v", entity)
	}

	cache.once.Do(func() {
		cache.env, cache.err = e.factories[entity]()
	})

	return cache.env, cache.err
}

// SelectOption is a functional option for the Select method
type SelectOption func(*selectionOptions)

type selectionOptions struct {
	unknownPaths []string
}

// WithUnknownPaths sets the explicit unknown paths for the selection
func WithUnknownPaths(paths ...string) SelectOption {
	return func(o *selectionOptions) {
		o.unknownPaths = paths
	}
}

// Selection is an interface for selecting entities based on a profile
type Selection interface {
	Select(*internalpb.SelectorEntity, ...SelectOption) (bool, string, error)
}

// EntitySelection is a struct that holds the compiled CEL expressions for a given entity type
type EntitySelection struct {
	env *cel.Env

	selector []*compiledSelector
	entity   minderv1.Entity
}

// Select return true if the entity matches all the compiled expressions and false otherwise
func (s *EntitySelection) Select(se *internalpb.SelectorEntity, userOpts ...SelectOption) (bool, string, error) {
	if se == nil {
		return false, "", fmt.Errorf("input entity is nil")
	}

	var opts selectionOptions
	for _, opt := range userOpts {
		opt(&opts)
	}

	for _, sel := range s.selector {
		entityMap, err := inputAsMap(se)
		if err != nil {
			return false, "", fmt.Errorf("failed to convert input to map: %w", err)
		}

		out, details, err := s.evalWithOpts(&opts, sel, entityMap)
		// check unknowns /before/ an error. Maybe we should try to special-case the one
		// error we get from the CEL library in this case and check for the rest?
		if s.detailHasUnknowns(sel, details) {
			return false, "", ErrResultUnknown
		}

		if err != nil {
			return false, "", fmt.Errorf("failed to evaluate Expression: %w", err)
		}

		if types.IsUnknown(out) {
			return false, "", ErrResultUnknown
		}

		if out.Type() != cel.BoolType {
			return false, "", fmt.Errorf("expression did not evaluate to a boolean: %v", out)
		}

		if !out.Value().(bool) {
			return false, sel.orig, nil
		}
	}

	return true, "", nil
}

func unknownAttributesFromOpts(unknownPaths []string) []*interpreter.AttributePattern {
	unknowns := make([]*interpreter.AttributePattern, 0, len(unknownPaths))

	for _, path := range unknownPaths {
		frags := strings.Split(path, ".")
		if len(frags) == 0 {
			continue
		}

		unknownAttr := interpreter.NewAttributePattern(frags[0])
		if len(frags) > 1 {
			for _, frag := range frags[1:] {
				unknownAttr = unknownAttr.QualString(frag)
			}
		}
		unknowns = append(unknowns, unknownAttr)
	}

	return unknowns
}

func (_ *EntitySelection) evalWithOpts(
	opts *selectionOptions, sel *compiledSelector, entityMap map[string]any,
) (ref.Val, *cel.EvalDetails, error) {
	unknowns := unknownAttributesFromOpts(opts.unknownPaths)
	if len(unknowns) > 0 {
		partialMap, err := cel.PartialVars(entityMap, unknowns...)
		if err != nil {
			return types.NewErr("failed to create partial value"), nil, fmt.Errorf("failed to create partial vars: %w", err)
		}

		return sel.program.Eval(partialMap)
	}

	return sel.program.Eval(entityMap)
}

func (s *EntitySelection) detailHasUnknowns(sel *compiledSelector, details *cel.EvalDetails) bool {
	if details == nil {
		return false
	}

	// TODO(jakub): We should also extract what the unknowns are and return them
	// there exists cel.AstToString() which prints the part that was not evaluated, but as a whole
	// (e.g. properties['is_fork'] == true) and not as a list of unknowns. We should either take a look
	// at its implementation or walk the AST ourselves
	residualAst, err := s.env.ResidualAst(sel.ast, details)
	if err != nil {
		return false
	}

	checked, err := cel.AstToCheckedExpr(residualAst)
	if err != nil {
		return false
	}

	return checked.GetExpr().GetConstExpr() == nil
}

func inputAsMap(se *internalpb.SelectorEntity) (map[string]any, error) {
	var value any

	key := se.GetEntityType().ToString()

	// FIXME(jakub): I tried to be smart and code something up using protoreflect and WhichOneOf but didn't
	// make it work. Maybe someone smarter than me can.
	// nolint:exhaustive
	switch se.GetEntityType() {
	case minderv1.Entity_ENTITY_REPOSITORIES:
		value = se.GetRepository()
	case minderv1.Entity_ENTITY_ARTIFACTS:
		value = se.GetArtifact()
	case minderv1.Entity_ENTITY_PULL_REQUESTS:
		value = se.GetPullRequest()
	default:
		return nil, fmt.Errorf("unsupported entity type [%d]: %s", se.GetEntityType(), se.GetEntityType().ToString())
	}

	return map[string]any{
		key:      value,
		"entity": se,
	}, nil
}
