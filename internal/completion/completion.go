package completion

import (
	"fmt"
	"io"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

//go:generate go run ./gen/gen.go . completion ./scripts/gomod_custom_func.sh

type ShellType uint8

const (
	BASH ShellType = iota
	POWERSHELL
	ZSH
)

var shellToString = map[ShellType]string{
	BASH:       "bash",
	POWERSHELL: "powershell",
	ZSH:        "zsh",
}

func GenerateCompletionScript(logger *logrus.Logger, rootCmd *cobra.Command, shell ShellType, writer io.Writer) error {
	if fileWriter, ok := writer.(*os.File); ok {
		logger.Debugf("Writing shell completion script for %s to '%s'.", shellToString[shell], fileWriter.Name())
	}

	var err error
	switch shell {
	case BASH:
		err = rootCmd.GenBashCompletion(writer)
	case POWERSHELL:
		err = rootCmd.GenPowerShellCompletion(writer)
	case ZSH:
		err = rootCmd.GenZshCompletion(writer)
	}
	if err != nil {
		return fmt.Errorf("Failed to write shell completion scripts: %v", err)
	}
	return nil
}
