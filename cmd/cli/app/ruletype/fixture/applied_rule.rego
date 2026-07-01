# SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
# SPDX-License-Identifier: Apache-2.0

package minder

# METADATA
# 
# title: 'Applied Rule'
# description: 'A minimal test rule type.'
# custom:
#   release_phase: alpha
#   severity:
#     value: low
#   def:
#     in_entity: repository
#
#     # How to gather data (Minimal REST example)
#     ingest:
#       type: rest
#       rest:
#         endpoint: '/repos/{{.Entity.Owner}}/{{.Entity.Name}}'
#         parse: 'json'
#
#     eval:
#       rego:
#         type: 'deny-by-default'
package minder

import rego.v1

# How to evaluate the data (Minimal Rego example that always passes)
default allow := false

allow := true