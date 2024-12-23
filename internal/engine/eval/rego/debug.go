// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package rego provides the rego rule evaluator
package rego

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/open-policy-agent/opa/ast/location"
	"github.com/open-policy-agent/opa/debug"
	"github.com/open-policy-agent/opa/rego"

	"github.com/mindersec/minder/internal/util/cli"
	"github.com/mindersec/minder/pkg/engine/v1/interfaces"
)

func makeEventHandler(ch chan<- *debug.Event) func(debug.Event) {
	return func(event debug.Event) {
		ch <- &event
	}
}

//nolint:unused
func makeTracingEventHandler(ch chan<- *debug.Event) func(debug.Event) {
	return func(event debug.Event) {
		fmt.Fprintf(os.Stderr, "%+v\n", event)
		ch <- &event
	}
}

var (
	errEmptySource  = errors.New("empty source code")
	errInvalidInstr = errors.New("invalid instruction")
	errInvalidBP    = errors.New("invalid breakpoint")
	errUserAbort    = errors.New("user abort")
)

// Debug implements an interactive debugger for REGO-based evaluators.
func (e *Evaluator) Debug(
	ctx context.Context,
	_ *interfaces.Result,
	input *Input,
	funcs ...func(*rego.Rego),
) error {
	allOpts := make([]func(*rego.Rego), 0, len(e.regoOpts)+len(funcs))
	allOpts = append(allOpts, e.regoOpts...)
	allOpts = append(allOpts, funcs...)

	ds, err := newDebugSession(
		withPrompt("(mindbg)"),
		withSource(e.cfg.Def),
		withInput(input),
		withQuery(e.reseval.getQueryString()),
		withOpts(allOpts...),
		// withTracingEventHandler(),
	)
	if err != nil {
		return fmt.Errorf("error initializing debugger: %w", err)
	}

	return ds.Start(ctx)
}

type debugSession struct {
	prompt  string
	src     string
	lines   int
	input   any
	query   string
	opts    []debug.LaunchOption
	ch      chan *debug.Event
	handler debug.EventHandler

	// fields initialized after starting the session
	session debug.Session
}

type debugSessionOption func(*debugSession) error

func withPrompt(prompt string) debugSessionOption {
	return func(ds *debugSession) error {
		ds.prompt = prompt
		return nil
	}
}

func withSource(src string) debugSessionOption {
	return func(ds *debugSession) error {
		if len(src) == 0 {
			return errEmptySource
		}
		ds.src = src
		ds.lines = len(strings.Split(src, "\n"))
		return nil
	}
}

func withInput(input any) debugSessionOption {
	return func(ds *debugSession) error {
		ds.input = input
		return nil
	}
}

func withQuery(query string) debugSessionOption {
	return func(ds *debugSession) error {
		ds.query = query
		return nil
	}
}

func withOpts(opts ...func(*rego.Rego)) debugSessionOption {
	return func(ds *debugSession) error {
		var res []debug.LaunchOption
		if ds.opts == nil {
			res = make([]debug.LaunchOption, 0, len(opts))
		} else {
			res = ds.opts
		}

		for _, opt := range opts {
			res = append(res, debug.RegoOption(opt))
		}

		ds.opts = res
		return nil
	}
}

//nolint:unused
func withTracingEventHandler() debugSessionOption {
	return func(ds *debugSession) error {
		// NOTE: this channel must be buffered, because REGO
		// interpreter emits several events that we're
		// currently handling in the same thread of execition
		// of the CLI interface.
		//
		// The solution would be handling CLI events and
		// debuggee events asynchronously, but we're not there
		// yet.
		ch := make(chan *debug.Event, 10)
		ds.ch = ch
		ds.handler = makeTracingEventHandler(ch)
		return nil
	}
}

func newDebugSession(
	opts ...debugSessionOption,
) (*debugSession, error) {
	ds := &debugSession{}

	for _, opt := range opts {
		if err := opt(ds); err != nil {
			return nil, err
		}
	}

	if ds.handler == nil {
		// NOTE: this channel must be buffered, because REGO
		// interpreter emits several events that we're
		// currently handling in the same thread of execition
		// of the CLI interface.
		//
		// The solution would be handling CLI events and
		// debuggee events asynchronously, but we're not there
		// yet.
		ch := make(chan *debug.Event, 10)
		ds.ch = ch
		ds.handler = makeEventHandler(ch)
	}

	return ds, nil
}

