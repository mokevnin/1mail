//go:build ignore

package main

import (
	"log"

	"entgo.io/ent/entc"
	"entgo.io/ent/entc/gen"
	"entgo.io/ent/schema/field"
)

func main() {
	err := entc.Generate("./schema", &gen.Config{
		Target:  ".",
		Package: "github.com/mokevnin/1mail/ent",
		IDType:  &field.TypeInfo{Type: field.TypeInt64},
		Features: []gen.Feature{
			gen.FeatureSnapshot,
			gen.FeatureUpsert,
			// Modifier exposes .Modify() on query builders, used by the analytics
			// dashboard for date_trunc day-bucketed engagement aggregates.
			gen.FeatureModifier,
			// VersionedMigration generates migrate.NamedDiff, which cmd/db uses to write
			// the next Atlas migration straight from the ent schema (the community Atlas
			// binary cannot read ent:// itself).
			gen.FeatureVersionedMigration,
		},
	})
	if err != nil {
		log.Fatal("running ent codegen:", err)
	}
}
