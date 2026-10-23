// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"context"
	"os"
	"runtime"
	"syscall"
	"testing"
	"time"
)

// This test sends SIGTERM to the process, so it must not run in parallel with
// other tests that install process-wide signal handlers.
//
//nolint:paralleltest
func TestServerContextStopsOnSIGTERM(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not support sending SIGTERM to the current process")
	}

	ctx, cancel := serverContext(context.Background())
	defer cancel()

	process, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatalf("find current process: %v", err)
	}
	if err := process.Signal(syscall.SIGTERM); err != nil {
		t.Fatalf("send SIGTERM: %v", err)
	}

	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("server context was not canceled after SIGTERM")
	}
}
