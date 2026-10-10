package testhelper

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// AgeOutbox makes every outbox row in this test's transaction look d older.
func (env *TestEnv) AgeOutbox(t *testing.T, d time.Duration) {
	t.Helper()
	_, err := env.SQLDB.ExecContext(t.Context(),
		`UPDATE watermill_domain_events SET created_at = created_at - make_interval(secs => $1)`, d.Seconds())
	require.NoError(t, err)
}

// AckOutbox moves a consumer group's cursor past every outbox row currently
// visible in this test's transaction, as if the group had processed them all.
func (env *TestEnv) AckOutbox(t *testing.T, group string) {
	t.Helper()
	_, err := env.SQLDB.ExecContext(t.Context(), `
INSERT INTO watermill_offsets_domain_events (consumer_group, offset_acked, last_processed_transaction_id)
SELECT $1, "offset", transaction_id FROM watermill_domain_events ORDER BY transaction_id DESC, "offset" DESC LIMIT 1
ON CONFLICT (consumer_group) DO UPDATE SET
  offset_acked = EXCLUDED.offset_acked, last_processed_transaction_id = EXCLUDED.last_processed_transaction_id`, group)
	require.NoError(t, err)
}
