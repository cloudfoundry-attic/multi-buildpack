package integration_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/cloudfoundry/libbuildpack/cutlass"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("running supply buildpacks before the ruby buildpack", func() {
	var app *cutlass.App
	AfterEach(func() {
		if app != nil {
			app.Destroy()
		}
		app = nil
	})

	runCmd := func(name string, args ...string) {
		cmd := exec.Command(name, args...)
		cmd.Dir = bpDir
		cmd.Stdout = GinkgoWriter
		cmd.Stderr = GinkgoWriter
		Expect(cmd.Run()).To(Succeed())
	}

	Context("the app is pushed", func() {
		var buildpackFile string
		BeforeEach(func() {
			app = cutlass.New(filepath.Join(bpDir, "fixtures", "fake_supply_ruby_app"))
			app.Buildpacks = []string{"multi-unpackaged-buildpack-" + cutlass.RandStringRunes(20)}

			buildpackFile = fmt.Sprintf("/tmp/%s.zip", app.Buildpacks[0])
			runCmd("zip", "-r", buildpackFile, "bin/", "src/", "scripts/", "manifest.yml", "VERSION")

			runCmd("cf", "create-buildpack", app.Buildpacks[0], buildpackFile, "100", "--enable")
		})
		AfterEach(func() {
			os.Remove(buildpackFile)
			runCmd("cf", "delete-buildpack", "-f", app.Buildpacks[0])
		})

		It("finds the supplied dependency in the runtime container", func() {
			Expect(app.Push()).To(Succeed())
			Eventually(func() ([]string, error) { return app.InstanceStates() }, 10*time.Second).Should(Equal([]string{"RUNNING"}))

			Expect(app.Stdout.String()).To(ContainSubstring("Running go build compile"))
			Expect(app.Stdout.String()).To(ContainSubstring("SUPPLYING DOTNET"))

			Expect(app.GetBody("/")).To(ContainSubstring("dotnet: 1.0.1"))
		})
	})
})
