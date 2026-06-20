package collect_test

import (
	"context"
	"testing"

	"github.com/go-faster/jx"
	"github.com/mokevnin/1mail/ent/event"
	"github.com/mokevnin/1mail/ent/trackingprofile"
	collectapi "github.com/mokevnin/1mail/gen/collect"
	"github.com/mokevnin/1mail/internal/testhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// collectKey is the per-workspace write-key seeded for workspace "acme" (id 1).
const collectKey = "omck_test_acme_collect_key"

// apiKey is the client-side SecuritySource supplying the x-collect-key value.
type apiKey struct{ key string }

func (k apiKey) ApiKeyAuth(context.Context, collectapi.OperationName) (collectapi.ApiKeyAuth, error) {
	return collectapi.ApiKeyAuth{APIKey: k.key}, nil
}

func client(t *testing.T, env *testhelper.TestEnv, key string) *collectapi.Client {
	t.Helper()
	c, err := collectapi.NewClient("http://local/collect", apiKey{key}, collectapi.WithClient(env.Transport(nil)))
	require.NoError(t, err)
	return c
}

func TestCollectKeyAuth(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	res, err := client(t, env, "").CollectEventsCreate(ctx, &collectapi.CollectEventsInput{})
	require.NoError(t, err)
	assert.IsType(t, &collectapi.CollectEventsCreateUnauthorized{}, res)

	res, err = client(t, env, "wrong-key").CollectEventsCreate(ctx, &collectapi.CollectEventsInput{})
	require.NoError(t, err)
	assert.IsType(t, &collectapi.CollectEventsCreateUnauthorized{}, res)
}

func TestCollectIdentifyAndEvents(t *testing.T) {
	env := testhelper.Setup(t)
	c := client(t, env, collectKey)
	ctx := context.Background()

	// Identify with typed traits — exercises map[string]jx.Raw -> map[string]any.
	ident, err := c.CollectIdentifyCreate(ctx, &collectapi.CollectIdentifyInput{
		VisitorId: "v1",
		Email:     collectapi.NewOptNilEmailAddress("trav@example.com"),
		Traits: collectapi.NewOptNilCollectIdentifyInputTraits(collectapi.CollectIdentifyInputTraits{
			"plan":   jx.Raw(`"pro"`),
			"visits": jx.Raw(`3`),
		}),
	})
	require.NoError(t, err)
	assert.IsType(t, &collectapi.CollectOkResponse{}, ident)

	profile, err := env.DB.TrackingProfile.Query().
		Where(trackingprofile.Email("trav@example.com")).Only(ctx)
	require.NoError(t, err)
	assert.Equal(t, "pro", profile.Traits["plan"])
	assert.Equal(t, float64(3), profile.Traits["visits"]) // JSON numbers → float64
	assert.Equal(t, int64(1), profile.WorkspaceID)        // scoped to the key's workspace

	// Events with properties — same conversion path.
	evRes, err := c.CollectEventsCreate(ctx, &collectapi.CollectEventsInput{
		Events: []collectapi.CollectEventInput{{
			VisitorId: "v1",
			Action:    "page_view",
			Properties: collectapi.NewOptNilCollectEventInputProperties(collectapi.CollectEventInputProperties{
				"url": jx.Raw(`"/pricing"`),
				"n":   jx.Raw(`5`),
			}),
		}},
	})
	require.NoError(t, err)
	assert.IsType(t, &collectapi.CollectEventsCreateNoContent{}, evRes)

	evt, err := env.DB.Event.Query().
		Where(event.Email("trav@example.com"), event.Action("page_view")).Only(ctx)
	require.NoError(t, err)
	assert.Equal(t, "/pricing", evt.Properties["url"])
	assert.Equal(t, float64(5), evt.Properties["n"])
	assert.Equal(t, int64(1), evt.WorkspaceID) // scoped to the key's workspace
}
