import { describe, it, vi, expect, afterEach, beforeEach } from 'vitest';
import { Loader, Platform, ReleaseType } from '../lib/modlist.types.js';
import { getMod as cfMod } from './curseforge/index.js';
import { getMod as mMod } from './modrinth/index.js';
import { chance } from 'jest-chance';
import { fetchModDetails } from './index.js';
import { UnknownPlatformException } from '../errors/UnknownPlatformException.js';
import { generateRemoteModDetails } from '../../test/modDetailsGenerator.js';

vi.mock('./modrinth/index.js');
vi.mock('./curseforge/index.js');

interface LocalTestContext {
  platform: Platform,
  id: string,
  allowedReleaseTypes: ReleaseType[],
  gameVersion: string,
  loader: Loader,
  allowFallback: boolean
}

describe('The repository facade', () => {
  beforeEach<LocalTestContext>((context) => {
    context.platform = chance.pickone(Object.values(Platform));
    context.id = chance.word();
    context.allowedReleaseTypes = chance.pickset(Object.values(ReleaseType), chance.integer({
      min: 1,
      max: Object.keys(ReleaseType).length
    }));
    context.gameVersion = chance.pickone(['1.16.5', '1.17.1', '1.18.1', '1.18.2', '1.19']);
    context.loader = chance.pickone(Object.values(Loader));
    context.allowFallback = chance.bool();
  });

  afterEach(() => {
    vi.resetAllMocks();
  });

  it<LocalTestContext>('throws an exception when an unknown platform is used', async (context) => {
    const invalidPlatform = chance.word();
    await expect(async () => {
      await fetchModDetails(
        invalidPlatform as Platform,
        context.id,
        context.allowedReleaseTypes,
        context.gameVersion,
        context.loader,
        context.allowFallback
      );
    }).rejects.toThrow(new UnknownPlatformException(invalidPlatform));
  });

  describe.each([
    [Platform.CURSEFORGE, cfMod],
    [Platform.MODRINTH, mMod]
  ])('when the platform is %s', (platform: Platform, implementation) => {
    it<LocalTestContext>('calls the correct implementation', async (context) => {
      const randomResult = generateRemoteModDetails().generated;
      vi.mocked(implementation).mockResolvedValueOnce(randomResult);
      await fetchModDetails(
        platform,
        context.id,
        context.allowedReleaseTypes,
        context.gameVersion,
        context.loader,
        context.allowFallback
      );

      expect(implementation).toBeCalledWith(
        context.id,
        context.allowedReleaseTypes,
        context.gameVersion,
        context.loader,
        context.allowFallback
      );
    });
  });
});
