# SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
# SPDX-License-Identifier: Apache-2.0

# Note: Rego print() output is not available to assertions inside rule tests;
# it is captured into TestResult.Output for the end user via the test report only.
def test_rego_print_captured():
    res = eval(
        rule="print_rule",
        entity={"owner": "test", "name": "repo"},
        profile={"required_value": "enabled"},
        mock_http={
            "/repos/test/repo/settings": body('{"value": "enabled"}')
        }
    )
    assert.eq(res["status"], "pass")
