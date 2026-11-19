//go:build entgen

// Command entgen is the pinned ent code generator used by scripts/entgen.sh.
//
// It exists because the stock `ent generate --target <dir>` rewrites the
// generated package path to the target directory. That is fine for a real
// target, but it makes an isolated staging directory unusable: every import in
// the staged code would point at the staging package instead of
// itsm-backend/ent. Here the package path is fixed while the target stays
// configurable, so a failed generation can never leave half-written files in
// the tracked ent/ tree.
package main

import (
	"flag"
	"log"
	"os"

	"entgo.io/ent/entc"
	"entgo.io/ent/entc/gen"
	"entgo.io/ent/schema/field"
)

func main() {
	target := flag.String("target", "", "isolated output directory for generated code")
	flag.Parse()
	if *target == "" {
		log.Fatal("entgen: -target is required")
	}
	if err := os.MkdirAll(*target, 0o755); err != nil {
		log.Fatalf("entgen: create target: %v", err)
	}
	cfg := &gen.Config{
		Package: "itsm-backend/ent",
		Target:  *target,
		// Matches the stock CLI default so staged output is byte-identical
		// to the committed ent/ tree when schemas have not changed.
		IDType: &field.TypeInfo{Type: field.TypeInt},
	}
	if err := entc.Generate("./ent/schema", cfg); err != nil {
		log.Fatalf("entgen: generate: %v", err)
	}
}
