import { chance } from 'jest-chance';
import { Mod, Platform } from '../src/lib/modlist.types.js';
import { GeneratorResult } from './test.types.js';
import { generateModInstall } from './modInstallGenerator.js';

export const generateModConfig = (overrides?: Partial<Mod>): GeneratorResult<Mod> => {

  const type = chance.pickone(Object.values(Platform));
  const id = chance.word();
  const install = generateModInstall().generated;
  const allowedReleaseTypes = chance.pickset(['release', 'beta', 'alpha'], chance.integer({ min: 1, max: 3 }));
  const name = chance.word();

  const generated: Mod = {
    type: type,
    id: id,
    installed: install,
    allowedReleaseTypes: allowedReleaseTypes,
    name: name,
    ...overrides
  };

  const expected: Mod = {
    ...overrides,
    type: type,
    id: id,
    installed: install,
    allowedReleaseTypes: allowedReleaseTypes,
    name: name
  };

  return {
    generated: generated,
    expected: expected
  };

};
