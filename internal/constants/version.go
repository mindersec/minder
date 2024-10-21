// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package constants

import (
	"runtime/debug"
	"strings"
	"text/template"
)

var (
	// CLIVersion is the version of the application. Note that this is
	// set at compile time using ldflags.
	CLIVersion = "no-info"
	// VerboseCLIVersion is the verbose version of the application.
	// Note that this is set up at init time.
	VerboseCLIVersion = ""
	// Revision is the git commit hash. Note that this is set at compile time
	// using ldflags.
	Revision = "no-info"
)

type versionStruct struct {
	Version   string
	GoVersion string
	Time      string
	Commit    string
	OS        string
	Arch      string
	Modified  bool
}

const (
	verboseTemplate = `Version: {{.Version}}
Go Version: {{.GoVersion}}
Git Commit: {{.Commit}}
Commit Date: {{.Time}}
OS/Arch: {{.OS}}/{{.Arch}}
Dirty: {{.Modified}}`
)

func init() {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}

	var vvs versionStruct

	vvs.Version = CLIVersion
	vvs.GoVersion = bi.GoVersion

	for _, kv := range bi.Settings {
		switch kv.Key {
		case "vcs.time":
			vvs.Time = kv.Value
		case "vcs.revision":
			vvs.Commit = kv.Value
			Revision = kv.Value
		case "vcs.modified":
			vvs.Modified = kv.Value == "true"
		case "GOOS":
			vvs.OS = kv.Value
		case "GOARCH":
			vvs.Arch = kv.Value
		}
	}

	VerboseCLIVersion = vvs.String()
}

func (vvs *versionStruct) String() string {
	stringBuilder := &strings.Builder{}
	tmpl := template.Must(template.New("version").Parse(verboseTemplate))
	err := tmpl.Execute(stringBuilder, vvs)
	if err != nil {
		panic(err)
	}

	return stringBuilder.String()
}
