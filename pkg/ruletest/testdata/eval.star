# SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
# SPDX-License-Identifier: Apache-2.0

def test_eval_success():
    res = eval(
        rule="branch_protection_reviews",
        entity={"owner": "test", "name": "repo"},
        profile={"required_reviews": 2},
        mock_http={
            "/repos/test/repo/branches/main/protection": body('{"required_pull_request_reviews": {"required_approving_review_count": 2}}')
        }
    )
    assert.eq(res["status"], "pass")

def test_eval_fail():
    res = eval(
        rule="branch_protection_reviews",
        entity={"owner": "test", "name": "repo"},
        profile={"required_reviews": 2},
        mock_http={
            "/repos/test/repo/branches/main/protection": body('{"required_pull_request_reviews": {"required_approving_review_count": 1}}')
        }
    )
    assert.eq(res["status"], "fail")
    assert.true(res["message"] != "")

def test_eval_error_404():
    res = eval(
        rule="branch_protection_reviews",
        entity={"owner": "test", "name": "repo"},
        profile={"required_reviews": 2},
        mock_http={
            "/repos/test/repo/branches/main/protection": body('{"message": "Not found"}').code(404)
        }
    )
    assert.eq(res["status"], "fail")
    assert.true(res["message"] != "")

# branch_protection_reviews declares provider_traits: [PROVIDER_TYPE_GITHUB].
# When the test provider doesn't implement that trait, rule type engine
# construction should fail (raising a Starlark error), not silently skip.
def test_fail_eval_missing_provider_trait():
    eval(
        rule="branch_protection_reviews",
        entity={"owner": "test", "name": "repo"},
        profile={"required_reviews": 2},
        provider_missing_traits=["PROVIDER_TYPE_GITHUB"],
    )
