//go:build ignore

package main

import (
	"log"
	"strings"
	"text/template"

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
		// The scoped client (ADR 0017): ent/template/scoped*.tmpl generate ent/scoped*.go.
		Templates: []*gen.Template{
			gen.MustParse(gen.NewTemplate("scoped").Funcs(scopedFuncs()).ParseDir("template")),
		},
	})
	if err != nil {
		log.Fatal("running ent codegen:", err)
	}
}

// scopedRef is a plain field that references another Workspace-owned entity.
type scopedRef struct {
	Field  *gen.Field
	Target *gen.Type
}

// scopedFuncs are the helpers ent/template/scoped.tmpl needs to find the
// workspace-mixin entities and the references between them.
func scopedFuncs() template.FuncMap {
	return template.FuncMap{
		// scopedNodes lists the entities that carry the workspace mixin.
		"scopedNodes": func(g *gen.Graph) []*gen.Type {
			checkScopedRefs(g)
			var out []*gen.Type
			for _, n := range g.Nodes {
				if isScoped(n) {
					out = append(out, n)
				}
			}
			return out
		},
		"scopedIs": isScoped,
		// scopedEdges lists the edges of n that point at another scoped entity.
		"scopedEdges": func(n *gen.Type) []*gen.Edge {
			var out []*gen.Edge
			for _, e := range n.EdgesWithID() {
				if isScoped(e.Type) {
					out = append(out, e)
				}
			}
			return out
		},
		// scopedEdgeMutable reports whether the edge can be changed on update.
		"scopedEdgeMutable": func(e *gen.Edge) bool {
			if e.Immutable {
				return false
			}
			f := e.Field()
			return f == nil || !f.Immutable
		},
		// scopedFieldRefs lists the annotated plain fields of n that hold a reference.
		"scopedFieldRefs": func(g *gen.Graph, n *gen.Type) []scopedRef {
			var out []scopedRef
			for _, f := range n.Fields {
				if t := annotatedTarget(g, f); t != nil {
					out = append(out, scopedRef{Field: f, Target: t})
				}
			}
			return out
		},
		// scopedRefTarget returns the scoped entity a field references (an edge
		// field or an annotated plain field), or nil.
		"scopedRefTarget": func(g *gen.Graph, f *gen.Field) *gen.Type {
			if f.IsEdgeField() {
				if e, err := f.Edge(); err == nil && isScoped(e.Type) {
					return e.Type
				}
				return nil
			}
			return annotatedTarget(g, f)
		},
	}
}

// checkScopedRefs fails the generation when a scoped entity has a plain id column
// that is neither an edge field nor annotated: its references would go unchecked.
func checkScopedRefs(g *gen.Graph) {
	for _, n := range g.Nodes {
		if !isScoped(n) {
			continue
		}
		for _, f := range n.Fields {
			if f.Name == "workspace_id" || f.IsEdgeField() || f.Type.Type != field.TypeInt64 || !strings.HasSuffix(f.Name, "_id") {
				continue
			}
			if ann, ok := f.Annotations["ScopedRef"].(map[string]any); ok && ann["Unchecked"] == true {
				continue
			}
			if annotatedTarget(g, f) == nil {
				log.Fatalf("%s.%s looks like a reference: annotate it with schema.ScopedRef{Entity: ...} so the scoped client verifies it", n.Name, f.Name)
			}
		}
	}
}

func isScoped(t *gen.Type) bool {
	for _, f := range t.Fields {
		if f.Name == "workspace_id" && f.Position != nil && f.Position.MixedIn {
			return true
		}
	}
	return false
}

func annotatedTarget(g *gen.Graph, f *gen.Field) *gen.Type {
	ann, ok := f.Annotations["ScopedRef"].(map[string]any)
	if !ok {
		return nil
	}
	name, _ := ann["Entity"].(string)
	if name == "" {
		return nil
	}
	for _, n := range g.Nodes {
		if n.Name == name {
			return n
		}
	}
	log.Fatalf("ScopedRef on %s names an unknown entity %q", f.Name, name)
	return nil
}
