package depgraph

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

func runCommand(logger *logrus.Logger, path string, args ...string) (string, error) {
	cmd := exec.Command(path, args...)
	logger.Debugf("Running command '%s %s'.", cmd.Path, strings.Join(cmd.Args, " "))
	raw, err := cmd.CombinedOutput()
	if err != nil {
		logger.WithError(err).Errorf("'%s %s' exited with an error", cmd.Path, strings.Join(cmd.Args, " "))
		logger.Errorf("Command output was: %s", raw)
		return "", fmt.Errorf("'%s %s' error", cmd.Path, strings.Join(cmd.Args, " "))
	}
	return string(raw), nil
}

func prepareOutputPath(logger *logrus.Logger, outputPath string, force bool) error {
	logger.Debugf("Preparing output path %q.", outputPath)
	if force {
		logger.Debug("Attempting to delete any pre-existing folder or file.")
		if err := os.RemoveAll(outputPath); err != nil {
			logger.WithError(err).Errorf("Could not clear existing file at %q.", outputPath)
			return fmt.Errorf("could not remove %q", outputPath)
		}
	}

	if _, err := os.Stat(outputPath); err == nil {
		logger.Errorf("The specified output path %q already exists.", outputPath)
		return fmt.Errorf("target file %q already exists", outputPath)
	} else if !os.IsNotExist(err) {
		logger.WithError(err).Errorf("Failed to check if %q already exists.", outputPath)
		return fmt.Errorf("could not stat about %q", outputPath)
	}

	logger.Debugf("Ensuring %q exists as a folder.", filepath.Dir(outputPath))
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		logger.WithError(err).Errorf("Failed to create output directory %q.", filepath.Dir(outputPath))
		return fmt.Errorf("could not create %q", filepath.Dir(outputPath))
	}
	return nil
}
