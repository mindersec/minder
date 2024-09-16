// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package main provides the entrypoint for the minder server
package main

import (
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"

	"github.com/mindersec/minder/cmd/server/app"
)

func main() {
	f, err := os.Create("output.prof")
	if err != nil {
		fmt.Println(err)
		return
	}
	runtime.SetCPUProfileRate(1000)
	pprof.StartCPUProfile(f)
	defer pprof.StopCPUProfile()

	app.Execute()
}
