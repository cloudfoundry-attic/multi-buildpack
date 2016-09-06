$: << 'cf_spec'
require 'spec_helper'
require_relative '../../lib/multi_buildpack_stager'
require 'fileutils'
require 'tmpdir'

describe MultiBuildpackStager do
  let (:build_dir) { Dir.mktmpdir}
  let (:cache_dir) { Dir.mktmpdir}

  let (:multi_buildpacks) do
    <<~MULTIBUILDPACKYAML
       buildpacks:
         - https://github.com/cloudfoundry/ruby-buildpack
         - https://github.com/cloudfoundry/go-buildpack
       MULTIBUILDPACKYAML
  end

  subject { described_class.new(build_dir, cache_dir) }

  after do
    FileUtils.rm_rf(build_dir)
    FileUtils.rm_rf(cache_dir)
  end

  describe '#buildpacks' do
    context 'with a multi-buildpack.yml file' do
      before do
        multi_buildpack_file = File.join(build_dir, 'multi-buildpack.yml')
        File.write(multi_buildpack_file, multi_buildpacks)
      end

      it 'returns an array of buildpacks' do
        expect(subject.buildpacks).to include('https://github.com/cloudfoundry/ruby-buildpack')
        expect(subject.buildpacks).to include('https://github.com/cloudfoundry/go-buildpack')
      end
    end

    context 'without a multi-buildpack.yml file' do
      it 'reports a helpful error message' do
        error_message = "A multi-buildpack.yml file must be provided at your app root to use this buildpack."
        expect { subject.buildpacks }.to raise_error(StandardError, error_message)
      end
    end
  end
end
