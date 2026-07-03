# SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
# SPDX-License-Identifier: Apache-2.0

def test_mock_fs_success():
    res = eval(
        rule="mock_fs_rule",
        entity={"owner": "test", "name": "repo", "clone_url": "https://github.com/test/repo.git"},
        mock_fs={
            "test.txt": "hello world\n"
        }
    )
    assert.eq(res["message"], "")
    assert.eq(res["status"], "pass")

def test_mock_fs_fail():
    res = eval(
        rule="mock_fs_rule",
        entity={"owner": "test", "name": "repo", "clone_url": "https://github.com/test/repo.git"},
        mock_fs={
            "test.txt": "goodbye world\n"
        }
    )
    assert.true(res["message"] != "")
    assert.eq(res["status"], "fail")
