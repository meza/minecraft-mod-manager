import { afterEach, describe, expect, it, vi } from 'vitest';
import { chance } from 'jest-chance';
import path from 'node:path';
import { default as Downloader } from 'nodejs-file-downloader';
import { downloadFile } from './downloader.js';
import { DownloadFailedException } from '../errors/DownloadFailedException.js';

vi.mock('nodejs-file-downloader');

describe('The downloader facade', () => {
  afterEach(() => {
    vi.resetAllMocks();
  });

  it('should invoke the downloader lib as expected', async () => {
    const url = chance.url();
    const destination = path.resolve(chance.word());

    // @ts-ignore
    vi.mocked(Downloader).mockImplementationOnce(() => ({
      download: vi.fn().mockResolvedValueOnce({ downloadStatus: 'COMPLETE', filePath: destination }),
      cancel: vi.fn()
    }));

    expect(async () => {
      await downloadFile(url, destination);
    }).not.toThrow();

    expect(vi.mocked(Downloader)).toHaveBeenCalledOnce();
    expect(vi.mocked(Downloader)).toHaveBeenCalledWith({
      url: url,
      directory: path.dirname(destination),
      filename: path.basename(destination),
      cloneFiles: false
    });

  });

  it('should throw an error if the download fails', async () => {
    const url = chance.url();
    const destination = path.resolve(chance.word());

    // @ts-ignore
    vi.mocked(Downloader).mockImplementationOnce(() => ({
      download: vi.fn().mockRejectedValueOnce(new Error('Download failed')),
      cancel: vi.fn()
    }));

    await expect(async () => {
      await downloadFile(url, destination);
    }).rejects.toThrow(new DownloadFailedException(url));

  });
});
