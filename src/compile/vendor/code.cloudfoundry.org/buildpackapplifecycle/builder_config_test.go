package buildpackapplifecycle_test

import (
	"code.cloudfoundry.org/buildpackapplifecycle"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("LifecycleBuilderConfig", func() {
	var builderConfig buildpackapplifecycle.LifecycleBuilderConfig
	var skipDetect bool

	BeforeEach(func() {
		skipDetect = false
	})

	JustBeforeEach(func() {
		builderConfig = buildpackapplifecycle.NewLifecycleBuilderConfig([]string{"ocaml-buildpack", "haskell-buildpack", "bash-buildpack"}, skipDetect, false)
	})

	Context("with defaults", func() {
		It("generates a script for running its builder", func() {
			commandFlags := []string{
				"-buildDir=/tmp/app",
				"-buildpackOrder=ocaml-buildpack,haskell-buildpack,bash-buildpack",
				"-buildpacksDir=/tmp/buildpacks",
				"-buildpacksDownloadDir=/tmp/buildpackdownloads",
				"-buildArtifactsCacheDir=/tmp/cache",
				"-outputDroplet=/tmp/droplet",
				"-outputMetadata=/tmp/result.json",
				"-outputBuildArtifactsCache=/tmp/output-cache",
				"-skipCertVerify=false",
				"-skipDetect=false",
			}

			Expect(builderConfig.Path()).To(Equal("/tmp/lifecycle/builder"))
			Expect(builderConfig.Args()).To(ConsistOf(commandFlags))
		})

		It("returns the path to the app bits", func() {
			Expect(builderConfig.BuildDir()).To(Equal("/tmp/app"))
		})

	})

	Context("with overrides", func() {
		BeforeEach(func() {
			skipDetect = true
		})

		JustBeforeEach(func() {
			builderConfig.Set("buildDir", "/some/build/dir")
			builderConfig.Set("outputDroplet", "/some/droplet")
			builderConfig.Set("outputMetadata", "/some/result/dir")
			builderConfig.Set("buildpacksDir", "/some/buildpacks/dir")
			builderConfig.Set("buildpacksDownloadDir", "/some/downloads/dir")
			builderConfig.Set("buildArtifactsCacheDir", "/some/cache/dir")
			builderConfig.Set("outputBuildArtifactsCache", "/some/cache-file")
			builderConfig.Set("skipCertVerify", "true")
			builderConfig.Set("skipDetect", "true")
		})

		It("generates a script for running its builder", func() {
			commandFlags := []string{
				"-buildDir=/some/build/dir",
				"-buildpackOrder=ocaml-buildpack,haskell-buildpack,bash-buildpack",
				"-buildpacksDir=/some/buildpacks/dir",
				"-buildpacksDownloadDir=/some/downloads/dir",
				"-buildArtifactsCacheDir=/some/cache/dir",
				"-outputDroplet=/some/droplet",
				"-outputMetadata=/some/result/dir",
				"-outputBuildArtifactsCache=/some/cache-file",
				"-skipCertVerify=true",
				"-skipDetect=true",
			}

			Expect(builderConfig.Path()).To(Equal("/tmp/lifecycle/builder"))
			Expect(builderConfig.Args()).To(ConsistOf(commandFlags))
		})

		It("returns the path to the app bits", func() {
			Expect(builderConfig.BuildDir()).To(Equal("/some/build/dir"))
		})
	})

	It("returns the path to a given system buildpack", func() {
		key := "my-buildpack/key/::"
		Expect(builderConfig.BuildpackPath(key)).To(Equal("/tmp/buildpacks/8b2f72a0702aed614f8b5d8f7f5b431b"))
	})

	It("returns the path to a given downloaded buildpack", func() {
		key := "https://github.com/cloudfoundry/ruby-buildpack"
		Expect(builderConfig.BuildpackPath(key)).To(Equal("/tmp/buildpackdownloads/21de62d118ecb1f46d868d24f00839ef"))
	})

	It("returns the path to the staging metadata", func() {
		Expect(builderConfig.OutputMetadata()).To(Equal("/tmp/result.json"))
	})
})
