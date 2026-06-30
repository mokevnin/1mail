package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/contact"
	"github.com/mokevnin/1mail/ent/event"
	"github.com/mokevnin/1mail/ent/visitor"
	"github.com/mokevnin/1mail/internal/events"
	"github.com/samber/lo"
)

type IdentifyInput struct {
	VisitorID string
	Email     *string
	Phone     *string
	SubjectID *string
	Traits    map[string]any
}

type CollectEventInput struct {
	VisitorID  string
	Action     string
	Properties map[string]any
	OccurredAt *time.Time
}

// IdentifyVisitor binds a Visitor to a Contact and asserts that Contact's alias keys
// (subject_id / email / phone). It upserts the Contact by any present alias key,
// auto-creates typed Custom fields from the traits, binds the device, and stitches
// the device's earlier anonymous Events onto the Contact so pre-identify behavior
// becomes visible to segmentation. A newly created Contact emits contact.created.
func IdentifyVisitor(ctx context.Context, bus *events.Bus, workspaceID int64, input IdentifyInput) error {
	visitorID := normalizeStringVal(input.VisitorID)
	if visitorID == "" {
		return errors.New("visitorId is required")
	}

	email := normalizeLower(input.Email)
	phone := normalizeOptional(input.Phone)
	subjectID := normalizeString(input.SubjectID)
	traits := normalizeTraits(input.Traits)
	if subjectID == "" && email == nil && phone == nil {
		return fmt.Errorf("identify requires subjectId, email, or phone")
	}

	return bus.WithinTx(ctx, func(tx *ent.Client, pub events.Publisher) error {
		c, created, err := upsertContact(ctx, tx, workspaceID, subjectID, email, phone, traits)
		if err != nil {
			return err
		}

		vis, err := findOrCreateVisitor(ctx, tx, workspaceID, visitorID)
		if err != nil {
			return err
		}
		if err := tx.Visitor.UpdateOneID(vis.ID).
			SetContactID(c.ID).
			SetLastSeenAt(time.Now()).
			Exec(ctx); err != nil {
			return err
		}

		// Stitch: attach the device's earlier anonymous events onto the Contact.
		if _, err := tx.Event.Update().
			Where(
				event.WorkspaceID(workspaceID),
				event.VisitorID(visitorID),
				event.ContactIDIsNil(),
			).
			SetContactID(c.ID).
			Save(ctx); err != nil {
			return err
		}

		if created {
			return pub.Publish(ctx, &events.ContactCreated{
				WorkspaceID: workspaceID,
				ContactID:   c.ID,
				Email:       lo.FromPtr(c.Email),
			})
		}
		return nil
	})
}

func CollectEvents(ctx context.Context, bus *events.Bus, workspaceID int64, evts []CollectEventInput) error {
	for _, evt := range evts {
		visitorID := normalizeStringVal(evt.VisitorID)
		// Resolve identity and publish in one transaction: the visitor upsert and the
		// outbox row commit together. The collected event is the customer's own — its
		// action and properties are stored as-is.
		if err := bus.WithinTx(ctx, func(tx *ent.Client, pub events.Publisher) error {
			res, err := resolveIdentity(ctx, tx, workspaceID, visitorID)
			if err != nil {
				return err
			}
			var occurred time.Time
			if evt.OccurredAt != nil {
				occurred = *evt.OccurredAt
			}
			return pub.Publish(ctx, &events.CollectedEvent{
				WorkspaceID: workspaceID,
				ContactID:   res.contactID,
				VisitorID:   visitorID,
				SubjectID:   res.subjectID,
				Email:       res.email,
				Phone:       res.phone,
				Action:      evt.Action,
				Properties:  evt.Properties,
				OccurredAt:  occurred,
			})
		}); err != nil {
			return err
		}
	}
	return nil
}

// identityResolution is the Contact a device resolves to at event-ingest time. A
// zero contactID means anonymous (the device is not yet identified); the event is
// recorded with a null contact_id and stitched onto a Contact at the next Identify.
type identityResolution struct {
	contactID int64
	subjectID string
	email     string
	phone     string
}

func resolveIdentity(ctx context.Context, client *ent.Client, workspaceID int64, visitorID string) (*identityResolution, error) {
	vis, err := findOrCreateVisitor(ctx, client, workspaceID, visitorID)
	if err != nil {
		return nil, err
	}
	if vis.ContactID == nil {
		return &identityResolution{}, nil // anonymous
	}
	c, err := client.Contact.Get(ctx, *vis.ContactID)
	if err != nil {
		return &identityResolution{}, nil // contact gone; treat as anonymous
	}
	return &identityResolution{
		contactID: c.ID,
		subjectID: lo.FromPtr(c.SubjectID),
		email:     lo.FromPtr(c.Email),
		phone:     lo.FromPtr(c.Phone),
	}, nil
}

