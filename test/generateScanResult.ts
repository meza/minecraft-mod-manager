import { chance } from 'jest-chance';
import { ScanResults } from '../src/actions/scan.js';
import { Platform, RemoteModDetails } from '../src/lib/modlist.types.js';
import { PlatformLookupResult } from '../src/repositories/index.js';
import { generatePlatformLookupResult } from './generatePlatformLookupResult.js';
import { generateRandomPlatform } from './generateRandomPlatform.js';
import { generateRemoteModDetails } from './generateRemoteDetails.js';
import { GeneratorResult } from './test.types.js';

export interface ScanResultGeneratorOverrides {
  name?: string | undefined;
  modId?: string | undefined;
  platform?: Platform | undefined;
}

export const generateScanResult = (overrides?: ScanResultGeneratorOverrides): GeneratorResult<ScanResults> => {
  const name = overrides?.name || chance.word();
  const platform = overrides?.platform || generateRandomPlatform();
  const modId = overrides?.modId || chance.word();

  const resolvedDetails: RemoteModDetails = generateRemoteModDetails({
    name: name
  } as Partial<RemoteModDetails>).generated;

  const localDetails: PlatformLookupResult = generatePlatformLookupResult(
    {
      platform: platform,
      modId: modId
    } as Partial<PlatformLookupResult>,
    { name: name } as Partial<RemoteModDetails>
  ).generated;

  return {
    generated: {
      allRemoteDetails: [resolvedDetails],
      localDetails: [localDetails],
      preferredDetails: resolvedDetails
    },
    expected: {
      allRemoteDetails: [resolvedDetails],
      localDetails: [localDetails],
      preferredDetails: resolvedDetails
    }
  };
};
