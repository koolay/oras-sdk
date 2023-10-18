package artifacts

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/slog"

	"github.com/koolay/oras-sdk/option"
)

func TestRunPull(t *testing.T) {
	defaultLog := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: true,
	}))
	slog.SetDefault(defaultLog)
	ctx := context.Background()
	username := os.Getenv("YOUR_AK")
	pssword := os.Getenv("YOUR_SK")

	opts := PullOptions{}
	opts.Output = "/tmp/oras-temp"
	opts.concurrency = 4
	opts.RawReference = "docker.io/library/busybox:latest"
	opts.Type = option.TargetTypeRemote
	opts.Debug = true
	opts.Verbose = true
	opts.Remote = option.NewRemote(false, username, pssword)
	err := RunPull(ctx, opts)
	assert.Nil(t, err)
}
