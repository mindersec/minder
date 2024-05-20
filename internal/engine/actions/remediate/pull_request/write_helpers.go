// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pull_request

import (
	"crypto/sha1" // #nosec G505 - we're not using sha1 for crypto, only to quickly compare contents
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/rs/zerolog/log"

	"github.com/stacklok/minder/internal/util"
)

type fsEntry struct {
	contentTemplate *util.SafeTemplate

	Path    string `json:"path"`
	Content string `json:"content"`
	Mode    string `json:"mode"`
}

func (fe *fsEntry) write(fs billy.Filesystem) error {
	dirOsMode, err := filemode.Dir.ToOSFileMode()
	if err != nil {
		return fmt.Errorf("cannot get directory mode: %w", err)
	}
	if err := fs.MkdirAll(filepath.Dir(fe.Path), dirOsMode); err != nil {
		return fmt.Errorf("cannot create directory: %w", err)
	}

	parsedGitMode, err := filemode.New(fe.Mode)
	if err != nil {
		return fmt.Errorf("cannot parse mode: %w", err)
	}
	parsedOsMode, err := parsedGitMode.ToOSFileMode()
	if err != nil {
		return fmt.Errorf("cannot convert file mode: %w", err)
	}

	f, err := fs.OpenFile(fe.Path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, parsedOsMode)
	if err != nil {
		return fmt.Errorf("cannot create file: %w", err)
	}
	defer func() {
		err := f.Close()
		if err != nil {
			log.Error().Err(err).Msg("failed to close file")
		}
	}()

	_, err = io.WriteString(f, fe.Content)
	if err != nil {
		return fmt.Errorf("cannot write to file: %w", err)
	}

	return nil
}

type fsChangeSet struct {
	fs      billy.Filesystem
	entries []*fsEntry
}

func (fcs *fsChangeSet) writeEntries() error {
	for i := range fcs.entries {
		entry := fcs.entries[i]

		if err := entry.write(fcs.fs); err != nil {
			return fmt.Errorf("cannot write entry %s: %w", entry.Path, err)
		}
	}

	return nil
}

func (fcs *fsChangeSet) hash() (string, error) {
	if fcs.entries == nil {
		return "", fmt.Errorf("no entries")
	}

	var combinedContents string

	for i := range fcs.entries {
		if len(fcs.entries[i].Content) == 0 {
			// just making sure we call sha1() after expandContents()
			return "", fmt.Errorf("content (index %d) is empty", i)
		}
		combinedContents += fcs.entries[i].Path + fcs.entries[i].Content
	}

	// #nosec G401 - we're not using sha1 for crypto, only to quickly compare contents
	return fmt.Sprintf("%x", sha1.Sum([]byte(combinedContents))), nil
}

func (fcs *fsChangeSet) writeSummary(out io.Writer) error {
	if fcs.entries == nil {
		return fmt.Errorf("no entries")
	}

	b, err := json.MarshalIndent(fcs.entries, "", "  ")
	if err != nil {
		return fmt.Errorf("cannot marshal entries: %w", err)
	}
	fmt.Fprintln(out, b)

	return nil
}
