package main_test

import (
	c "compile"
	"io/ioutil"
	"os"
	"path/filepath"

	"bytes"

	"code.cloudfoundry.org/buildpackapplifecycle"
	bp "github.com/cloudfoundry/libbuildpack"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

//go:generate mockgen -source=vendor/github.com/cloudfoundry/libbuildpack/manifest.go --destination=mocks_manifest_test.go --package=main_test --imports=.=github.com/cloudfoundry/libbuildpack
//go:generate mockgen -source=vendor/code.cloudfoundry.org/buildpackapplifecycle/buildpackrunner/runner.go --destination=mocks_runner_test.go --package=main_test

var _ = Describe("Compile", func() {
	var (
		err          error
		buildDir     string
		cacheDir     string
		compiler     *c.MultiCompiler
		buildpacks   []string
		downloadsDir string
		mockCtrl     *gomock.Controller
		mockManifest *MockManifest
		mockRunner   *MockRunner
		buffer       *bytes.Buffer

		logger bp.Logger
	)

	BeforeEach(func() {
		buildDir, err = ioutil.TempDir("", "build")
		Expect(err).To(BeNil())

		cacheDir, err = ioutil.TempDir("", "cache")
		Expect(err).To(BeNil())

		downloadsDir, err = ioutil.TempDir("", "downloads")
		Expect(err).To(BeNil())

		buffer = new(bytes.Buffer)
		logger = bp.NewLogger()
		logger.SetOutput(buffer)

		buildpacks = []string{}

		mockCtrl = gomock.NewController(GinkgoT())
		mockManifest = NewMockManifest(mockCtrl)
		mockRunner = NewMockRunner(mockCtrl)
	})

	JustBeforeEach(func() {
		bpc := &bp.Compiler{
			BuildDir: buildDir,
			CacheDir: cacheDir,
			Manifest: mockManifest,
			Log:      logger,
		}

		compiler = &c.MultiCompiler{
			Compiler:     bpc,
			Buildpacks:   buildpacks,
			DownloadsDir: downloadsDir,
			Runner:       mockRunner,
		}
	})

	AfterEach(func() {
		err = os.RemoveAll(buildDir)
		Expect(err).To(BeNil())

		err = os.RemoveAll(cacheDir)
		Expect(err).To(BeNil())

		err = os.RemoveAll(downloadsDir)
		Expect(err).To(BeNil())

	})

	Describe("RemoveUnusedCache", func() {
		var (
			existingDirs       []string
			buildpackCacheDirs []string
		)

		BeforeEach(func() {
			buildpacks = []string{"first_buildpack", "second_buildpack"}
		})

		JustBeforeEach(func() {
			for _, bp := range compiler.Buildpacks {
				buildpackCacheDirs = append(buildpackCacheDirs, compiler.CacheDir(bp))
				err = os.MkdirAll(compiler.CacheDir(bp), 0755)
				Expect(err).To(BeNil())
			}
		})

		AfterEach(func() {
			existingDirs = []string{}
			buildpackCacheDirs = []string{}
		})

		Context("there are no unused cache directories", func() {
			It("does not remove anything", func() {
				compiler.RemoveUnusedCache()

				dirs, err := ioutil.ReadDir(cacheDir)
				Expect(err).To(BeNil())

				for _, dir := range dirs {
					existingDirs = append(existingDirs, filepath.Join(cacheDir, dir.Name()))
				}

				Expect(existingDirs).To(ConsistOf(buildpackCacheDirs))
			})
		})

		Context("there are unused cache directories", func() {
			BeforeEach(func() {
				err = os.MkdirAll(filepath.Join(cacheDir, "an_unused_cache_dir"), 0755)
				Expect(err).To(BeNil())
			})

			It("removes the unused directory", func() {
				compiler.RemoveUnusedCache()

				dirs, err := ioutil.ReadDir(cacheDir)
				Expect(err).To(BeNil())

				for _, dir := range dirs {
					existingDirs = append(existingDirs, filepath.Join(cacheDir, dir.Name()))
				}

				Expect(existingDirs).To(ConsistOf(buildpackCacheDirs))
			})
		})
	})

	Describe("RunBuildpacks", func() {
		Context("a list of buildpacks is provided", func() {
			BeforeEach(func() {
				buildpacks = []string{"third_buildpack", "fourth_buildpack"}
			})

			It("runs all the buildpacks", func() {
				call0 := mockRunner.EXPECT().Run(gomock.Any()).Do(func(config *buildpackapplifecycle.LifecycleBuilderConfig) {
					Expect(config.BuildDir()).To(Equal(buildDir))
					Expect(config.BuildpackOrder()).To(ConsistOf(buildpacks[0]))
					Expect(config.OutputDroplet()).To(Equal("/dev/null"))
					Expect(config.BuildpacksDir()).To(Equal(downloadsDir))
					Expect(config.BuildArtifactsCacheDir()).To(Equal(compiler.CacheDir(buildpacks[0])))
				})

				mockRunner.EXPECT().Run(gomock.Any()).Do(func(config *buildpackapplifecycle.LifecycleBuilderConfig) {
					Expect(config.BuildDir()).To(Equal(buildDir))
					Expect(config.BuildpackOrder()).To(ConsistOf(buildpacks[1]))
					Expect(config.OutputDroplet()).To(Equal("/dev/null"))
					Expect(config.BuildpacksDir()).To(Equal(downloadsDir))
					Expect(config.BuildArtifactsCacheDir()).To(Equal(compiler.CacheDir(buildpacks[1])))

				}).After(call0)

				err = compiler.RunBuildpacks()
				Expect(err).To(BeNil())

				Expect(buffer.String()).To(ContainSubstring("-----> Running builder for buildpack third_buildpack"))
				Expect(buffer.String()).To(ContainSubstring("-----> Running builder for buildpack fourth_buildpack"))

			})
		})

		Context("a list of buildpacks is empty", func() {
			It("returns without calling runner.Run", func() {
				mockRunner.EXPECT().Run(gomock.Any()).Times(0)

				err = compiler.RunBuildpacks()
				Expect(err).To(BeNil())

				Expect(buffer.String()).To(Equal(""))
			})
		})
	})
})
