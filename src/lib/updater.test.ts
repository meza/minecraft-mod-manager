import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { generateModInstall } from '../../test/modInstallGenerator.js';
import { chance } from 'jest-chance';
import path from 'node:path';
import { updateMod } from './updater.js';
import { downloadFile } from './downloader.js';
import fs from 'node:fs/promises';
import { Logger } from './Logger.js';

vi.mock('node:fs/promises');
vi.mock('./downloader.js');
vi.mock('./Logger.js');

const assumeDownloadSuccessful = () => {
  vi.mocked(downloadFile).mockResolvedValueOnce();
};

const assumeDownloadFailed = () => {
  vi.mocked(downloadFile).mockRejectedValueOnce({});
};

describe('The updater module', () => {
  let logger: Logger;

  beforeEach(() => {
    logger = new Logger({} as never);
  });
  afterEach(() => {
    vi.resetAllMocks();
  });

  it('should safely update to a new file', async () => {
    const randomMod = generateModInstall().generated;
    const randomModsFolder = chance.word();
    const randomOldPath = chance.word();
    const originalPath = path.resolve(randomModsFolder, randomOldPath);
    const expectedNewPath = path.resolve(randomModsFolder, randomMod.fileName);

    assumeDownloadSuccessful();

    await updateMod(randomMod, originalPath, randomModsFolder, logger);

    expect(vi.mocked(fs.rename)).toHaveBeenCalledWith(originalPath, `${originalPath}.bak`);
    expect(vi.mocked(downloadFile)).toHaveBeenCalledWith(randomMod.downloadUrl, expectedNewPath);
    expect(vi.mocked(fs.rm)).toHaveBeenCalledWith(`${originalPath}.bak`);

    expect(vi.mocked(fs.rename)).toHaveBeenCalledOnce();
    expect(vi.mocked(downloadFile)).toHaveBeenCalledOnce();
    expect(vi.mocked(fs.rm)).toHaveBeenCalledOnce();

  });

  it('should roll back on download failure', async () => {
    const randomMod = generateModInstall().generated;
    const randomModsFolder = chance.word();
    const randomOldPath = chance.word();
    const originalPath = path.resolve(randomModsFolder, randomOldPath);
    const expectedNewPath = path.resolve(randomModsFolder, randomMod.fileName);

    assumeDownloadFailed();

    await updateMod(randomMod, originalPath, randomModsFolder, logger);

    expect(vi.mocked(fs.rename)).toHaveBeenNthCalledWith(1, originalPath, `${originalPath}.bak`);
    expect(vi.mocked(downloadFile)).toHaveBeenCalledWith(randomMod.downloadUrl, expectedNewPath);
    expect(vi.mocked(fs.rename)).toHaveBeenNthCalledWith(2, `${originalPath}.bak`, originalPath);

    expect(logger.log).toHaveBeenCalledWith(`Download of ${randomMod.name} failed, restoring the original`);

    expect(vi.mocked(fs.rename)).toHaveBeenCalledTimes(2);
    expect(vi.mocked(downloadFile)).toHaveBeenCalledOnce();
    expect(vi.mocked(fs.rm)).not.toHaveBeenCalled();

  });

});
