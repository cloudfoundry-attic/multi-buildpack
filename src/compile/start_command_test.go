package main_test

import (
	"io/ioutil"
	"os"
	"path/filepath"

	c "compile"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("WriteStartCommand", func() {
	var (
		stagingInfoDir  string
		stagingInfoFile string
		outputDir       string
		outputFile      string
		err             error
	)

	BeforeEach(func() {
		stagingInfoDir, err = ioutil.TempDir("", "contents")
		Expect(err).To(BeNil())
		stagingInfoFile = filepath.Join(stagingInfoDir, "staging_info.yml")

		outputDir, err = ioutil.TempDir("", "release")
		Expect(err).To(BeNil())
		outputFile = filepath.Join(outputDir, "multi-buildpack-release.yml")
	})

	AfterEach(func() {
		err = os.RemoveAll(stagingInfoDir)
		Expect(err).To(BeNil())

		err = os.RemoveAll(outputDir)
		Expect(err).To(BeNil())
	})

	Context("staging_info.yml exists", func() {
		BeforeEach(func() {
			content := `{"detected_buildpack":"some_buildpack","start_command":"run_thing arg1 arg2"}`
			err = ioutil.WriteFile(stagingInfoFile, []byte(content), 0644)
			Expect(err).To(BeNil())
		})

		It("writes the intended release output to multi-buildpack-release.yml ", func() {
			err = c.WriteStartCommand(stagingInfoFile, outputFile)

			Expect(err).To(BeNil())

			data, err := ioutil.ReadFile(outputFile)
			Expect(err).To(BeNil())
			Expect(string(data)).To(Equal("default_process_types:\n  web: run_thing arg1 arg2\n"))
		})
	})

	Context("staging_info.yml is malformed", func() {
		BeforeEach(func() {
			content := `{"detected_buildpack" "some_buildpack "start_command run_thing arg1 arg2"}`
			err = ioutil.WriteFile(stagingInfoFile, []byte(content), 0644)
			Expect(err).To(BeNil())
		})

		It("returns an error", func() {
			err = c.WriteStartCommand(stagingInfoFile, outputFile)

			Expect(err).NotTo(BeNil())
		})
	})

	Context("staging_info.yml does not exist", func() {
		It("returns an error", func() {
			err = c.WriteStartCommand(stagingInfoFile, outputFile)

			Expect(err).NotTo(BeNil())
		})
	})
})
