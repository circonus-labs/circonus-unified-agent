//go:build windows
// +build windows

package execd

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/circonus-labs/circonus-unified-agent/cua"
)

func (e *Execd) Gather(ctx context.Context, acc cua.Accumulator) error {
	if e.process == nil {
		return nil
	}

	switch e.Signal {
	case "STDIN":
		if osStdin, ok := e.process.Stdin.(*os.File); ok {
			_ = osStdin.SetWriteDeadline(time.Now().Add(1 * time.Second))
		}
		if _, err := io.WriteString(e.process.Stdin, "\n"); err != nil {
			return fmt.Errorf("Error writing to stdin: %w", err)
		}
	case "none":
	default:
		return fmt.Errorf("invalid signal: %s", e.Signal)
	}

	return nil
}
