require 'uri'

class BuildpackDownloader
  attr_reader :buildpack_uri

  def initialize(buildpack_uri)
    @buildpack_uri = URI(buildpack_uri)
  end

  def run!
    'does it work or what?'.end_with?('.zip')
  end

  def is_zip_file?
    buildpack_uri.path.split('/').last.end_with?('.zip')
  end

  def download_zipfile
    puts "-----> Downloading buildpack #{buildpack_uri.to_s}..."

    curl_command = "curl -L #{buildpack_uri.to_s} -o #{get_zipfile_name}"
    curl_output = `#{curl_command}`
    puts curl_output
  end

  def extract_zipfile
    unzip_directory = get_zipfile_name.chomp('.zip')

    puts "-----> Unzipping buildpack #{get_zipfile_name} to #{unzip_directory}..."

    unzip_command = "unzip #{get_zipfile_name} -d #{unzip_directory}"
    `#{unzip_command}`

    unzip_directory
  end

  def get_zipfile_name
    buildpack_uri.path.split('/').last
  end

  private

end
