//go:build e2e

package e2e

import (
	"fmt"
	"os"
	"testing"
)

// env is the shared application and Mailpit; each test isolates itself with env.NewWorkspace.
var env *Env

func TestMain(m *testing.M) {
	os.Exit(run(m))
}

func run(m *testing.M) int {
	e, stop, err := Boot()
	if err != nil {
		fmt.Fprintln(os.Stderr, "e2e: boot:", err)
		return 1
	}
	defer stop()
	env = e
	return m.Run()
}
