require 'fileutils'
require 'yaml'

class MultiBuildpackStager

  attr_reader :build_dir, :cache_dir, :buildpack_downloads_dir

  def initialize(build_dir, cache_dir)
    @build_dir = build_dir
    @cache_dir = cache_dir
    @buildpack_downloads_dir = File.join(build_dir, "multi-buildpack-downloads-#{Random.rand(1000000)}")
    FileUtils.mkdir_p(buildpack_downloads_dir)
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

    compile_command = "/tmp/lifecycle/builder"
    compile_command += " --skipDetect=true --buildpacksDir=#{buildpack_downloads_dir}"
    compile_command += " --buildpackOrder=#{buildpack} --outputDroplet=/dev/null"
    compile_command += " --buildDir=#{build_dir} --buildArtifactsCacheDir=#{cache_dir}"

    compile_output = `#{compile_command}`
    puts compile_output
  end

  def run!
    buildpacks.each do |buildpack|
      run_builder(buildpack)
    end

    puts "-----> Removing buildpack downloads directory #{buildpack_downloads_dir}..."
    FileUtils.rm_rf(buildpack_downloads_dir)
  end
end