func (ds *debugSession) waitFor(
	ctx context.Context,
	eventTypes ...debug.EventType,
) *debug.Event {
	for {
		select {
		case e := <-ds.ch:
			if slices.Contains(eventTypes, e.Type) {
				return e
			}
		case <-ctx.Done():
			return nil
		}
	}
}

func (ds *debugSession) startDebugger(
	ctx context.Context,
) error {
	debugger := debug.NewDebugger(
		debug.SetEventHandler(ds.handler),
	)
	// This combination of flags provides roughly the same user
	// experience as one would have while debugging imperative
	// languages using a standard debugger like lldb or gdb.
	//
	// Specifically, `StopOnEntry` stops when entering an
	// expression, which is like stepping through some, but not
	// all, lines and even inside the same line multiple times in
	// the case of list/set comprehensions, while `StopOnFail`
	// results in stopping at all expressions producing a `false`
	// value, which is similar to the previous case in that it
	// stops every time a check fails during a list/set
	// comprehension.
	//
	// The previous descriptions must be taken with a grain of
	// salt and are likely missing useful cases. That said, the
	// described cases are hardly seen when debugging imperative
	// languages, which is the user experience we want to provide
	// at the moment. Of course, this might change in the future.
	launchProps := debug.LaunchEvalProperties{
		LaunchProperties: debug.LaunchProperties{
			StopOnEntry:  false,
			StopOnFail:   false,
			StopOnResult: true,
			EnablePrint:  true,
			RuleIndexing: false,
		},
		Input: ds.input,
		Query: ds.query,
	}

	session, err := debugger.LaunchEval(ctx, launchProps, ds.opts...)
	if err != nil {
		return err
	}

	ds.session = session

	return nil
}

