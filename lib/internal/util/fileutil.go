package util

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

// PrepareOutputPath ensures that the directory containing the specified path exist. It also checks
// that the full path does not refer to an existing file or directory. If such a path already exists
// an error is returned unless the `force` parameter is set to `true` in which case we delete it.
func PrepareOutputPath(logger *logrus.Logger, outputPath string, force bool) (*os.File, error) {
	logger.Debugf("Preparing output path %q.", outputPath)

	// Perform target file sanity checks.
	var sanityCheckErr error
	if force {
		logger.Debug("Attempting to delete any pre-existing folder or file.")
		if err := os.RemoveAll(outputPath); err != nil {
			logger.WithError(err).Errorf("Could not clear existing file at %q.", outputPath)
			sanityCheckErr = fmt.Errorf("could not remove %q", outputPath)
		}
	}
	if _, err := os.Stat(outputPath); err == nil {
		logger.Errorf("The specified output path %q already exists.", outputPath)
		sanityCheckErr = fmt.Errorf("target file %q already exists", outputPath)
	} else if !os.IsNotExist(err) {
		logger.WithError(err).Errorf("Failed to check if %q already exists.", outputPath)
		sanityCheckErr = fmt.Errorf("could not stat about %q", outputPath)
	}
	if sanityCheckErr != nil {
		return nil, sanityCheckErr
	}

	logger.Debugf("Ensuring %q exists as a folder.", filepath.Dir(outputPath))
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		logger.WithError(err).Errorf("Failed to create output directory %q.", filepath.Dir(outputPath))
		return nil, fmt.Errorf("could not create %q", filepath.Dir(outputPath))
	}
	out, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logger.WithError(err).Errorf("Could not create output file %q.", outputPath)
		return nil, err
	}
	return out, nil
}
