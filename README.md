# Cloud Foundry Experimental Multi-buildpack

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

## Disclaimer

This buildpack is intended as a proof-of-concept to generate user feedback for first class multi-buildpack support.
It is not intended for production usage.
