package middleware

// Phase 07 accepts M2M traffic through the same TSID-like JWT validator used
// for user traffic. A token is treated as M2M when the validated claims contain
// amr=client_credentials or a service-looking subject such as svc-orders.
//
// The split is intentionally claim-based instead of endpoint-based: the
// Warehouse BC consumes the protocol contract exposed by IAM, not IAM's
// implementation language or internal account model.
