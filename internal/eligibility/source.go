package eligibility

import (
	"fmt"
	"strings"
)

// NormalizeDestination is the single definition of "normalized destination":
// lower-cased and trimmed. Suppression and Unsubscribe rows store destinations
// this way, and every lookup must apply it so the point-lookup path (Check) and
// the batch SQL path (Predicate, which folds case via lower()) agree. Keep all
// write and lookup sites routed through here so they cannot drift.
func NormalizeDestination(dest string) string {
	return strings.ToLower(strings.TrimSpace(dest))
}

// Channel identifiers. Email is the only built channel; sms is reserved.
const ChannelEmail = "email"

// Sending sources — the unit a destination unsubscribes from. A source is
// automatic, never hand-authored: it IS the sender. All broadcasts share one
// scope; each automation is its own; "everything" is the reserved deliberate
// "leave entirely" opt-out (only ever an unsubscribe scope, never a send source).
const (
	SourceBroadcasts = "broadcasts"
	SourceEverything = "everything"
)

// AutomationSource is the per-automation unsubscribe scope.
func AutomationSource(automationID int64) string {
	return fmt.Sprintf("automation:%d", automationID)
}
