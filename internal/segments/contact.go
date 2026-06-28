package segments

// This file is the application-side adapter that binds the generic engine to the
// project's Contact entity. It is the only file in the package that imports the
// generated ent code; when extracting segments.go into a standalone library,
// this file stays behind in the app.

import (
	"github.com/mokevnin/1mail/ent/contact"
	"github.com/mokevnin/1mail/ent/predicate"
)

// ContactSchema whitelists the contact fields a segment rule may target and maps
// them to columns. Custom fields are addressed as "custom:<key>".
func ContactSchema() Schema {
	return Schema{
		Columns: map[string]string{
			"email":      contact.FieldEmail,
			"first_name": contact.FieldFirstName,
			"last_name":  contact.FieldLastName,
			"time_zone":  contact.FieldTimeZone,
			"status":     contact.FieldStatus,
		},
		JSONColumns: map[string]string{
			"custom:": contact.FieldCustomFields,
		},
	}
}

// ContactPredicate compiles a stored segment definition into a contact predicate
// for use in Contact.Query().Where(...).
func ContactPredicate(def string) (predicate.Contact, error) {
	g, err := Parse(def)
	if err != nil {
		return nil, err
	}
	p, err := Compile(g, ContactSchema())
	if err != nil {
		return nil, err
	}
	return predicate.Contact(p), nil
}

// ValidateContactDefinition checks a definition against the contact schema.
func ValidateContactDefinition(def string) error {
	return Validate(def, ContactSchema())
}
