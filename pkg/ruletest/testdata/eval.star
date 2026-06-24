# SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
# SPDX-License-Identifier: Apache-2.0

def test_eval_success():
    res = eval(
        rule="rule_type_sample.yaml",
        entity={"required_pull_request_reviews": {"required_approving_review_count": 2}},
        profile={"required_reviews": 2},
    )
    assert.eq(res["status"], "pass")

def test_eval_fail():
    res = eval(
        rule="rule_type_sample.yaml",
        entity={"required_pull_request_reviews": {"required_approving_review_count": 1}},
        profile={"required_reviews": 2},
    )
    assert.eq(res["status"], "fail")
    assert.true(res["message"] != "")
