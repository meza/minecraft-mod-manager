import { chance } from 'jest-chance';
import { Mod } from '../src/lib/modlist.types.js';
import { GeneratorResult } from './test.types.js';
import { generateRandomPlatform } from './generateRandomPlatform.js';

export const generateModConfig = (overrides?: Partial<Mod>): GeneratorResult<Mod> => {

  const type = generateRandomPlatform();
  const id = chance.word();
  const allowedReleaseTypes = chance.pickset(['release', 'beta', 'alpha'], chance.integer({ min: 1, max: 3 }));
  const name = chance.word();

  const generated: Mod = {
    type: type,
    id: id,
    allowedReleaseTypes: allowedReleaseTypes,
    name: name,
    ...overrides
  };

  const expected: Mod = {
    ...overrides,
    type: type,
    id: id,
    allowedReleaseTypes: allowedReleaseTypes,
    name: name
  };

  return {
    generated: generated,
    expected: expected
  };

};