//nolint:gocyclo
func (ds *debugSession) Start(ctx context.Context) error {
	err := ds.startDebugger(ctx)
	if err != nil {
		return fmt.Errorf("error launching debugger: %w", err)
	}

	thr := debug.ThreadID(1)
	fmt.Printf("%s ", ds.prompt)
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		var b strings.Builder
		switch {
		case line == "":
			// There's nothing to do here, but it is
			// useful to let the user spam enter to see if
			// it's working.
		case line == "r":
			err = ds.startDebugger(ctx)
			if err != nil {
				return fmt.Errorf("error restarting debugger: %w", err)
			}
			fmt.Fprintf(&b, "Restarted")
		case line == "c":
			if err := ds.session.ResumeAll(); err != nil {
				return fmt.Errorf("error resuming execution: %w", err)
			}

			evt := ds.waitFor(ctx,
				debug.ExceptionEventType,
				debug.StoppedEventType,
				debug.StdoutEventType,
				debug.TerminatedEventType,
			)
			switch evt.Type {
			case debug.ExceptionEventType:
				fmt.Fprintf(&b, "\nException\n")
				if err := printLocals(&b, ds.session, evt.Thread); err != nil {
					return fmt.Errorf("error printing locals: %w", err)
				}
			case debug.StoppedEventType:
				fmt.Fprintf(&b, "\nStopped\n")
				if err := printLocals(&b, ds.session, evt.Thread); err != nil {
					return fmt.Errorf("error printing locals: %w", err)
				}
			case debug.StdoutEventType:
				fmt.Fprintf(&b, "\nFinished\n")
				if err := printLocals(&b, ds.session, evt.Thread); err != nil {
					return fmt.Errorf("error printing locals: %w", err)
				}
				fmt.Fprintf(&b, "\nResult: ")
				err := printVar(&b,
					fmt.Sprintf("%s.*", RegoQueryPrefix),
					ds.session,
					evt.Thread,
				)
				if err != nil {
					return fmt.Errorf("error printing variable: %w", err)
				}
			case debug.TerminatedEventType:
				fmt.Fprintf(&b, "\nTerminated\n")
			}
		case line == "q":
			return errUserAbort

		case line == "locals":
			if err := printLocals(&b, ds.session, thr); err != nil {
				return fmt.Errorf("error printing locals: %w", err)
			}
		case line == "bp":
			bps, err := ds.session.Breakpoints()
			if err != nil {
				return fmt.Errorf("error getting breakpoints: %w", err)
			}
			printBreakpoints(&b, bps)
		case line == "bt":
			stack, err := ds.session.StackTrace(thr)
			if err != nil {
				return fmt.Errorf("error getting stack trace: %w", err)
			}
			printStackTrace(&b, stack, 10)
		case line == "list", line == "l":
			stack, err := ds.session.StackTrace(thr)
			if err != nil {
				return fmt.Errorf("error getting stack trace: %w", err)
			}
			printSource(&b, ds.src, stack)
		case line == "trs":
			threads, err := ds.session.Threads()
			if err != nil {
				return fmt.Errorf("error getting threads: %w", err)
			}
			printThreads(&b, threads)

		// "clearall" command currently removes all
		// breakpoints, both user-defined and internal
		// ones. This is not desirable for the very same
		// reasons described in the comment related to the
		// "next" command.
		case line == "cla",
			line == "clearall":
			if err := ds.session.ClearBreakpoints(); err != nil {
				return fmt.Errorf("error clearing breakpoints: %w", err)
			}

		// "next" is a bit quirky, since it requires a few
		// steps to function, namely:
		//
		// * adding a so called "internal breakpoint"
		// * running until it's reached, and finally
		// * removing the breakpoint
		//
		// Internal breakpoints should be managed separately
		// from user-defined breakpoints, as the user should
		// neither see them nor be allowed to remove them
		// since it could invalidate some assumptions the code
		// does around them.
		//
		// TODO: add two lists of breakpoints to
		// `debugSession` struct and add routines to manage
		// them.
		case line == "n",
			line == "next":
			stack, err := ds.session.StackTrace(thr)
			if err != nil {
				return fmt.Errorf("error getting stack trace: %w", err)
			}
			if loc := getCurrentLocation(stack); loc != nil {
				// Unfortunately, getting the column
				// right is tricky, since source-level
				// breakpoints only look at line
				// numbers in the REGO interpreter, so
				// the safest assumption is starting
				// from 0.
				//
				// It would be great if the frame
				// struct contained details about the
				// position in the source.
				nextloc := location.Location{
					Row: loc.Row + 1, // let's hope it always exists...
					Col: 0,
				}

				// add internal breakpoint
				bp, err := ds.session.AddBreakpoint(nextloc)
				if err != nil {
					return fmt.Errorf("error setting breakpoint: %w", err)
				}

				// resume execution
				if err := ds.session.Resume(thr); err != nil {
					return fmt.Errorf("error resuming execution: %w", err)
				}

				evt := ds.waitFor(ctx, debug.StoppedEventType)
				stack, err := ds.session.StackTrace(evt.Thread)
				if err != nil {
					return fmt.Errorf("error getting stack trace: %w", err)
				}

				// clear internal breakpoint, even if
				// we stopped for another reason.
				if _, err := ds.session.RemoveBreakpoint(bp.ID()); err != nil {
					return fmt.Errorf("error removing breakpoing: %w", err)
				}

				printSource(&b, ds.src, stack)
			}
		case line == "s",
			line == "sv":
			if err := ds.session.StepOver(thr); err != nil {
				return fmt.Errorf("error on step-over: %w", err)
			}
			evt := ds.waitFor(ctx, debug.StoppedEventType)
			stack, err := ds.session.StackTrace(evt.Thread)
			if err != nil {
				return fmt.Errorf("error getting stack trace: %w", err)
			}
			printSource(&b, ds.src, stack)
		case line == "si":
			if err := ds.session.StepIn(thr); err != nil {
				return fmt.Errorf("error on step-in: %w", err)
			}
			evt := ds.waitFor(ctx, debug.StoppedEventType)
			stack, err := ds.session.StackTrace(evt.Thread)
			if err != nil {
				return fmt.Errorf("error getting stack trace: %w", err)
			}
			printSource(&b, ds.src, stack)
		case line == "so":
			if err := ds.session.StepOut(thr); err != nil {
				return fmt.Errorf("error on step-out: %w", err)
			}
			evt := ds.waitFor(ctx, debug.StoppedEventType)
			stack, err := ds.session.StackTrace(evt.Thread)
			if err != nil {
				return fmt.Errorf("error getting stack trace: %w", err)
			}
			printSource(&b, ds.src, stack)

		case line == "h",
			line == "help":
			printHelp(&b)

		case strings.HasPrefix(line, "p"):
			varname, err := toVarName(line)
			if err != nil {
				fmt.Fprintln(&b, err)
				continue
			}
			// printVar function accepts a regexp as
			// variable name, allowing the caller to match
			// multiple variables.
			//
			// We don't want to expose this functionality
			// to the user, as the general case (fetching
			// a specific variable) becomes awkward,
			// requiring the user to specify the full
			// regex.
			//
			// To solve this, we always wrap the received
			// variable name in ^ and $.
			r := fmt.Sprintf("^%s$", varname)
			if err := printVar(&b, r, ds.session, thr); err != nil {
				return fmt.Errorf("error printing variables: %w", err)
			}

		case strings.HasPrefix(line, "b"):
			loc, err := toLocation(line, ds.lines)
			if err != nil {
				fmt.Fprintln(&b, err)
			} else {
				bp, err := ds.session.AddBreakpoint(*loc)
				if err != nil {
					return fmt.Errorf("error setting breakpoint: %w", err)
				}
				fmt.Fprintln(&b)
				printBreakpoint(&b, bp)
			}

		// "clear" command currently allows removing all
		// breakpoints, both user-defined and internal
		// ones. This is not desirable for the very same
		// reasons described in the comment related to the
		// "next" command.
		case strings.HasPrefix(line, "cl "),
			strings.HasPrefix(line, "clear "):
			ids := make([]debug.BreakpointID, 0)
			bps, err := ds.session.Breakpoints()
			if err != nil {
				return fmt.Errorf("error gettin breakpoints: %w", err)
			}
			for _, bp := range bps {
				ids = append(ids, bp.ID())
			}
			id, err := toBreakpointID(line, ids)
			if err != nil {
				fmt.Fprintln(&b, err)
			} else {
				if _, err := ds.session.RemoveBreakpoint(id); err != nil {
					return fmt.Errorf("error removing breakpoint: %w", err)
				}
			}
		default:
			fmt.Fprintf(&b, "Invalid command: %s\nPress h for help\n", line)
		}

		output := b.String()
		if output != "" {
			fmt.Printf("%s\n%s ", output, ds.prompt)
		} else {
			fmt.Printf("%s ", ds.prompt)
		}
	}

	return scanner.Err()
}

