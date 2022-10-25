import path from 'path';
import Downloader from 'nodejs-file-downloader';
import { DownloadFailedException } from '../errors/DownloadFailedException.js';

export const downloadFile = async (url: string, destination: string) => {
  // eslint-disable-next-line @typescript-eslint/ban-ts-comment
  // @ts-ignore
  const downloader = new Downloader({
    url: url,
    directory: path.dirname(destination),
    filename: path.basename(destination),
    cloneFiles: false,
    maxAttempts: 3
  });
  try {
    await downloader.download();
  } catch (_) {
    throw new DownloadFailedException(url);
    // TODO handle failed download
  }
};
