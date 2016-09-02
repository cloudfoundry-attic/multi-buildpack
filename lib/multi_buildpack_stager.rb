require 'fileutils'
require 'yaml'
require 'uri'

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
    buildpack_urls = YAML.load_file(multi_buildpack_file)['buildpacks']
    download_buildpacks(buildpack_urls)
  end

  def download_buildpacks(buildpack_urls)
    buildpacks = []

    Dir.chdir(buildpack_downloads_dir) do
      buildpack_urls.each do |buildpack_url|

        parsed_uri = URI(buildpack_url)

        if is_zip_file(parsed_uri)
          buildpacks.push(download_zip_file(parsed_uri))
        else
          buildpacks.push(clone_buildpack_repo(parsed_uri))
        end

      end
    end

    buildpacks
  end

  def is_zip_file(uri)
    uri.path.split('/').last.end_with?('.zip')
  end

  def download_zip_file(uri)
    puts "-----> Downloading buildpack #{uri.to_s}..."

    output_file = uri.path.split('/').last
    unzip_directory = output_file.chomp('.zip')

    curl_command = "curl -L #{uri.to_s} -o #{output_file}"
    curl_output = `#{curl_command}`
    puts curl_output

    puts "-----> Unzipping buildpack #{output_file} to #{unzip_directory}..."
    unzip_command = "unzip #{output_file} -d #{unzip_directory}"
    `#{unzip_command}`

    unzip_directory
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
