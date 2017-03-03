require 'sinatra'
require 'yaml'

get '/' do
  supplied_file = Dir["#{ENV['HOME']}/../deps/*/supplied"].first

  File.read(supplied_file).strip
end
