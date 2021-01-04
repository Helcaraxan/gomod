package testutil

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/Helcaraxan/gomod/internal/logger"
)

func TestLogger(t *testing.T) *zap.Logger {
	out := &syncBuffer{Builder: strings.Builder{}}
	t.Cleanup(func() {
		if t.Failed() {
			fmt.Fprintf(os.Stderr, out.String())
		}
	})

	fmt.Fprintf(out, "--- Test %s ---", t.Name())
	return zap.New(zapcore.NewCore(logger.NewGoModEncoder(), out, zapcore.DebugLevel))
}

type syncBuffer struct {
	strings.Builder
}

func (s *syncBuffer) Sync() error { return nil }
