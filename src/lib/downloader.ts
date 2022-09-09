import path from 'path';
import Downloader from 'nodejs-file-downloader';

export const downloadFile = async (url: string, destination: string) => {
  // @ts-ignore
  const downloader = new Downloader({
    url: url,
    directory: path.dirname(destination),
    filename: path.basename(destination),
    cloneFiles: false
  });
  try {
    await downloader.download();
  } catch (error) {
    throw new Error(`Error downloading file: ${error}`);
  }
};
