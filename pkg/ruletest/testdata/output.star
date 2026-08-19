# SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
# SPDX-License-Identifier: Apache-2.0

# These tests demonstrate that res["output"] is populated from the rule's rego
# `output` expression, allowing test authors to assert on structured evaluation
# data rather than just status/message.

def test_output_present_on_fail():
    res = eval(
        rule="check_required_setting",
        entity={"owner": "test", "name": "repo"},
        profile={"required_value": "enabled"},
        mock_http={
            "/repos/test/repo/settings": body('{"value": "disabled"}')
        }
    )
    assert.eq(res["status"], "fail")
    assert.eq(res["output"]["actual"], "disabled")
    assert.eq(res["output"]["expected"], "enabled")

def test_output_absent_on_pass():
    res = eval(
        rule="check_required_setting",
        entity={"owner": "test", "name": "repo"},
        profile={"required_value": "enabled"},
        mock_http={
            "/repos/test/repo/settings": body('{"value": "enabled"}')
        }
    )
    assert.eq(res["status"], "pass")
    assert.true("output" not in res)
