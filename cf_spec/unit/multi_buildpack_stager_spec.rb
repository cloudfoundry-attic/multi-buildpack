$: << 'cf_spec'
require 'spec_helper'
require_relative '../../lib/multi_buildpack_stager'
require 'fileutils'
require 'tmpdir'
require 'digest'

describe MultiBuildpackStager do
  let!(:build_dir) { Dir.mktmpdir}
  let!(:cache_dir) { Dir.mktmpdir}
  after { FileUtils.rm_rf(build_dir); FileUtils.rm_rf(cache_dir) }

  let (:multi_buildpacks) do
    <<~MULTIBUILDPACKYAML
       buildpacks:
         - https://github.com/cloudfoundry/ruby-buildpack
         - https://github.com/cloudfoundry/go-buildpack
       MULTIBUILDPACKYAML
  end

  subject { described_class.new(build_dir, cache_dir) }

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

  describe '#run!' do
    before { Thread.current[:log] = StringIO.new }
    after { Thread.current[:log] = nil }

    let(:commands) { [] }
    before { allow(subject).to receive(:system) { |*args| commands << args; 'fake stdout' } }

    let(:ruby_md5) { Digest::MD5.hexdigest 'https://github.com/cloudfoundry/ruby-buildpack' }
    let(:go_md5) { Digest::MD5.hexdigest 'https://github.com/cloudfoundry/go-buildpack' }

    context 'with a multi-buildpack.yml file' do
      before do
        multi_buildpack_file = File.join(build_dir, 'multi-buildpack.yml')
        File.write(multi_buildpack_file, multi_buildpacks)
      end

      it 'calls lifecycle/builder with buildpack-specific cache_dir' do
        subject.run!
        expect(subject).to have_received(:system).twice

        cache_dirs = commands.map do |args|
          args.detect{|a|a.start_with? '--buildArtifactsCacheDir='}
        end

        expect(cache_dirs).to match ["--buildArtifactsCacheDir=#{cache_dir}/#{ruby_md5}", "--buildArtifactsCacheDir=#{cache_dir}/#{go_md5}"]
      end

      it 'removes any cache_dirs that are no longer required' do
        FileUtils.mkdir_p File.join(cache_dir, 'extra_dir')

        subject.run!

        cache_dirs = Dir.entries(cache_dir).reject { |d| %w(. ..).include? d }
        expect(cache_dirs).to match [go_md5, ruby_md5]
      end

      it 'leaves file in cache_dirs across runs' do
        FileUtils.mkdir_p File.join(cache_dir, ruby_md5)
        File.write(File.join(cache_dir, ruby_md5, 'a_file.txt'), 'some text')

        subject.run!

        expect(File.read(File.join(cache_dir, ruby_md5, 'a_file.txt'))).to eq 'some text'
      end
    end
  end
end
