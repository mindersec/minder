# Tests that rules with param_schema defaults work without explicit params.
# Before the fix, omitting params= would cause a panic because the
# template `index .Params "branch"` would receive a nil map.

def test_param_defaults_applied():
    """Eval should pass when params is omitted and param_schema has defaults."""
    res = eval(
        rule="param_defaults_rule",
        entity={"owner": "test", "name": "repo", "type": "repository", "default_branch": "main"},
        mock_http={
            "/repos/test/repo/branches/main/protection": body('{"allow_deletions": {"enabled": false}}')
        }
    )
    assert.eq(res["status"], "pass")

def test_param_defaults_with_explicit_params():
    """Eval should use explicit params when provided."""
    res = eval(
        rule="param_defaults_rule",
        entity={"owner": "test", "name": "repo", "type": "repository", "default_branch": "main"},
        params={"branch": "develop"},
        mock_http={
            "/repos/test/repo/branches/develop/protection": body('{"allow_deletions": {"enabled": false}}')
        }
    )
    assert.eq(res["status"], "pass")
