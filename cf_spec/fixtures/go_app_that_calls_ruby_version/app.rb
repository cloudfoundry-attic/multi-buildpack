require 'sinatra'

get '/' do
  "RUBY_VERSION IS #{RUBY_VERSION}, ruby -v is #{`ruby -v`}"
end
