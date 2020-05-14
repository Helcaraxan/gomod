package util

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
)

func RunCommand(log *zap.Logger, path string, cmd string, args ...string) (stdout []byte, stderr []byte, err error) {
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

	if log.Core().Enabled(zap.DebugLevel) {
		execCmd.Stdout = io.MultiWriter(execCmd.Stdout, os.Stdout)
	}
	if log.Core().Enabled(zap.WarnLevel) {
		execCmd.Stderr = io.MultiWriter(execCmd.Stderr, os.Stderr)
	}

	log = log.With(zap.Strings("args", append([]string{execCmd.Path}, execCmd.Args...)))
	log.Debug("Running command.")
	err = execCmd.Run()
	log.Debug("Finished running.", zap.ByteString("stdout", stdoutBuffer.Bytes()), zap.ByteString("stderr", stderrBuffer.Bytes()))
	if err != nil {
		log.Error("Command exited with an error.", zap.Error(err))
		return stdoutBuffer.Bytes(), stderrBuffer.Bytes(), fmt.Errorf("failed to run '%s %s: %s", cmd, strings.Join(args, " "), err)
	}
	return stdoutBuffer.Bytes(), stderrBuffer.Bytes(), nil
}
