package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"code.cloudfoundry.org/buildpackapplifecycle"
	"code.cloudfoundry.org/buildpackapplifecycle/buildpackrunner"
	"github.com/cloudfoundry/libbuildpack"
)

type Runner interface {
	Run() (string, error)
}

// MultiCompiler a struct to compile this buildpack
type MultiCompiler struct {
	Compiler     *libbuildpack.Compiler
	Buildpacks   []string
	DownloadsDir string
	Runner       Runner
}

func main() {
	logger := libbuildpack.NewLogger()

	compiler, err := libbuildpack.NewCompiler(os.Args[1:], logger)
	err = compiler.CheckBuildpackValid()
	if err != nil {
		os.Exit(10)
	}

	buildpacks, err := GetBuildpacks(compiler.BuildDir, logger)
	if err != nil {
		os.Exit(11)
	}

	mc, err := NewMultiCompiler(compiler, buildpacks)
	if err != nil {
		os.Exit(12)
	}

	err = mc.Compile()
	if err != nil {
		os.Exit(13)
	}

	compiler.StagingComplete()
}

// NewMultiCompiler creates a new MultiCompiler
func NewMultiCompiler(compiler *libbuildpack.Compiler, buildpacks []string) (*MultiCompiler, error) {
	downloadsDir, err := ioutil.TempDir("", "downloads")
	if err != nil {
		return nil, err
	}
	mc := &MultiCompiler{
		Compiler:     compiler,
		Buildpacks:   buildpacks,
		DownloadsDir: downloadsDir,
		Runner:       nil,
	}
	return mc, nil
}

// Compile this buildpack
func (c *MultiCompiler) Compile() error {
	config, err := c.NewLifecycleBuilderConfig()
	if err != nil {
		c.Compiler.Log.Error("Unable to set up runner config: %s", err.Error())
		return err
	}

	c.Runner = buildpackrunner.New(&config)

	stagingInfoFile, err := c.RunBuildpacks()
	if err != nil {
		c.Compiler.Log.Error("Unable to run all buildpacks: %s", err.Error())
		return err
	}

	err = WriteStartCommand(stagingInfoFile, "/tmp/multi-buildpack-release.yml")
	if err != nil {
		c.Compiler.Log.Error("Unable to write start command: %s", err.Error())
		return err
	}

	err = libbuildpack.WriteProfileD(c.Compiler.BuildDir, "00000000-multi.sh", "mv .deps ../deps && export DEPS_DIR=$HOME/../deps\n")
	if err != nil {
		c.Compiler.Log.Warning("Unable create .profile.d/00000000-multi.sh script: %s", err.Error())
		return err
	}

	err = c.CleanupStagingArea()
	if err != nil {
		c.Compiler.Log.Warning("Unable to clean staging container: %s", err.Error())
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
	if err := cfg.Set("buildDir", c.Compiler.BuildDir); err != nil {
		return cfg, err
	}

	if err := cfg.Set("buildArtifactsCacheDir", c.Compiler.CacheDir); err != nil {
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

	c.Compiler.Log.BeginStep("Running buildpacks:")
	c.Compiler.Log.Info(strings.Join(c.Buildpacks, "\n"))

	return c.Runner.Run()
}

// CleanupStagingArea moves prepares the staging container to be tarred by the old lifecycle
func (c *MultiCompiler) CleanupStagingArea() error {
	if err := os.RemoveAll(c.DownloadsDir); err != nil {
		c.Compiler.Log.Warning("Unable to remove downloaded buildpacks: %s", err.Error())
	}

	depsDirs, err := filepath.Glob(filepath.Join(os.TempDir(), "contents*", "deps"))
	if err != nil {
		return err
	}

	if len(depsDirs) != 1 {
		return fmt.Errorf("found %d deps dirs, expected 1", len(depsDirs))
	}
	return os.Rename(depsDirs[0], filepath.Join(c.Compiler.BuildDir, ".deps"))
}
