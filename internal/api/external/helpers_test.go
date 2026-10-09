package external_test

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

// mustParseID parses an API EntityId back into the primary key it renders.
func mustParseID(t *testing.T, id string) int64 {
	t.Helper()
	n, err := strconv.ParseInt(id, 10, 64)
	require.NoError(t, err)
	return n
}