func toLocation(line string, lineCount int) (*location.Location, error) {
	num, ok := strings.CutPrefix(line, "b ")
	if !ok {
		return nil, fmt.Errorf(`%w: "%s"`, errInvalidInstr, line)
	}
	i, err := strconv.ParseInt(num, 10, 64)
	if err != nil {
		return nil, fmt.Errorf(`%w: invalid line "%s": %s`, errInvalidBP, num, err)
	}
	if i < 1 || int(i) > lineCount {
		return nil, fmt.Errorf("%w: invalid line %d", errInvalidBP, i)
	}
	return &location.Location{File: "minder.rego", Row: int(i)}, nil
}

func toBreakpointID(line string, ids []debug.BreakpointID) (debug.BreakpointID, error) {
	num1, ok1 := strings.CutPrefix(line, "cl ")
	num2, ok2 := strings.CutPrefix(line, "clear ")
	if !ok1 && !ok2 {
		return debug.BreakpointID(-1), fmt.Errorf(`%w: "%s"`, errInvalidInstr, line)
	}

	var num string
	if !ok1 {
		num = num2
	}
	if !ok2 {
		num = num1
	}

	i, err := strconv.ParseInt(num, 10, 64)
	if err != nil {
		return debug.BreakpointID(-1), fmt.Errorf(
			`%w: invalid breakpoint id %s`,
			errInvalidBP, num,
		)
	}

	if i < 1 {
		return debug.BreakpointID(-1), fmt.Errorf(
			"%w: negative line id",
			errInvalidBP,
		)
	}

	if !slices.Contains(ids, debug.BreakpointID(i)) {
		return debug.BreakpointID(-1), fmt.Errorf(
			"%w: breakpoint does not exist",
			errInvalidBP,
		)
	}

	return debug.BreakpointID(i), nil
}

func toVarName(line string) (string, error) {
	varname, ok := strings.CutPrefix(line, "p ")
	if !ok {
		return "", fmt.Errorf(`%w: "%s"`, errInvalidInstr, line)
	}
	return varname, nil
}

