
 require 'yaml'
 require 'uri'


class MultiBuildpackStager

  attr_reader :build_dir, :cache_dir

  def initialize(build_dir, cache_dir)
    @build_dir = build_dir
    @cache_dir = cache_dir
  end

  def get_buildpack_urls
    multi_buildpack_file = File.join(build_dir, 'multibuildpack.yml')
    YAML.load_file(multi_buildpack_file)['buildpacks']
  end

  def clone_buildpack(git_url)
    puts "=====> Cloning buildpack #{git_url}..."

    parsed_uri = URI(git_url)

    buildpack_name = parsed_uri.path.split('/').last
    git_branch = parsed_uri.fragment

    parsed_uri.fragment = nil
    git_clone_url = parsed_uri.to_s

    git_clone_command = "git clone #{git_clone_url}"
    git_clone_command += " --branch #{git_branch}" unless git_branch.nil?

    puts git_clone_command
    puts `#{git_clone_command}`

    if File.exist? "#{buildpack_name}/.gitmodules"
      puts "=====> Detected git submodules. Initializing..."

      Dir.chdir(buildpack_name) do
        puts `git submodule update --init --recursive`
      end
    end
    buildpack_name
  end

  def run_compile(buildpack_dir)
    puts "=====> Running compile for buildpack #{buildpack_dir}..."

    Dir.chdir(buildpack_dir) do
      compile_command = "bin/compile #{build_dir} #{cache_dir}"
      puts `#{compile_command}`
    end
  end

  def write_to_release_file(buildpack_dir)
    puts "=====> Running release for buildpack #{buildpack_dir}..."

    release_output = ''

    Dir.chdir(buildpack_dir) do
      release_command = "bin/release #{build_dir}"
      release_output = `#{release_command}`
    end

    release_output_file  = File.join(build_dir, 'last_pack_release.out')
    File.write(release_output_file, release_output)
  end

  def run!
    buildpacks_to_run = get_buildpack_urls

    buildpack_names = buildpacks_to_run.map do |buildpack_url|
      buildpack_dir = clone_buildpack(buildpack_url)
      run_compile(buildpack_dir)
      buildpack_dir
    end

    write_to_release_file(buildpack_names.last)
  end
end

