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
		panic(err)
	}

	buildpacks, err := GetBuildpacks(compiler.BuildDir, logger)
	if err != nil {
		panic(err)
	}

	mc, err := NewMultiCompiler(compiler, buildpacks)
	if err != nil {
		panic(err)
	}

	err = mc.Compile()
	if err != nil {
		panic(err)
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
	newBuildDir, err := c.MoveBuildDir()
	if err != nil {
		c.Compiler.Log.Error("Unable to move app directory: %s", err.Error())
		return err
	}

	config, err := c.NewLifecycleBuilderConfig(newBuildDir)
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

	err = libbuildpack.WriteProfileD(newBuildDir, "00000000-multi.sh", "mv .deps ../deps && export DEPS_DIR=$HOME/../deps\n")
	if err != nil {
		c.Compiler.Log.Warning("Unable create .profile.d/00000000-multi.sh script: %s", err.Error())
		return err
	}

	err = c.CleanupStagingArea(newBuildDir)
	if err != nil {
		c.Compiler.Log.Warning("Unable to clean staging container: %s", err.Error())
		return err
	}

	return nil
}

func (c *MultiCompiler) MoveBuildDir() (string, error) {
	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		return "", err
	}

	newDir := filepath.Join(tempDir, "app")

	c.Compiler.Log.BeginStep("Staging app in %s", newDir)
	err = os.Rename(c.Compiler.BuildDir, newDir)
	if err != nil {
		return "", err
	}

	err = os.Symlink(newDir, c.Compiler.BuildDir)
	if err != nil {
		fmt.Println(err.Error())
		return "", err
	}

	return newDir, nil
}

func (c *MultiCompiler) NewLifecycleBuilderConfig(buildDir string) (buildpackapplifecycle.LifecycleBuilderConfig, error) {
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
	if err := cfg.Set("buildDir", buildDir); err != nil {
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
func (c *MultiCompiler) CleanupStagingArea(newBuildDir string) error {
	err := os.RemoveAll(c.DownloadsDir)
	if err != nil {
		c.Compiler.Log.Warning("Unable to remove downloaded buildpacks: %s", err.Error())
	}

	oldDepsDir, err := filepath.Abs(filepath.Join(newBuildDir, "..", "deps"))
	if err != nil {
		return err
	}

	err = os.Rename(oldDepsDir, filepath.Join(newBuildDir, ".deps"))
	if err != nil {
		return err
	}

	err = os.Remove(c.Compiler.BuildDir)
	if err != nil {
		return err
	}

	return os.Rename(newBuildDir, c.Compiler.BuildDir)
}
