package run

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/rjeczalik/notify"

	"github.com/sourcegraph/sourcegraph/dev/sg/internal/secrets"
)

type ConfigCommand interface {
	// Getters for common fields
	GetName() string
	GetContinueWatchOnExit() bool
	GetIgnoreStdout() bool
	GetIgnoreStderr() bool
	GetPreamble() string
	GetEnv() map[string]string
	GetBinaryLocation() (string, error)
	GetExternalSecrets() map[string]secrets.ExternalSecret
	GetExec(context.Context) (*exec.Cmd, error)

	// Start a file watcher on the relevant filesystem sub-tree for this command
	StartWatch(context.Context) (<-chan struct{}, error)
}

func WatchPaths(ctx context.Context, paths []string) (<-chan struct{}, error) {
	// Set up the watchers.
	restart := make(chan struct{})
	events := make(chan notify.EventInfo, 1)

	// Do nothing if no watch paths are configured
	if len(paths) == 0 {
		return restart, nil
	}

	for _, path := range paths {
		if err := notify.Watch(path, events, notify.All); err != nil {
			return nil, err
		}
	}

	// Start watching for changes to the source tree
	go func() {
		defer close(events)
		defer notify.Stop(events)

		for {
			select {
			case <-ctx.Done():
				return
			case <-events:
				restart <- struct{}{}
			}

		}
	}()

	return restart, nil
}

type noBinaryError struct {
	name string
	err  error
}

func (e noBinaryError) Error() string {
	return fmt.Sprintf("no-binary-error: %s has no binary", e.name)
}

func (e noBinaryError) Unwrap() error {
	return e.err
}

func (e noBinaryError) Wrap(err error) error {
	e.err = err
	return e
}
