package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/trackingprofile"
	"github.com/mokevnin/1mail/ent/trackingvisitor"
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

func IdentifyVisitor(ctx context.Context, client *ent.Client, workspaceID int64, input IdentifyInput) error {
	visitorID := normalizeStringVal(input.VisitorID)
	if visitorID == "" {
		return errors.New("visitorId is required")
	}

	email := normalizeLower(input.Email)
	phone := normalizeOptional(input.Phone)
	subjectID := normalizeString(input.SubjectID)
	traits := normalizeTraits(input.Traits)
	canonicalSubjectID, err := deriveCanonicalSubjectID(subjectID, email, phone)
	if err != nil {
		return err
	}

	profile, err := upsertProfile(ctx, client, workspaceID, canonicalSubjectID, email, phone, traits)
	if err != nil {
		return err
	}

	visitor, err := findOrCreateVisitor(ctx, client, workspaceID, visitorID)
	if err != nil {
		return err
	}

	return client.TrackingVisitor.UpdateOneID(visitor.ID).
		SetProfileID(profile.ID).
		SetLastSeenAt(time.Now()).
		Exec(ctx)
}

func CollectEvents(ctx context.Context, client *ent.Client, workspaceID int64, events []CollectEventInput) error {
	for _, evt := range events {
		visitorID := normalizeStringVal(evt.VisitorID)
		resolution, err := resolveTrackingIdentity(ctx, client, workspaceID, visitorID)
		if err != nil {
			return err
		}

		q := client.Event.Create().
			SetWorkspaceID(workspaceID).
			SetSubjectID(resolution.subjectID).
			SetAction(evt.Action)
		if resolution.email != "" {
			q = q.SetEmail(resolution.email)
		}
		if resolution.phone != "" {
			q = q.SetPhone(resolution.phone)
		}
		if evt.Properties != nil {
			q = q.SetProperties(evt.Properties)
		}
		if evt.OccurredAt != nil {
			q = q.SetOccurredAt(*evt.OccurredAt)
		}
		if _, err := q.Save(ctx); err != nil {
			return err
		}
	}
	return nil
}

type trackingResolution struct {
	subjectID string
	email     string
	phone     string
}

func resolveTrackingIdentity(ctx context.Context, client *ent.Client, workspaceID int64, visitorID string) (*trackingResolution, error) {
	visitor, err := findOrCreateVisitor(ctx, client, workspaceID, visitorID)
	if err != nil {
		return nil, err
	}
	if visitor.ProfileID == nil {
		return &trackingResolution{subjectID: "visitor:" + visitorID}, nil
	}
	profile, err := client.TrackingProfile.Get(ctx, *visitor.ProfileID)
	if err != nil {
		return &trackingResolution{subjectID: "visitor:" + visitorID}, nil
	}
	res := &trackingResolution{subjectID: profile.SubjectID}
	if profile.Email != nil {
		res.email = *profile.Email
	}
	if profile.Phone != nil {
		res.phone = *profile.Phone
	}
	return res, nil
}

func findOrCreateVisitor(ctx context.Context, client *ent.Client, workspaceID int64, visitorID string) (*ent.TrackingVisitor, error) {
	existing, err := client.TrackingVisitor.Query().
		Where(trackingvisitor.VisitorID(visitorID), trackingvisitor.WorkspaceID(workspaceID)).
		First(ctx)
	if err == nil {
		return client.TrackingVisitor.UpdateOneID(existing.ID).
			SetLastSeenAt(time.Now()).
			Save(ctx)
	}
	if !ent.IsNotFound(err) {
		return nil, err
	}
	return client.TrackingVisitor.Create().
		SetWorkspaceID(workspaceID).
		SetVisitorID(visitorID).
		SetLastSeenAt(time.Now()).
		Save(ctx)
}

func upsertProfile(ctx context.Context, client *ent.Client, workspaceID int64, subjectID string, email, phone *string, traits map[string]any) (*ent.TrackingProfile, error) {
	existing, err := findProfile(ctx, client, workspaceID, subjectID, email, phone)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		merged := mergeTraits(existing.Traits, traits)
		q := client.TrackingProfile.UpdateOneID(existing.ID).SetTraits(merged)
		if email != nil && existing.Email == nil {
			q = q.SetEmail(*email)
		}
		if phone != nil && existing.Phone == nil {
			q = q.SetPhone(*phone)
		}
		return q.Save(ctx)
	}
	q := client.TrackingProfile.Create().
		SetWorkspaceID(workspaceID).
		SetSubjectID(subjectID).
		SetTraits(traits)
	if email != nil {
		q = q.SetEmail(*email)
	}
	if phone != nil {
		q = q.SetPhone(*phone)
	}
	return q.Save(ctx)
}

func findProfile(ctx context.Context, client *ent.Client, workspaceID int64, subjectID string, email, phone *string) (*ent.TrackingProfile, error) {
	if subjectID != "" {
		p, err := client.TrackingProfile.Query().
			Where(trackingprofile.SubjectID(subjectID), trackingprofile.WorkspaceID(workspaceID)).First(ctx)
		if err == nil {
			return p, nil
		}
		if !ent.IsNotFound(err) {
			return nil, err
		}
	}
	if email != nil {
		p, err := client.TrackingProfile.Query().
			Where(trackingprofile.Email(*email), trackingprofile.WorkspaceID(workspaceID)).First(ctx)
		if err == nil {
			return p, nil
		}
		if !ent.IsNotFound(err) {
			return nil, err
		}
	}
	if phone != nil {
		p, err := client.TrackingProfile.Query().
			Where(trackingprofile.Phone(*phone), trackingprofile.WorkspaceID(workspaceID)).First(ctx)
		if err == nil {
			return p, nil
		}
		if !ent.IsNotFound(err) {
			return nil, err
		}
	}
	return nil, nil
}

func deriveCanonicalSubjectID(subjectID string, email, phone *string) (string, error) {
	if subjectID != "" {
		return subjectID, nil
	}
	if email != nil {
		return "email:" + *email, nil
	}
	if phone != nil {
		return "phone:" + *phone, nil
	}
	return "", fmt.Errorf("identify requires subjectId, email, or phone")
}

func mergeTraits(existing, incoming map[string]any) map[string]any {
	return lo.Assign(existing, incoming)
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
