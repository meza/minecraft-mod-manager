import { chance } from 'jest-chance';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { generatePlatformLookupResult } from '../../test/generatePlatformLookupResult.js';
import { generateRandomPlatform } from '../../test/generateRandomPlatform.js';
import { generateRemoteModDetails } from '../../test/generateRemoteDetails.js';
import { UnknownPlatformException } from '../errors/UnknownPlatformException.js';
import { Loader, Platform, ReleaseType } from '../lib/modlist.types.js';
import { Curseforge } from './curseforge/index.js';
import { LookupInput, PlatformLookupResult, fetchModDetails, lookup } from './index.js';
import { Modrinth } from './modrinth/index.js';

vi.mock('./modrinth/index.js', () => {
  const Modrinth = vi.fn();
  Modrinth.prototype.lookup = vi.fn();
  Modrinth.prototype.fetchMod = vi.fn();
  return { Modrinth: Modrinth };
});
vi.mock('./curseforge/index.js', () => {
  const Curseforge = vi.fn();
  Curseforge.prototype.lookup = vi.fn();
  Curseforge.prototype.fetchMod = vi.fn();
  return { Curseforge: Curseforge };
});

export interface RepositoryTestContext {
  platform: Platform;
  id: string;
  allowedReleaseTypes: ReleaseType[];
  gameVersion: string;
  loader: Loader;
  allowFallback: boolean;
  version?: string;
}

const curseforge = new Curseforge();
const modrinth = new Modrinth();

const getLookupImplementation = (platform: Platform) => {
  if (platform === Platform.MODRINTH) {
    return vi.mocked(modrinth.lookup);
  }

  return vi.mocked(curseforge.lookup);
};

