package main

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/containers/podman/v5/pkg/systemd/parser"
	"github.com/containers/podman/v5/pkg/systemd/quadlet"
	"github.com/containers/podman/v5/version/rawversion"
)

var (
	verboseFlag bool // True if -v passed
	dryRunFlag  bool // True if -dryrun is used
	versionFlag bool // True if -version is used
)

var (
	void struct{}
	// Key: Extension
	// Value: Processing order for resource naming dependencies
	supportedExtensions = map[string]int{
		".container": 4,
		".volume":    2,
		".kube":      4,
		".network":   2,
		".image":     1,
		".build":     3,
		".pod":       5,
	}
)

func Logf(format string, a ...interface{}) {
	s := fmt.Sprintf(format, a...)
	line := fmt.Sprintf("quadlet-generator[%d]: %s", os.Getpid(), s)

	if dryRunFlag {
		fmt.Fprintf(os.Stderr, "%s\n", line)
		os.Stderr.Sync()
	}
}

var debugEnabled = false

func enableDebug() {
	debugEnabled = true
}

func Debugf(format string, a ...interface{}) {
	if debugEnabled {
		Logf(format, a...)
	}
}

type searchPaths struct {
	sorted []string
	// map to store paths so we can quickly check if we saw them already and not loop in case of symlinks
	visitedDirs map[string]struct{}
}

func newSearchPaths() *searchPaths {
	return &searchPaths{
		sorted:      make([]string, 0),
		visitedDirs: make(map[string]struct{}, 0),
	}
}

func (s *searchPaths) Add(path string) {
	s.sorted = append(s.sorted, path)
	s.visitedDirs[path] = struct{}{}
}

func (s *searchPaths) Visited(path string) bool {
	_, visited := s.visitedDirs[path]
	return visited
}

func getUnitDirs(unitsDir string) []string {
	paths := newSearchPaths()
	appendSubPaths(paths, unitsDir)
	return paths.sorted
}

func appendSubPaths(paths *searchPaths, path string) {
	resolvedPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			Debugf("Error occurred resolving path %q: %s", path, err)
		}
		// Despite the failure add the path to the list for logging purposes
		// This is the equivalent of adding the path when info==nil below
		paths.Add(path)
		return
	}

	if skipPath(paths, resolvedPath) {
		return
	}

	// Add the current directory
	paths.Add(resolvedPath)

	// Read the contents of the directory
	entries, err := os.ReadDir(resolvedPath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			Debugf("Error occurred walking sub directories %q: %s", path, err)
		}
		return
	}

	// Recursively run through the contents of the directory
	for _, entry := range entries {
		fullPath := filepath.Join(resolvedPath, entry.Name())
		appendSubPaths(paths, fullPath)
	}
}

func skipPath(paths *searchPaths, path string) bool {
	// If the path is already in the map no need to read it again
	if paths.Visited(path) {
		return true
	}

	// Don't traverse drop-in directories
	if strings.HasSuffix(path, ".d") {
		return true
	}

	stat, err := os.Stat(path)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			Debugf("Error occurred resolving path %q: %s", path, err)
		}
		return true
	}

	// Not a directory nothing to add
	return !stat.IsDir()
}

func isExtSupported(filename string) bool {
	ext := filepath.Ext(filename)
	_, ok := supportedExtensions[ext]
	return ok
}

var seen = make(map[string]struct{})

func loadUnitsFromDir(sourcePath string) ([]*parser.UnitFile, error) {
	var prevError error
	files, err := os.ReadDir(sourcePath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		return []*parser.UnitFile{}, nil
	}

	var units []*parser.UnitFile

	for _, file := range files {
		name := file.Name()
		if _, ok := seen[name]; !ok && isExtSupported(name) {
			path := path.Join(sourcePath, name)

			Debugf("Loading source unit file %s", path)

			if f, err := parser.ParseUnitFile(path); err != nil {
				err = fmt.Errorf("error loading %q, %w", path, err)
				if prevError == nil {
					prevError = err
				} else {
					prevError = fmt.Errorf("%s\n%s", prevError, err)
				}
			} else {
				seen[name] = void
				units = append(units, f)
			}
		}
	}

	return units, prevError
}

