import path from 'path';
import * as fs from 'fs/promises';
import { glob } from 'glob';
import { chance } from 'jest-chance';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { fileExists } from './config.js';
import { notIgnored } from './ignore.js';

interface LocalTestContext {
  configLocation: string;
}

vi.mock('fs/promises');
vi.mock('glob');
vi.mock('./config.js');

describe('The ignore module', () => {
  beforeEach<LocalTestContext>((context) => {
    context.configLocation = path.resolve('/', 'modlist.json');
    vi.resetAllMocks();
  });

  describe("when the ignore file doesn't exist", () => {
    it<LocalTestContext>('returns the full input', async ({ configLocation }) => {
      vi.mocked(fileExists).mockResolvedValueOnce(false);

      const files: string[] = chance.n(() => chance.word() + '.jar', chance.integer({ min: 1, max: 10 }));
      vi.mocked(glob.sync).mockReturnValue(files.map((file) => path.resolve('/', file)));

      const result = await notIgnored(configLocation, files);
      expect(result).toEqual(files);
    });
  });

  it<LocalTestContext>('passes on all patterns to the glob', async ({ configLocation }) => {
    const firstPattern = chance.word();
    const secondPattern = chance.word();
    const thirdPattern = chance.word();

    const expectedDirectory = configLocation;

    vi.mocked(fileExists).mockResolvedValueOnce(true);
    vi.mocked(fs.readFile).mockResolvedValueOnce(`${firstPattern}\n${secondPattern}\n${thirdPattern}\n`);
    vi.mocked(glob.sync).mockReturnValue([]);
    await notIgnored(configLocation, []);

    expect(vi.mocked(glob.sync)).toHaveBeenCalledTimes(4);
    expect(vi.mocked(glob.sync)).toHaveBeenNthCalledWith(1, '**/*.disabled', {
      cwd: expectedDirectory,
      absolute: true
    });
    expect(vi.mocked(glob.sync)).toHaveBeenNthCalledWith(2, firstPattern, {
      cwd: expectedDirectory,
      absolute: true
    });
    expect(vi.mocked(glob.sync)).toHaveBeenNthCalledWith(3, secondPattern, {
      cwd: expectedDirectory,
      absolute: true
    });
    expect(vi.mocked(glob.sync)).toHaveBeenNthCalledWith(4, thirdPattern, {
      cwd: expectedDirectory,
      absolute: true
    });
  });

  it<LocalTestContext>('can return the not ignored files', async () => {
    vi.mocked(fileExists).mockResolvedValueOnce(true);
    vi.mocked(fs.readFile).mockResolvedValueOnce('doesntmatter');
    vi.mocked(glob.sync).mockReturnValue(['/mods/a.jar', '/mods/c.jar']);

    const actual = await notIgnored('/', ['/mods/a.jar', '/mods/b.jar', '/mods/c.jar']);

    expect(actual).toEqual(['/mods/b.jar']);
  });
});
