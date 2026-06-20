package service

import (
	"regexp"
	"strings"
)

var (
	slugNonAlnum = regexp.MustCompile(`[^a-z0-9]+`)
	slugTrim     = regexp.MustCompile(`^-+|-+$`)
)

// Slugify converts an arbitrary string into a URL-safe slug: lowercased ASCII
// alphanumerics with runs of other characters collapsed to single hyphens.
// Returns "" when the input has no usable characters (callers should fall back).
func Slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = slugNonAlnum.ReplaceAllString(s, "-")
	s = slugTrim.ReplaceAllString(s, "")
	return s
}
