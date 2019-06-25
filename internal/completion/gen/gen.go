package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/sirupsen/logrus"
)

func main() {
	if len(os.Args) < 4 {
		logrus.Fatal("Not enough arguments. Use: go run gen.go -- <output-dir> <output-package> <input-files...>")
	}
	outputPackage := os.Args[2]
	outputDir, err := filepath.Abs(os.Args[1])
	if err != nil {
		logrus.Fatalf("Could not resolve output path %q to an absolute path.", os.Args[1])
	}

	info, err := os.Stat(outputDir)
	switch {
	case err == nil:
		if !info.IsDir() {
			logrus.Fatalf("Selected output path %q exists but is not a directory.", outputDir)
		}
	case os.IsNotExist(err):
		if err = os.MkdirAll(outputDir, 0755); err != nil {
			logrus.Fatalf("Failed to ensure that output directory %q exists.", outputDir)
		}
	default:
		logrus.Fatalf("Could not retrieve information about output path %q.", outputDir)
	}

	for _, file := range os.Args[3:] {
		if err = processFile(outputDir, outputPackage, file); err != nil {
			logrus.WithError(err).Fatalf("Unable to process file %q.", file)
		}
	}
}

const constTemplate = `// Code generated. DO NOT EDIT.

package %s

const %s = ` + "`%s`\n"

func processFile(outputDir string, outputPackage string, inputPath string) error {
	if filepath.Ext(inputPath) != ".sh" {
		logrus.Warnf("Skipping %q as it does not have a '.sh' extension.", inputPath)
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
