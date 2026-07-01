package jobs

import (
	"context"
	"log/slog"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

// errorHandler is river's centralized error/panic sink. Without it, a job that
// exhausts its retries is discarded silently and a worker panic only reaches
// river's default logger — both invisible in production. It logs each with
// enough context (kind, attempt, queue) to alert on, and is the natural seam for
// wiring Sentry later. It never cancels the job (SetCancelled stays false), so
// river keeps following the configured retry schedule.
type errorHandler struct {
	logger *slog.Logger
}

var _ river.ErrorHandler = (*errorHandler)(nil)

func (h *errorHandler) HandleError(_ context.Context, job *rivertype.JobRow, err error) *river.ErrorHandlerResult {
	h.logger.Error("river job errored",
		slog.String("kind", job.Kind),
		slog.String("queue", job.Queue),
		slog.Int64("job_id", job.ID),
		slog.Int("attempt", job.Attempt),
		slog.Int("max_attempts", job.MaxAttempts),
		slog.Any("error", err),
	)
	return &river.ErrorHandlerResult{}
}

func (h *errorHandler) HandlePanic(_ context.Context, job *rivertype.JobRow, panicVal any, trace string) *river.ErrorHandlerResult {
	h.logger.Error("river job panicked",
		slog.String("kind", job.Kind),
		slog.String("queue", job.Queue),
		slog.Int64("job_id", job.ID),
		slog.Int("attempt", job.Attempt),
		slog.Any("panic", panicVal),
		slog.String("trace", trace),
	)
	return &river.ErrorHandlerResult{}
}
