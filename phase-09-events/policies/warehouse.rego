# The authorization policy of the Warehouse BC: the rules you wrote in
# Phase 07, given here unchanged. Nothing in this phase touches
# authorization; the identity envelope keeps doing its job while the
# integration style underneath changes.
#
# The input document the BC sends for every decision:
#
#   {
#     "action": "article:read" | "article:create" | ...,
#     "principal": {                     # absent when the caller has no identity
#       "kind": "user" | "m2m",
#       "email": "alice@example.com",    # set for users
#       "service_id": "svc-orders"       # set for M2M services
#     }
#   }

package warehouse

import rego.v1

# article_creators lists the principals allowed to create articles.
article_creators := {
	"alice@example.com", # Alice Demo, the warehouse manager
	"svc-orders", # the orders service (M2M)
}

default allow := false # deny by default: the cases nobody thought about land here

# every authenticated caller may read
allow if {
	input.action == "article:read"
	input.principal.kind in {"user", "m2m"}
}

# only principals on the creators list may create
allow if {
	input.action == "article:create"
	principal_id in article_creators
}

# users are identified by email, M2M services by their service id
principal_id := input.principal.email if input.principal.kind == "user"

principal_id := input.principal.service_id if input.principal.kind == "m2m"

# "No identity, no decision" needs no rule of its own: with no principal in
# the input, every body above fails to match and the default answers false.
