package integration_test

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/cloudfoundry/libbuildpack"
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

	Context("the app is pushed once", func() {
		BeforeEach(func() {
			app = cutlass.New(filepath.Join(bpDir, "fixtures", "fake_supply_ruby_app"))
			app.Buildpacks = []string{"multi_buildpack"}
		})

		It("finds the supplied dependency in the runtime container", func() {
			PushAppAndConfirm(app)
			Expect(app.Stdout.String()).To(ContainSubstring("Multi Buildpack version"))
			Expect(app.Stdout.String()).To(ContainSubstring("SUPPLYING DOTNET"))
			Expect(app.GetBody("/")).To(ContainSubstring("dotnet: 1.0.1"))
		})
	})

	Context("an app is pushed multiple times", func() {
		var tmpDir, randomRunes string
		BeforeEach(func() {
			var err error
			tmpDir, err = cutlass.CopyFixture(filepath.Join(bpDir, "fixtures", "test_cache_ruby_app"))
			Expect(err).To(BeNil())
			app = cutlass.New(tmpDir)
			app.Buildpacks = []string{"multi_buildpack"}

			randomRunes = cutlass.RandStringRunes(32)
			Expect(ioutil.WriteFile(filepath.Join(tmpDir, "RANDOM_NUMBER"), []byte(randomRunes), 0644)).To(Succeed())
		})
		AfterEach(func() { os.RemoveAll(tmpDir) })

		It("pushes successfully both times with same buildpacks", func() {
			libbuildpack.NewYAML().Write(filepath.Join(tmpDir, "multi-buildpack.yml"), map[string][]string{
				"buildpacks": []string{
					"https://buildpacks.cloudfoundry.org/fixtures/supply-cache-new.zip",
					"https://github.com/cloudfoundry/ruby-buildpack",
				},
			})
			PushAppAndConfirm(app)
			Expect(app.GetBody("/")).To(ContainSubstring(randomRunes))

			Expect(ioutil.WriteFile(filepath.Join(tmpDir, "RANDOM_NUMBER"), []byte("some string"), 0644)).To(Succeed())
			PushAppAndConfirm(app)
			Expect(app.GetBody("/")).To(ContainSubstring(randomRunes))
		})

		It("pushes successfully both times with diffenent non-final buildpacks", func() {
			libbuildpack.NewYAML().Write(filepath.Join(tmpDir, "multi-buildpack.yml"), map[string][]string{
				"buildpacks": []string{
					"https://buildpacks.cloudfoundry.org/fixtures/supply-cache-new.zip",
					"https://buildpacks.cloudfoundry.org/fixtures/num-cache-new.zip",
					"https://github.com/cloudfoundry/ruby-buildpack",
				},
			})
			PushAppAndConfirm(app)
			Expect(app.GetBody("/")).To(ContainSubstring(randomRunes))
			Expect(app.Stdout.String()).To(ContainSubstring("THERE ARE 3 CACHE DIRS"))

			libbuildpack.NewYAML().Write(filepath.Join(tmpDir, "multi-buildpack.yml"), map[string][]string{
				"buildpacks": []string{
					"https://buildpacks.cloudfoundry.org/fixtures/num-cache-new.zip",
					"https://github.com/cloudfoundry/ruby-buildpack",
				},
			})
			PushAppAndConfirm(app)
			Expect(app.GetBody("/")).To(ContainSubstring("supply2"))
			Expect(app.Stdout.String()).To(ContainSubstring("THERE ARE 2 CACHE DIRS"))
		})
	})
})
