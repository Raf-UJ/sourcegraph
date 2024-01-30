package perforce

import (
	"context"
	"fmt"
	"github.com/sourcegraph/log"
	"github.com/sourcegraph/sourcegraph/internal/wrexec"
	"io"
	"os/exec"
)

type p4Options struct {
	arguments []string

	overrideEnvironment []string // these environment variables will override any existing environment variables with the same name
	environment         []string

	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer

	logger log.Logger
}

type P4OptionFunc func(*p4Options)

func WithArguments(args ...string) P4OptionFunc {
	return func(o *p4Options) {
		o.arguments = append(o.arguments, args...)
	}
}

func WithHost(p4port string) P4OptionFunc {
	return func(o *p4Options) {
		o.overrideEnvironment = append(o.overrideEnvironment,
			fmt.Sprintf("P4PORT=%s", p4port),
		)
	}
}

func WithAuthentication(user, password string) P4OptionFunc {
	return func(o *p4Options) {
		o.overrideEnvironment = append(o.overrideEnvironment,
			fmt.Sprintf("P4USER=%s", user),
			fmt.Sprintf("P4PASSWD=%s", password),
		)
	}
}

func WithEnvironment(env ...string) P4OptionFunc {
	return func(o *p4Options) {
		o.environment = append(o.environment, env...)
	}
}

func WithLogger(logger log.Logger) P4OptionFunc {
	return func(o *p4Options) {
		o.logger = logger
	}
}

func WithAlternateHomeDir(dir string) P4OptionFunc {
	return func(o *p4Options) {
		o.overrideEnvironment = append(o.overrideEnvironment,
			fmt.Sprintf("HOME=%s", dir),
		)
	}
}

func WithClient(client string) P4OptionFunc {
	return func(o *p4Options) {
		o.overrideEnvironment = append(o.overrideEnvironment,
			fmt.Sprintf("P4CLIENT=%s", client),
		)
	}
}

func WithStderr(stderr io.Writer) P4OptionFunc {
	return func(o *p4Options) {
		o.stderr = stderr
	}
}

func WithStdin(stdin io.Reader) P4OptionFunc {
	return func(o *p4Options) {
		o.stdin = stdin
	}
}

func WithStdout(stdout io.Writer) P4OptionFunc {
	return func(o *p4Options) {
		o.stdout = stdout
	}
}

func NewBaseCommand(ctx context.Context, cwd string, options ...P4OptionFunc) wrexec.Cmder {
	opts := p4Options{}

	for _, option := range options {
		option(&opts)
	}

	c := exec.CommandContext(ctx, "p4", opts.arguments...)
	c.Env = append(c.Env,
		fmt.Sprintf("P4CLIENTPATH=%s", cwd),
	)

	c.Env = append(c.Env, opts.environment...)
	c.Env = append(c.Env, opts.overrideEnvironment...)

	if opts.stdin != nil {
		c.Stdin = opts.stdin
	}

	if opts.stdout != nil {
		c.Stdout = opts.stdout
	}

	if opts.stderr != nil {
		c.Stderr = opts.stderr
	}

	c.Dir = cwd

	logger := log.NoOp()
	if opts.logger != nil {
		logger = opts.logger
	}

	return wrexec.Wrap(ctx, logger, c)
}
