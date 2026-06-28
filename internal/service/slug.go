package service

import "github.com/gosimple/slug"

// Slugify converts an arbitrary string into a URL-safe slug: lowercased, with
// Unicode transliterated to ASCII (e.g. "Привет Мир" -> "privet-mir") and runs
// of other characters collapsed to single hyphens. Returns "" when the input
// has no usable characters (callers should fall back). Thin wrapper over
// gosimple/slug so we don't maintain transliteration tables by hand.
func Slugify(s string) string {
	return slug.Make(s)
}
