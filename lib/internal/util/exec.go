package util

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/sirupsen/logrus"
)

func RunCommand(logger *logrus.Logger, cmd string, args ...string) (stdout []byte, stderr []byte, err error) {
	var stdoutBuf, stderrBuf bytes.Buffer
	execCmd := exec.Command(cmd, args...)

	if logger.GetLevel() < logrus.InfoLevel {
		execCmd.Stdout = &stdoutBuf
	} else {
		cmdStdout, pipeErr := execCmd.StdoutPipe()
		if pipeErr != nil {
			logger.WithError(pipeErr).Errorf("Failed to set up stdout pipe for '%s %s'", cmd, strings.Join(execCmd.Args, " "))
			return nil, nil, err
		}
		liveStdout := io.TeeReader(cmdStdout, &stdoutBuf)
		go func() {
			_, copyErr := io.Copy(os.Stdout, liveStdout)
			if copyErr != nil {
				logger.WithError(copyErr).Error("Sub-process stdout pipe broke.")
			}
		}()
	}
	if logger.GetLevel() < logrus.WarnLevel {
		execCmd.Stderr = &stderrBuf
	} else {
		cmdStderr, pipeErr := execCmd.StderrPipe()
		if pipeErr != nil {
			logger.WithError(pipeErr).Errorf("Failed to set up stderr pipe for '%s %s'", cmd, strings.Join(execCmd.Args, " "))
			return nil, nil, pipeErr
		}
		liveStderr := io.TeeReader(cmdStderr, &stderrBuf)
		go func() {
			_, copyErr := io.Copy(os.Stdout, liveStderr)
			if copyErr != nil {
				logger.WithError(copyErr).Error("Sub-process stderr pipe broke.")
			}
		}()
	}

	logger.Debugf("Running command '%s %s'.", execCmd.Path, strings.Join(execCmd.Args, " "))
	err = execCmd.Run()
	logger.Debugf("Content of stdout was: %s", stdoutBuf.Bytes())
	logger.Debugf("Content of stderr was: %s", stderrBuf.Bytes())
	if err != nil {
		logger.WithError(err).Errorf("'%s %s' exited with an error", execCmd.Path, strings.Join(execCmd.Args, " "))
		return stdoutBuf.Bytes(), stderrBuf.Bytes(), fmt.Errorf("failed to run '%s %s: %s", cmd, strings.Join(args, " "), err)
	}
	return stdoutBuf.Bytes(), stderrBuf.Bytes(), nil
}
