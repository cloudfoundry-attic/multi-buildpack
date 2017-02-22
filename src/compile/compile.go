package main

import (
	"crypto/md5"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"code.cloudfoundry.org/buildpackapplifecycle"
	"code.cloudfoundry.org/buildpackapplifecycle/buildpackrunner"
	"github.com/cloudfoundry/libbuildpack"
)

// MultiCompiler a struct to compile this buildpack
type MultiCompiler struct {
	Compiler     *libbuildpack.Compiler
	Buildpacks   []string
	DownloadsDir string
	Runner       buildpackrunner.Runner
}

func main() {
	buildDir := os.Args[1]
	cacheDir := os.Args[2]

	logger := libbuildpack.NewLogger()

	compiler, err := libbuildpack.NewCompiler(buildDir, cacheDir, logger)
	err = compiler.CheckBuildpackValid()
	if err != nil {
		panic(err)
	}

	buildpacks, err := GetBuildpacks(buildDir, logger)
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
		Runner:       buildpackrunner.New(),
	}
	return mc, nil
}

// Compile this buildpack
func (c *MultiCompiler) Compile() error {
	err := c.RemoveUnusedCache()
	if err != nil {
		c.Compiler.Log.Warning("Unable to clean unused cache directories: %s", err.Error())
	}

	stagingInfoFile, err := c.RunBuildpacks()
	if err != nil {
		c.Compiler.Log.Error("Unable to run all buildpacks: %s", err.Error())
		return err
	}

	err = WriteStartCommand(stagingInfoFile, "/tmp/multi-buildpack-release.yml")
	if err != nil {
		c.Compiler.Log.Error("Unable to write start command: ")
	}

	c.Compiler.Log.BeginStep("Removing buildpack downloads directory %s", c.DownloadsDir)
	err = os.RemoveAll(c.DownloadsDir)
	if err != nil {
		c.Compiler.Log.Warning("Unable to remove downloaded buildpacks: %s", err.Error())
	}
	return nil
}

func (c *MultiCompiler) RunBuildpacks() (string, error) {
	var stagingInfoFile string

	for _, buildpack := range c.Buildpacks {
		c.Compiler.Log.BeginStep("Running builder for buildpack %s", buildpack)

		config, err := c.newLifecycleBuilderConfig(c.DownloadsDir, buildpack, c.Compiler.BuildDir)
		if err := config.Validate(); err != nil {
			return "", err
		}

		stagingInfoFile, err = c.Runner.Run(&config)
		if err != nil {
			c.Compiler.Log.Error(err.Error())

			// FIXME Should probably return here
			// return err
		}
	}

	return stagingInfoFile, nil
}

func (c *MultiCompiler) newLifecycleBuilderConfig(downloadsDir, buildpack, buildDir string) (buildpackapplifecycle.LifecycleBuilderConfig, error) {
	cfg := buildpackapplifecycle.NewLifecycleBuilderConfig([]string{}, true, false)
	if err := cfg.Set("buildpacksDir", downloadsDir); err != nil {
		return cfg, err
	}
	if err := cfg.Set("buildpackOrder", buildpack); err != nil {
		return cfg, err
	}
	if err := cfg.Set("outputDroplet", "/dev/null"); err != nil {
		return cfg, err
	}
	if err := cfg.Set("buildDir", buildDir); err != nil {
		return cfg, err
	}

	if err := cfg.Set("buildArtifactsCacheDir", c.CacheDir(buildpack)); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func (c *MultiCompiler) CacheDir(buildpack string) string {
	md5sum := fmt.Sprintf("%x", md5.Sum([]byte(buildpack)))
	return filepath.Join(c.Compiler.CacheDir, string(md5sum[:]))
}

// RemoveUnusedCache removes no longer required cache directories
func (c *MultiCompiler) RemoveUnusedCache() error {
	dirs, err := ioutil.ReadDir(c.Compiler.CacheDir)
	if err != nil {
		return err
	}

	for _, dir := range dirs {
		if c.dirUnused(dir) {
			err = os.RemoveAll(filepath.Join(c.Compiler.CacheDir, dir.Name()))
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *MultiCompiler) dirUnused(dir os.FileInfo) bool {
	var neededCacheDirs []string

	for _, bp := range c.Buildpacks {
		neededCacheDirs = append(neededCacheDirs, c.CacheDir(bp))
	}

	for _, i := range neededCacheDirs {
		if i == filepath.Join(c.Compiler.CacheDir, dir.Name()) {
			return false
		}
	}

	return true
}
