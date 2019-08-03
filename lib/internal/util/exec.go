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

	stdoutWriter := &bufferedTeeWriter{logger: logger}
	stderrWriter := &bufferedTeeWriter{logger: logger}

	execCmd := exec.Command(cmd, args...)
	execCmd.Dir = path
	execCmd.Stdout = stdoutWriter
	execCmd.Stderr = stderrWriter

	if logger.GetLevel() >= logrus.DebugLevel {
		stdoutWriter.printer = os.Stdout
	}
	if logger.GetLevel() >= logrus.WarnLevel {
		stderrWriter.printer = os.Stderr
	}

	logger.Debugf("Running command '%s %s'.", execCmd.Path, strings.Join(execCmd.Args, " "))
	err = execCmd.Run()
	logger.Debugf("Content of stdout was: %s", stdoutWriter.buffer.Bytes())
	logger.Debugf("Content of stderr was: %s", stderrWriter.buffer.Bytes())
	if err != nil {
		logger.WithError(err).Errorf("'%s %s' exited with an error", execCmd.Path, strings.Join(execCmd.Args, " "))
		return stdoutWriter.buffer.Bytes(), stderrWriter.buffer.Bytes(), fmt.Errorf("failed to run '%s %s: %s", cmd, strings.Join(args, " "), err)
	}
	return stdoutWriter.buffer.Bytes(), stderrWriter.buffer.Bytes(), nil
}

type bufferedTeeWriter struct {
	logger  *logrus.Logger
	buffer  bytes.Buffer
	printer io.Writer
}

func (w *bufferedTeeWriter) Write(b []byte) (int, error) {
	n, err := w.buffer.Write(b)
	if err != nil {
		w.logger.WithError(err).Error("Could not write to output buffer.")
		return n, err
	}
	if w.printer != nil {
		if _, err = w.printer.Write(b); err != nil {
			w.logger.WithError(err).Warn("Terminal output pipe broke. Printed output may be incomplete.")
		}
	}
	return n, nil
}
