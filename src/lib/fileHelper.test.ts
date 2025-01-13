import { BigIntStats, Stats } from 'node:fs';
import path from 'path';
import * as fs from 'fs/promises';
import { chance } from 'jest-chance';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { generateModsJson } from '../../test/modlistGenerator.js';
import { getModFiles } from './fileHelper.js';
import { notIgnored } from './ignore.js';
import { ModsJson } from './modlist.types.js';

vi.mock('./ignore.js');
vi.mock('fs/promises');

interface LocalTestContext {
  configLocation: string;
  rootDir: string;
  configuration: ModsJson;
}

describe('The file helper module', () => {
  beforeEach<LocalTestContext>((context) => {
    vi.resetAllMocks();
    context.rootDir = path.resolve('/', chance.word());
    context.configLocation = path.resolve(context.rootDir, chance.word());
    context.configuration = generateModsJson({ modsFolder: 'mods' }).generated;
  });

  it<LocalTestContext>('can handle a relative mods folder', async ({ configLocation, rootDir, configuration }) => {
    vi.mocked(fs.readdir).mockResolvedValueOnce([]);

    await getModFiles(configLocation, configuration);

    expect(fs.readdir).toHaveBeenCalledWith(path.resolve(rootDir, 'mods'));
  });

  it<LocalTestContext>('ignores all directories', async ({ configLocation, rootDir, configuration }) => {
    const foundFiles = chance.n(
      () => {
        return path.resolve(rootDir, 'mods', chance.word() + '.jar');
      },
      chance.integer({ min: 2, max: 20 })
    );
    const mockStatSync = vi.mocked(fs.stat);

    mockStatSync.mockResolvedValue({
      isDirectory: () => true
    } as unknown as Stats | BigIntStats);

    vi.mocked(fs.readdir).mockResolvedValueOnce(foundFiles);

    const actual = await getModFiles(configLocation, configuration);

    expect(actual).toEqual([]);
  });

  it<LocalTestContext>('applies the ignore filter', async ({ configLocation, rootDir, configuration }) => {
    const foundFiles = chance.n(
      () => {
        return path.resolve(rootDir, 'mods', chance.word());
      },
      chance.integer({ min: 2, max: 20 })
    );
    const notIgnoredFiles = chance.n(chance.word, chance.integer({ min: 2, max: 20 }));
    vi.mocked(fs.readdir).mockResolvedValueOnce(foundFiles);
    vi.mocked(notIgnored).mockResolvedValueOnce(notIgnoredFiles);

    const actual = await getModFiles(configLocation, configuration);

    expect(notIgnored).toHaveBeenCalledWith(rootDir, foundFiles);
    expect(actual).toEqual(notIgnoredFiles);
  });
});
