package testhelper

import (
	"encoding/json"
	"testing"

	"github.com/riverqueue/river/rivertype"
	"github.com/stretchr/testify/require"
)

// The river queue is a library table, not an ent entity, and tests run jobs inline,
// so a test that needs a job to be waiting in the queue puts one there and reads it
// back through these helpers instead of querying the table itself. Both ride the
// test's transaction.

// EnqueueJob puts one available job of kind with args into the queue.
func (env *TestEnv) EnqueueJob(t *testing.T, kind string, args any) {
	t.Helper()
	body, err := json.Marshal(args)
	require.NoError(t, err)
	_, err = env.SQLDB.ExecContext(t.Context(),
		`INSERT INTO river_job (kind, args, state, max_attempts, queue) VALUES ($1, $2, $3, 3, 'default')`,
		kind, body, string(rivertype.JobStateAvailable))
	require.NoError(t, err)
}

// JobsOf returns the args of the queued jobs of kind, oldest first.
func (env *TestEnv) JobsOf(t *testing.T, kind string) []map[string]any {
	t.Helper()
	rows, err := env.SQLDB.QueryContext(t.Context(), `SELECT args FROM river_job WHERE kind = $1 ORDER BY id`, kind)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	var out []map[string]any
	for rows.Next() {
		var raw []byte
		require.NoError(t, rows.Scan(&raw))
		var args map[string]any
		require.NoError(t, json.Unmarshal(raw, &args))
		out = append(out, args)
	}
	require.NoError(t, rows.Err())
	return out
}
