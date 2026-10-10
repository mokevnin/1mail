package site_test

import (
	"context"
	"encoding/csv"
	"io"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	siteapi "github.com/mokevnin/1mail/gen/site"
	"github.com/mokevnin/1mail/internal/fixtures"
	"github.com/mokevnin/1mail/internal/testhelper"
)

func auditExport(t *testing.T, c *siteapi.Client, slug string) siteapi.SiteAuditExportRes {
	t.Helper()
	res, err := c.SiteAuditExport(context.Background(), siteapi.SiteAuditExportParams{Slug: slug})
	require.NoError(t, err)
	return res
}

func auditCSV(t *testing.T, res siteapi.SiteAuditExportRes) [][]string {
	t.Helper()
	ok, isOK := res.(*siteapi.SiteAuditExportOKHeaders)
	require.Truef(t, isOK, "export answered %T", res)
	assert.Contains(t, ok.ContentDisposition, "attachment")
	assert.Contains(t, ok.ContentDisposition, ".csv")
	body, err := io.ReadAll(ok.Response.Data)
	require.NoError(t, err)
	rows, err := csv.NewReader(strings.NewReader(string(body))).ReadAll()
	require.NoError(t, err)
	return rows
}

func TestAuditExportIsTheWholeLogAsCSV(t *testing.T) {
	env := testhelper.Setup(t)
	promoteMary(t, env)

	rows := auditCSV(t, auditExport(t, env.SiteActor(t, fixtures.OwnerJohnEmail), fixtures.AcmeSlug))
	require.Len(t, rows, 3, "header, the new entry, the fixture entry")
	header := rows[0]
	assert.Equal(t, []string{
		"id", "occurred_at", "actor_kind", "actor_id", "actor_name", "action",
		"target_type", "target_id", "target_name", "diff", "request_id", "ip", "user_agent",
	}, header)

	col := func(row []string, name string) string {
		for i, h := range header {
			if h == name {
				return row[i]
			}
		}
		t.Fatalf("no column %q", name)
		return ""
	}
	assert.Equal(t, "membership.update", col(rows[1], "action"), "newest first")
	assert.Equal(t, "John", col(rows[1], "actor_name"))
	assert.JSONEq(t, `{"role":{"from":"member","to":"admin"}}`, col(rows[1], "diff"))
	assert.JSONEq(t, `{"role":{"from":"admin","to":"member"}}`, col(rows[2], "diff"))
}

func TestAuditExportSpansPagesAndNeverLeaksAnotherTenant(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()
	s := env.DB.Scoped(fixtures.AcmeID)
	const extra = 250 // more than one internal page
	for i := range extra {
		require.NoError(t, s.AuditEntry().Create().
			SetEntryKey("bulk-"+strconv.Itoa(i)).SetOccurredAt(time.Now()).
			SetActorKind("system").SetAction("tag.create").SetTargetType("tag").
			Exec(ctx))
	}

	rows := auditCSV(t, auditExport(t, env.SiteActor(t, fixtures.OwnerJohnEmail), fixtures.AcmeSlug))
	assert.Len(t, rows, 1+extra+1, "header, every extra entry and the fixture entry; none of Globex's")
}

func TestAuditExportNeutralisesSpreadsheetFormulas(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()
	require.NoError(t, env.DB.Scoped(fixtures.AcmeID).AuditEntry().Create().
		SetEntryKey("formula").SetOccurredAt(time.Now()).
		SetActorKind("user").SetActorName("=HYPERLINK(\"http://evil\")").
		SetAction("tag.create").SetTargetType("tag").SetTargetName("+cmd").
		Exec(ctx))

	rows := auditCSV(t, auditExport(t, env.SiteActor(t, fixtures.OwnerJohnEmail), fixtures.AcmeSlug))
	for _, cell := range rows[1] {
		assert.False(t, strings.ContainsAny(cell[:min(1, len(cell))], "=+-@\t\r"), "cell %q starts a formula", cell)
	}
	assert.Contains(t, rows[1], "'=HYPERLINK(\"http://evil\")")
	assert.Contains(t, rows[1], "'+cmd")
}

func TestAuditExportIsOwnerAndAdminOnlyAndLicensed(t *testing.T) {
	env := testhelper.Setup(t)
	mary := env.SiteActor(t, fixtures.MemberMaryEmail)
	assert.IsType(t, &siteapi.SiteAuditExportForbidden{}, auditExport(t, mary, fixtures.AcmeSlug), "a member is refused")

	oscar := env.SiteActor(t, fixtures.OutsiderOscarEmail)
	assert.IsType(t, &siteapi.SiteAuditExportNotFound{}, auditExport(t, oscar, fixtures.AcmeSlug), "a stranger finds no workspace")

	promoteMary(t, env)
	assert.IsType(t, &siteapi.SiteAuditExportOKHeaders{}, auditExport(t, mary, fixtures.AcmeSlug), "an admin exports")
}

func TestAuditExportWithoutLicenseIsPaymentRequired(t *testing.T) {
	env := testhelper.Setup(t, testhelper.WithoutLicense())
	owner := env.SiteActor(t, fixtures.OwnerJohnEmail)
	assert.IsType(t, &siteapi.SiteAuditExportPaymentRequired{}, auditExport(t, owner, fixtures.AcmeSlug))
}
