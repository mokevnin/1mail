package jobs_test

import (
	"context"
	"testing"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/automationrun"
	"github.com/mokevnin/1mail/ent/contact"
	"github.com/mokevnin/1mail/ent/tag"
	"github.com/mokevnin/1mail/internal/fixtures"
	"github.com/mokevnin/1mail/internal/testhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Fixture automation 104 applies "engaged" then removes "vip"; run 1100 enrolls
// contact 1 (who has vip and newsletter). Tag steps send nothing.
func TestAutomationTagSteps(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()
	fs := &fakeSender{}

	drive(t, env, fakeResolver{sender: fs}, 1100)

	assert.Empty(t, fs.sent)
	names, err := env.DB.Tag.Query().
		Where(tag.HasContactsWith(contact.ID(fixtures.ContactAliceID))).
		Order(ent.Asc(tag.FieldName)).
		Select(tag.FieldName).
		Strings(ctx)
	require.NoError(t, err)
	assert.Equal(t, []string{"engaged", "newsletter"}, names)
	assert.Equal(t, automationrun.StatusCompleted, env.DB.AutomationRun.GetX(ctx, 1100).Status)
}
