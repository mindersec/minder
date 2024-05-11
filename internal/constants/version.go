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
