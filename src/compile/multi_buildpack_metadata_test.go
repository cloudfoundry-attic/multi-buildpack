package main_test

import (
	"io/ioutil"
	"os"
	"path/filepath"

	c "compile"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

//go:generate mockgen -source=vendor/github.com/cloudfoundry/libbuildpack/logger.go --destination=mocks_logger_test.go --package=main_test

var _ = Describe("GetBuildpacks", func() {
	var (
		buildpacks []string
		buildDir   string
		err        error
		mockCtrl   *gomock.Controller
		logger     *MockLogger
	)

	BeforeEach(func() {
		buildDir, err = ioutil.TempDir("", "build")
		Expect(err).To(BeNil())

		mockCtrl = gomock.NewController(GinkgoT())
		logger = NewMockLogger(mockCtrl)
	})

	AfterEach(func() {
		err = os.RemoveAll(buildDir)
		Expect(err).To(BeNil())
	})

	Context("multi-buildpack.yml exists", func() {
		BeforeEach(func() {
			content := "buildpacks:\n- some-buildpack\n- some-other-buildpack"
			err = ioutil.WriteFile(filepath.Join(buildDir, "multi-buildpack.yml"), []byte(content), 0444)
			Expect(err).To(BeNil())
		})

		It("returns the list of buildpacks provided in multi-buildpack.yml", func() {
			buildpacks, err = c.GetBuildpacks(buildDir, logger)

			Expect(err).To(BeNil())
			Expect(buildpacks).To(Equal([]string{"some-buildpack", "some-other-buildpack"}))
		})
	})

	Context("multi-buildpack.yml is malformed", func() {
		BeforeEach(func() {
			content := "strange unparseable stuff"
			err = ioutil.WriteFile(filepath.Join(buildDir, "multi-buildpack.yml"), []byte(content), 0444)
			Expect(err).To(BeNil())
		})

		It("returns an error", func() {
			logger.EXPECT().Error(gomock.Any()).AnyTimes()

			_, err := c.GetBuildpacks(buildDir, logger)
			Expect(err).ToNot(BeNil())
		})

		It("informs the user", func() {
			logger.EXPECT().Error("The multi-buildpack.yml file is malformed.")
			c.GetBuildpacks(buildDir, logger)
		})
	})

	Context("multi-buildpack.yml does not exist", func() {
		It("returns an error", func() {
			logger.EXPECT().Error(gomock.Any()).AnyTimes()

			_, err := c.GetBuildpacks(buildDir, logger)
			Expect(err).ToNot(BeNil())
		})

		It("informs the user", func() {
			logger.EXPECT().Error("A multi-buildpack.yml file must be provided at your app root to use this buildpack.")
			c.GetBuildpacks(buildDir, logger)
		})
	})
})
