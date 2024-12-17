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
	"strconv"
	"strings"
	"time"

	"github.com/open-policy-agent/opa/ast/location"
	"github.com/open-policy-agent/opa/debug"
	"github.com/open-policy-agent/opa/rego"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/mindersec/minder/internal/util/cli"
	"github.com/mindersec/minder/pkg/engine/v1/interfaces"
)

type eventHandler struct {
	ch chan *debug.Event
}

func newEventHandler() *eventHandler {
	return &eventHandler{
		ch: make(chan *debug.Event),
	}
}

func (eh *eventHandler) HandleEvent(event debug.Event) {
	eh.ch <- &event
}

// Actual client interface

func (eh *eventHandler) NextBlocking() *debug.Event {
	return <-eh.ch
}

func (eh *eventHandler) WaitFor(
	ctx context.Context,
	eventType debug.EventType,
) *debug.Event {
	for {
		select {
		case e := <-eh.ch:
			if e.Type == eventType {
				return e
			}
		case <-ctx.Done():
			return nil
		}
	}
}

var (
	errInvalidInstr = errors.New("invalid instruction")
	errInvalidBP    = errors.New("invalid breakpoint")
)

func (e *Evaluator) Debug(
	ctx context.Context,
	pol map[string]any,
	entity protoreflect.ProtoMessage,
	res *interfaces.Result,
	input *Input,
	funcs ...func(*rego.Rego),
) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	eh := newEventHandler()
	debugger := debug.NewDebugger(
		debug.SetEventHandler(eh.HandleEvent),
	)
	launchProps := debug.LaunchEvalProperties{
		LaunchProperties: debug.LaunchProperties{
			StopOnResult: true,
			StopOnFail:   true,
			EnablePrint:  true,
		},
		Input: input,
		Query: e.reseval.getQueryString(),
	}

	regoOpts := make([]debug.LaunchOption, 0, len(e.regoOpts)+len(funcs))
	for _, f := range e.regoOpts {
		regoOpts = append(regoOpts, debug.RegoOption(f))
	}
	for _, f := range funcs {
		regoOpts = append(regoOpts, debug.RegoOption(f))
	}

	session, err := debugger.LaunchEval(ctx, launchProps, regoOpts...)
	if err != nil {
		return fmt.Errorf("error launching debugger: %w", err)
	}

	// initial breakpoint
	if _, err := session.AddBreakpoint(location.Location{File: "minder.rego", Row: 1}); err != nil {
		return fmt.Errorf("error setting breakpoint: %w", err)
	}

	thr := debug.ThreadID(1)
	fmt.Print("(mindbg) ")
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()

		var b strings.Builder
		switch {
		case line == "":

		case line == "r",
			line == "c":
			if err := session.Resume(thr); err != nil {
				return fmt.Errorf("error resuming execution: %w", err)
			}

		EVENTS:
			for {
				evt := eh.NextBlocking()
				switch evt.Type {
				case debug.ExceptionEventType:
					fmt.Fprintf(&b, "\nException %+v\n", evt)
					printVars(&b, session, thr)
					break EVENTS
				case debug.StoppedEventType:
					fmt.Fprintf(&b, "\nStopped %+v\n", evt)
					return errors.New("execution stopped")
				case debug.StdoutEventType:
					fmt.Fprintf(&b, "\nFinished %+v\n", evt)
					break EVENTS
				}
			}
		case line == "locals":
			printVars(&b, session, thr)
		case line == "bp":
			bps, err := session.Breakpoints()
			if err != nil {
				return fmt.Errorf("error getting breakpoints: %w", err)
			}
			printBreakpoints(&b, bps)
		case line == "list", line == "l":
			stack, err := session.StackTrace(thr)
			if err != nil {
				return fmt.Errorf("error getting stack trace: %w", err)
			}
			printStackTrace(&b, e.cfg.Def, stack)
		case line == "trs":
			threads, err := session.Threads()
			if err != nil {
				return fmt.Errorf("error getting threads: %w", err)
			}
			printThreads(&b, threads)
		case line == "cla",
			line == "clearall":
			if err := session.ClearBreakpoints(); err != nil {
				return fmt.Errorf("error clearing breakpoints: %w", err)
			}
		// "next" is a bit quirky, since it requires adding an
		// internal breakpoint, running until it's reached,
		// and finally removing the breakpoint.
		case line == "n",
			line == "next":
			stack, err := session.StackTrace(thr)
			if err != nil {
				return fmt.Errorf("error getting stack trace: %w", err)
			}
			if loc := getCurrentLocation(stack); loc != nil {
				loc.Row += 1 // let's hope it always exists...
				loc.Col = 0

				// add internal breakpoint
				bp, err := session.AddBreakpoint(*loc)
				if err != nil {
					return fmt.Errorf("error setting breakpoint: %w", err)
				}

				// resume execution
				if err := session.Resume(thr); err != nil {
					return fmt.Errorf("error resuming execution: %w", err)
				}

				ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
				defer cancel()
				evt := eh.WaitFor(ctx, debug.StoppedEventType)
				stack, err := session.StackTrace(evt.Thread)
				if err != nil {
					return fmt.Errorf("error getting stack trace: %w", err)
				}

				// clear internal breakpoint, even if
				// we stopped for another reason.
				session.RemoveBreakpoint(bp.ID())

				printStackTrace(&b, e.cfg.Def, stack)
			}
		case line == "s",
			line == "sv":
			go func() {
				if err := session.StepOver(thr); err != nil {
					panic(err)
				}
			}()
			ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()
			evt := eh.WaitFor(ctx, debug.StoppedEventType)
			stack, err := session.StackTrace(evt.Thread)
			if err != nil {
				return fmt.Errorf("error getting stack trace: %w", err)
			}
			printStackTrace(&b, e.cfg.Def, stack)
		case line == "si":
			go func() {
				if err := session.StepIn(thr); err != nil {
					panic(err)
				}
			}()
			ctx, cancel := context.WithTimeout(ctx, time.Second)
			defer cancel()
			evt := eh.WaitFor(ctx, debug.StoppedEventType)
			stack, err := session.StackTrace(evt.Thread)
			if err != nil {
				return fmt.Errorf("error getting stack trace: %w", err)
			}
			printStackTrace(&b, e.cfg.Def, stack)
		case line == "so":
			go func() {
				if err := session.StepOut(thr); err != nil {
					panic(err)
				}
			}()
			ctx, cancel := context.WithTimeout(ctx, time.Second)
			defer cancel()
			evt := eh.WaitFor(ctx, debug.StoppedEventType)
			stack, err := session.StackTrace(evt.Thread)
			if err != nil {
				return fmt.Errorf("error getting stack trace: %w", err)
			}
			printStackTrace(&b, e.cfg.Def, stack)
		case line == "q":
			return fmt.Errorf("user abort")
		case line == "h",
			line == "help":
			printHelp(&b)
		case strings.HasPrefix(line, "p"):
			varname, err := toVarName(line)
			if err != nil {
				fmt.Fprintln(&b, err)
				continue
			}
			printVar(&b, varname, session, thr)
		case strings.HasPrefix(line, "b"):
			loc, err := toLocation(line)
			if err != nil {
				fmt.Fprintln(&b, err)
				continue
			}
			bp, err := session.AddBreakpoint(*loc)
			if err != nil {
				return fmt.Errorf("error setting breakpoint: %w", err)
			}
			fmt.Fprintln(&b)
			printBreakpoint(&b, bp)
		case strings.HasPrefix(line, "cl "),
			strings.HasPrefix(line, "clear "):
			id, err := toBreakpointID(line)
			if err != nil {
				fmt.Fprintln(&b, err)
				continue
			}
			session.RemoveBreakpoint(id)
		}

		output := b.String()
		if output != "" {
			fmt.Printf("%s\n(mindbg) ", output)
		} else {
			fmt.Printf("(mindbg) ")
		}
	}

	return scanner.Err()
}

