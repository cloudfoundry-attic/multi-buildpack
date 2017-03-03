$: << 'cf_spec'
require 'spec_helper'

describe 'running fake-buildpack supply before the ruby buildpack' do
  let(:buildpack) { ENV.fetch('SHARED_HOST')=='true' ? 'multi_buildpack' : 'multi-test-buildpack' }
  let(:app) { Machete.deploy_app(app_name, buildpack: buildpack) }
  let(:app_name) { 'fake_supply_ruby_app' }
  let(:browser) { Machete::Browser.new(app) }

  subject(:app) { Machete.deploy_app(app_name) }

  after { Machete::CF::DeleteApp.new.execute(app) }

  it 'finds the supplied "dependency" in the runtime container' do
    expect(app).to be_running
    expect(app).to have_logged "SUPPLYING"

    browser.visit_path('/')
    expect(browser).to have_body('always-detects-buildpack')
  end
end
