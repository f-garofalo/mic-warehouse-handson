# The authorization policy of the Warehouse BC. The rules live HERE, as
# policy code, not in Go: the BC compiles this file in and asks it on every
# guarded route. In Phase 08 the same rules leave the binary entirely,
# served by a policy engine and changeable without a rebuild.
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
#
# TODO Phase 07 Part 2 — the rules:
#   - no identity, no decision: when the input has no principal, no rule
#     below may match, so the default answers (check that yours hold this);
#   - "article:read": every authenticated caller may read;
#   - "article:create": only principals in article_creators. Users are
#     identified by their email, M2M services by their service id
#     (principal.kind tells you which one you have);
#   - any other action: denied. Deny by default is the only safe default
#     for an authorization policy.

package warehouse

import rego.v1

# article_creators lists the principals allowed to create articles.
article_creators := {
	"alice@example.com", # Alice Demo, the warehouse manager
	"svc-orders", # the orders service (M2M)
}

default allow := false # deny everything until you build it
