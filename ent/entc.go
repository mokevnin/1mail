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
		},
	})
	if err != nil {
		log.Fatal("running ent codegen:", err)
	}
}
