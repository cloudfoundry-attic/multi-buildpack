require 'fileutils'
require 'digest'
require 'yaml'

class MultiBuildpackStager

  attr_reader :build_dir, :buildpack_downloads_dir

  def initialize(build_dir, cache_dir)
    @build_dir = build_dir
    @cache_dir = cache_dir
    @buildpack_downloads_dir = File.join(build_dir, "multi-buildpack-downloads-#{Random.rand(1000000)}")
    FileUtils.mkdir_p(buildpack_downloads_dir)
  end

  def cache_dir(buildpack = nil)
    return @cache_dir unless buildpack
    File.expand_path File.join(@cache_dir, Digest::MD5.hexdigest(buildpack))
  end

  def buildpacks
    multi_buildpack_file = File.join(build_dir, 'multi-buildpack.yml')

    unless File.exist?(multi_buildpack_file)
      error_message = "A multi-buildpack.yml file must be provided at your app root to use this buildpack."
      raise error_message
    end

    YAML.load_file(multi_buildpack_file)['buildpacks']
  end

  def run_builder(buildpack)
    puts "-----> Running builder for buildpack #{buildpack}..."
    FileUtils.mkdir_p(cache_dir(buildpack))

    system(
      "/tmp/lifecycle/builder",
      "--skipDetect=true",
      "--buildpacksDir=#{buildpack_downloads_dir}",
      "--buildpackOrder=#{buildpack}",
      "--outputDroplet=/dev/null",
      "--buildDir=#{build_dir}",
      "--buildArtifactsCacheDir=#{cache_dir(buildpack)}"
    )
  end

  def run!
    cleanup!
    buildpacks.each do |buildpack|
      run_builder(buildpack)
    end

    puts "-----> Removing buildpack downloads directory #{buildpack_downloads_dir}..."
    FileUtils.rm_rf(buildpack_downloads_dir)
  end

  private

  def cleanup!
    cache_dirs = Dir.entries(cache_dir).reject { |d| %w(. ..).include? d }.map { |d| File.expand_path File.join(cache_dir, d) }
    buildpack_cache_dirs = buildpacks.map { |b| cache_dir(b) }

    (cache_dirs - buildpack_cache_dirs).each do |dir|
      FileUtils.rm_rf dir
    end
  end
end
