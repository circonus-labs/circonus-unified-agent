package shim

import (
	"bufio"
	"context"
	"errors"
	"io"
	"log"
	"testing"
	"time"

	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/stretchr/testify/require"
)

func TestShimSetsUpLogger(t *testing.T) {
	stderrReader, stderrWriter := io.Pipe()
	stdinReader, stdinWriter := io.Pipe()

	runErroringInputPlugin(t, 40*time.Second, stdinReader, nil, stderrWriter)

	_, _ = stdinWriter.Write([]byte("\n"))

	// <-metricProcessed

	r := bufio.NewReader(stderrReader)
	out, err := r.ReadString('\n')
	require.NoError(t, err)
	require.Contains(t, out, "Error in plugin: intentional")

	stdinWriter.Close()
}

func runErroringInputPlugin(t *testing.T, interval time.Duration, stdin io.Reader, stdout, stderr io.Writer) (metricProcessed chan bool, exited chan bool) {
	metricProcessed = make(chan bool, 1)
	exited = make(chan bool, 1)
	inp := &erroringInput{}

	shim := New()
	if stdin != nil {
		shim.stdin = stdin
	}
	if stdout != nil {
		shim.stdout = stdout
	}
	if stderr != nil {
		shim.stderr = stderr
		log.SetOutput(stderr)
	}
	_ = shim.AddInput(inp)
	go func() {
		err := shim.Run(context.Background(), interval)
		require.NoError(t, err)
		exited <- true
	}()
	return metricProcessed, exited
}

type erroringInput struct {
}

func (i *erroringInput) SampleConfig() string {
	return ""
}

func (i *erroringInput) Description() string {
	return ""
}

func (i *erroringInput) Gather(ctx context.Context, acc cua.Accumulator) error {
	acc.AddError(errors.New("intentional"))
	return nil
}

func (i *erroringInput) Start(acc cua.Accumulator) error {
	return nil
}

func (i *erroringInput) Stop() {
}
