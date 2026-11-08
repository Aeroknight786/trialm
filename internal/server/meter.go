package server

import (
	"context"
	"fmt"
	"io"
	"time"
)

// Meter shows how many requests the local route has served, on one line:
// redrawn in place on a terminal, printed only when it changes elsewhere.
func Meter(ctx context.Context, output io.Writer, s *Server, inPlace bool) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	last := int64(-1)
	for {
		if n := s.bridgeN.Load(); n != last {
			last = n
			line := fmt.Sprintf("Requests %d · ctrl-c stops · the engine stays warm 15 min", n)
			if inPlace {
				fmt.Fprint(output, "\r\x1b[2K"+line)
			} else {
				fmt.Fprintln(output, line)
			}
		}
		select {
		case <-ctx.Done():
			if inPlace {
				fmt.Fprintln(output)
			}
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
