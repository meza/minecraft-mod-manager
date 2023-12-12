import { chance } from 'jest-chance';
import { Mod } from '../src/lib/modlist.types.js';
import { GeneratorResult } from './test.types.js';
import { generateRandomPlatform } from './generateRandomPlatform.js';

export const generateModConfig = (overrides?: Partial<Mod>): GeneratorResult<Mod> => {

  const type = generateRandomPlatform();
  const id = chance.hash({ length: 25 });
  const allowedReleaseTypes = chance.pickset(['release', 'beta', 'alpha'], chance.integer({ min: 1, max: 3 }));
  const name = chance.word();
  const allowVersionFallback = chance.bool();

  const generated: Mod = {
    type: type,
    id: id,
    allowedReleaseTypes: allowedReleaseTypes,
    name: name,
    allowVersionFallback: allowVersionFallback,
    ...overrides
  };

  const expected: Mod = {
    ...overrides,
    type: type,
    id: id,
    allowedReleaseTypes: allowedReleaseTypes,
    allowVersionFallback: allowVersionFallback,
    name: name
  };

  return {
    generated: generated,
    expected: expected
  };

};
