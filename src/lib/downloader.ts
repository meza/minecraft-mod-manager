import path from 'path';
import Downloader from 'nodejs-file-downloader';
import { DownloadFailedException } from '../errors/DownloadFailedException.js';

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
  } catch (_) {
    throw new DownloadFailedException(url);
  }
};
