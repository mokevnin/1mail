package testhelper

import (
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverdatabasesql"
	"github.com/stretchr/testify/require"
)

// Tests run jobs inline, so a test that needs a job to be waiting in the queue puts
// one there and reads it back through these helpers. They use river's own insert-only
// client over the test's connection (the same txdb connection the ent client rides),
// so nothing here touches the river table directly.

// maxQueuedJobs bounds a JobsOf read; a test queues a handful.
const maxQueuedJobs = 1000

func (env *TestEnv) riverClient(t *testing.T) *river.Client[*sql.Tx] {
	t.Helper()
	rc, err := river.NewClient(riverdatabasesql.New(env.SQLDB), &river.Config{})
	require.NoError(t, err)
	return rc
}

// EnqueueJob puts one available job of args into the queue.
func (env *TestEnv) EnqueueJob(t *testing.T, args river.JobArgs) {
	t.Helper()
	_, err := env.riverClient(t).Insert(t.Context(), args, nil)
	require.NoError(t, err)
}

// JobsOf returns the args of the queued jobs of kind, oldest first.
func (env *TestEnv) JobsOf(t *testing.T, kind string) []map[string]any {
	t.Helper()
	res, err := env.riverClient(t).JobList(t.Context(), river.NewJobListParams().Kinds(kind).OrderBy(river.JobListOrderByID, river.SortOrderAsc).First(maxQueuedJobs))
	require.NoError(t, err)
	var out []map[string]any
	for _, job := range res.Jobs {
		var args map[string]any
		require.NoError(t, json.Unmarshal(job.EncodedArgs, &args))
		out = append(out, args)
	}
	return out
}
