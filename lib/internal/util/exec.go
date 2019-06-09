package util

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/sirupsen/logrus"
)

func RunCommand(logger *logrus.Logger, quiet bool, path string, args ...string) ([]byte, error) {
	cmd := exec.Command(path, args...)

	if !quiet {
		errStream, err := cmd.StderrPipe()
		if err != nil {
			return nil, fmt.Errorf("unable to retrieve the stderr pipe for '%s %s", cmd.Path, strings.Join(cmd.Args, " "))
		}
		go func() {
			_, copyErr := io.Copy(os.Stdout, errStream)
			if copyErr != nil {
				fmt.Fprintf(os.Stderr, "Subprocess output pipe broke down: %v\n", copyErr)
			}
		}()
	}

	logger.Debugf("Running command '%s %s'.", cmd.Path, strings.Join(cmd.Args, " "))
	raw, err := cmd.Output()
	if err != nil {
		logger.WithError(err).Errorf("'%s %s' exited with an error", cmd.Path, strings.Join(cmd.Args, " "))
		logger.Errorf("Command output was: %s", raw)
		return nil, fmt.Errorf("'%s %s' error", cmd.Path, strings.Join(cmd.Args, " "))
	}
	return raw, nil
}
