import { beforeEach, describe, it, vi, expect } from 'vitest';
import { Logger } from '../lib/Logger.js';
import { prune, PruneOptions } from './prune.js';
import { ModInstall, ModsJson } from '../lib/modlist.types.js';
import { generateModsJson } from '../../test/modlistGenerator.js';
import { ensureConfiguration, readLockFile } from '../lib/config.js';
import { chance } from 'jest-chance';
import { fileIsManaged } from '../lib/configurationHelper.js';
import { shouldPruneFiles } from '../interactions/shouldPruneFiles.js';
import path from 'path';
import fs from 'fs/promises';
import { getModFiles } from '../lib/fileHelper.js';

interface LocalTestContext {
  logger: Logger;
  configuration: ModsJson;
  installations: ModInstall[];
  options: PruneOptions;
}

vi.mock('../lib/config.js');
vi.mock('../lib/Logger.js');
vi.mock('../lib/configurationHelper.js');
vi.mock('../interactions/shouldPruneFiles.js');
vi.mock('../lib/fileHelper.js');
vi.mock('fs/promises');

describe('The prune action', () => {
  beforeEach<LocalTestContext>((context) => {
    vi.resetAllMocks();
    context.logger = new Logger({} as never);
    context.configuration = generateModsJson().generated;
    context.installations = [];
    context.options = {
      debug: false,
      quiet: false,
      config: 'config.json',
      force: false
    };

    vi.mocked(ensureConfiguration).mockResolvedValueOnce(context.configuration);
    vi.mocked(readLockFile).mockResolvedValueOnce(context.installations);

  });

  it<LocalTestContext>('notifies about no files in the mods folder', async ({ options, logger }) => {
    vi.mocked(getModFiles).mockResolvedValueOnce([]);

    await prune(options, logger);

    const logMessage = vi.mocked(logger.log).mock.calls[0][0];

    expect(logMessage).toMatchInlineSnapshot('"You have no files in your mods folder."');
  });

  it<LocalTestContext>('notifies about no unmanaged files in the mods folder', async ({ options, logger }) => {
    vi.mocked(getModFiles).mockResolvedValueOnce(chance.n(chance.word, chance.integer({ min: 2, max: 10 })));
    vi.mocked(fileIsManaged).mockReturnValue(true);

    await prune(options, logger);

    const logMessage = vi.mocked(logger.log).mock.calls[0][0];

    expect(logMessage).toMatchInlineSnapshot('"You have no unmanaged mods in your mods folder."');
  });

  it<LocalTestContext>('deletes the files', async ({ options, logger, configuration }) => {
    const file1 = chance.word();
    const file2 = chance.word();
    const file3 = chance.word();

    const expectedFile1 = path.resolve(configuration.modsFolder, file1);
    const expectedFile2 = path.resolve(configuration.modsFolder, file2);
    const expectedFile3 = path.resolve(configuration.modsFolder, file3);

    vi.mocked(getModFiles).mockResolvedValueOnce([file1, file2, file3]);
    vi.mocked(fileIsManaged).mockReturnValue(false);
    vi.mocked(shouldPruneFiles).mockResolvedValueOnce(true);

    await prune(options, logger);

    expect(fs.rm).toHaveBeenCalledTimes(3);
    expect(fs.rm).toHaveBeenNthCalledWith(1, expectedFile1, { force: true });
    expect(fs.rm).toHaveBeenNthCalledWith(2, expectedFile2, { force: true });
    expect(fs.rm).toHaveBeenNthCalledWith(3, expectedFile3, { force: true });

    expect(vi.mocked(logger.log).mock.calls[0][0]).toContain('Deleted: ');
    expect(vi.mocked(logger.log).mock.calls[0][0]).toContain(expectedFile1);
    expect(vi.mocked(logger.log).mock.calls[1][0]).toContain('Deleted: ');
    expect(vi.mocked(logger.log).mock.calls[1][0]).toContain(expectedFile2);
    expect(vi.mocked(logger.log).mock.calls[2][0]).toContain('Deleted: ');
    expect(vi.mocked(logger.log).mock.calls[2][0]).toContain(expectedFile3);
  });

  it<LocalTestContext>('doesn\'t remove files if not asked to', async ({ options, logger }) => {
    const file1 = chance.word();
    const file2 = chance.word();
    const file3 = chance.word();

    vi.mocked(getModFiles).mockResolvedValueOnce([file1, file2, file3]);
    vi.mocked(fileIsManaged).mockReturnValue(false);
    vi.mocked(shouldPruneFiles).mockResolvedValueOnce(false);

    await prune(options, logger);

    expect(fs.rm).not.toHaveBeenCalled();
    expect(logger.log).not.toHaveBeenCalled();
  });
});