func loadUnitDropins(unit *parser.UnitFile, sourcePaths []string) error {
	var prevError error
	reportError := func(err error) {
		if prevError != nil {
			err = fmt.Errorf("%s\n%s", prevError, err)
		}
		prevError = err
	}

	dropinDirs := []string{}
	unitDropinPaths := unit.GetUnitDropinPaths()

	for _, sourcePath := range sourcePaths {
		for _, dropinPath := range unitDropinPaths {
			dropinDirs = append(dropinDirs, path.Join(sourcePath, dropinPath))
		}
	}

	var dropinPaths = make(map[string]string)
	for _, dropinDir := range dropinDirs {
		dropinFiles, err := os.ReadDir(dropinDir)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				reportError(fmt.Errorf("error reading directory %q, %w", dropinDir, err))
			}

			continue
		}

		for _, dropinFile := range dropinFiles {
			dropinName := dropinFile.Name()
			if filepath.Ext(dropinName) != ".conf" {
				continue // Only *.conf supported
			}

			if _, ok := dropinPaths[dropinName]; ok {
				continue // We already saw this name
			}

			dropinPaths[dropinName] = path.Join(dropinDir, dropinName)
		}
	}

	dropinFiles := make([]string, len(dropinPaths))
	i := 0
	for k := range dropinPaths {
		dropinFiles[i] = k
		i++
	}

	// Merge in alpha-numerical order
	sort.Strings(dropinFiles)

	for _, dropinFile := range dropinFiles {
		dropinPath := dropinPaths[dropinFile]

		Debugf("Loading source drop-in file %s", dropinPath)

		if f, err := parser.ParseUnitFile(dropinPath); err != nil {
			reportError(fmt.Errorf("error loading %q, %w", dropinPath, err))
		} else {
			unit.Merge(f)
		}
	}

	return prevError
}

func isImageID(imageName string) bool {
	// All sha25:... names are assumed by podman to be fully specified
	if strings.HasPrefix(imageName, "sha256:") {
		return true
	}

	// However, podman also accepts image ids as pure hex strings,
	// but only those of length 64 are unambiguous image ids
	if len(imageName) != 64 {
		return false
	}

	for _, c := range imageName {
		if !unicode.Is(unicode.Hex_Digit, c) {
			return false
		}
	}

	return true
}

func isUnambiguousName(imageName string) bool {
	// Fully specified image ids are unambiguous
	if isImageID(imageName) {
		return true
	}

	// Otherwise we require a fully qualified name
	firstSlash := strings.Index(imageName, "/")
	if firstSlash == -1 {
		// No domain or path, not fully qualified
		return false
	}

	// What is before the first slash can be a domain or a path
	domain := imageName[:firstSlash]

	// If its a domain (has dot or port or is "localhost") it is considered fq
	if strings.ContainsAny(domain, ".:") || domain == "localhost" {
		return true
	}

	return false
}

// warns if input is an ambiguous name, i.e. a partial image id or a short
// name (i.e. is missing a registry)
//
// Examples:
//   - short names: "image:tag", "library/fedora"
//   - fully qualified names: "quay.io/image", "localhost/image:tag",
//     "server.org:5000/lib/image", "sha256:..."
//
// We implement a simple version of this from scratch here to avoid
// a huge dependency in the generator just for a warning.
func warnIfAmbiguousName(unit *parser.UnitFile, group string) {
	imageName, ok := unit.Lookup(group, quadlet.KeyImage)
	if !ok {
		return
	}
	if strings.HasSuffix(imageName, ".build") || strings.HasSuffix(imageName, ".image") {
		return
	}
	if !isUnambiguousName(imageName) {
		Logf("Warning: %s specifies the image \"%s\" which not a fully qualified image name. This is not ideal for performance and security reasons. See the podman-pull manpage discussion of short-name-aliases.conf for details.", unit.Filename, imageName)
	}
}

func generateUnitsInfoMap(units []*parser.UnitFile) map[string]*quadlet.UnitInfo {
	unitsInfoMap := make(map[string]*quadlet.UnitInfo)
	for _, unit := range units {
		var serviceName string
		var containers []string
		var resourceName string

		switch {
		case strings.HasSuffix(unit.Filename, ".container"):
			serviceName = quadlet.GetContainerServiceName(unit)
		case strings.HasSuffix(unit.Filename, ".volume"):
			serviceName = quadlet.GetVolumeServiceName(unit)
		case strings.HasSuffix(unit.Filename, ".kube"):
			serviceName = quadlet.GetKubeServiceName(unit)
		case strings.HasSuffix(unit.Filename, ".network"):
			serviceName = quadlet.GetNetworkServiceName(unit)
		case strings.HasSuffix(unit.Filename, ".image"):
			serviceName = quadlet.GetImageServiceName(unit)
		case strings.HasSuffix(unit.Filename, ".build"):
			serviceName = quadlet.GetBuildServiceName(unit)
			// Prefill resouceNames for .build files. This is significantly less complex than
			// pre-computing all resourceNames for all Quadlet types (which is rather complex for a few
			// types), but still breaks the dependency cycle between .volume and .build ([Volume] can
			// have Image=some.build, and [Build] can have Volume=some.volume:/some-volume)
			resourceName = quadlet.GetBuiltImageName(unit)
		case strings.HasSuffix(unit.Filename, ".pod"):
			serviceName = quadlet.GetPodServiceName(unit)
			containers = make([]string, 0)
		default:
			Logf("Unsupported file type %q", unit.Filename)
			continue
		}

		unitsInfoMap[unit.Filename] = &quadlet.UnitInfo{
			ServiceName:       serviceName,
			ContainersToStart: containers,
			ResourceName:      resourceName,
		}
	}

	return unitsInfoMap
}

