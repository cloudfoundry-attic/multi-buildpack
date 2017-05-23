package main_test

import (
	c "compile"
	"io/ioutil"
	"os"
	"path/filepath"

	"bytes"

	"github.com/cloudfoundry/libbuildpack"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

//go:generate mockgen -source=compile.go --destination=mocks_test.go --package=main_test

var _ = Describe("Compile", func() {
	var (
		err          error
		buildDir     string
		cacheDir     string
		compiler     *c.MultiCompiler
		buildpacks   []string
		downloadsDir string
		mockCtrl     *gomock.Controller
		mockRunner   *MockRunner
		buffer       *bytes.Buffer

		logger *libbuildpack.Logger
	)

	BeforeEach(func() {
		buildDir, err = ioutil.TempDir("", "build")
		Expect(err).To(BeNil())

		cacheDir, err = ioutil.TempDir("", "cache")
		Expect(err).To(BeNil())

		downloadsDir, err = ioutil.TempDir("", "downloads")
		Expect(err).To(BeNil())

		buffer = new(bytes.Buffer)
		logger = libbuildpack.NewLogger(buffer)

		buildpacks = []string{}

		mockCtrl = gomock.NewController(GinkgoT())
		mockRunner = NewMockRunner(mockCtrl)
	})

	JustBeforeEach(func() {
		compiler = &c.MultiCompiler{
			BuildDir:     buildDir,
			CacheDir:     cacheDir,
			Log:          logger,
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

	Describe("NewLifecycleBuilderConfig", func() {
		BeforeEach(func() {
			buildpacks = []string{"a", "b", "c"}
		})

		It("sets the correct properties on the config object", func() {
			config, err := compiler.NewLifecycleBuilderConfig()
			Expect(err).To(BeNil())

			Expect(config.BuildDir()).To(Equal(buildDir))
			Expect(config.BuildpackOrder()).To(Equal(buildpacks))
			Expect(config.OutputDroplet()).To(Equal("/dev/null"))
			Expect(config.BuildpacksDir()).To(Equal(downloadsDir))
			Expect(config.BuildArtifactsCacheDir()).To(Equal(cacheDir))
		})
	})

	Describe("RunBuildpacks", func() {
		Context("a list of buildpacks is provided", func() {
			BeforeEach(func() {
				buildpacks = []string{"third_buildpack", "fourth_buildpack"}
			})

			JustBeforeEach(func() {
				mockRunner.EXPECT().Run().Return("fourth/staging_info.yml", nil)
			})

			It("returns the location of the last staging_info.yml", func() {
				stagingInfo, err := compiler.RunBuildpacks()
				Expect(err).To(BeNil())
				Expect(stagingInfo).To(Equal("fourth/staging_info.yml"))
			})
		})

		Context("a list of buildpacks is empty", func() {
			It("returns without calling runner.Run", func() {
				mockRunner.EXPECT().Run().Times(0)

				stagingInfo, err := compiler.RunBuildpacks()
				Expect(err).To(BeNil())

				Expect(stagingInfo).To(Equal(""))
				Expect(buffer.String()).To(Equal(""))
			})
		})
	})

	Describe("CleanupStagingArea", func() {
		var (
			contentsDir string
			depsDir     string
		)

		BeforeEach(func() {
			contentsDir, err = ioutil.TempDir("", "contents")
			Expect(err).To(BeNil())

			depsDir = filepath.Join(contentsDir, "deps")
			err = os.MkdirAll(depsDir, 0755)
			Expect(err).To(BeNil())

			err = ioutil.WriteFile(filepath.Join(depsDir, "dep1.txt"), []byte("x1"), 0644)
			Expect(err).To(BeNil())

			err = ioutil.WriteFile(filepath.Join(depsDir, "dep2.txt"), []byte("x2"), 0644)
			Expect(err).To(BeNil())

			Expect(downloadsDir).To(BeADirectory())
		})

		AfterEach(func() {
			err = os.RemoveAll(contentsDir)
			Expect(err).To(BeNil())
		})

		Context("there are no errors", func() {
			It("deletes the directory containing the downloaded buildpacks", func() {
				Expect(compiler.CleanupStagingArea()).To(Succeed())
				Expect(downloadsDir).NotTo(BeADirectory())
			})

			It("it moves /tmp/<contents>/deps to <buildDir>/.deps", func() {
				buildDepsDir := filepath.Join(buildDir, ".deps")
				Expect(compiler.CleanupStagingArea()).To(Succeed())

				Expect(buildDepsDir).To(BeADirectory())
				Expect(ioutil.ReadFile(filepath.Join(buildDepsDir, "dep1.txt"))).To(Equal([]byte("x1")))
				Expect(ioutil.ReadFile(filepath.Join(buildDepsDir, "dep2.txt"))).To(Equal([]byte("x2")))
			})
		})

		Context("there is an existing deps dir", func() {
			var (
				oldContentsDir string
				oldDepsDir     string
			)

			JustBeforeEach(func() {
				oldContentsDir, err = ioutil.TempDir("", "contents")
				Expect(err).To(BeNil())

				oldDepsDir = filepath.Join(oldContentsDir, "deps")
				Expect(os.MkdirAll(oldDepsDir, 0755)).To(Succeed())

				compiler.ExistingDepsDirs = []string{oldDepsDir}
			})

			AfterEach(func() {
				Expect(os.RemoveAll(oldContentsDir)).To(Succeed())
			})

			It("it moves /tmp/<contents>/deps to <buildDir>/.deps", func() {
				buildDepsDir := filepath.Join(buildDir, ".deps")
				Expect(compiler.CleanupStagingArea()).To(Succeed())

				Expect(buildDepsDir).To(BeADirectory())
				Expect(ioutil.ReadFile(filepath.Join(buildDepsDir, "dep1.txt"))).To(Equal([]byte("x1")))
				Expect(ioutil.ReadFile(filepath.Join(buildDepsDir, "dep2.txt"))).To(Equal([]byte("x2")))
			})

			It("it does not move the old deps dir", func() {
				Expect(compiler.CleanupStagingArea()).To(Succeed())
				Expect(oldDepsDir).To(BeADirectory())
			})
		})
	})
})
