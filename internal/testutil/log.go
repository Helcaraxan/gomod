package testutil

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"go.uber.org/zap/zapcore"

	"github.com/Helcaraxan/gomod/internal/logger"
)

func TestLogger(t *testing.T) *logger.Builder {
	out := &syncBuffer{Builder: strings.Builder{}}
	t.Cleanup(func() {
		if t.Failed() {
			fmt.Fprintf(os.Stderr, out.String())
		}
	})

	fmt.Fprintf(out, "--- Test %s ---", t.Name())
	dl := logger.NewBuilder(out)
	dl.SetDomainLevel("all", zapcore.DebugLevel)
	return dl
}

type syncBuffer struct {
	strings.Builder
}

func (s *syncBuffer) Sync() error { return nil }
