package modules

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Helcaraxan/gomod/lib/internal/util"
)

// Module represents the data returned by 'go list -m --json' for a Go module.
type Module struct {
	Main      bool         // is this the main module?
	Indirect  bool         // is it an indirect dependency?
	Path      string       // module path
	Replace   *Module      // replaced by this module
	Version   string       // module version
	Time      *time.Time   // time version was created
	Updates   *Module      // available update, if any (with -u)
	GoMod     string       // the path to this module's go.mod file
	GoVersion string       // the Go version associated with the module
	Error     *ModuleError // error loading module
}

// ModuleError represents the data that is returned whenever Go tooling was unable to load a given
// module's information.
type ModuleError struct {
	Err string // the error itself
}

func RetrieveModuleInformation(logger *logrus.Logger, modulePath string) (*Module, map[string]*Module, error) {
	logger.Debug("Retrieving module information via 'go list'")
	raw, _, err := util.RunCommand(logger, modulePath, "go", "list", "-json", "-m", "all")
	if err != nil {
		return nil, nil, err
	}
	raw = bytes.ReplaceAll(bytes.TrimSpace(raw), []byte("\n}\n"), []byte("\n},\n"))
	raw = append([]byte("[\n"), raw...)
	raw = append(raw, []byte("\n]")...)

	var moduleList []*Module
	if err = json.Unmarshal(raw, &moduleList); err != nil {
		return nil, nil, fmt.Errorf("Unable to retrieve information from 'go list': %v", err)
	}

	var main *Module
	modules := map[string]*Module{}
	for _, module := range moduleList {
		if module.Error != nil {
			logger.Warnf("Unable to retrieve information for module %q: %s", module.Path, module.Error.Err)
			continue
		}

		if module.Main {
			main = module
		}
		modules[module.Path] = module
		if module.Replace != nil {
			modules[module.Replace.Path] = module
		}
	}
	if main == nil || len(main.Path) == 0 {
		logger.Error("Unable to determine the module of the current codebase.")
		return nil, nil, errors.New("could not determine main module")
	}
	return main, modules, nil
}
