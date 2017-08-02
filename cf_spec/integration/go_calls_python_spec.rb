$: << 'cf_spec'
require 'yaml'
require 'spec_helper'

describe 'running supply python buildpack before the go buildpack' do
  let(:buildpack) { ENV.fetch('SHARED_HOST')=='true' ? 'multi_buildpack' : 'multi-test-buildpack' }
  let(:app) { Machete.deploy_app(app_name, buildpack: buildpack) }
  let(:browser) { Machete::Browser.new(app) }

  after { Machete::CF::DeleteApp.new.execute(app) }

  RSpec.shared_examples "go app can call out to a python hello world app" do
    it 'finds the supplied dependency in the runtime container' do
      expect(app).to be_running
      expect(app).to have_logged "Multi Buildpack version"
      expect(app).to have_logged expected_log_for_python

      browser.visit_path('/')
      expect(browser).to have_body(/\[\{"hello":"world"\}\]/)
    end
  end

  context 'an app is pushed which uses pip dependencies' do
    let (:app_name) { 'go_calls_python' }
    let (:expected_log_for_python) { 'Installing python-' }

    it_behaves_like "go app can call out to a python hello world app"
  end

  context 'an app is pushed which uses miniconda' do
    let (:app_name) { 'go_calls_python_miniconda' }
    let (:expected_log_for_python) { 'Installing Miniconda' }

    it_behaves_like "go app can call out to a python hello world app"
  end

  context 'an app is pushed which uses NLTK corpus' do
    let (:app_name) { 'go_calls_python_nltk' }

    it 'downloads the corpora and works as expected' do
      expect(app).to be_running
      expect(app).to have_logged "Multi Buildpack version"
      expect(app).to have_logged "Downloading NLTK corpora..."

      browser.visit_path('/')
      expect(browser).to have_body(/The Fulton County Grand Jury said Friday an investigation of Atlanta's recent primary election produced/)
    end
  end
end
