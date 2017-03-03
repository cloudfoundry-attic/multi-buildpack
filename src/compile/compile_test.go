package main_test

import (
	c "compile"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"

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

	Describe("RunBuildpacks", func() {
		var (
			stagingInfo string
			newBuildDir string
		)

		BeforeEach(func() {
			newBuildDir = "/tmp/abcd1234/app"
		})
		Context("a list of buildpacks is provided", func() {
			BeforeEach(func() {
				buildpacks = []string{"third_buildpack", "fourth_buildpack"}
			})

			JustBeforeEach(func() {
				mockRunner.EXPECT().Run(gomock.Any()).Do(func(config *buildpackapplifecycle.LifecycleBuilderConfig) {
					Expect(config.BuildDir()).To(Equal(newBuildDir))
					Expect(config.BuildpackOrder()).To(Equal(buildpacks))
					Expect(config.OutputDroplet()).To(Equal("/dev/null"))
					Expect(config.BuildpacksDir()).To(Equal(downloadsDir))
					Expect(config.BuildArtifactsCacheDir()).To(Equal(cacheDir))
				}).Return("fourth/staging_info.yml", nil)
			})

			It("returns the location of the last staging_info.yml", func() {
				stagingInfo, err = compiler.RunBuildpacks(newBuildDir)
				Expect(err).To(BeNil())
				Expect(stagingInfo).To(Equal("fourth/staging_info.yml"))
			})
		})

		Context("a list of buildpacks is empty", func() {
			It("returns without calling runner.Run", func() {
				mockRunner.EXPECT().Run(gomock.Any()).Times(0)

				stagingInfo, err = compiler.RunBuildpacks(newBuildDir)
				Expect(err).To(BeNil())

				Expect(stagingInfo).To(Equal(""))
				Expect(buffer.String()).To(Equal(""))
			})
		})
	})

	Describe("MoveBuildDir", func() {
		BeforeEach(func() {
			err = ioutil.WriteFile(filepath.Join(buildDir, "file1.txt"), []byte("test"), 0644)
			Expect(err).To(BeNil())

			err = ioutil.WriteFile(filepath.Join(buildDir, "file2.txt"), []byte("test2"), 0644)
			Expect(err).To(BeNil())
		})

		Context("there are no errors", func() {
			It("moves the build dir to the new location", func() {
				newDir, err := compiler.MoveBuildDir()
				Expect(err).To(BeNil())

				Expect(newDir).NotTo(Equal(buildDir))
				Expect(ioutil.ReadFile(filepath.Join(newDir, "file1.txt"))).To(Equal([]byte("test")))
				Expect(ioutil.ReadFile(filepath.Join(newDir, "file2.txt"))).To(Equal([]byte("test2")))
			})

			It("the old build dir location is a symlink to the new build dir", func() {
				newDir, err := compiler.MoveBuildDir()
				Expect(err).To(BeNil())

				buildDirInfo, err := os.Lstat(buildDir)
				Expect(err).To(BeNil())
				Expect(buildDirInfo.Mode() & os.ModeSymlink).NotTo(Equal(0000))

				symlinkDest, err := os.Readlink(buildDir)
				Expect(err).To(BeNil())
				Expect(symlinkDest).To(Equal(newDir))
			})

			It("the new directory is of the form /<temp>/<8+ char>/app", func() {
				newDir, err := compiler.MoveBuildDir()
				Expect(err).To(BeNil())

				dirRegex := regexp.MustCompile(`\/.{3,}\/[A-Za-z0-9]{8,}\/app`)
				Expect(dirRegex.Match([]byte(newDir))).To(BeTrue())
			})
		})
	})

	Describe("CleanupStagingArea", func() {
		var (
			newBuildRoot string
			newBuildDir  string
			newDepsDir   string
		)

		BeforeEach(func() {
			newBuildRoot, err = ioutil.TempDir("", "return")
			Expect(err).To(BeNil())

			newBuildDir = filepath.Join(newBuildRoot, "app")
			err = os.MkdirAll(newBuildDir, 0755)
			Expect(err).To(BeNil())

			err = ioutil.WriteFile(filepath.Join(newBuildDir, "file1.txt"), []byte("test"), 0644)
			Expect(err).To(BeNil())

			err = ioutil.WriteFile(filepath.Join(newBuildDir, "file2.txt"), []byte("test2"), 0644)
			Expect(err).To(BeNil())

			newDepsDir = filepath.Join(newBuildRoot, "deps")
			err = os.MkdirAll(newDepsDir, 0755)
			Expect(err).To(BeNil())

			err = ioutil.WriteFile(filepath.Join(newDepsDir, "dep1.txt"), []byte("x1"), 0644)
			Expect(err).To(BeNil())

			err = ioutil.WriteFile(filepath.Join(newDepsDir, "dep2.txt"), []byte("x2"), 0644)
			Expect(err).To(BeNil())

			Expect(downloadsDir).To(BeADirectory())
		})

		AfterEach(func() {
			err = os.RemoveAll(newBuildDir)
			Expect(err).To(BeNil())
		})

		Context("there are no errors", func() {
			It("returns the build dir to the previous location", func() {
				err := compiler.CleanupStagingArea(newBuildDir)
				Expect(err).To(BeNil())

				Expect(ioutil.ReadFile(filepath.Join(buildDir, "file1.txt"))).To(Equal([]byte("test")))
				Expect(ioutil.ReadFile(filepath.Join(buildDir, "file2.txt"))).To(Equal([]byte("test2")))
			})

			It("deletes the directory containing the downloaded buildpacks", func() {
				err := compiler.CleanupStagingArea(newBuildDir)
				Expect(err).To(BeNil())

				Expect(downloadsDir).NotTo(BeADirectory())
			})

			It("it moves <buildDir>/../deps to <buildDir>/.deps", func() {
				depsDir := filepath.Join(buildDir, ".deps")

				err := compiler.CleanupStagingArea(newBuildDir)
				Expect(err).To(BeNil())

				Expect(depsDir).To(BeADirectory())
				Expect(ioutil.ReadFile(filepath.Join(depsDir, "dep1.txt"))).To(Equal([]byte("x1")))
				Expect(ioutil.ReadFile(filepath.Join(depsDir, "dep2.txt"))).To(Equal([]byte("x2")))
			})
		})
	})
})
