// $: << 'cf_spec'
// require 'yaml'
// require 'spec_helper'
//
// describe 'running supply go buildpack before the ruby buildpack' do
//   let(:buildpack) { ENV.fetch('SHARED_HOST')=='true' ? 'multi_buildpack' : 'multi-test-buildpack' }
//   let(:app) { Machete.deploy_app(app_name, buildpack: buildpack) }
//   let(:browser) { Machete::Browser.new(app) }
//
//   after { Machete::CF::DeleteApp.new.execute(app) }
//
//   context 'the app is pushed' do
//     let (:app_name) { 'rails5' }
//
//     it 'finds the supplied dependency in the runtime container' do
//       expect(app).to be_running
//       expect(app).to have_logged "Multi Buildpack version"
//       expect(app).to have_logged "Nodejs Buildpack version"
//       expect(app).to have_logged "Installing node 8."
//
//       browser.visit_path('/')
//       expect(browser).to have_body(/Ruby version: ruby 2\.\d+\.\d+/)
//       expect(browser).to have_body(/Node version: v8\.\d+\.\d+/)
//       expect(browser).to have_body(/\/home\/vcap\/deps\/0\/node/)
//       expect(app).to have_logged "Skipping install of nodejs since it has been supplied"
//     end
//   end
// end
