package libbuildpack

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Masterminds/semver"
)

const dateFormat = "2006-01-02"
const thirtyDays = time.Hour * 24 * 30

type Manifest interface {
	DefaultVersion(depName string) (Dependency, error)
	FetchDependency(dep Dependency, outputFile string) error
	InstallDependency(dep Dependency, outputDir string) error
	Version() (string, error)
	Language() string
	CheckStackSupport() error
	RootDir() string
	CheckBuildpackVersion(cacheDir string)
	StoreBuildpackMetadata(cacheDir string)
	AllDependencyVersions(string) []string
	InstallOnlyVersion(depName string, installDir string) error
}

type Dependency struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
}

type DeprecationDate struct {
	Name        string `yaml:"name"`
	VersionLine string `yaml:"version_line"`
	Date        string `yaml:"date"`
	Link        string `yaml:"link"`
}

type ManifestEntry struct {
	Dependency Dependency `yaml:",inline"`
	URI        string     `yaml:"uri"`
	MD5        string     `yaml:"md5"`
	CFStacks   []string   `yaml:"cf_stacks"`
}

type manifest struct {
	LanguageString  string            `yaml:"language"`
	DefaultVersions []Dependency      `yaml:"default_versions"`
	ManifestEntries []ManifestEntry   `yaml:"dependencies"`
	Deprecations    []DeprecationDate `yaml:"dependency_deprecation_dates"`
	manifestRootDir string
	currentTime     time.Time
}

type BuildpackMetadata struct {
	Language string `yaml:"language"`
	Version  string `yaml:"version"`
}

func NewManifest(bpDir string, currentTime time.Time) (Manifest, error) {
	var m manifest

	err := NewYAML().Load(filepath.Join(bpDir, "manifest.yml"), &m)
	if err != nil {
		return nil, err
	}

	m.manifestRootDir, err = filepath.Abs(bpDir)
	if err != nil {
		return nil, err
	}

	m.currentTime = currentTime

	return &m, nil
}

func (m *manifest) RootDir() string {
	return m.manifestRootDir
}

func (m *manifest) CheckBuildpackVersion(cacheDir string) {
	var md BuildpackMetadata

	err := NewYAML().Load(filepath.Join(cacheDir, "BUILDPACK_METADATA"), &md)
	if err != nil {
		return
	}

	if md.Language != m.Language() {
		return
	}

	version, err := m.Version()
	if err != nil {
		return
	}

	if md.Version != version {
		Log.Warning("buildpack version changed from %s to %s", md.Version, version)
	}

	return
}

func (m *manifest) StoreBuildpackMetadata(cacheDir string) {
	logOutput := Log.GetOutput()
	Log.SetOutput(ioutil.Discard)
	defer Log.SetOutput(logOutput)

	version, err := m.Version()
	if err != nil {
		return
	}

	md := BuildpackMetadata{Language: m.Language(), Version: version}

	if exists, _ := FileExists(cacheDir); exists {
		_ = NewYAML().Write(filepath.Join(cacheDir, "BUILDPACK_METADATA"), &md)
	}
}

func (m *manifest) Language() string {
	return m.LanguageString
}

func (m *manifest) Version() (string, error) {
	version, err := ioutil.ReadFile(filepath.Join(m.manifestRootDir, "VERSION"))
	if err != nil {
		return "", fmt.Errorf("unable to read VERSION file %s", err)
	}

	return strings.TrimSpace(string(version)), nil
}

func (m *manifest) CheckStackSupport() error {
	requiredStack := os.Getenv("CF_STACK")

	if len(m.ManifestEntries) == 0 {
		return nil
	}
	for _, entry := range m.ManifestEntries {
		for _, stack := range entry.CFStacks {
			if stack == requiredStack {
				return nil
			}
		}
	}
	return fmt.Errorf("required stack %s was not found", requiredStack)
}

func (m *manifest) DefaultVersion(depName string) (Dependency, error) {
	var defaultVersion Dependency
	var err error
	numDefaults := 0

	for _, dep := range m.DefaultVersions {
		if depName == dep.Name {
			defaultVersion = dep
			numDefaults++
		}
	}

	if numDefaults == 0 {
		err = fmt.Errorf("no default version for %s", depName)
	} else if numDefaults > 1 {
		err = fmt.Errorf("found %d default versions for %s", numDefaults, depName)
	}

	if err != nil {
		Log.Error(defaultVersionsError)
		return Dependency{}, err
	}

	return defaultVersion, nil
}

