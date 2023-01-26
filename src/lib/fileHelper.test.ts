import { beforeEach, describe, vi, it, expect } from 'vitest';
import * as fs from 'fs/promises';
import { notIgnored } from './ignore.js';
import { chance } from 'jest-chance';
import path from 'path';
import { getModFiles } from './fileHelper.js';

vi.mock('./ignore.js');
vi.mock('fs/promises');

interface LocalTestContext {
  configLocation: string;
  rootDir: string;
}

describe('The file helper module', () => {
  beforeEach<LocalTestContext>((context) => {
    context.rootDir = path.resolve('/', chance.word());
    context.configLocation = path.resolve(context.rootDir, chance.word());
  });

  it<LocalTestContext>('can handle a relative mods folder', async ({ configLocation, rootDir }) => {
    vi.mocked(fs.readdir).mockResolvedValueOnce([]);

    await getModFiles(configLocation, 'mods');

    expect(fs.readdir).toHaveBeenCalledWith(path.resolve(rootDir, 'mods'));

  });

  it<LocalTestContext>('can handle an absolute mods folder', async ({ configLocation }) => {
    vi.mocked(fs.readdir).mockResolvedValueOnce([]);

    await getModFiles(configLocation, '/absolute/mods');

    expect(fs.readdir).toHaveBeenCalledWith('/absolute/mods');

  });

  it<LocalTestContext>('returns properly when no files are found', async ({ configLocation }) => {
    vi.mocked(fs.readdir).mockResolvedValueOnce([]);

    const actual = await getModFiles(configLocation, chance.word());

    expect(actual).toEqual([]);

  });

  it<LocalTestContext>('applies the ignore filter', async ({ configLocation, rootDir }) => {
    const foundFiles = chance.n(() => {
      return path.resolve(rootDir, 'mods', chance.word());
    }, chance.integer({ min: 2, max: 20 }));
    const notIgnoredFiles = chance.n(chance.word, chance.integer({ min: 2, max: 20 }));
    vi.mocked(fs.readdir).mockResolvedValueOnce(foundFiles);
    vi.mocked(notIgnored).mockResolvedValueOnce(notIgnoredFiles);

    const actual = await getModFiles(configLocation, 'mods');

    expect(notIgnored).toHaveBeenCalledWith(rootDir, foundFiles);
    expect(actual).toEqual(notIgnoredFiles);
  });
});
