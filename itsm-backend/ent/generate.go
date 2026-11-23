package ent

// Deprecated: use `make entgen` from the repository root. This directive
// writes in place and bypasses the isolated staging + SHA-256 sync pipeline;
// it is kept only for compatibility with legacy scripts. See
// docs/delivery/entc-codegen-runbook.md.
//go:generate go run -mod=mod entgo.io/ent/cmd/ent generate ./schema
