# Cloud Foundry Multi-buildpack

[![CF Slack](https://www.google.com/s2/favicons?domain=www.slack.com) Join us on Slack](https://cloudfoundry.slack.com/messages/buildpacks/)

The multi-buildpack buildpack provides older Cloud Foundry deployments with the same multi-buildpack support that is available in Cloud Foundry's CAPI v3 API. See [Understanding Buildpacks](https://docs.cloudfoundry.org/buildpacks/understand-buildpacks.html) for more info.

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

- It will use the app start command given by the final buildpack (the last buildpack in your `multi-buildpack.yml`).

- The multi-buildpack buildpack will not work with system buildpacks. You must use URLs as shown above. Ex. the following `multi-buildpack.yml` file will **not** work:

```yaml
buildpacks:
  - ruby_buildpack
  - go_buildpack
```

### Testing

Buildpacks use the [Cutlass](https://github.com/cloudfoundry/libbuildpack/cutlass) framework for running integration tests.

To test this buildpack, run the following command from the buildpack's directory:

1. Source the .envrc file in the buildpack directory.

   ```bash
   source .envrc
   ```
   To simplify the process in the future, install [direnv](https://direnv.net/) which will automatically source .envrc when you change directories.

1. Run unit tests

    ```bash
    ./scripts/unit.sh
    ```

1. Run integration tests

    ```bash
    ./scripts/integration.sh
    ```

More information can be found on Github [cutlass](https://github.com/cloudfoundry/libbuildpack/cutlass).

### Contributing

Find our guidelines [here](./CONTRIBUTING.md).

### Help and Support

Join the #buildpacks channel in our [Slack community](http://slack.cloudfoundry.org/) if you need any further assistance.

### Reporting Issues

Please fill out the issue template fully if you'd like to start an issue for the buildpack.

### Active Development

The project backlog is on [Pivotal Tracker](https://www.pivotaltracker.com/projects/1042066)
