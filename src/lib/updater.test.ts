import fs from 'node:fs/promises';
import path from 'node:path';
import { chance } from 'jest-chance';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { generateModInstall } from '../../test/modInstallGenerator.js';
import { downloadFile } from './downloader.js';
import { updateMod } from './updater.js';

vi.mock('node:fs/promises');
vi.mock('./downloader.js');

const assumeDownloadSuccessful = () => {
  vi.mocked(downloadFile).mockResolvedValueOnce();
};

describe('The updater module', () => {
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

    await updateMod(randomMod, originalPath, randomModsFolder);

    expect(vi.mocked(downloadFile)).toHaveBeenCalledWith(randomMod.downloadUrl, expectedNewPath);
    expect(vi.mocked(fs.rm)).toHaveBeenCalledWith(originalPath);
  });

  it('should safely update to the same file file', async () => {
    const randomMod = generateModInstall().generated;
    const randomModsFolder = chance.word();
    const originalPath = path.resolve(randomModsFolder, randomMod.fileName);
    const expectedNewPath = path.resolve(randomModsFolder, randomMod.fileName);

    assumeDownloadSuccessful();

    await updateMod(randomMod, originalPath, randomModsFolder);

    expect(vi.mocked(downloadFile)).toHaveBeenCalledWith(randomMod.downloadUrl, expectedNewPath);
    expect(vi.mocked(fs.rm)).not.toHaveBeenCalled();
  });
});