func findOrCreateVisitor(ctx context.Context, client *ent.Client, workspaceID int64, visitorID string) (*ent.Visitor, error) {
	existing, err := client.Visitor.Query().
		Where(visitor.VisitorID(visitorID), visitor.WorkspaceID(workspaceID)).
		First(ctx)
	if err == nil {
		return client.Visitor.UpdateOneID(existing.ID).
			SetLastSeenAt(time.Now()).
			Save(ctx)
	}
	if !ent.IsNotFound(err) {
		return nil, err
	}
	return client.Visitor.Create().
		SetWorkspaceID(workspaceID).
		SetVisitorID(visitorID).
		SetLastSeenAt(time.Now()).
		Save(ctx)
}

// upsertContact resolves a Contact by any present alias key (subject_id → email →
// phone), filling in alias keys it was missing and merging typed Custom field
// values. Returns whether a new Contact was created. Aliases already present are
// never overwritten (identity is additive here).
func upsertContact(ctx context.Context, client *ent.Client, workspaceID int64, subjectID string, email, phone *string, traits map[string]any) (*ent.Contact, bool, error) {
	typed, err := EnsureCustomFields(ctx, client, workspaceID, traits)
	if err != nil {
		return nil, false, err
	}

	existing, err := findContact(ctx, client, workspaceID, subjectID, email, phone)
	if err != nil {
		return nil, false, err
	}
	if existing != nil {
		q := client.Contact.UpdateOneID(existing.ID)
		if subjectID != "" && existing.SubjectID == nil {
			q.SetSubjectID(subjectID)
		}
		if email != nil && existing.Email == nil {
			q.SetEmail(*email)
		}
		if phone != nil && existing.Phone == nil {
			q.SetPhone(*phone)
		}
		if len(typed) > 0 {
			q.SetCustomFields(lo.Assign(existing.CustomFields, typed))
		}
		c, err := q.Save(ctx)
		return c, false, err
	}

	q := client.Contact.Create().SetWorkspaceID(workspaceID)
	if subjectID != "" {
		q.SetSubjectID(subjectID)
	}
	if email != nil {
		q.SetEmail(*email)
	}
	if phone != nil {
		q.SetPhone(*phone)
	}
	if len(typed) > 0 {
		q.SetCustomFields(typed)
	}
	c, err := q.Save(ctx)
	return c, true, err
}

// ResolveContactID resolves an existing Contact by any present alias key (subject_id
// → email → phone) and returns its id, or 0 if none matches. It never creates a
// Contact — used by event ingest to attach an event to a Contact by stable identity
// when one already exists, leaving it anonymous (0) otherwise.
func ResolveContactID(ctx context.Context, client *ent.Client, workspaceID int64, subjectID string, email, phone *string) (int64, error) {
	c, err := findContact(ctx, client, workspaceID, normalizeStringVal(subjectID), normalizeLower(email), normalizeOptional(phone))
	if err != nil {
		return 0, err
	}
	if c == nil {
		return 0, nil
	}
	return c.ID, nil
}

func findContact(ctx context.Context, client *ent.Client, workspaceID int64, subjectID string, email, phone *string) (*ent.Contact, error) {
	if subjectID != "" {
		c, err := client.Contact.Query().
			Where(contact.SubjectID(subjectID), contact.WorkspaceID(workspaceID)).First(ctx)
		if err == nil {
			return c, nil
		}
		if !ent.IsNotFound(err) {
			return nil, err
		}
	}
	if email != nil {
		c, err := client.Contact.Query().
			Where(contact.Email(*email), contact.WorkspaceID(workspaceID)).First(ctx)
		if err == nil {
			return c, nil
		}
		if !ent.IsNotFound(err) {
			return nil, err
		}
	}
	if phone != nil {
		c, err := client.Contact.Query().
			Where(contact.Phone(*phone), contact.WorkspaceID(workspaceID)).First(ctx)
		if err == nil {
			return c, nil
		}
		if !ent.IsNotFound(err) {
			return nil, err
		}
	}
	return nil, nil
}

func normalizeString(s *string) string {
	if s == nil {
		return ""
	}
	return strings.TrimSpace(*s)
}

func normalizeStringVal(s string) string {
	return strings.TrimSpace(s)
}

func normalizeLower(s *string) *string {
	if s == nil {
		return nil
	}
	v := strings.ToLower(strings.TrimSpace(*s))
	if v == "" {
		return nil
	}
	return &v
}

func normalizeOptional(s *string) *string {
	if s == nil {
		return nil
	}
	v := strings.TrimSpace(*s)
	if v == "" {
		return nil
	}
	return &v
}

func normalizeTraits(t map[string]any) map[string]any {
	if t == nil {
		return map[string]any{}
	}
	return t
}
