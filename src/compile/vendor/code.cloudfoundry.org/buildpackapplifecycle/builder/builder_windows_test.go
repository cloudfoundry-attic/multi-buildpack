package main_test

import (
	"os/exec"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var buildpackFixtures = filepath.Join("fixtures", "buildpacks", "windows")
var appFixtures = filepath.Join("fixtures", "apps")

func cp(src string, dst string) {
	session, err := gexec.Start(exec.Command("powershell", "-Command", "Copy-Item", "-Recurse", "-Force", src, dst),
		GinkgoWriter,
		GinkgoWriter)
	Expect(err).ToNot(HaveOccurred())

	Eventually(session).Should(gexec.Exit(0))
}
