package integration_test

import (
	"path/filepath"

	"github.com/cloudfoundry/libbuildpack/cutlass"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("running supply ruby buildpack before the go buildpack", func() {
	var app *cutlass.App
	AfterEach(func() {
		if app != nil {
			app.Destroy()
		}
		app = nil
	})

	Context("the app is pushed", func() {
		BeforeEach(func() {
			app = cutlass.New(filepath.Join(bpDir, "fixtures", "go_calls_nodejs"))
			app.Buildpack = "multi_buildpack"
		})

		It("finds the supplied dependency in the runtime container", func() {
			PushAppAndConfirm(app)

			Expect(app.Stdout.String()).To(ContainSubstring("Multi Buildpack version"))
			Expect(app.Stdout.String()).To(ContainSubstring("Nodejs Buildpack version"))

			Expect(app.GetBody("/")).To(MatchRegexp("INFO hello world"))
		})
	})
})
