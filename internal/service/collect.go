package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/event"
	"github.com/mokevnin/1mail/ent/visitor"
	"github.com/mokevnin/1mail/internal/contacts"
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
	visitorID := strings.TrimSpace(input.VisitorID)
	if visitorID == "" {
		return errors.New("visitorId is required")
	}

	return bus.WithinTx(ctx, func(tx *ent.Client, pub events.Publisher) error {
		res, err := contacts.UpsertIn(ctx, tx, pub, workspaceID, contacts.Attributes{
			SubjectID:    input.SubjectID,
			Email:        input.Email,
			Phone:        input.Phone,
			CustomFields: input.Traits,
		})
		if errors.Is(err, contacts.ErrIdentityRequired) {
			return errors.New("identify requires subjectId, email, or phone")
		}
		if err != nil {
			return err
		}
		c := res.Contact

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

		return nil
	})
}

func CollectEvents(ctx context.Context, bus *events.Bus, workspaceID int64, evts []CollectEventInput) error {
	for _, evt := range evts {
		visitorID := strings.TrimSpace(evt.VisitorID)
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

// ResolveContactID resolves an existing Contact by any present alias key (subject_id
// → email → phone) and returns its id, or 0 if none matches. It never creates a
// Contact — used by event ingest to attach an event to a Contact by stable identity
// when one already exists, leaving it anonymous (0) otherwise.
func ResolveContactID(ctx context.Context, client *ent.Client, workspaceID int64, subjectID string, email, phone *string) (int64, error) {
	c, err := contacts.Resolve(ctx, client, workspaceID, &subjectID, email, phone)
	if err != nil || c == nil {
		return 0, err
	}
	return c.ID, nil
}