describe('The repository facade', () => {
  beforeEach<RepositoryTestContext>((context) => {
    vi.resetAllMocks();

    context.platform = generateRandomPlatform();
    context.id = chance.word();
    context.allowedReleaseTypes = chance.pickset(
      Object.values(ReleaseType),
      chance.integer({
        min: 1,
        max: Object.keys(ReleaseType).length
      })
    );
    context.gameVersion = chance.pickone(['1.16.5', '1.17.1', '1.18.1', '1.18.2', '1.19']);
    context.loader = chance.pickone(Object.values(Loader));
    context.allowFallback = chance.bool();
    if (chance.bool()) {
      context.version = chance.word();
    }
  });

  describe('when fetching mod details', () => {
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
    ])('and the platform is %s', (platform: Platform, implementation) => {
      it<RepositoryTestContext>(`calls the correct implementation for ${platform}`, async (context) => {
        const randomResult = generateRemoteModDetails().generated;
        vi.mocked(implementation).mockResolvedValueOnce(randomResult);
        await fetchModDetails(
          platform,
          context.id,
          context.allowedReleaseTypes,
          context.gameVersion,
          context.loader,
          context.allowFallback,
          context.version
        );

        expect(implementation).toBeCalledWith(
          context.id,
          context.allowedReleaseTypes,
          context.gameVersion,
          context.loader,
          context.allowFallback,
          context.version
        );
      });
    });
  });

  describe('when looking up mods', () => {
    it('can handle an empty input', async () => {
      const actual = await lookup([]);
      expect(actual).toEqual([]);
      expect(vi.mocked(curseforge.lookup)).not.toHaveBeenCalled();
      expect(vi.mocked(modrinth.lookup)).not.toHaveBeenCalled();
    });

    it.each([
      [Platform.CURSEFORGE, curseforge.lookup],
      [Platform.MODRINTH, modrinth.lookup]
    ])('finds the appropriate repository for %s', async (platform: Platform, implementation) => {
      const randomHashes = [chance.n(chance.hash, chance.integer({ min: 1, max: 5 }))];
      const input: LookupInput = {
        platform: platform,
        hash: randomHashes
      };

      vi.mocked(implementation).mockResolvedValueOnce([]); //no processing done

      const actual = await lookup([input]);

      expect(vi.mocked(implementation)).toBeCalledWith(randomHashes);
      expect(actual).toEqual([]);
    });

    it('ignores incorrect platforms', async () => {
      const input1: LookupInput = {
        platform: chance.word(),
        hash: [chance.n(chance.hash, chance.integer({ min: 1, max: 5 }))]
      };
      const input2: LookupInput = {
        platform: chance.word(),
        hash: [chance.n(chance.hash, chance.integer({ min: 1, max: 5 }))]
      };

      const actual = await lookup([input1, input2]);

      expect(actual).toEqual([]);
      expect(vi.mocked(curseforge.lookup)).not.toHaveBeenCalled();
      expect(vi.mocked(modrinth.lookup)).not.toHaveBeenCalled();
    });

    it<RepositoryTestContext>('processes a single result properly', async ({ platform }) => {
      const randomHash = chance.hash();
      const input: LookupInput = {
        platform: platform,
        hash: [randomHash]
      };
      const implementation = getLookupImplementation(platform);
      const modId = chance.word();
      const remoteModDetails = generateRemoteModDetails({
        hash: randomHash
      }).generated;
      const lookupResult = generatePlatformLookupResult({
        mod: remoteModDetails,
        modId: modId,
        platform: platform
      }).generated;
      const result: PlatformLookupResult[] = [lookupResult];

      implementation.mockResolvedValueOnce(result);

      const actual = await lookup([input]);

      expect(actual.length).toEqual(1);
      expect(actual[0].sha1Hash).toEqual(randomHash);
      expect(actual[0].hits.length).toEqual(1);
      expect(actual[0].hits[0]).toBe(lookupResult);
    });

    describe('and there are multiple results to the same hash', () => {
      it<RepositoryTestContext>('processes multiple results properly', async () => {
        const randomHash = chance.hash();
        const input: LookupInput[] = [
          {
            platform: Platform.CURSEFORGE,
            hash: [randomHash]
          },
          {
            platform: Platform.MODRINTH,
            hash: [randomHash]
          }
        ];
        const remoteModDetails1 = generateRemoteModDetails({
          hash: randomHash
        }).generated;
        const lookupResult1 = generatePlatformLookupResult({
          mod: remoteModDetails1,
          modId: chance.word(),
          platform: Platform.CURSEFORGE
        }).generated;

        const remoteModDetails2 = generateRemoteModDetails({
          hash: randomHash
        }).generated;
        const lookupResult2 = generatePlatformLookupResult({
          mod: remoteModDetails2,
          modId: chance.word(),
          platform: Platform.MODRINTH
        }).generated;

        vi.mocked(curseforge.lookup).mockResolvedValueOnce([lookupResult1]);
        vi.mocked(modrinth.lookup).mockResolvedValueOnce([lookupResult2]);

        const actual = await lookup(input);

        expect(actual.length).toEqual(1);
        expect(actual[0].sha1Hash).toEqual(randomHash);
        expect(actual[0].hits.length).toEqual(2);
        expect(actual[0].hits).toContainEqual(lookupResult1);
        expect(actual[0].hits).toContainEqual(lookupResult2);
      });

      it<RepositoryTestContext>('ignores results wihtout a download url', async () => {
        const randomHash = chance.hash();
        const input: LookupInput[] = [
          {
            platform: Platform.CURSEFORGE,
            hash: [randomHash]
          },
          {
            platform: Platform.MODRINTH,
            hash: [randomHash]
          }
        ];
        const remoteModDetails1 = generateRemoteModDetails({
          hash: randomHash
        }).generated;

        // @ts-ignore
        remoteModDetails1.downloadUrl = null;

        const lookupResult1 = generatePlatformLookupResult({
          mod: remoteModDetails1,
          modId: chance.word(),
          platform: Platform.CURSEFORGE
        }).generated;

        const remoteModDetails2 = generateRemoteModDetails({
          hash: randomHash
        }).generated;
        const lookupResult2 = generatePlatformLookupResult({
          mod: remoteModDetails2,
          modId: chance.word(),
          platform: Platform.MODRINTH
        }).generated;

        vi.mocked(curseforge.lookup).mockResolvedValueOnce([lookupResult1]);
        vi.mocked(modrinth.lookup).mockResolvedValueOnce([lookupResult2]);

        const actual = await lookup(input);

        expect(actual.length).toEqual(1);
        expect(actual[0].sha1Hash).toEqual(randomHash);
        expect(actual[0].hits.length).toEqual(1);
        expect(actual[0].hits).toContainEqual(lookupResult2);
      });
    });

    describe('when a lookup fails', () => {
      it('ignores the failure', async () => {
        const randomHash = chance.hash();

        const lookupResult2 = generatePlatformLookupResult(
          {
            platform: Platform.CURSEFORGE
          },
          {
            hash: randomHash
          }
        ).generated;

        vi.mocked(modrinth.lookup).mockRejectedValueOnce(new Error('test error'));
        vi.mocked(curseforge.lookup).mockResolvedValueOnce([lookupResult2]);

        const actual = await lookup([
          {
            platform: Platform.MODRINTH,
            hash: [randomHash]
          },
          {
            platform: Platform.CURSEFORGE,
            hash: [randomHash]
          }
        ]);

        expect(actual.length).toEqual(1);
        expect(actual[0].sha1Hash).toEqual(randomHash);
        expect(actual[0].hits.length).toEqual(1);
        expect(actual[0].hits).toContainEqual(lookupResult2);
      });
    });
  });
});
