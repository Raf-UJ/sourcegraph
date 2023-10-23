package ci

import (
	bk "github.com/sourcegraph/sourcegraph/dev/ci/internal/buildkite"
	"github.com/sourcegraph/sourcegraph/dev/ci/internal/ci/operations"
)

func finalizeReleasePatch(_ Config) operations.Operation {
	cmds := []bk.StepOpt{}
	cmds = append(cmds, bk.Agent("queue", "bazel"))

	cmds = append(cmds,
		bazelAnnouncef("bazel run //tools/release:finalize_release_patch"),
		// bk.Cmd(bazelCmd("run //tools/release:finalize_release_patch")),
		bk.Cmd(`echo "pretending to finalize"`),
	)

	return func(pipeline *bk.Pipeline) {
		pipeline.AddStep(":bazel: Finalize patch release",
			cmds...,
		)
	}
}