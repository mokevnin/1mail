// Package resources holds goverter-generated mappers from ent entities to
// external API resources, plus the custom conversions goverter cannot infer.
// The generated implementation (ConverterImpl) lives in converter_gen.go;
// consumers instantiate it (see internal/api/external).
package resources

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/go-faster/jx"
	"github.com/mokevnin/1mail/ent"
	externalapi "github.com/mokevnin/1mail/gen/external"
)

// Converter maps ent entities to external API resources. goverter generates the
// implementation; the extends below cover the conversions it can't infer.
//
// goverter:converter
// goverter:output:file ./converter_gen.go
// goverter:matchIgnoreCase
// goverter:useZeroValueOnPointerInconsistency
// goverter:extend entityID
// goverter:extend timestamp
// goverter:extend optNilString
// goverter:extend optNilEmailAddress
// goverter:extend optNilTimeZone
// goverter:extend optNilTimestamp
// goverter:extend contactCustomFields
type Converter interface {
	ContactToResource(source *ent.Contact) externalapi.ContactResource
	ApiTokenToInfo(source *ent.ApiToken) externalapi.ApiTokenInfo
}

func entityID(id int64) externalapi.EntityId {
	return externalapi.EntityId(strconv.FormatInt(id, 10))
}

func timestamp(t time.Time) externalapi.Timestamp {
	return externalapi.Timestamp(t)
}

func optNilString(v *string) externalapi.OptNilString {
	if v == nil {
		return externalapi.OptNilString{}
	}
	return externalapi.NewOptNilString(*v)
}

func optNilEmailAddress(v *string) externalapi.OptNilEmailAddress {
	if v == nil {
		return externalapi.OptNilEmailAddress{}
	}
	return externalapi.NewOptNilEmailAddress(externalapi.EmailAddress(*v))
}

func optNilTimeZone(v *string) externalapi.OptNilTimeZoneName {
	if v == nil {
		return externalapi.OptNilTimeZoneName{}
	}
	return externalapi.NewOptNilTimeZoneName(externalapi.TimeZoneName(*v))
}

func optNilTimestamp(v *time.Time) externalapi.OptNilTimestamp {
	if v == nil {
		return externalapi.OptNilTimestamp{}
	}
	return externalapi.NewOptNilTimestamp(externalapi.Timestamp(*v))
}

func contactCustomFields(m map[string]any) externalapi.OptNilContactResourceCustomFields {
	if len(m) == 0 {
		return externalapi.OptNilContactResourceCustomFields{}
	}
	fields := make(externalapi.ContactResourceCustomFields, len(m))
	for k, v := range m {
		b, err := json.Marshal(v)
		if err != nil {
			continue
		}
		fields[k] = jx.Raw(b)
	}
	return externalapi.NewOptNilContactResourceCustomFields(fields)
}
