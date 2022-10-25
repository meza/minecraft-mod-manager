import { ModrinthFile } from '../src/repositories/modrinth/index.js';
import { GeneratorResult } from './test.types.js';
import { chance } from 'jest-chance';

export const generateModrinthFile = (overrides?: Partial<ModrinthFile>): GeneratorResult<ModrinthFile> => {

  const hashes = {
    sha1: chance.hash({ length: 40, casing: 'upper' }),
    sha512: chance.hash({ length: 128, casing: 'upper' })
  };
  const url = chance.url();
  const fileName = chance.word();

  const generated: ModrinthFile = {
    hashes: hashes,
    url: url,
    filename: fileName,
    ...overrides
  };

  const expected: ModrinthFile = {
    hashes: hashes,
    url: url,
    filename: fileName,
    ...overrides
  };

  return {
    generated: generated,
    expected: expected
  };
};
