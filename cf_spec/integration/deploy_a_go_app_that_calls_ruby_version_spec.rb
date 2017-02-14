$: << 'cf_spec'
require 'spec_helper'

describe 'A go app that calls ruby version' do
  let(:buildpack) { ENV.fetch('SHARED_HOST')=='true' ? 'multi_buildpack' : 'multi-test-buildpack' }
  let(:app) { Machete.deploy_app(app_name, buildpack: buildpack) }
  let(:app_name) { 'go_app_that_calls_ruby_version' }
  let(:browser) { Machete::Browser.new(app) }

  subject(:app) { Machete.deploy_app(app_name) }

  after { Machete::CF::DeleteApp.new.execute(app) }

  it 'reports the ruby version specified in the Gemfile' do
    expect(app).to be_running

    browser.visit_path('/')
    expect(browser).to have_body(/The ruby version is: ruby 2\.3\.1/)
  end
end
