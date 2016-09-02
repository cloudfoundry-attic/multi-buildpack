require 'fileutils'
require 'yaml'
require 'uri'
require_relative 'buildpack_downloader'

class MultiBuildpackStager

  attr_reader :build_dir, :cache_dir, :buildpack_downloads_dir

  def initialize(build_dir, cache_dir)
    @build_dir = build_dir
    @cache_dir = cache_dir
    @buildpack_downloads_dir = File.join(build_dir, "multibuildpack-downloads-#{Random.rand(1000000)}")
    FileUtils.mkdir_p(buildpack_downloads_dir)
  end

  def get_buildpacks
    multi_buildpack_file = File.join(build_dir, 'multibuildpack.yml')
    buildpack_uris = YAML.load_file(multi_buildpack_file)['buildpacks']
    download_buildpacks(buildpack_uris)
  end

  def download_buildpacks(buildpack_uris)
    buildpacks = []

    Dir.chdir(buildpack_downloads_dir) do
      buildpack_uris.each do |buildpack_uri|

        downloader = BuildpackDownloader.new(buildpack_uri)
        buildpack = downloader.run!
        buildpacks.push(buildpack)

      end
    end

    buildpacks
  end

  def clone_buildpack_repo(git_url)
    puts "-----> Cloning buildpack #{git_url}..."

    buildpack_name = git_url.path.split('/').last
    git_branch = git_url.fragment
    git_url.fragment = nil

    git_clone_command = "git clone --depth 1 --recursive #{git_url.to_s} #{buildpack_name}"
    git_clone_command += " --branch #{git_branch}" unless git_branch.nil?

    puts git_clone_command
    puts `#{git_clone_command}`

    buildpack_name
  end

  def run_compile(buildpack)
    puts "-----> Running compile for buildpack #{buildpack}..."

    buildpack_dir = File.join(buildpack_downloads_dir, buildpack)

    Dir.chdir(buildpack_dir) do
      compile_command = "bin/compile #{build_dir} #{cache_dir}"
      compile_output = `#{compile_command}`
      puts compile_output
    end
  end

  def write_to_release_file(buildpack)
    puts "-----> Running release for buildpack #{buildpack}..."

    buildpack_dir = File.join(buildpack_downloads_dir, buildpack)
    release_output = ''

    Dir.chdir(buildpack_dir) do
      release_command = "bin/release #{build_dir}"
      release_output = `#{release_command}`
    end

    release_output_file = File.join(build_dir, 'last_pack_release.out')
    File.write(release_output_file, release_output)
  end

  def run!
    buildpacks = get_buildpacks

    buildpacks.each do |buildpack|
      run_compile(buildpack)
    end

    write_to_release_file(buildpacks.last)

    puts "-----> Removing buildpack downloads directory #{buildpack_downloads_dir}..."
    FileUtils.rm_rf(buildpack_downloads_dir)
  end
end