func toLocation(line string) (*location.Location, error) {
	num, ok := strings.CutPrefix(line, "b ")
	if !ok {
		return nil, fmt.Errorf(`%w: "%s"`, errInvalidInstr, line)
	}
	i, err := strconv.ParseUint(num, 10, 64)
	if err != nil {
		return nil, fmt.Errorf(`%w: invalid line %s`, errInvalidBP, num)
	}
	return &location.Location{File: "minder.rego", Row: int(i)}, nil
}

func toBreakpointID(line string) (debug.BreakpointID, error) {
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

	i, err := strconv.ParseUint(num, 10, 64)
	if err != nil {
		return debug.BreakpointID(-1), fmt.Errorf(`%w: invalid breakpoint id %s`, errInvalidBP, num)
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
	fmt.Fprintln(b)
	for _, bp := range bps {
		printBreakpoint(b, bp)
	}
}

func printBreakpoint(b *strings.Builder, bp debug.Breakpoint) {
	fmt.Fprintf(b, "Breakpoint %d set at %s:%d\n", bp.ID(), bp.Location().File, bp.Location().Row)
}

func printThreads(b *strings.Builder, threads []debug.Thread) {
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

func printStackTrace(b *strings.Builder, src string, stack debug.StackTrace) {
	if len(stack) == 0 {
		printSource(b, src)
		return
	}

	lines := strings.Split(src, "\n")
	padding := int64(math.Floor(math.Log10(float64(len(lines)))) + 1)

	fmt.Fprintln(b)
	frame := stack[0]
	if loc := frame.Location(); loc != nil {
		fmt.Fprintf(b, "Frame %d at %s:%d.%d\n", frame.ID(), loc.File, loc.Row, loc.Col)

		for idx, line := range strings.Split(src, "\n") {
			fmt.Fprintf(b, "%*d: %s", padding, idx+1, line)
			if idx+1 == loc.Row {
				theline := strings.Split(string(loc.Text), "\n")[0]
				fmt.Fprintf(b, "\n%s%s",
					strings.Repeat(" ", loc.Col+int(padding)+2-1),
					cli.SimpleBoldStyle.Render(strings.Repeat("^", len(theline))),
				)
			}
			fmt.Fprintln(b)
		}
	}
}

func printSource(b *strings.Builder, source string) {
	fmt.Fprintln(b)
	lines := strings.Split(source, "\n")
	padding := int64(math.Floor(math.Log10(float64(len(lines)))) + 1)
	for idx, line := range lines {
		fmt.Fprintf(b, "%*d: %s\n", padding, idx+1, line)
	}
}

func printVars(b *strings.Builder, s debug.Session, thrID debug.ThreadID) error {
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
		printVariablesInScope(b, varname, s, scope.VariablesReference())
	}

	return nil
}

func printVariablesInScope(
	b *strings.Builder,
	varname string,
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
		if v.Name() == varname {
			var b1 strings.Builder
			varToString(&b1, v, s)
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

func varToString(b *strings.Builder, v debug.Variable, s debug.Session) error {
	switch v.Type() {
	case "array":
		fmt.Fprint(b, "[\n  ")
		elems, err := s.Variables(v.VariablesReference())
		if err != nil {
			return err
		}
		for i, elem := range elems {
			varToString(b, elem, s)
			if i < len(elems)-1 {
				fmt.Fprintf(b, ",\n  ")
			}
		}
		fmt.Fprint(b, "\n]")
	case "object":
		fmt.Fprint(b, "{\n  ")
		fields, err := s.Variables(v.VariablesReference())
		if err != nil {
			return err
		}
		for i, field := range fields {
			fmt.Fprintf(b, "  %s: ", field.Name())
			varToString(b, field, s)
			if i < len(fields)-1 {
				fmt.Fprintf(b, ",\n  ")
			}
		}
		fmt.Fprint(b, "\n}")
	default:
		fmt.Fprint(b, v.Value())
	}

	return nil
}

var helpMsg = `
Available commands:
    r/c ----------- continue
    b <num> ------- set breakpoint at line <num>
    bp ------------ show breakpoints
    clear/cl <id> - clear breakpoint with id <id>
    clearall/cla -- clear all breakpoints
    trs ----------- print threads
    s/sv ---------- step over
    so ------------ step out
    si ------------ step into
    list/l -------- list source
    locals -------- print local variables
    q ------------- quit
    help/h -------- print help
`

func printHelp(b *strings.Builder) {
	fmt.Fprintln(b, helpMsg)
}
