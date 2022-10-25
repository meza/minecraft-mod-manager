import { describe, vi, expect, it } from 'vitest';
import { fileExists } from './config.js';
import { getHash } from './hash.js';
import { chance } from 'jest-chance';
import fs from 'node:fs/promises';

vi.mock('./config.js');
vi.mock('node:fs/promises');

describe('The hash module', () => {
  it('throws an error if the file does not exist', async () => {
    vi.mocked(fileExists).mockResolvedValueOnce(false);
    const randomFile = chance.word();
    await expect(async () => {
      await getHash(randomFile);
    }).rejects.toThrow(new Error(`File (${randomFile}) does not exist, can't determine the hash`));
  });

  it('returns the hash of the file', async () => {
    const randomFile = chance.word();
    vi.mocked(fileExists).mockResolvedValueOnce(true);
    vi.mocked(fs.readFile).mockResolvedValueOnce('this is the file contents');

    await expect(getHash(randomFile)).resolves.toEqual('6ea6ab9b67e8d51b9d3e6dc877521431926b2fa5');

  });
});
