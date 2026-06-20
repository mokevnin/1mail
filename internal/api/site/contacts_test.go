package site_test

import (
	"context"
	"testing"

	siteapi "github.com/mokevnin/1mail/gen/site"
	"github.com/mokevnin/1mail/internal/testhelper"
	"github.com/stretchr/testify/require"
)

// noJWT supplies an empty JWT cookie value.
type noJWT struct{}

func (noJWT) ApiKeyAuth(context.Context, siteapi.OperationName) (siteapi.ApiKeyAuth, error) {
	return siteapi.ApiKeyAuth{}, nil
}

// Site contacts require a valid JWT cookie (generated SecurityHandler). Without
// one the request is rejected; the typed client surfaces the 401 as an error.
func TestSiteContactsRequireAuth(t *testing.T) {
	env := testhelper.Setup(t)
	c, err := siteapi.NewClient("http://local/site", noJWT{}, siteapi.WithClient(env.Transport(nil)))
	require.NoError(t, err)

	_, err = c.SiteContactsList(context.Background(), siteapi.SiteContactsListParams{})
	require.Error(t, err)
}
