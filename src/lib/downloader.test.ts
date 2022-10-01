import { afterEach, describe, it, vi, expect } from 'vitest';
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

    try {
      await downloadFile(url, destination);
      expect.fail('Expected the Download Failed Exception to be thrown');
    } catch (e) {
      expect(e).toBeInstanceOf(DownloadFailedException);
      expect((e as DownloadFailedException).message).toBe(`Error downloading file: ${url}`);
    }
  });
});
