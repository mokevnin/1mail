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
	"github.com/mokevnin/1mail/internal/automations"
)

// automationSteps decodes the stored definition into the typed steps DTO. A
// malformed or empty definition yields no steps rather than an error.
func automationSteps(def string) []externalapi.AutomationStep {
	stored, err := automations.Decode(def)
	if err != nil {
		return nil
	}
	steps := make([]externalapi.AutomationStep, 0, len(stored))
	for _, s := range stored {
		step := externalapi.AutomationStep{Type: externalapi.AutomationStepType(s.Type)}
		switch s.Type {
		case automations.StepWait:
			step.Seconds = externalapi.NewOptInt32(int32(s.Seconds))
		case automations.StepApplyTag, automations.StepRemoveTag:
			step.Tag = externalapi.NewOptString(s.Tag)
		default:
			step.Subject = externalapi.NewOptString(s.Subject)
			step.Body = externalapi.NewOptString(s.Body)
		}
		steps = append(steps, step)
	}
	return steps
}

// AutomationSteps converts the typed steps DTO to the automations module's steps.
func AutomationSteps(steps []externalapi.AutomationStep) []automations.Step {
	out := make([]automations.Step, 0, len(steps))
	for _, s := range steps {
		out = append(out, automations.Step{
			Type:    string(s.Type),
			Subject: s.Subject.Or(""),
			Body:    s.Body.Or(""),
			Seconds: int(s.Seconds.Or(0)),
			Tag:     s.Tag.Or(""),
		})
	}
	return out
}

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
// goverter:extend optString
// goverter:extend optNilEmailAddress
// goverter:extend optNilTimeZone
// goverter:extend optNilTimestamp
// goverter:extend optNilEntityID
// goverter:extend contactCustomFields
type Converter interface {
	ContactToResource(source *ent.Contact) externalapi.ContactResource
	BroadcastToResource(source *ent.Broadcast) externalapi.BroadcastResource
	ApiTokenToInfo(source *ent.ApiToken) externalapi.ApiTokenInfo
	SegmentToResource(source *ent.Segment) externalapi.SegmentResource
	EmailTemplateToResource(source *ent.EmailTemplate) externalapi.TemplateResource
	CustomFieldToResource(source *ent.CustomField) externalapi.CustomFieldResource
	SendingDomainToResource(source *ent.SendingDomain) externalapi.SendingDomainResource
	TagToResource(source *ent.Tag) externalapi.TagResource
	// goverter:map Definition Steps | automationSteps
	AutomationToResource(source *ent.Automation) externalapi.AutomationResource
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

func optString(v *string) externalapi.OptString {
	if v == nil {
		return externalapi.OptString{}
	}
	return externalapi.NewOptString(*v)
}

func optNilEmailAddress(v *string) externalapi.OptNilEmailAddress {
	if v == nil {
		return externalapi.OptNilEmailAddress{}
	}
	return externalapi.NewOptNilEmailAddress(externalapi.EmailAddress(*v))
}

func optNilEntityID(v *int64) externalapi.OptNilEntityId {
	if v == nil {
		return externalapi.OptNilEntityId{}
	}
	return externalapi.NewOptNilEntityId(entityID(*v))
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
