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
// goverter:extend intToInt32
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

func intToInt32(v int) int32 {
	return int32(v)
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
