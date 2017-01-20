require 'bundler/setup'
require 'machete'
require 'machete/matchers'

`mkdir -p log`
Machete.logger = Machete::Logger.new("log/integration.log")

RSpec.configure do |config|
  config.color = true
  config.tty = true

  config.filter_run_excluding :cached => ENV['BUILDPACK_MODE'] == 'uncached'
  config.filter_run_excluding :uncached => ENV['BUILDPACK_MODE'] == 'cached'
end

module Kernel
  @@io_semaphore = Mutex.new

  [ :printf, :p, :print, :puts ].each do |io_write|
    hidden_io_write = "__#{io_write}__"
    alias_method hidden_io_write, io_write

    define_method(io_write) do |*args|
      @@io_semaphore.synchronize do
        if log = Thread.current[:log]
          log.__send__(io_write, *args)
        else
          self.__send__(hidden_io_write, *args)
        end
      end
    end
  end
end
