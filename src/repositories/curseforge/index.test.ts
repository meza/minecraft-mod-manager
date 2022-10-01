import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { chance } from 'jest-chance';
import { CurseforgeModFile, getMod } from './index.js';
import { Loader, Platform, ReleaseType } from '../../lib/modlist.types.js';
import { RepositoryTestContext } from '../index.test.js';
import { CouldNotFindModException } from '../../errors/CouldNotFindModException.js';
import { generateCurseforgeModFile } from '../../../test/generateCurseforgeModFile.js';
import { NoRemoteFileFound } from '../../errors/NoRemoteFileFound.js';

const assumeFailedModFetch = () => {
  vi.stubGlobal('fetch', () => {
    return Promise.resolve({
      status: chance.pickone([401, 404, 500])
    });
  });
};

const assumeSuccessfulModFetch = (modName: string, latestFiles: CurseforgeModFile[]) => {
  vi.stubGlobal('fetch', () => {
    return Promise.resolve({
      status: 200,
      json: () => Promise.resolve({
        data: {
          name: modName,
          latestFiles: latestFiles
        }
      })
    });
  });
};

describe('The Curseforge repository', () => {

  beforeEach<RepositoryTestContext>((context) => {
    context.platform = Platform.CURSEFORGE;
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

  it<RepositoryTestContext>('throws an error when the mod details could not be fetched', async (context) => {
    assumeFailedModFetch();

    await expect(async () => {
      await getMod(
        context.id,
        context.allowedReleaseTypes,
        context.gameVersion,
        context.loader,
        context.allowFallback
      );
    }).rejects.toThrow(new CouldNotFindModException(context.id, context.platform));
  });

  it<RepositoryTestContext>('throws an error when no files match the requested game version', async (context) => {
    const randomName = chance.word();
    const randomFile = generateCurseforgeModFile({
      sortableGameVersions: [{
        gameVersionName: 'nothing',
        gameVersion: 'also nothing'
      }]
    });
    assumeSuccessfulModFetch(randomName, [randomFile.generated]);

    await expect(async () => {
      await getMod(
        context.id,
        context.allowedReleaseTypes,
        context.gameVersion,
        context.loader,
        context.allowFallback
      );
    }).rejects.toThrow(new NoRemoteFileFound(randomName, context.platform));

  });

  it<RepositoryTestContext>('throws an error when no files are available', async (context) => {
    const randomName = chance.word();
    const randomFile = generateCurseforgeModFile({
      isAvailable: false,
      sortableGameVersions: [{
        gameVersionName: context.gameVersion,
        gameVersion: context.gameVersion
      }]
    });
    assumeSuccessfulModFetch(randomName, [randomFile.generated]);

    await expect(async () => {
      await getMod(
        context.id,
        context.allowedReleaseTypes,
        context.gameVersion,
        context.loader,
        context.allowFallback
      );
    }).rejects.toThrow(new NoRemoteFileFound(randomName, context.platform));
  });

  it<RepositoryTestContext>('throws an error when no files match the release type', async (context) => {
    const randomName = chance.word();
    const randomFile = generateCurseforgeModFile({
      isAvailable: true,
      releaseType: 2, // (2 = beta)
      sortableGameVersions: [{
        gameVersionName: context.gameVersion,
        gameVersion: context.gameVersion
      }]
    });
    assumeSuccessfulModFetch(randomName, [randomFile.generated]);

    await expect(async () => {
      await getMod(
        context.id,
        [ReleaseType.RELEASE],
        context.gameVersion,
        context.loader,
        context.allowFallback
      );
    }).rejects.toThrow(new NoRemoteFileFound(randomName, context.platform));

  });

  describe.each([1, 2, 3, 5, 6, 7, 8, 9, 11, 12, 13, 14, 15])('when the file status is %i', (status) => {
    it<RepositoryTestContext>('throws an error', async (context) => {
      const randomName = chance.word();
      const randomFile = generateCurseforgeModFile({
        isAvailable: true,
        fileStatus: status,
        sortableGameVersions: [{
          gameVersionName: context.gameVersion,
          gameVersion: context.gameVersion
        }]
      });
      assumeSuccessfulModFetch(randomName, [randomFile.generated]);

      await expect(async () => {
        await getMod(
          context.id,
          [ReleaseType.RELEASE, ReleaseType.BETA, ReleaseType.ALPHA],
          context.gameVersion,
          context.loader,
          context.allowFallback
        );
      }).rejects.toThrow(new NoRemoteFileFound(randomName, context.platform));
    });
  });
});
