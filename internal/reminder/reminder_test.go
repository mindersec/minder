// Copyright 2024 Stacklok, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package reminder

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	reminderconfig "github.com/stacklok/minder/internal/config/reminder"
)

func Test_cursorStateBackup(t *testing.T) {
	t.Parallel()

	tempDirPath := t.TempDir()
	cursorFilePath := filepath.Join(tempDirPath, "cursor")
	repoListCursor := map[projectProviderPair]string{
		{
			ProjectId: generateUUIDFromNum(t, 1),
			Provider:  "github",
		}: "repo-cursor-1",
		{
			ProjectId: generateUUIDFromNum(t, 2),
			Provider:  "gitlab",
		}: "repo-cursor-2",
	}
	projectCursor := "project-cursor"

	r := &reminder{
		cfg: &reminderconfig.Config{
			CursorFile: cursorFilePath,
		},
		projectListCursor: projectCursor,
		repoListCursor:    repoListCursor,
	}

	ctx := context.Background()

	err := r.storeCursorState(ctx)
	require.NoError(t, err)

	// Set cursors to empty values to check if they are restored
	r.projectListCursor = ""
	r.repoListCursor = nil

	err = r.restoreCursorState(ctx)
	require.NoError(t, err)

	require.Equal(t, projectCursor, r.projectListCursor)
	require.Equal(t, len(repoListCursor), len(r.repoListCursor))
	for k, v := range repoListCursor {
		require.Equal(t, v, r.repoListCursor[k])
	}
}

func generateUUIDFromNum(t *testing.T, num int) uuid.UUID {
	t.Helper()

	numberStr := fmt.Sprintf("%d", num)

	uuidStr := fmt.Sprintf("00000000-0000-0000-0000-%012s", numberStr)

	u, err := uuid.Parse(uuidStr)
	if err != nil {
		t.Errorf("error parsing UUID: %v", err)
	}

	return u
}
