package modules

// PackageInfo represents the data returned by 'go list --json' for a Go package. It's content is
// extracted directly from the Go documentation.
type PackageInfo struct {
	Dir           string      // directory containing package sources
	ImportPath    string      // import path of package in dir
	ImportComment string      // path in import comment on package statement
	Name          string      // package name
	Doc           string      // package documentation string
	Target        string      // install path
	Shlib         string      // the shared library that contains this package (only set when -linkshared)
	Goroot        bool        // is this package in the Go root?
	Standard      bool        // is this package part of the standard Go library?
	Stale         bool        // would 'go install' do anything for this package?
	StaleReason   string      // explanation for Stale==true
	Root          string      // Go root or Go path dir containing this package
	ConflictDir   string      // this directory shadows Dir in $GOPATH
	BinaryOnly    bool        // binary-only package (no longer supported)
	ForTest       string      // package is only for use in named test
	Export        string      // file containing export data (when using -export)
	Module        *ModuleInfo // info about package's containing module, if any (can be nil)
	Match         []string    // command-line patterns matching this package
	DepOnly       bool        // package is only a dependency, not explicitly listed

	// Source files
	GoFiles         []string // .go source files (excluding CgoFiles, TestGoFiles, XTestGoFiles)
	CgoFiles        []string // .go source files that import "C"
	CompiledGoFiles []string // .go files presented to compiler (when using -compiled)
	IgnoredGoFiles  []string // .go source files ignored due to build constraints
	CFiles          []string // .c source files
	CXXFiles        []string // .cc, .cxx and .cpp source files
	MFiles          []string // .m source files
	HFiles          []string // .h, .hh, .hpp and .hxx source files
	FFiles          []string // .f, .F, .for and .f90 Fortran source files
	SFiles          []string // .s source files
	SwigFiles       []string // .swig files
	SwigCXXFiles    []string // .swigcxx files
	SysoFiles       []string // .syso object files to add to archive
	TestGoFiles     []string // _test.go files in package
	XTestGoFiles    []string // _test.go files outside package

	// Cgo directives
	CgoCFLAGS    []string // cgo: flags for C compiler
	CgoCPPFLAGS  []string // cgo: flags for C preprocessor
	CgoCXXFLAGS  []string // cgo: flags for C++ compiler
	CgoFFLAGS    []string // cgo: flags for Fortran compiler
	CgoLDFLAGS   []string // cgo: flags for linker
	CgoPkgConfig []string // cgo: pkg-config names

	// Dependency information
	Imports      []string          // import paths used by this package
	ImportMap    map[string]string // map from source import to ImportPath (identity entries omitted)
	Deps         []string          // all (recursively) imported dependencies
	TestImports  []string          // imports from TestGoFiles
	XTestImports []string          // imports from XTestGoFiles

	// Error information
	Incomplete bool            // this package or a dependency has an error
	Error      *PackageError   // error loading package
	DepsErrors []*PackageError // errors loading dependencies
}

type PackageError struct {
	ImportStack []string // shortest path from package named on command line to this one
	Pos         string   // position of error (if present, file:line:col)
	Err         string   // the error itself
}
