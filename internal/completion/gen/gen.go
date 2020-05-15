package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/Helcaraxan/gomod/internal/logger"
)

var log = zap.New(zapcore.NewCore(logger.NewGoModEncoder(), os.Stdout, zapcore.DebugLevel))

func main() {
	if len(os.Args) < 4 {
		log.Fatal("Not enough arguments. Use: go run gen.go -- <output-dir> <output-package> <input-files...>")
	}
	outputPackage := os.Args[2]
	outputDir, err := filepath.Abs(os.Args[1])
	if err != nil {
		log.Fatal("Could not resolve absolute path.", zap.String("path", os.Args[1]))
	}

	info, err := os.Stat(outputDir)
	switch {
	case err == nil:
		if !info.IsDir() {
			log.Fatal("Selected output path exists but is not a directory.", zap.String("path", outputDir))
		}
	case os.IsNotExist(err):
		if err = os.MkdirAll(outputDir, 0755); err != nil {
			log.Fatal("Failed to ensure that output directory exists.", zap.String("path", outputDir))
		}
	default:
		log.Fatal("Could not retrieve information about output path.", zap.String("path", outputDir))
	}

	for _, file := range os.Args[3:] {
		if err = processFile(outputDir, outputPackage, file); err != nil {
			log.Fatal("Unable to process file.", zap.String("file", file), zap.Error(err))
		}
	}
}

const constTemplate = `// Code generated. DO NOT EDIT.

package %s

const %s = ` + "`%s`\n"

func processFile(outputDir string, outputPackage string, inputPath string) error {
	if filepath.Ext(inputPath) != ".sh" {
		log.Warn("Skipping file as it does not have a '.sh' extension.", zap.String("file", inputPath))
		return nil
	}

	raw, err := ioutil.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("unable to read file %q passed as argument", inputPath)
	}
	content := strings.TrimSpace(string(raw)) + "\n"

	outputPath := getFilename(outputDir, inputPath)
	err = ioutil.WriteFile(outputPath, []byte(fmt.Sprintf(constTemplate, outputPackage, getVariableName(inputPath), content)), 0644)
	if err != nil {
		return fmt.Errorf("failed to write file generated from %q to %q", inputPath, outputPath)
	}
	return nil
}

func getFilename(outputDir string, inputPath string) string {
	outputPath := strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath)) + ".go"
	return filepath.Join(outputDir, strings.ToLower(outputPath))
}

func getVariableName(inputPath string) string {
	var varname string
	wasUnderscore := true
	for _, r := range strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath)) {
		switch {
		case r == '_':
			wasUnderscore = true
		case wasUnderscore:
			wasUnderscore = false
			varname += string(unicode.ToUpper(r))
		default:
			varname += string(unicode.ToLower(r))
		}
	}
	return varname
}
