package events

import (
	"context"
	"testing"

	"entgo.io/ent/dialect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The tx driver hands ent no-op nested transactions: Bus.WithinTx owns the real
// commit/rollback, so ent's own Begin/Commit/Rollback/Close must be inert.
func TestTxDriverNestedTransactionsAreInert(t *testing.T) {
	d := newTxDriver(nil)
	assert.Equal(t, dialect.Postgres, d.Dialect())
	require.NoError(t, d.Close())

	tx, err := d.(*txDriver).BeginTx(context.Background(), nil)
	require.NoError(t, err)
	require.NoError(t, tx.Commit())
	require.NoError(t, tx.Rollback())
}
