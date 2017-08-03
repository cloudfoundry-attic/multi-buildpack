package integration_test

import (
	"path/filepath"

	"github.com/cloudfoundry/libbuildpack/cutlass"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("running supply go buildpack before the ruby buildpack", func() {
	var app *cutlass.App
	AfterEach(func() {
		if app != nil {
			app.Destroy()
		}
		app = nil
	})

	Context("the app is pushed", func() {
		BeforeEach(func() {
			app = cutlass.New(filepath.Join(bpDir, "fixtures", "rails5"))
			app.Buildpack = "multi_buildpack"
		})

		It("finds the supplied dependency in the runtime container", func() {
			PushAppAndConfirm(app)

			Expect(app.Stdout.String()).To(ContainSubstring("Multi Buildpack version"))
			Expect(app.Stdout.String()).To(ContainSubstring("Nodejs Buildpack version"))
			Expect(app.Stdout.String()).To(ContainSubstring("Installing node 8."))

			body, err := app.GetBody("/")
			Expect(err).To(BeNil())
			Expect(body).To(ContainSubstring("Ruby version: ruby 2."))
			Expect(body).To(ContainSubstring("Node version: v8."))
			Expect(body).To(ContainSubstring("/home/vcap/deps/0/node"))

			Expect(app.Stdout.String()).To(ContainSubstring("Skipping install of nodejs since it has been supplied"))
		})
	})
})
