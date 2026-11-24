# SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
# SPDX-License-Identifier: Apache-2.0

def test_pr_without_dependency_changes():
    res = eval(
        rule="pr_vulnerability_check",
        entity={"repo_owner": "test", "repo_name": "repo", "number": 1},
        mock_http={
            "/repos/test/repo/pulls/1/files": body('[{"filename": "README.md", "status": "modified"}]'),
        },
    )
    assert.eq(res["status"], "pass")

def test_pr_list_files_error():
    res = eval(
        rule="pr_vulnerability_check",
        entity={"repo_owner": "test", "repo_name": "repo", "number": 1},
        mock_http={
            "/repos/test/repo/pulls/1/files": body('{"message": "Not Found"}').code(404),
        },
    )
    assert.eq(res["status"], "error")
