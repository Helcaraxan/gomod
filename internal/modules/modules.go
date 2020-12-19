package modules

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/Helcaraxan/gomod/internal/util"
)

// ModuleInfo represents the data returned by 'go list -m --json' for a Go module. It's content is
// extracted directly from the Go documentation.
type ModuleInfo struct {
	Main      bool         // is this the main module?
	Indirect  bool         // is it an indirect dependency?
	Path      string       // module path
	Replace   *ModuleInfo  // replaced by this module
	Version   string       // module version
	Time      *time.Time   // time version was created
	Dir       string       // location of the module's source
	Update    *ModuleInfo  // available update, if any (with -u)
	GoMod     string       // the path to this module's go.mod file
	GoVersion string       // the Go version associated with the module
	Error     *ModuleError // error loading module
}

// ModuleError represents the data that is returned whenever Go tooling was unable to load a given
// module's information.
type ModuleError struct {
	Err string // the error itself
}

// Retrieve the Module information for all dependencies of the Go module found at the specified path.
func GetDependencies(log *zap.Logger, moduleDir string) (*ModuleInfo, map[string]*ModuleInfo, error) {
	return retrieveModuleInformation(log, moduleDir, "all")
}

// Retrieve the Module information for all dependencies of the Go module found at the specified
// path, including any potentially available updates. This requires internet connectivity in order
// to return the results. Lack of connectivity should result in an error being returned but this is
// not a hard guarantee.
func GetDependenciesWithUpdates(log *zap.Logger, moduleDir string) (*ModuleInfo, map[string]*ModuleInfo, error) {
	if err := connectivityCheck(); err != nil {
		log.Error("No connectivity.", zap.Error(err))
		return nil, nil, err
	}
	return retrieveModuleInformation(log, moduleDir, "all", "-versions", "-u")
}

// Retrieve the Module information for the specified target module which must be a dependency of the
// Go module found at the specified path.
func GetModule(log *zap.Logger, moduleDir string, targetModule string) (*ModuleInfo, error) {
	module, _, err := retrieveModuleInformation(log, moduleDir, targetModule)
	return module, err
}

// Retrieve the Module information for the specified target module which must be a dependency of the
// Go module found at the specified path, including any potentially available updates. This requires
// internet connectivity in order to return the results. Lack of connectivity should result in an
// error being returned but this is not a hard guarantee.
func GetModuleWithUpdate(log *zap.Logger, moduleDir string, targetModule string) (*ModuleInfo, error) {
	if err := connectivityCheck(); err != nil {
		log.Error("No connectivity.", zap.Error(err))
		return nil, err
	}
	module, _, err := retrieveModuleInformation(log, moduleDir, targetModule, "-versions", "-u")
	return module, err
}

func retrieveModuleInformation(
	log *zap.Logger,
	moduleDir string,
	targetModule string,
	extraGoListArgs ...string,
) (*ModuleInfo, map[string]*ModuleInfo, error) {
	log.Debug("Ensuring module information is available locally by running 'go mod download'.")
	_, _, err := util.RunCommand(log, moduleDir, "go", "mod", "download")
	if err != nil {
		log.Error("Failed to run 'go mod download'.", zap.Error(err))
		return nil, nil, err
	}

	log.Debug("Retrieving module information via 'go list'")
	goListArgs := append([]string{"list", "-json", "-m"}, extraGoListArgs...)
	if targetModule == "" {
		targetModule = "all"
	}
	goListArgs = append(goListArgs, targetModule)

	raw, _, err := util.RunCommand(log, moduleDir, "go", goListArgs...)
	if err != nil {
		log.Error("Failed to list modules in dependency graph via 'go list'.", zap.Error(err))
		return nil, nil, err
	}
	raw = bytes.ReplaceAll(bytes.TrimSpace(raw), []byte("\n}\n"), []byte("\n},\n"))
	raw = append([]byte("[\n"), raw...)
	raw = append(raw, []byte("\n]")...)

	var moduleList []*ModuleInfo
	if err = json.Unmarshal(raw, &moduleList); err != nil {
		return nil, nil, fmt.Errorf("Unable to retrieve information from 'go list': %v", err)
	}

	var main *ModuleInfo
	modules := map[string]*ModuleInfo{}
	for _, module := range moduleList {
		if module.Error != nil {
			log.Warn("Unable to retrieve information for module", zap.String("module", module.Path), zap.String("error", module.Error.Err))
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
	if len(modules) == 0 {
		return nil, nil, errors.New("unable to load any module information")
	}
	return main, modules, nil
}

var httpClient = &http.Client{Timeout: 5 * time.Second}

func connectivityCheck() error {
	resp, err := httpClient.Get("https://proxy.golang.org/")
	if err != nil {
		return fmt.Errorf("failed to ping https://proxy.golang.org/: %s", err.Error())
	} else if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ping to https://proxy.golang.org/ returned a non-200 status code")
	}
	return nil
}