func printBreakpoints(b *strings.Builder, bps []debug.Breakpoint) {
	if len(bps) == 0 {
		return
	}
	fmt.Fprintln(b)
	for _, bp := range bps {
		printBreakpoint(b, bp)
	}
}

func printBreakpoint(b *strings.Builder, bp debug.Breakpoint) {
	fmt.Fprintf(b, "Breakpoint %d set at %s:%d\n",
		bp.ID(),
		bp.Location().File,
		bp.Location().Row,
	)
}

func printThreads(b *strings.Builder, threads []debug.Thread) {
	if len(threads) == 0 {
		return
	}
	fmt.Fprintln(b)
	for _, thread := range threads {
		fmt.Fprintf(b, "Thread %d\n", thread.ID())
	}
}

func getCurrentLocation(stack debug.StackTrace) *location.Location {
	if len(stack) == 0 {
		return nil
	}

	frame := stack[0]
	return frame.Location()
}

func printStackTrace(b *strings.Builder, stack debug.StackTrace, limit int) {
	if len(stack) == 0 {
		return
	}

	fmt.Fprintln(b)
	for _, frame := range stack[:limit] {
		if loc := frame.Location(); loc != nil {
			fmt.Fprintf(b, "Frame %d at %s:%d.%d\n",
				frame.ID(),
				loc.File,
				loc.Row,
				loc.Col,
			)
		}
	}
	if len(stack) > limit {
		fmt.Fprintf(b, "...\n")
	}
}

func printSource(b *strings.Builder, src string, stack debug.StackTrace) {
	if len(stack) == 0 {
		printSourceSimple(b, src)
		return
	}

	lines := strings.Split(src, "\n")
	padding := int64(math.Floor(math.Log10(float64(len(lines)))) + 1)

	fmt.Fprintln(b)
	frame := stack[0]
	if loc := frame.Location(); loc != nil {
		fmt.Fprintf(b, "Frame %d at %s:%d.%d\n",
			frame.ID(),
			loc.File,
			loc.Row,
			loc.Col,
		)

		for idx, line := range strings.Split(src, "\n") {
			fmt.Fprintf(b, "%*d: %s", padding, idx+1, line)
			if idx+1 == loc.Row {
				// `theline` is the very first line of
				// the expression starting at the
				// given position.
				//
				// In REGO expressions can span
				// multiple lines (for example, rules
				// do), but we really are interested
				// in underlining only the first line
				// of the given expression.
				//
				// For weird underlyining starting
				// from column 0 of the line, see
				// comment on setting source-level
				// breakpoints.
				theline := strings.Split(line, "\n")[0]
				fmt.Fprintf(b, "\n%s%s",
					strings.Repeat(" ", int(padding)+2+loc.Col-1),
					cli.SimpleBoldStyle.Render(strings.Repeat("^", len(theline)-loc.Col+1)),
				)
			}
			fmt.Fprintln(b)
		}
	}
}

func printSourceSimple(b *strings.Builder, source string) {
	fmt.Fprintln(b)
	lines := strings.Split(source, "\n")
	padding := int64(math.Floor(math.Log10(float64(len(lines)))) + 1)
	for idx, line := range lines {
		fmt.Fprintf(b, "%*d: %s\n", padding, idx+1, line)
	}
}

func printLocals(b *strings.Builder, s debug.Session, thrID debug.ThreadID) error {
	trace, err := s.StackTrace(thrID)
	if err != nil {
		return fmt.Errorf("error getting stacktrace: %w", err)
	}

	if len(trace) == 0 {
		fmt.Fprintln(b, "No locals")
		return nil
	}

	// The first trace in the list is the one related to the
	// current stack frame.
	scopes, err := s.Scopes(trace[0].ID())
	if err != nil {
		return fmt.Errorf("error getting scopes: %w", err)
	}

	for _, scope := range scopes {
		vars, err := s.Variables(scope.VariablesReference())
		if err != nil {
			return fmt.Errorf("error getting variables: %w", err)
		}
		for _, v := range vars {
			fmt.Fprintf(b, "%s %s = %s\n", v.Type(), v.Name(), v.Value())
		}
	}

	return nil
}

