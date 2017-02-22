package main_test

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"

	c "compile"

	"github.com/cloudfoundry/libbuildpack"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("GetBuildpacks", func() {
	var (
		buildpacks []string
		buildDir   string
		err        error
		buffer     *bytes.Buffer
		logger     libbuildpack.Logger
	)

	BeforeEach(func() {
		buildDir, err = ioutil.TempDir("", "build")
		Expect(err).To(BeNil())

		buffer = new(bytes.Buffer)
		logger = libbuildpack.NewLogger()
		logger.SetOutput(buffer)
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
			_, err := c.GetBuildpacks(buildDir, logger)
			Expect(err).ToNot(BeNil())
		})

		It("informs the user", func() {
			c.GetBuildpacks(buildDir, logger)

			Expect(buffer.String()).To(Equal("       **ERROR** The multi-buildpack.yml file is malformed.\n"))
		})
	})

	Context("multi-buildpack.yml does not exist", func() {
		It("returns an error", func() {
			_, err := c.GetBuildpacks(buildDir, logger)
			Expect(err).ToNot(BeNil())
		})

		It("informs the user", func() {
			c.GetBuildpacks(buildDir, logger)

			Expect(buffer.String()).To(Equal("       **ERROR** A multi-buildpack.yml file must be provided at your app root to use this buildpack.\n"))
		})
	})
})
