# METADATA
#
# title: Test ruletype in Rego format
# description: |
#   A longer description of this ruletype
# custom:
#   release_phase: alpha
#   short_failure_message: This failed
#   guidance: |
#     You should do better
#   def:
#     in_entity: pull_request
#     ingest:
#       type: diff
#       diff:
#         type: full
#     eval:
#       data_sources: [{name: ds_a}, {name: ghapi_comments}]
#       rego:
#         type: constraints
package minder

import rego.v1

violations contains {"msg": "a simple violation"} if {
	input.creator == "banned"
}

violations contains {"msg": msg} if {
	some comment in minder.datasource.ghapi_comments.pr_comment({
		"owner": input.properties["github/repo_owner"],
		"repo": input.properties["github/repo_name"],
		"pr": input.properties["github/pr_number"],
	})
	contains(comment.body, "badword")

	msg = sprintf("Comment %d is naughty", [comment.id])
}
