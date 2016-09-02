require '../../lib/buildpack_downloader'

describe BuildpackDownloader do
  describe 'download zip file' do
    let(:zipfile_uri) { 'https://example.com/buildpacks/intercal-buildpack-v1.2.3.zip' }
    let(:downloader) { BuildpackDownloader.new(zipfile_uri) }

    subject { downloader.run! }

    describe '#is_zip_file?' do
      subject { downloader.is_zip_file? }

      context 'uri is for a zip file' do
        it 'gives true for a <some uri>.zip' do
          expect(subject).to eq true
        end
      end

      context 'uri is for something other than a zip file' do
        let(:zipfile_uri) { 'https://example.com/buildpacks/actually_is_not_a_buildpack_at_all.txt' }

        it 'gives false for a <some uri>.txt' do
          expect(subject).to eq false
        end
      end
    end

    describe '#get_zipfile_name' do
      subject { downloader.get_zipfile_name }

      it 'finds the zip file name from the URI' do
        expect(subject).to eq 'intercal-buildpack-v1.2.3.zip'
      end
    end

    describe '#download_zipfile' do
      subject { downloader.download_zipfile }
      before do
        allow(downloader).to receive(:`).and_return('hello from curl')
      end

      it 'uses curl to download the file' do
        expect(downloader).to receive(:`).with('curl -L https://example.com/buildpacks/intercal-buildpack-v1.2.3.zip -o intercal-buildpack-v1.2.3.zip')

        subject
      end

      it 'says it is downloading' do
        expect { subject }.to output(/-----> Downloading buildpack https:\/\/example.com\/buildpacks\/intercal-buildpack-v1\.2\.3\.zip\.\.\./).to_stdout
      end

      it 'shows the output from curl' do
        expect { subject }.to output(/hello from curl/).to_stdout
      end

    end

    describe '#extract_zipfile' do
      subject { downloader.extract_zipfile }
      before do
        allow(downloader).to receive(:`)
      end

      it 'tells us what is extracted to where' do
        expect { subject }.to output(/-----> Unzipping buildpack intercal-buildpack-v1\.2\.3\.zip to intercal-buildpack-v1\.2\.3\.\.\./).to_stdout
      end

      it 'uses unzip to extract the file' do
        expect(downloader).to receive(:`).with('unzip intercal-buildpack-v1.2.3.zip -d intercal-buildpack-v1.2.3')

        subject
      end

      it 'returns the directory it unzipped to' do
        expect(subject).to eq 'intercal-buildpack-v1.2.3'
      end
    end
  end

  describe 'download git repo'
end
