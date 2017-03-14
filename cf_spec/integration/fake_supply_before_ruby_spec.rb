$: << 'cf_spec'
require 'yaml'
require 'spec_helper'

describe 'running supply buildpacks before the ruby buildpack' do
  let(:buildpack) { ENV.fetch('SHARED_HOST')=='true' ? 'multi_buildpack' : 'multi-test-buildpack' }
  let(:app) { Machete.deploy_app(app_name, buildpack: buildpack) }
  let(:browser) { Machete::Browser.new(app) }

  after { Machete::CF::DeleteApp.new.execute(app) }

  context 'the app is pushed once' do
    let (:app_name) { 'fake_supply_ruby_app' }

    it 'finds the supplied "dependency" in the runtime container' do
      expect(app).to be_running
      expect(app).to have_logged "SUPPLYING"

      browser.visit_path('/')
      expect(browser).to have_body('always-detects-buildpack')
    end
  end

  context 'the app is pushed  multiple times' do
    let(:app_name) { 'test_cache_ruby_app' }
    let(:rand) { String(Random.rand(10000000)) }

    before do
      File.write("cf_spec/fixtures/test_cache_ruby_app/RANDOM_NUMBER", rand)
      multi_buildpack = {'buildpacks' => buildpacks }
      File.write("cf_spec/fixtures/test_cache_ruby_app/multi-buildpack.yml", multi_buildpack.to_yaml)
    end

    after do
      FileUtils.rm_rf("cf_spec/fixtures/test_cache_ruby_app/RANDOM_NUMBER")
      FileUtils.rm_rf("cf_spec/fixtures/test_cache_ruby_app/multi-buildpack.yml")
    end

    context "with the same buildpacks" do
      let(:buildpacks) {["https://buildpacks.cloudfoundry.org/fixtures/supply-cache.zip", "https://github.com/cloudfoundry/ruby-buildpack"]}

      it 'uses the cached files' do
        expect(app).to be_running
        expect(app).to have_logged "SUPPLYING"

        browser.visit_path('/')
        expect(browser).to have_body(rand)

        File.write("cf_spec/fixtures/test_cache_ruby_app/RANDOM_NUMBER", "not a number")

        Machete.push(app)
        expect(app).to be_running

        browser.visit_path('/')
        expect(browser).to have_body(rand)
      end
    end

    context "with different non-final buildpacks" do
      let(:buildpacks) {["https://buildpacks.cloudfoundry.org/fixtures/supply-cache.zip", "https://buildpacks.cloudfoundry.org/fixtures/num-cache-dirs.zip", "https://github.com/cloudfoundry/ruby-buildpack"]}

      it 'removes the unused cache dir' do
        expect(app).to be_running
        expect(app).to have_logged "THERE ARE 3 CACHE DIRS"

        browser.visit_path('/')
        expect(browser).to have_body("#{rand}supply2")

        multi_buildpack = {'buildpacks' => (buildpacks - ["https://buildpacks.cloudfoundry.org/fixtures/supply-cache.zip"] )}
        File.write("cf_spec/fixtures/test_cache_ruby_app/multi-buildpack.yml", multi_buildpack.to_yaml)

        Machete.push(app)
        expect(app).to be_running
        expect(app).to have_logged "THERE ARE 2 CACHE DIRS"

        browser.visit_path('/')
        expect(browser).to have_body("supply2")
      end
    end
  end
end