func printVar(
	b *strings.Builder,
	varname string,
	s debug.Session,
	thrID debug.ThreadID,
) error {
	r, err := regexp.Compile(varname)
	if err != nil {
		return fmt.Errorf("error instantiating regex: %w", err)
	}

	trace, err := s.StackTrace(thrID)
	if err != nil {
		return fmt.Errorf("error getting stacktrace: %w", err)
	}

	if len(trace) == 0 {
		return nil
	}

	// The first trace in the list is the one related to the
	// current stack frame.
	scopes, err := s.Scopes(trace[0].ID())
	if err != nil {
		return fmt.Errorf("error getting scopes: %w", err)
	}

	for _, scope := range scopes {
		if err := printVariablesInScope(b, r, s, scope.VariablesReference()); err != nil {
			return err
		}
	}

	return nil
}

func printVariablesInScope(
	b *strings.Builder,
	r *regexp.Regexp,
	s debug.Session,
	varRef debug.VarRef,
) error {
	if varRef == 0 {
		return nil
	}

	vars, err := s.Variables(varRef)
	if err != nil {
		return fmt.Errorf("error getting variables: %w", err)
	}
	for _, v := range vars {
		if r.MatchString(v.Name()) {
			var b1 strings.Builder
			if err := varToString(&b1, v, s, 0); err != nil {
				return err
			}
			fmt.Fprintf(b, "%s %s = %s\n", v.Type(), v.Name(), b1.String())

			// We break early here despite the fact that
			// multiple variables might match the given
			// `varname`. This is done to honour lexical
			// scope, showing just the only variable that
			// is actually being used for evaluation in
			// the given frame.
			return nil
		}
	}

	return nil
}

func varToString(
	b *strings.Builder,
	v debug.Variable,
	s debug.Session,
	indentation int,
) error {
	padding := strings.Repeat(" ", indentation)
	switch v.Type() {
	case "array":
		return elementsToString(b, v, s, indentation, "[", "]",
			func(elem debug.Variable) error {
				fmt.Fprintf(b, "  %s", padding)
				err := varToString(b, elem, s, indentation)
				if err != nil {
					return err
				}
				fmt.Fprintf(b, ",\n")
				return nil
			},
		)
	case "set":
		return elementsToString(b, v, s, indentation, "{", "}",
			func(elem debug.Variable) error {
				fmt.Fprintf(b, "  %s", padding)
				err := varToString(b, elem, s, indentation)
				if err != nil {
					return err
				}
				fmt.Fprintf(b, ",\n")
				return nil
			},
		)
	case "object":
		return elementsToString(b, v, s, indentation, "{", "}",
			func(elem debug.Variable) error {
				fmt.Fprintf(b, "  %s%s: ", padding, elem.Name())
				err := varToString(b, elem, s, indentation)
				if err != nil {
					return err
				}
				fmt.Fprintf(b, ",\n")
				return nil
			},
		)
	default:
		fmt.Fprintf(b, "%s%s", padding, v.Value())
	}

	return nil
}

func elementsToString(
	b *strings.Builder,
	v debug.Variable,
	s debug.Session,
	indentation int,
	leftDelimiter string,
	rightDelimiter string,
	formatter func(debug.Variable) error,
) error {
	padding := strings.Repeat(" ", indentation)
	fmt.Fprintf(b, "%s%s\n", padding, leftDelimiter)
	elems, err := s.Variables(v.VariablesReference())
	if err != nil {
		return err
	}
	for _, elem := range elems {
		if err := formatter(elem); err != nil {
			return err
		}
	}
	fmt.Fprintf(b, "%s%s", padding, rightDelimiter)

	return nil
}

var helpMsg = `
Controlling execution:
    c ------------- continue
    r ------------- restart debugging session
    q ------------- quit

Printing:
    bt ------------ print stack trace (top 10)
    trs ----------- print threads
    list/l -------- list source
    locals -------- print local variables

Breakpoints:
    bp ------------ show breakpoints
    b <num> ------- set breakpoint at line <num>
    clear/cl <id> - clear breakpoint with id <id>
    clearall/cla -- clear all breakpoints

Stepping:
    n/next--------- next line
    s/sv ---------- step over
    so ------------ step out
    si ------------ step into

Help:
    help/h -------- print help
`

func printHelp(b *strings.Builder) {
	fmt.Fprint(b, helpMsg)
}
