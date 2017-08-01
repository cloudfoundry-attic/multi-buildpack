// $: << 'cf_spec'
// require 'yaml'
// require 'spec_helper'
// require 'open3'
//
// describe 'running supply buildpacks before the ruby buildpack' do
//   let(:buildpack) { "multi-unpackaged-buildpack-#{rand(1000)}" }
//   let(:app) { Machete.deploy_app(app_name, buildpack: buildpack, skip_verify_version: true) }
//   let(:browser) { Machete::Browser.new(app) }
//
//   before do
//     buildpack_file = "/tmp/#{buildpack}.zip"
//     Open3.capture2e('zip','-r',buildpack_file,'bin/','src/','manifest.yml','VERSION')[1].success? or raise 'Could not create unpackaged buildpack zip file'
//     Open3.capture2e('cf', 'create-buildpack', buildpack, buildpack_file, '100', '--enable')[1].success? or raise 'Could not upload buildpack'
//     FileUtils.rm buildpack_file
//   end
//   after do
//     Machete::CF::DeleteApp.new.execute(app)
//     Open3.capture2e('cf', 'delete-buildpack', '-f', buildpack)
//   end
//
//   context 'the app is pushed' do
//     let (:app_name) { 'fake_supply_ruby_app' }
//
//     it 'finds the supplied dependency in the runtime container' do
//       expect(app).to be_running
//       expect(app).to have_logged(/Running go build compile/)
//       expect(app).to have_logged "SUPPLYING DOTNET"
//
//       browser.visit_path('/')
//       expect(browser).to have_body(/dotnet: 1.0.1/)
//     end
//   end
// end
