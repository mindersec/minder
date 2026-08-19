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

# branch_protection_reviews declares provider_traits: [github]. When
# provider_traits_present lists other traits but excludes it, the rule type
# is not applicable to the test provider, and eval() reports a skip rather
# than pass/fail/error. This also exercises parseProviderTraitsList against
# real short trait names (the same spelling used in rule type YAML), not
# just the trivial empty-list case.
def test_skip_eval_missing_provider_trait():
    res = eval(
        rule="branch_protection_reviews",
        entity={"owner": "test", "name": "repo"},
        profile={"required_reviews": 2},
        provider_traits_present=["git", "rest"],
    )
    assert.eq(res["status"], "skip")
    assert.true(res["message"] != "")
