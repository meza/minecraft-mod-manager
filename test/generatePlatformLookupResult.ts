import { chance } from 'jest-chance';
import { RemoteModDetails } from '../src/lib/modlist.types.js';
import { PlatformLookupResult } from '../src/repositories/index.js';
import { generateRandomPlatform } from './generateRandomPlatform.js';
import { generateRemoteModDetails } from './generateRemoteDetails.js';
import { GeneratorResult } from './test.types.js';

export const generatePlatformLookupResult = (
  overrides?: Partial<PlatformLookupResult>,
  fileOverrides?: Partial<RemoteModDetails>
): GeneratorResult<PlatformLookupResult> => {
  const platform = overrides?.platform || generateRandomPlatform();
  const modId = overrides?.modId || chance.word();
  const mod = overrides?.mod || generateRemoteModDetails(fileOverrides).generated;
  return {
    generated: {
      modId: modId,
      platform: platform,
      mod: mod
    },
    expected: {
      modId: modId,
      platform: platform,
      mod: mod
    }
  };
};
