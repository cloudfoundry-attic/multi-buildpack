$: << 'cf_spec'
require 'yaml'
require 'spec_helper'

describe 'running supply buildpacks before the python buildpack' do
  let(:buildpack) { ENV.fetch('SHARED_HOST')=='true' ? 'multi_buildpack' : 'multi-test-buildpack' }
  let(:app) { Machete.deploy_app(app_name, buildpack: buildpack) }
  let(:browser) { Machete::Browser.new(app) }

  after { Machete::CF::DeleteApp.new.execute(app) }

  context 'a simple app is pushed once' do
    let (:app_name) { 'fake_supply_python_app' }

    it 'finds the supplied dependency in the runtime container' do
      expect(app).to be_running
      expect(app).to have_logged "SUPPLYING DOTNET"

      browser.visit_path('/')
      expect(browser).to have_body(/dotnet: 1.0.1/)
    end
  end

  context 'an app is pushed multiple times' do
    let(:buildpacks) {["https://buildpacks.cloudfoundry.org/fixtures/supply-cache-new.zip", "https://github.com/cloudfoundry/python-buildpack#develop"]}

    before do
      multi_buildpack = {'buildpacks' => buildpacks }
      File.write("cf_spec/fixtures/#{app_name}/multi-buildpack.yml", multi_buildpack.to_yaml)
    end

    after do
      FileUtils.rm_rf("cf_spec/fixtures/#{app_name}/multi-buildpack.yml")
    end

    context 'the app has a git dependency in requirements.txt' do
      let(:app_name)   { "flask_git_req"}

      it 'pushes successfully both times' do
        expect(app).to be_running

        browser.visit_path('/')
        expect(browser).to have_body('Hello, World!')

        multi_buildpack = {'buildpacks' => ( ["https://github.com/cloudfoundry/binary-buildpack"] + buildpacks )}
        File.write("cf_spec/fixtures/#{app_name}/multi-buildpack.yml", multi_buildpack.to_yaml)

        Machete.push(app)
        expect(app).to be_running

        browser.visit_path('/')
        expect(browser).to have_body('Hello, World!')
      end

    end

    context 'the app uses miniconda' do
      let(:app_name)   { "miniconda_python_3"}

      it 'uses the miniconda cache for the second push' do
        expect(app).to be_running(120)

        # Check that scipy was installed in the logs
        expect(app).to have_logged("scipy")

        browser.visit_path('/')
        expect(browser).to have_body('numpy: 1.10.4')
        expect(browser).to have_body('scipy: 0.17.0')
        expect(browser).to have_body('sklearn: 0.17.1')
        expect(browser).to have_body('pandas: 0.18.0')
        expect(browser).to have_body('python-version3')


        multi_buildpack = {'buildpacks' => ( ["https://github.com/cloudfoundry/binary-buildpack"] + buildpacks )}
        File.write("cf_spec/fixtures/#{app_name}/multi-buildpack.yml", multi_buildpack.to_yaml)

        Machete.push(app)
        expect(app).to be_running(120)

        # Check that scipy was not re-installed in the logs
        expect(app).to_not have_logged("scipy")

        browser.visit_path('/')
        expect(browser).to have_body('numpy: 1.10.4')
        expect(browser).to have_body('scipy: 0.17.0')
        expect(browser).to have_body('sklearn: 0.17.1')
        expect(browser).to have_body('pandas: 0.18.0')
        expect(browser).to have_body('python-version3')
      end
    end
  end
end
