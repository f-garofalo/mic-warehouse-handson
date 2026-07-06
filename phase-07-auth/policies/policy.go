// Package policies embeds the Warehouse BC authorization policy
// (warehouse.rego) and evaluates it in-process with OPA's Rego engine.
//
// Given code: the rules live in warehouse.rego, not here. This file only
// compiles the policy at startup and answers queries against it.
package policies

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/open-policy-agent/opa/rego"
)

//go:embed warehouse.rego
var warehouseSource string

var warehouseQuery rego.PreparedEvalQuery

func init() {
	q, err := rego.New(
		rego.Query("data.warehouse.allow"),
		rego.Module("warehouse.rego", warehouseSource),
	).PrepareForEval(context.Background())
	if err != nil {
		// A policy that does not compile must fail loudly, not silently deny.
		panic(fmt.Sprintf("policies: warehouse.rego did not compile: %v", err))
	}
	warehouseQuery = q
}

// Allow evaluates data.warehouse.allow against the given input document.
// Evaluation errors fail closed.
func Allow(input map[string]any) bool {
	rs, err := warehouseQuery.Eval(context.Background(), rego.EvalInput(input))
	if err != nil || len(rs) == 0 || len(rs[0].Expressions) == 0 {
		return false
	}
	allowed, ok := rs[0].Expressions[0].Value.(bool)
	return ok && allowed
}
