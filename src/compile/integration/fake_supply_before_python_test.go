package integration_test

import (
	"os"
	"path/filepath"

	"github.com/cloudfoundry/libbuildpack"
	"github.com/cloudfoundry/libbuildpack/cutlass"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("running supply buildpacks before the python buildpack", func() {
	var app *cutlass.App
	AfterEach(func() {
		if app != nil {
			app.Destroy()
		}
		app = nil
	})

	Context("a simple app is pushed once", func() {
		BeforeEach(func() {
			app = cutlass.New(filepath.Join(bpDir, "fixtures", "fake_supply_python_app"))
			app.Buildpacks = []string{"multi_buildpack"}
		})

		It("finds the supplied dependency in the runtime container", func() {
			PushAppAndConfirm(app)
			Expect(app.Stdout.String()).To(ContainSubstring("Supplying Dotnet Core"))
			Expect(app.GetBody("/")).To(MatchRegexp(`dotnet: \d+\.\d+\.\d+`))
		})
	})

	Context("an app is pushed multiple times", func() {
		var tmpDir string
		BeforeEach(func() {
			var err error
			tmpDir, err = cutlass.CopyFixture(filepath.Join(bpDir, "fixtures", "flask_git_req"))
			Expect(err).To(BeNil())
			app = cutlass.New(tmpDir)
			app.Buildpacks = []string{"multi_buildpack"}
		})
		AfterEach(func() { os.RemoveAll(tmpDir) })

		It("pushes successfully both times", func() {
			libbuildpack.NewYAML().Write(filepath.Join(tmpDir, "multi-buildpack.yml"), map[string][]string{
				"buildpacks": []string{
					"https://buildpacks.cloudfoundry.org/fixtures/supply-cache-new.zip",
					"https://github.com/cloudfoundry/python-buildpack#master",
				},
			})
			PushAppAndConfirm(app)
			Expect(app.GetBody("/")).To(ContainSubstring("Hello, World!"))

			libbuildpack.NewYAML().Write(filepath.Join(tmpDir, "multi-buildpack.yml"), map[string][]string{
				"buildpacks": []string{
					"https://github.com/cloudfoundry/binary-buildpack",
					"https://buildpacks.cloudfoundry.org/fixtures/supply-cache-new.zip",
					"https://github.com/cloudfoundry/python-buildpack#master",
				},
			})
			PushAppAndConfirm(app)
			Expect(app.GetBody("/")).To(ContainSubstring("Hello, World!"))
		})
	})

	Context("the app uses miniconda", func() {
		BeforeEach(func() {
			app = cutlass.New(filepath.Join(bpDir, "fixtures", "miniconda_python_3"))
			app.Buildpacks = []string{"multi_buildpack"}
			app.Memory = "1GB"
			app.Disk = "2GB"
		})

		It("uses miniconda", func() {
			PushAppAndConfirm(app)
			Expect(app.Stdout.String()).To(ContainSubstring("scipy"))

			body, err := app.GetBody("/")
			Expect(err).To(BeNil())
			Expect(body).To(ContainSubstring("numpy: 1.10.4"))
			Expect(body).To(ContainSubstring("scipy: 0.17.0"))
			Expect(body).To(ContainSubstring("sklearn: 0.17.1"))
			Expect(body).To(ContainSubstring("pandas: 0.18.0"))
			Expect(body).To(ContainSubstring("python-version3"))
		})
	})
})
