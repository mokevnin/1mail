// Package resources holds goverter-generated mappers from ent entities to site
// API resources (the ogen *Resource DTOs), plus the custom conversions goverter
// cannot infer on its own. The generated implementation (ConverterImpl) lives in
// converter_gen.go; consumers instantiate it (see internal/api/site).
package resources

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/go-faster/jx"
	"github.com/mokevnin/1mail/ent"
	siteapi "github.com/mokevnin/1mail/gen/site"
)

// Converter maps ent entities to site API resources. goverter generates the
// implementation; named string/time casts are inferred, the extends below cover
// the conversions it can't (ids, optionals, the event property bag).
//
// goverter:converter
// goverter:output:file ./converter_gen.go
// goverter:matchIgnoreCase
// goverter:useZeroValueOnPointerInconsistency
// goverter:extend entityID
// goverter:extend timestamp
// goverter:extend optNilString
// goverter:extend optNilEmailAddress
// goverter:extend optNilEntityID
// goverter:extend optNilTimeZone
// goverter:extend optNilTimestamp
// goverter:extend contactCustomFields
// goverter:extend eventProperties
// goverter:extend broadcastStats
type Converter interface {
	ContactToResource(source *ent.Contact) siteapi.SiteContactResource
	SegmentToResource(source *ent.Segment) siteapi.SiteSegmentResource
	TokenToResource(source *ent.ApiToken) siteapi.SiteApiTokenResource
	UserToResource(source *ent.User) *siteapi.SiteUserResource
	WorkspaceToResource(source *ent.Workspace) siteapi.SiteWorkspaceResource
	EventToResource(source *ent.Event) siteapi.SiteEventResource

	// The flat stat counters on the broadcast are folded into the nested Stats
	// object; map the whole source through broadcastStats.
	// goverter:map . Stats
	BroadcastToResource(source *ent.Broadcast) siteapi.SiteBroadcastResource

	EmailTemplateToResource(source *ent.EmailTemplate) siteapi.SiteEmailTemplateResource
	AutomationToResource(source *ent.Automation) siteapi.SiteAutomationResource
	SuppressionToResource(source *ent.Suppression) siteapi.SiteSuppressionResource
}

func entityID(id int64) siteapi.EntityId {
	return siteapi.EntityId(strconv.FormatInt(id, 10))
}

func timestamp(t time.Time) siteapi.Timestamp {
	return siteapi.Timestamp(t)
}

func optNilString(v *string) siteapi.OptNilString {
	if v == nil {
		return siteapi.OptNilString{}
	}
	return siteapi.NewOptNilString(*v)
}

func optNilEmailAddress(v *string) siteapi.OptNilEmailAddress {
	if v == nil {
		return siteapi.OptNilEmailAddress{}
	}
	return siteapi.NewOptNilEmailAddress(siteapi.EmailAddress(*v))
}

func optNilEntityID(v *int64) siteapi.OptNilEntityId {
	if v == nil {
		return siteapi.OptNilEntityId{}
	}
	return siteapi.NewOptNilEntityId(entityID(*v))
}

// broadcastStats folds the broadcast's flat delivery counters into the nested
// Stats DTO and computes the derived engagement rates. Rates are ratios in [0,1];
// a zero denominator yields 0 (see ratio). It is wired as a goverter extend so the
// `. -> Stats` mapping on BroadcastToResource routes through it.
func broadcastStats(b ent.Broadcast) siteapi.SiteBroadcastStats {
	return siteapi.SiteBroadcastStats{
		RecipientsTotal:   int32(b.RecipientsTotal),
		SentCount:         int32(b.SentCount),
		OpenedCount:       int32(b.OpenedCount),
		ClickedCount:      int32(b.ClickedCount),
		UnsubscribedCount: int32(b.UnsubscribedCount),
		FailedCount:       int32(b.FailedCount),
		DeliveryRate:      ratio(b.SentCount, b.RecipientsTotal),
		OpenRate:          ratio(b.OpenedCount, b.SentCount),
		ClickRate:         ratio(b.ClickedCount, b.SentCount),
		ClickToOpenRate:   ratio(b.ClickedCount, b.OpenedCount),
		UnsubscribeRate:   ratio(b.UnsubscribedCount, b.SentCount),
		FailureRate:       ratio(b.FailedCount, b.RecipientsTotal),
	}
}

// ratio is num/denom as a float32 in [0,1], guarding against a zero denominator.
func ratio(num, denom int) float32 {
	if denom <= 0 {
		return 0
	}
	return float32(num) / float32(denom)
}

func optNilTimeZone(v *string) siteapi.OptNilTimeZoneName {
	if v == nil {
		return siteapi.OptNilTimeZoneName{}
	}
	return siteapi.NewOptNilTimeZoneName(siteapi.TimeZoneName(*v))
}

func optNilTimestamp(v *time.Time) siteapi.OptNilTimestamp {
	if v == nil {
		return siteapi.OptNilTimestamp{}
	}
	return siteapi.NewOptNilTimestamp(siteapi.Timestamp(*v))
}

func contactCustomFields(m map[string]string) siteapi.OptNilSiteContactResourceCustomFields {
	if m == nil {
		return siteapi.OptNilSiteContactResourceCustomFields{}
	}
	return siteapi.NewOptNilSiteContactResourceCustomFields(siteapi.SiteContactResourceCustomFields(m))
}

func eventProperties(m map[string]any) siteapi.OptNilSiteEventResourceProperties {
	if len(m) == 0 {
		return siteapi.OptNilSiteEventResourceProperties{}
	}
	props := make(siteapi.SiteEventResourceProperties, len(m))
	for k, v := range m {
		b, err := json.Marshal(v)
		if err != nil {
			continue
		}
		props[k] = jx.Raw(b)
	}
	return siteapi.NewOptNilSiteEventResourceProperties(props)
}
