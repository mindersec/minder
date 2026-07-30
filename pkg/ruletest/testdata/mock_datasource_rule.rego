# SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
# SPDX-License-Identifier: Apache-2.0

# METADATA
# 
# title: Mock Datasource Rule
# description: 'Uses a data source to fetch some data and eval against it.'
# custom:
#   release_phase: beta
#   severity:
#     value: high
#   def:
#     in_entity: repository
#     rule_schema:
#       properties:
#         required_value:
#           type: string
#           default: "hello"
#     ingest:
#       type: builtin
#       builtin:
#         method: Passthrough
#     eval:
#       rego:
#         type: deny-by-default
package minder

import rego.v1

default allow := false

allow if {
  data_val := minder.datasource.mock_ds.get_val({})
  data_val.body == input.profile.required_value
}
