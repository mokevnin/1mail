// Package resources holds goverter-generated mappers from ent entities to
// external API resources, plus the custom conversions goverter cannot infer.
// The generated implementation (ConverterImpl) lives in converter_gen.go;
// consumers instantiate it (see internal/api/external).
package resources

import (
	"strconv"
	"time"

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

func contactCustomFields(m map[string]string) externalapi.OptNilContactResourceCustomFields {
	if m == nil {
		return externalapi.OptNilContactResourceCustomFields{}
	}
	return externalapi.NewOptNilContactResourceCustomFields(externalapi.ContactResourceCustomFields(m))
}
