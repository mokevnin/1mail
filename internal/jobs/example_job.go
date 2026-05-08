package jobs

import (
	"context"
	"fmt"

	"github.com/riverqueue/river"
)

type ExampleArgs struct {
	Message string `json:"message"`
}

func (ExampleArgs) Kind() string { return "example" }

type ExampleWorker struct {
	river.WorkerDefaults[ExampleArgs]
}

func (w *ExampleWorker) Work(ctx context.Context, job *river.Job[ExampleArgs]) error {
	fmt.Printf("example job: %s\n", job.Args.Message)
	return nil
}
