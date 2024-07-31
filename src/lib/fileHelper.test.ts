import { beforeEach, describe, vi, it, expect } from 'vitest';
import * as fs from 'fs/promises';
import { generateModsJson } from '../../test/modlistGenerator.js';
import { notIgnored } from './ignore.js';
import { chance } from 'jest-chance';
import path from 'path';
import { getModFiles } from './fileHelper.js';
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

  it<LocalTestContext>('applies the ignore filter', async ({ configLocation, rootDir, configuration }) => {
    const foundFiles = chance.n(() => {
      return path.resolve(rootDir, 'mods', chance.word());
    }, chance.integer({ min: 2, max: 20 }));
    const notIgnoredFiles = chance.n(chance.word, chance.integer({ min: 2, max: 20 }));
    vi.mocked(fs.readdir).mockResolvedValueOnce(foundFiles);
    vi.mocked(notIgnored).mockResolvedValueOnce(notIgnoredFiles);

    const actual = await getModFiles(configLocation, configuration);

    expect(notIgnored).toHaveBeenCalledWith(rootDir, foundFiles);
    expect(actual).toEqual(notIgnoredFiles);
  });
});
