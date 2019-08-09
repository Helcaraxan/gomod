package util

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

func RunCommand(logger *logrus.Logger, path string, cmd string, args ...string) (stdout []byte, stderr []byte, err error) {
	if !filepath.IsAbs(path) {
		if path, err = filepath.Abs(path); err != nil {
			return nil, nil, err
		}
	}

	stdoutBuffer := &bytes.Buffer{}
	stderrBuffer := &bytes.Buffer{}

	execCmd := exec.Command(cmd, args...)
	execCmd.Dir = path
	execCmd.Stdout = stdoutBuffer
	execCmd.Stderr = stderrBuffer

	if logger.GetLevel() >= logrus.DebugLevel {
		execCmd.Stdout = io.MultiWriter(execCmd.Stdout, os.Stdout)
	}
	if logger.GetLevel() >= logrus.WarnLevel {
		execCmd.Stderr = io.MultiWriter(execCmd.Stderr, os.Stderr)
	}

	logger.Debugf("Running command '%s %s'.", execCmd.Path, strings.Join(execCmd.Args, " "))
	err = execCmd.Run()
	logger.Debugf("Content of stdout was: %s", stdoutBuffer.Bytes())
	logger.Debugf("Content of stderr was: %s", stderrBuffer.Bytes())
	if err != nil {
		logger.WithError(err).Errorf("'%s %s' exited with an error", execCmd.Path, strings.Join(execCmd.Args, " "))
		return stdoutBuffer.Bytes(), stderrBuffer.Bytes(), fmt.Errorf("failed to run '%s %s: %s", cmd, strings.Join(args, " "), err)
	}
	return stdoutBuffer.Bytes(), stderrBuffer.Bytes(), nil
}
