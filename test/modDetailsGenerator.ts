import { ModDetails } from '../src/lib/modlist.types.js';
import { chance } from 'jest-chance';
import { GeneratorResult } from './test.types.js';

export const generateModDetails = (overrides?: Partial<ModDetails>): GeneratorResult<ModDetails> => {
  const generated: ModDetails = {
    name: chance.word(),
    fileName: chance.word(),
    downloadUrl: chance.url(),
    releaseDate: chance.date({ string: true }),
    hash: chance.hash(),
    ...overrides
  };

  return {
    generated: generated,
    expected: generated
  };

};
