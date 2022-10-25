import { chance } from 'jest-chance';
import { Loader, ModsJson } from '../src/lib/modlist.types.js';
import { GeneratorResult } from './test.types.js';

export const generateModsJson = (overrides?: Partial<ModsJson>): GeneratorResult<ModsJson> => {

  const allowedReleasesNumber = chance.integer({ min: 1, max: 3 });

  const modsFolder = chance.word();
  const allowedReleases = chance.pickset(['release', 'beta', 'alpha'], allowedReleasesNumber);
  const gameVersion = chance.word();
  const loader = chance.pickone(Object.values(Loader)) as Loader;
  const allowVersionFallback = chance.bool();

  const generated: ModsJson = {

    modsFolder: modsFolder,
    defaultAllowedReleaseTypes: allowedReleases,
    gameVersion: gameVersion,
    loader: loader,
    allowVersionFallback: allowVersionFallback,
    mods: [],
    ...overrides
  };

  const expected: ModsJson = {
    modsFolder: modsFolder,
    defaultAllowedReleaseTypes: allowedReleases,
    gameVersion: gameVersion,
    loader: loader,
    allowVersionFallback: allowVersionFallback,
    mods: [],
    ...overrides
  };

  return {
    generated: generated,
    expected: expected
  };

};
