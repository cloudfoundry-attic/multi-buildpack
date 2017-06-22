# Cloud Foundry Experimental Multi-buildpack

[![CF Slack](https://www.google.com/s2/favicons?domain=www.slack.com) Join us on Slack](https://cloudfoundry.slack.com/messages/buildpacks/)

This buildpack allows you to run multiple buildpacks in a single staging container.

## Usage

- This buildpack looks for a `multi-buildpack.yml` file in the root of the application directory with structure:

```yaml
buildpacks:
  - https://github.com/cloudfoundry/go-buildpack
  - https://github.com/cloudfoundry/ruby-buildpack/releases/download/v1.6.23/ruby_buildpack-cached-v1.6.23.zip
  - https://github.com/cloudfoundry/nodejs-buildpack#v1.5.18
  - https://github.com/cloudfoundry/python-buildpack#develop
```

- The multi-buildpack will download + run all the buildpacks in this list in the specified order.

- It will use the app start command given by the last buildpack run.

## Details

- This will not work with system buildpacks. Ex. the following `multi-buildpack.yml` file will not work:

```yaml
buildpacks:
  - ruby_buildpack
  - go_buildpack
```

- The multi-buildpack will run the `bin/compile` and `bin/release` scripts for each specified buildpack.

### Testing
Buildpacks use the [Machete](https://github.com/cloudfoundry/machete) framework for running integration tests.

To test a buildpack, run the following command from the buildpack's directory:

```
BUNDLE_GEMFILE=cf.Gemfile bundle exec buildpack-build
```

More options can be found on Machete's [Github page.](https://github.com/cloudfoundry/machete)

### Contributing

Find our guidelines [here](./CONTRIBUTING.md).

### Help and Support

Join the #buildpacks channel in our [Slack community](http://slack.cloudfoundry.org/) if you need any further assistance.

### Reporting Issues

Please fill out the issue template fully if you'd like to start an issue for the buildpack.

### Active Development

The project backlog is on [Pivotal Tracker](https://www.pivotaltracker.com/projects/1042066)

## Disclaimer

This buildpack is intended as a proof-of-concept to generate user feedback for first class multi-buildpack support.
It is not intended for production usage.
