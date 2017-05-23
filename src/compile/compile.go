package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"code.cloudfoundry.org/buildpackapplifecycle"
	"code.cloudfoundry.org/buildpackapplifecycle/buildpackrunner"
	"github.com/cloudfoundry/libbuildpack"
)

type Runner interface {
	Run() (string, error)
}

// MultiCompiler a struct to compile this buildpack
type MultiCompiler struct {
	BuildDir         string
	CacheDir         string
	Log              *libbuildpack.Logger
	Buildpacks       []string
	DownloadsDir     string
	Runner           Runner
	ExistingDepsDirs []string
}

func main() {
	logger := libbuildpack.NewLogger(os.Stdout)

	buildpackDir, err := libbuildpack.GetBuildpackDir()
	if err != nil {
		logger.Error("Unable to determine buildpack directory: %s", err.Error())
		os.Exit(8)
	}

	manifest, err := libbuildpack.NewManifest(buildpackDir, logger, time.Now())
	if err != nil {
		logger.Error("Unable to load buildpack manifest: %s", err.Error())
		os.Exit(9)
	}

	stager := libbuildpack.NewStager(os.Args[1:], logger, manifest)
	err = stager.CheckBuildpackValid()
	if err != nil {
		os.Exit(10)
	}

	buildpacks, err := GetBuildpacks(stager.BuildDir(), logger)
	if err != nil {
		os.Exit(11)
	}

	mc, err := NewMultiCompiler(stager.BuildDir(), stager.CacheDir(), buildpacks, logger)
	if err != nil {
		os.Exit(12)
	}

	err = mc.Compile()
	if err != nil {
		os.Exit(13)
	}

	stager.StagingComplete()
}

// NewMultiCompiler creates a new MultiCompiler
func NewMultiCompiler(buildDir, cacheDir string, buildpacks []string, logger *libbuildpack.Logger) (*MultiCompiler, error) {
	downloadsDir, err := ioutil.TempDir("", "downloads")
	if err != nil {
		return nil, err
	}
	mc := &MultiCompiler{
		BuildDir:         buildDir,
		CacheDir:         cacheDir,
		Buildpacks:       buildpacks,
		DownloadsDir:     downloadsDir,
		Log:              logger,
		Runner:           nil,
		ExistingDepsDirs: []string{},
	}
	return mc, nil
}

// Compile this buildpack
func (c *MultiCompiler) Compile() error {
	config, err := c.NewLifecycleBuilderConfig()
	if err != nil {
		c.Log.Error("Unable to set up runner config: %s", err.Error())
		return err
	}

	c.ExistingDepsDirs, err = filepath.Glob(filepath.Join(os.TempDir(), "contents*", "deps"))
	if err != nil {
		c.Log.Error("Unable to locate directories: %s", err.Error())
		return err
	}

	c.Runner = buildpackrunner.New(&config)

	stagingInfoFile, err := c.RunBuildpacks()
	if err != nil {
		c.Log.Error("Unable to run all buildpacks: %s", err.Error())
		return err
	}

	err = WriteStartCommand(stagingInfoFile, "/tmp/multi-buildpack-release.yml")
	if err != nil {
		c.Log.Error("Unable to write start command: %s", err.Error())
		return err
	}

	profiledDir := filepath.Join(c.BuildDir, ".profile.d")
	err = os.MkdirAll(profiledDir, 0755)
	if err != nil {
		return err
	}

	profileScript := `
	set -e

	if [ -d .deps ]; then
		mkdir -p ../deps
		mv .deps/* ../deps/
		rm -rf .deps
	fi
	DIR=$(dirname $HOME)
	export DEPS_DIR=$DIR/deps
	`

	err = ioutil.WriteFile(filepath.Join(profiledDir, "00000000_multi.sh"), []byte(profileScript), 0755)

	if err != nil {
		c.Log.Error("Unable create .profile.d/00000000_multi.sh script: %s", err.Error())
		return err
	}

	err = c.CleanupStagingArea()
	if err != nil {
		c.Log.Warning("Unable to clean staging container: %s", err.Error())
		return err
	}

	return nil
}

func (c *MultiCompiler) NewLifecycleBuilderConfig() (buildpackapplifecycle.LifecycleBuilderConfig, error) {
	cfg := buildpackapplifecycle.NewLifecycleBuilderConfig([]string{}, true, false)
	if err := cfg.Set("buildpacksDir", c.DownloadsDir); err != nil {
		return cfg, err
	}
	if err := cfg.Set("buildpackOrder", strings.Join(c.Buildpacks, ",")); err != nil {
		return cfg, err
	}
	if err := cfg.Set("outputDroplet", "/dev/null"); err != nil {
		return cfg, err
	}
	if err := cfg.Set("buildDir", c.BuildDir); err != nil {
		return cfg, err
	}

	if err := cfg.Set("buildArtifactsCacheDir", c.CacheDir); err != nil {
		return cfg, err
	}

	if err := cfg.Validate(); err != nil {
		return cfg, err
	}

	return cfg, nil
}

// RunBuildpacks calls the builder
func (c *MultiCompiler) RunBuildpacks() (string, error) {
	if len(c.Buildpacks) == 0 {
		return "", nil
	}

	c.Log.BeginStep("Running buildpacks:")
	c.Log.Info(strings.Join(c.Buildpacks, "\n"))

	return c.Runner.Run()
}

// CleanupStagingArea moves prepares the staging container to be tarred by the old lifecycle
func (c *MultiCompiler) CleanupStagingArea() error {
	if err := os.RemoveAll(c.DownloadsDir); err != nil {
		c.Log.Warning("Unable to remove downloaded buildpacks: %s", err.Error())
	}

	depsDirs, err := filepath.Glob(filepath.Join(os.TempDir(), "contents*", "deps"))
	if err != nil {
		return err
	}

	depsDirs = removeAll(depsDirs, c.ExistingDepsDirs)

	if len(depsDirs) != 1 {
		return fmt.Errorf("found %d deps dirs, expected 1", len(depsDirs))
	}
	return os.Rename(depsDirs[0], filepath.Join(c.BuildDir, ".deps"))
}

func removeAll(s []string, r []string) []string {
	remove := func(s []string, r string) []string {
		for i, v := range s {
			if v == r {
				return append(s[:i], s[i+1:]...)
			}
		}
		return s
	}

	for _, v := range r {
		s = remove(s, v)
	}
	return s
}