func main() {
	if err := process(); err != nil {
		Logf("%s", err.Error())
		os.Exit(1)
	}
	os.Exit(0)
}

func process() error {
	var prevError error

	flag.Parse()

	if versionFlag {
		fmt.Printf("%s\n", rawversion.RawVersion)
		return prevError
	}

	if verboseFlag || dryRunFlag {
		enableDebug()
	}

	reportError := func(err error) {
		if prevError != nil {
			err = fmt.Errorf("%s\n%s", prevError, err)
		}
		prevError = err
	}

	unitsDir := flag.Arg(0)
	sourcePathsMap := getUnitDirs(unitsDir)

	var units []*parser.UnitFile
	for _, d := range sourcePathsMap {
		if result, err := loadUnitsFromDir(d); err != nil {
			reportError(err)
		} else {
			units = append(units, result...)
		}
	}

	if len(units) == 0 {
		Debugf("No files parsed from %s", sourcePathsMap)
		return prevError
	}

	for _, unit := range units {
		if err := loadUnitDropins(unit, sourcePathsMap); err != nil {
			reportError(err)
		}
	}

	// Sort unit files according to potential inter-dependencies, with Volume and Network units
	// taking precedence over all others.
	sort.Slice(units, func(i, j int) bool {
		getOrder := func(i int) int {
			ext := filepath.Ext(units[i].Filename)
			order, ok := supportedExtensions[ext]
			if !ok {
				return 0
			}
			return order
		}
		return getOrder(i) < getOrder(j)
	})

	// Generate the PodsInfoMap to allow containers to link to their pods and add themselves to the pod's containers list
	unitsInfoMap := generateUnitsInfoMap(units)

	for _, unit := range units {
		var service *parser.UnitFile
		var err error

		switch {
		case strings.HasSuffix(unit.Filename, ".container"):
			warnIfAmbiguousName(unit, quadlet.ContainerGroup)
			service, err = quadlet.ConvertContainer(unit, false, unitsInfoMap)
		case strings.HasSuffix(unit.Filename, ".volume"):
			warnIfAmbiguousName(unit, quadlet.VolumeGroup)
			service, err = quadlet.ConvertVolume(unit, unit.Filename, unitsInfoMap, false)
		case strings.HasSuffix(unit.Filename, ".kube"):
			service, err = quadlet.ConvertKube(unit, unitsInfoMap, false)
		case strings.HasSuffix(unit.Filename, ".network"):
			service, err = quadlet.ConvertNetwork(unit, unit.Filename, unitsInfoMap, false)
		case strings.HasSuffix(unit.Filename, ".image"):
			warnIfAmbiguousName(unit, quadlet.ImageGroup)
			service, err = quadlet.ConvertImage(unit, unitsInfoMap, false)
		case strings.HasSuffix(unit.Filename, ".build"):
			service, err = quadlet.ConvertBuild(unit, unitsInfoMap, false)
		case strings.HasSuffix(unit.Filename, ".pod"):
			service, err = quadlet.ConvertPod(unit, unit.Filename, unitsInfoMap, false)
		default:
			Logf("Unsupported file type %q", unit.Filename)
			continue
		}

		if err != nil {
			reportError(fmt.Errorf("converting %q: %w", unit.Filename, err))
			continue
		}

		if dryRunFlag {
			data, err := service.ToString()
			if err != nil {
				reportError(fmt.Errorf("parsing %s: %w", service.Path, err))
				continue
			}
			fmt.Printf("---%s---\n%s\n", service.Path, data)
			continue
		}
	}
	return prevError
}

func init() {
	flag.BoolVar(&verboseFlag, "v", false, "Print debug information")
	flag.BoolVar(&dryRunFlag, "dryrun", false, "Run in dryrun mode printing debug information")
	flag.BoolVar(&versionFlag, "version", false, "Print version information and exit")
}