func (m *manifest) InstallDependency(dep Dependency, outputDir string) error {
	Log.BeginStep("Installing %s %s", dep.Name, dep.Version)

	tmpDir, err := ioutil.TempDir("", "downloads")
	if err != nil {
		return err
	}
	tmpFile := filepath.Join(tmpDir, "archive")

	entry, err := m.getEntry(dep)
	if err != nil {
		return err
	}

	err = m.FetchDependency(dep, tmpFile)
	if err != nil {
		return err
	}

	err = m.warnNewerPatch(dep)
	if err != nil {
		return err
	}

	err = m.warnEndOfLife(dep)
	if err != nil {
		return err
	}

	err = os.MkdirAll(outputDir, 0755)
	if err != nil {
		return err
	}

	if strings.HasSuffix(entry.URI, ".zip") {
		return ExtractZip(tmpFile, outputDir)
	}

	return ExtractTarGz(tmpFile, outputDir)
}

func (m *manifest) warnNewerPatch(dep Dependency) error {
	versions := m.AllDependencyVersions(dep.Name)

	v, err := semver.NewVersion(dep.Version)
	if err != nil {
		return err
	}

	constraint := fmt.Sprintf("%d.%d.x", v.Major(), v.Minor())
	latest, err := FindMatchingVersion(constraint, versions)
	if err != nil {
		return err
	}

	if latest != dep.Version {
		Log.Warning(outdatedDependencyWarning(dep, latest))
	}

	return nil
}

func (m *manifest) warnEndOfLife(dep Dependency) error {
	v, err := semver.NewVersion(dep.Version)
	if err != nil {
		return err
	}

	for _, deprecation := range m.Deprecations {
		if deprecation.Name != dep.Name {
			continue
		}

		versionLine, err := semver.NewConstraint(deprecation.VersionLine)
		if err != nil {
			return err
		}

		eolTime, err := time.Parse(dateFormat, deprecation.Date)
		if err != nil {
			return err
		}
		if versionLine.Check(v) && eolTime.Sub(m.currentTime) < thirtyDays {
			Log.Warning(endOfLifeWarning(dep.Name, deprecation.VersionLine, deprecation.Date, deprecation.Link))
		}
	}
	return nil
}

func (m *manifest) FetchDependency(dep Dependency, outputFile string) error {
	entry, err := m.getEntry(dep)
	if err != nil {
		return err
	}

	filteredURI, err := filterURI(entry.URI)
	if err != nil {
		return err
	}

	if m.isCached() {
		r := strings.NewReplacer("/", "_", ":", "_", "?", "_", "&", "_")
		source := filepath.Join(m.manifestRootDir, "dependencies", r.Replace(filteredURI))
		Log.Info("Copy [%s]", source)
		err = CopyFile(source, outputFile)
	} else {
		Log.Info("Download [%s]", filteredURI)
		err = downloadFile(entry.URI, outputFile)
	}
	if err != nil {
		return err
	}

	err = checkMD5(outputFile, entry.MD5)
	if err != nil {
		os.Remove(outputFile)
		return err
	}

	return nil
}

func (m *manifest) AllDependencyVersions(depName string) []string {
	var depVersions []string

	for _, e := range m.ManifestEntries {
		if e.Dependency.Name == depName {
			depVersions = append(depVersions, e.Dependency.Version)
		}
	}

	return depVersions
}

func (m *manifest) InstallOnlyVersion(depName string, installDir string) error {
	depVersions := m.AllDependencyVersions(depName)

	if len(depVersions) > 1 {
		return fmt.Errorf("more than one version of %s found", depName)
	} else if len(depVersions) == 0 {
		return fmt.Errorf("no versions of %s found", depName)
	}

	dep := Dependency{Name: depName, Version: depVersions[0]}
	return m.InstallDependency(dep, installDir)
}

func (m *manifest) getEntry(dep Dependency) (*ManifestEntry, error) {
	for _, e := range m.ManifestEntries {
		if e.Dependency == dep {
			return &e, nil
		}
	}

	Log.Error(dependencyMissingError(m, dep))
	return nil, fmt.Errorf("dependency %s %s not found", dep.Name, dep.Version)
}

func (m *manifest) isCached() bool {
	dependenciesDir := filepath.Join(m.manifestRootDir, "dependencies")

	isCached, err := FileExists(dependenciesDir)
	if err != nil {
		Log.Warning("Error determining if buildpack is cached: %s", err.Error())
	}

	return isCached
}
