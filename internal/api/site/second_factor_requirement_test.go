package site_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	siteapi "github.com/mokevnin/1mail/gen/site"
	"github.com/mokevnin/1mail/internal/events"
	"github.com/mokevnin/1mail/internal/fixtures"
	"github.com/mokevnin/1mail/internal/testhelper"
)

// Umbrella's requirement started 2026-03-01: Rita's Membership predates it (grace
// ends 2026-03-08), Nina joined 2026-03-05 (grace ends 2026-03-12). Fixture times
// are local times.
var (
	ritaGraceEnds = time.Date(2026, 3, 1, 0, 0, 0, 0, time.Local).Add(7 * 24 * time.Hour)
	ninaGraceEnds = time.Date(2026, 3, 5, 0, 0, 0, 0, time.Local).Add(7 * 24 * time.Hour)
)

type requirementClock struct{ t time.Time }

func (c *requirementClock) now() time.Time { return c.t }

func requirementEnv(t *testing.T, at time.Time) *testhelper.TestEnv {
	t.Helper()
	c := &requirementClock{t: at}
	return testhelper.Setup(t, testhelper.WithClock(c.now))
}

func setRequirement(t *testing.T, c *siteapi.Client, slug string, on bool) siteapi.SiteWorkspacesSetSecondFactorRequirementRes {
	t.Helper()
	res, err := c.SiteWorkspacesSetSecondFactorRequirement(t.Context(),
		&siteapi.SiteSecondFactorRequirementInput{Required: on},
		siteapi.SiteWorkspacesSetSecondFactorRequirementParams{Slug: slug})
	require.NoError(t, err)
	return res
}

func workspaceOf(t *testing.T, c *siteapi.Client, slug string) siteapi.SiteWorkspaceResource {
	t.Helper()
	list, err := c.SiteWorkspacesList(t.Context())
	require.NoError(t, err)
	for _, w := range list {
		if w.Slug == slug {
			return w
		}
	}
	require.Failf(t, "workspace not listed", "%s", slug)
	return siteapi.SiteWorkspaceResource{}
}

// siteGet sends a GET as the fixture user and returns the status and decoded body.
func siteGet(t *testing.T, env *testhelper.TestEnv, email, path string) (int, map[string]any) {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil)
	req.AddCookie(&http.Cookie{Name: "JWT", Value: env.SiteToken(t, email, nil)})
	env.Server.ServeHTTP(rec, req)
	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	return rec.Code, body
}

func TestOwnerAndAdminSwitchTheRequirementAndAMemberCannot(t *testing.T) {
	at := time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)
	c := &requirementClock{t: at}
	env := testhelper.Setup(t, testhelper.WithClock(c.now))
	john := env.SiteActor(t, fixtures.OwnerJohnEmail)

	res := setRequirement(t, john, fixtures.AcmeSlug, true)
	require.IsType(t, &siteapi.SiteWorkspaceResource{}, res)
	w := res.(*siteapi.SiteWorkspaceResource)
	assert.True(t, time.Time(w.SecondFactorRequiredAt.Value).Equal(at))
	assert.True(t, time.Time(w.SecondFactorGraceEndsAt.Value).Equal(at.Add(7*24*time.Hour)),
		"an existing Membership gets 7 days from the requirement's start")

	entries := env.OutboxEvents(t, events.NameAuditEntry)
	require.Len(t, entries, 1)
	e := entries[0].(*events.AuditEntry)
	assert.Equal(t, events.ActionWorkspaceUpdate, e.Action)
	assert.Contains(t, e.Diff, "second_factor_required_at")

	// Switching it on again keeps the start, so no one's grace restarts.
	c.t = at.Add(48 * time.Hour)
	again := setRequirement(t, env.SiteActor(t, fixtures.OwnerJohnEmail), fixtures.AcmeSlug, true).(*siteapi.SiteWorkspaceResource)
	assert.True(t, time.Time(again.SecondFactorRequiredAt.Value).Equal(at))
	assert.Len(t, env.OutboxEvents(t, events.NameAuditEntry), 1)

	mary := env.SiteActor(t, fixtures.MemberMaryEmail)
	assert.IsType(t, &siteapi.SiteWorkspacesSetSecondFactorRequirementForbidden{}, setRequirement(t, mary, fixtures.AcmeSlug, false))
	oscar := env.SiteActor(t, fixtures.OutsiderOscarEmail)
	assert.IsType(t, &siteapi.SiteWorkspacesSetSecondFactorRequirementNotFound{}, setRequirement(t, oscar, fixtures.AcmeSlug, false))

	sam := env.SiteActor(t, fixtures.SecondFactorSamEmail)
	cleared := setRequirement(t, sam, fixtures.UmbrellaSlug, false)
	require.IsType(t, &siteapi.SiteWorkspaceResource{}, cleared, "an admin may clear it")
	assert.False(t, cleared.(*siteapi.SiteWorkspaceResource).SecondFactorRequiredAt.Set)
	assert.Len(t, env.OutboxEvents(t, events.NameAuditEntry), 2)
}

