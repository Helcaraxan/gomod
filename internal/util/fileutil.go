package util

import (
	"fmt"
	"os"
	"path/filepath"

	"go.uber.org/zap"

	"github.com/Helcaraxan/gomod/internal/logger"
)

// PrepareOutputPath ensures that the directory containing the specified path exist. It also checks
// that the full path does not refer to an existing file or directory. If such a path already exists
// an error is returned.
func PrepareOutputPath(log *logger.Logger, outputPath string) (*os.File, error) {
	l := log.With(zap.String("output-path", outputPath))
	log.Debug("Preparing output path.")

	// Perform target file sanity checks.
	var sanityCheckErr error
	if _, err := os.Stat(outputPath); err == nil {
		l.Error("The specified output path already exists.")
		sanityCheckErr = fmt.Errorf("target file %q already exists", outputPath)
	} else if !os.IsNotExist(err) {
		l.Error("Failed to check if output path already exists.", zap.Error(err))
		sanityCheckErr = fmt.Errorf("could not stat about %q", outputPath)
	}
	if sanityCheckErr != nil {
		return nil, sanityCheckErr
	}

	l.Debug("Ensuring output path folder exists.")
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		l.Error("Failed to create output directory.", zap.Error(err))
		return nil, fmt.Errorf("could not create %q", filepath.Dir(outputPath))
	}
	out, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		l.Error("Could not create output file.", zap.Error(err))
		return nil, err
	}
	return out, nil
}
