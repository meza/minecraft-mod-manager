import { chance } from 'jest-chance';
import { InitializeOptions } from '../src/interactions/initializeConfig.js';
import { Loader, Platform } from '../src/lib/modlist.types.js';
import { GeneratorResult } from './test.types.js';

export const generateInitializeOptions = (
  overrides?: Partial<InitializeOptions>
): GeneratorResult<InitializeOptions> => {
  const loader = chance.pickone(Object.values(Loader));
  const gameVersion = chance.word();
  const defaultAllowedReleaseTypes = chance.pickset(
    Object.values(Platform),
    chance.integer({
      min: 1,
      max: Object.keys(Platform).length
    })
  );
  const modsFolder = chance.word();
  const config = chance.word() + '.json';

  const generated = {
    loader: loader,
    gameVersion: gameVersion,
    defaultAllowedReleaseTypes: defaultAllowedReleaseTypes.join(','),
    modsFolder: modsFolder,
    config: config,
    ...overrides
  };

  const expected = {
    loader: loader,
    gameVersion: gameVersion,
    defaultAllowedReleaseTypes: defaultAllowedReleaseTypes,
    modsFolder: modsFolder,
    config: config,
    ...overrides
  };

  return {
    generated: generated,
    expected: expected
  };
};
