package site_test

import (
	"context"
	"testing"

	siteapi "github.com/mokevnin/1mail/gen/site"
	"github.com/mokevnin/1mail/internal/testhelper"
	"github.com/stretchr/testify/require"
)

// Site contacts require a valid JWT cookie (generated SecurityHandler). Without
// one the request is rejected; the typed client surfaces the 401 as an error.
func TestSiteContactsRequireAuth(t *testing.T) {
	env := testhelper.Setup(t)
	c := env.SiteAnonymous(t)

	_, err := c.SiteContactsList(context.Background(), siteapi.SiteContactsListParams{})
	require.Error(t, err)
}
