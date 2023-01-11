import { beforeEach, describe, expect, it, vi } from 'vitest';
import { Loader, Platform, ReleaseType } from '../lib/modlist.types.js';
import { chance } from 'jest-chance';
import { fetchModDetails } from './index.js';
import { UnknownPlatformException } from '../errors/UnknownPlatformException.js';
import { generateRemoteModDetails } from '../../test/generateRemoteDetails.js';
import { Curseforge } from './curseforge/index.js';
import { Modrinth } from './modrinth/index.js';

vi.mock('./modrinth/index.js');
vi.mock('./curseforge/index.js');

export interface RepositoryTestContext {
  platform: Platform,
  id: string,
  allowedReleaseTypes: ReleaseType[],
  gameVersion: string,
  loader: Loader,
  allowFallback: boolean
}

const curseforge = new Curseforge();
const modrinth = new Modrinth();

describe('The repository facade', () => {
  beforeEach<RepositoryTestContext>((context) => {
    vi.resetAllMocks();

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

  it<RepositoryTestContext>('throws an exception when an unknown platform is used', async (context) => {
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
    [Platform.CURSEFORGE, curseforge.fetchMod],
    [Platform.MODRINTH, modrinth.fetchMod]
  ])('when the platform is %s', (platform: Platform, implementation) => {
    it<RepositoryTestContext>('calls the correct implementation', async (context) => {
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
