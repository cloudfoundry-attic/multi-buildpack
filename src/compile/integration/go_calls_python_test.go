package integration_test

import (
	"path/filepath"

	"github.com/cloudfoundry/libbuildpack/cutlass"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("running supply python buildpack before the go buildpack", func() {
	var app *cutlass.App
	AfterEach(func() {
		if app != nil {
			app.Destroy()
		}
		app = nil
	})

	pushApp := func(fixture string) {
		app = cutlass.New(filepath.Join(bpDir, "fixtures", fixture))
		app.Buildpacks = []string{"multi_buildpack"}
		PushAppAndConfirm(app)
	}

	It("an app is pushed which uses pip dependencies", func() {
		pushApp("go_calls_python")

		Expect(app.Stdout.String()).To(ContainSubstring("Multi Buildpack version"))
		Expect(app.Stdout.String()).To(ContainSubstring("Installing python-"))

		Expect(app.GetBody("/")).To(ContainSubstring(`[{"hello":"world"}]`))
	})

	It("an app is pushed which uses miniconda", func() {
		pushApp("go_calls_python_miniconda")

		Expect(app.Stdout.String()).To(ContainSubstring("Multi Buildpack version"))
		Expect(app.Stdout.String()).To(ContainSubstring("Installing Miniconda"))

		Expect(app.GetBody("/")).To(ContainSubstring(`[{"hello":"world"}]`))
	})

	It("an app is pushed which uses NLTK corpus", func() {
		pushApp("go_calls_python_nltk")

		Expect(app.Stdout.String()).To(ContainSubstring("Multi Buildpack version"))
		Expect(app.Stdout.String()).To(ContainSubstring("Downloading NLTK corpora..."))

		Expect(app.GetBody("/")).To(ContainSubstring("The Fulton County Grand Jury said Friday an investigation of Atlanta's recent primary election produced"))
	})
})
