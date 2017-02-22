package main

import (
	"os"
	"path/filepath"

	"github.com/cloudfoundry/libbuildpack"
)

// Config is a struct to parse multi-buildpack.yml
type MultiBuildpackMetadata struct {
	Buildpacks []string `yaml:"buildpacks"`
}

// NewConfig returns parsed config object
func GetBuildpacks(dir string, logger libbuildpack.Logger) ([]string, error) {
	metadata := &MultiBuildpackMetadata{}

	err := libbuildpack.NewYAML().Load(filepath.Join(dir, "multi-buildpack.yml"), metadata)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Error("A multi-buildpack.yml file must be provided at your app root to use this buildpack.")
		} else {
			logger.Error("The multi-buildpack.yml file is malformed.")
		}
		return nil, err
	}

	return metadata.Buildpacks, nil
}
