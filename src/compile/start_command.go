package main

import (
	"encoding/json"
	"io/ioutil"

	"code.cloudfoundry.org/buildpackapplifecycle/buildpackrunner"
	"github.com/cloudfoundry/libbuildpack"
)

func WriteStartCommand(stagingInfoFile string, outputFile string) error {
	var stagingInfo buildpackrunner.DeaStagingInfo

	stagingData, err := ioutil.ReadFile(stagingInfoFile)
	if err != nil {
		return err
	}

	err = json.Unmarshal(stagingData, &stagingInfo)
	if err != nil {
		return err
	}

	var webStartCommand = map[string]string{
		"web": stagingInfo.StartCommand,
	}

	release := buildpackrunner.Release{
		DefaultProcessTypes: webStartCommand,
	}

	return libbuildpack.NewYAML().Write(outputFile, &release)
}
