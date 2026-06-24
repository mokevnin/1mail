// Package convert holds small generic adapters between the ogen-generated
// Opt*/OptNil* option types and the plain Go pointers that ent setters expect.
// They replace the per-package optNil* helpers that the site and external API
// handlers previously duplicated.
package convert

// getter is implemented by every ogen Opt*/OptNil* value type.
type getter[T any] interface {
	Get() (T, bool)
}

// Ptr returns a pointer to the option's value, or nil when it is unset/null.
func Ptr[T any, O getter[T]](o O) *T {
	if v, ok := o.Get(); ok {
		return &v
	}
	return nil
}

// StringPtr is like Ptr but coerces named string option values (e.g. a
// TimeZoneName scalar) down to *string, the type ent's Nillable setters take.
func StringPtr[T ~string, O getter[T]](o O) *string {
	if v, ok := o.Get(); ok {
		s := string(v)
		return &s
	}
	return nil
}
