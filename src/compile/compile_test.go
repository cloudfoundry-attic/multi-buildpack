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
				call0 := mockRunner.EXPECT().Run(gomock.Any()).Do(func(config *buildpackapplifecycle.LifecycleBuilderConfig) {
					Expect(config.BuildDir()).To(Equal(newBuildDir))
					Expect(config.BuildpackOrder()).To(ConsistOf(buildpacks[0]))
					Expect(config.OutputDroplet()).To(Equal("/dev/null"))
					Expect(config.BuildpacksDir()).To(Equal(downloadsDir))
					Expect(config.BuildArtifactsCacheDir()).To(Equal(compiler.CacheDir(buildpacks[0])))
				}).Return("third/staging_info.yml", nil)

				mockRunner.EXPECT().Run(gomock.Any()).Do(func(config *buildpackapplifecycle.LifecycleBuilderConfig) {
					Expect(config.BuildDir()).To(Equal(newBuildDir))
					Expect(config.BuildpackOrder()).To(ConsistOf(buildpacks[1]))
					Expect(config.OutputDroplet()).To(Equal("/dev/null"))
					Expect(config.BuildpacksDir()).To(Equal(downloadsDir))
					Expect(config.BuildArtifactsCacheDir()).To(Equal(compiler.CacheDir(buildpacks[1])))

				}).Return("fourth/staging_info.yml", nil).After(call0)
			})

			It("runs all the buildpacks", func() {
				_, err = compiler.RunBuildpacks(newBuildDir)
				Expect(err).To(BeNil())

				Expect(buffer.String()).To(ContainSubstring("-----> Running builder for buildpack third_buildpack"))
				Expect(buffer.String()).To(ContainSubstring("-----> Running builder for buildpack fourth_buildpack"))

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
		var newBuildDir string

		BeforeEach(func() {
			newBuildDir, err = ioutil.TempDir("", "return")
			Expect(err).To(BeNil())

			err = ioutil.WriteFile(filepath.Join(newBuildDir, "file1.txt"), []byte("test"), 0644)
			Expect(err).To(BeNil())

			err = ioutil.WriteFile(filepath.Join(newBuildDir, "file2.txt"), []byte("test2"), 0644)
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
		})
	})
})