func TestGraceEndsSevenDaysAfterTheLaterOfTheRequirementAndTheMembership(t *testing.T) {
	env := requirementEnv(t, time.Date(2026, 3, 4, 0, 0, 0, 0, time.Local))

	rita := workspaceOf(t, env.SiteActor(t, fixtures.UmbrellaOwnerRitaEmail), fixtures.UmbrellaSlug)
	assert.True(t, time.Time(rita.SecondFactorGraceEndsAt.Value).Equal(ritaGraceEnds), "existing Membership")

	nina := workspaceOf(t, env.SiteActor(t, fixtures.UmbrellaMemberNinaEmail), fixtures.UmbrellaSlug)
	assert.True(t, time.Time(nina.SecondFactorGraceEndsAt.Value).Equal(ninaGraceEnds), "new Membership")

	sam := workspaceOf(t, env.SiteActor(t, fixtures.SecondFactorSamEmail), fixtures.UmbrellaSlug)
	assert.True(t, sam.SecondFactorRequiredAt.Set)
	assert.False(t, sam.SecondFactorGraceEndsAt.Set, "a User with a Second factor has no deadline")

	initech := workspaceOf(t, env.SiteActor(t, fixtures.UmbrellaOwnerRitaEmail), fixtures.InitechSlug)
	assert.False(t, initech.SecondFactorGraceEndsAt.Set, "a Workspace without the requirement")

	code, _ := siteGet(t, env, fixtures.UmbrellaOwnerRitaEmail, "/site/workspaces/umbrella/tags")
	assert.Equal(t, http.StatusOK, code, "in grace the Workspace is reachable")
}

func TestAfterGraceTheRequiringWorkspaceAloneIsWithheld(t *testing.T) {
	env := requirementEnv(t, ritaGraceEnds.Add(time.Hour))

	code, body := siteGet(t, env, fixtures.UmbrellaOwnerRitaEmail, "/site/workspaces/umbrella/tags")
	assert.Equal(t, http.StatusForbidden, code)
	assert.Equal(t, "second_factor_required", body["code"])

	for _, path := range []string{
		"/site/workspaces/initech/tags",
		"/site/workspaces",
		"/site/me",
		"/site/me/second-factor",
	} {
		code, _ := siteGet(t, env, fixtures.UmbrellaOwnerRitaEmail, path)
		assert.Equal(t, http.StatusOK, code, path)
	}

	code, _ = siteGet(t, env, fixtures.UmbrellaMemberNinaEmail, "/site/workspaces/umbrella/tags")
	assert.Equal(t, http.StatusOK, code, "Nina's grace runs from her later Membership")

	code, _ = siteGet(t, env, fixtures.SecondFactorSamEmail, "/site/workspaces/umbrella/tags")
	assert.Equal(t, http.StatusOK, code, "one Second factor satisfies every Workspace")
}

func TestANewMembershipIsWithheldOnceItsOwnGraceEnds(t *testing.T) {
	env := requirementEnv(t, ninaGraceEnds.Add(time.Hour))
	code, body := siteGet(t, env, fixtures.UmbrellaMemberNinaEmail, "/site/workspaces/umbrella/tags")
	assert.Equal(t, http.StatusForbidden, code)
	assert.Equal(t, "second_factor_required", body["code"])
}
