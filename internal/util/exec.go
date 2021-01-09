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

	"github.com/Helcaraxan/gomod/internal/logger"
)

func RunCommand(log *logger.Logger, path string, cmd string, args ...string) (stdout []byte, stderr []byte, err error) {
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
		execCmd.Stdout = io.MultiWriter(execCmd.Stdout, os.Stderr)
		execCmd.Stderr = io.MultiWriter(execCmd.Stderr, os.Stderr)
	}

	log.Debug("Running command.", zap.Strings("args", append([]string{execCmd.Path}, execCmd.Args...)))
	err = execCmd.Run()
	log.Debug(
		"Finished running.",
		zap.Strings("args", append([]string{execCmd.Path}, execCmd.Args...)),
		zap.ByteString("stdout", stdoutBuffer.Bytes()),
		zap.ByteString("stderr", stderrBuffer.Bytes()),
	)
	if err != nil {
		log.Error("Command exited with an error.", zap.Strings("args", append([]string{execCmd.Path}, execCmd.Args...)), zap.Error(err))
		return stdoutBuffer.Bytes(), stderrBuffer.Bytes(), fmt.Errorf("failed to run '%s %s: %s", cmd, strings.Join(args, " "), err)
	}
	return stdoutBuffer.Bytes(), stderrBuffer.Bytes(), nil
}
